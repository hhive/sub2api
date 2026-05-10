package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	imagePlaygroundAdapterContentType     = "application/json"
	imagePlaygroundAdapterMaxErrorMessage = 640
)

type openAIImagesTaskAdapter struct {
	handler              *OpenAIGatewayHandler
	loadAPIKey           func(context.Context, int64) (*service.APIKey, error)
	loadSubscription     func(context.Context, int64, int64) (*service.UserSubscription, error)
	validateSubscription func(*service.UserSubscription, *service.Group) (bool, error)
	maintainSubscription func(*service.UserSubscription)
	touchAPIKey          func(context.Context, int64) error
	invokeImages         func(*gin.Context)
}

type imagePlaygroundHandlerResult struct {
	status  int
	headers http.Header
	body    []byte
}

type ImagePlaygroundSubscriptionLoader interface {
	GetActiveSubscription(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error)
}

type ImagePlaygroundSubscriptionAuthorizer interface {
	ImagePlaygroundSubscriptionLoader
	ValidateAndCheckLimits(sub *service.UserSubscription, group *service.Group) (bool, error)
	DoWindowMaintenance(sub *service.UserSubscription)
}

func NewImagePlaygroundOpenAIImagesTaskExecutor(handler *OpenAIGatewayHandler, subscriptionAuthorizer ImagePlaygroundSubscriptionAuthorizer) service.ImagePlaygroundTaskExecutor {
	adapter := &openAIImagesTaskAdapter{handler: handler}
	if subscriptionAuthorizer != nil {
		adapter.loadSubscription = subscriptionAuthorizer.GetActiveSubscription
		adapter.validateSubscription = subscriptionAuthorizer.ValidateAndCheckLimits
		adapter.maintainSubscription = subscriptionAuthorizer.DoWindowMaintenance
	}
	if handler != nil && handler.apiKeyService != nil {
		adapter.touchAPIKey = handler.apiKeyService.TouchLastUsed
	}
	return adapter
}

func (a *openAIImagesTaskAdapter) ExecuteImagePlaygroundTask(ctx context.Context, task service.ImagePlaygroundTask) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	apiKey, err := a.loadTaskAPIKey(ctx, task.APIKeyID)
	if err != nil {
		return nil, service.ImagePlaygroundTaskExecutionError{Code: "api_key_load_failed", Message: sanitizeImagePlaygroundErrorMessage(err.Error())}
	}
	if err := validateImagePlaygroundTaskAPIKey(task, apiKey); err != nil {
		return nil, err
	}
	subscription, err := a.loadActiveSubscription(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	a.touchTaskAPIKey(ctx, apiKey.ID)

	result, err := a.invoke(ctx, task, apiKey, subscription)
	if err != nil {
		return nil, err
	}
	if result.status < 200 || result.status >= 300 {
		return nil, service.ImagePlaygroundTaskExecutionError{
			Code:    fmt.Sprintf("handler_status_%d", result.status),
			Message: buildImagePlaygroundHandlerErrorMessage(result),
		}
	}
	if !imagePlaygroundResponseContainsImage(result.body) {
		return nil, service.ImagePlaygroundTaskExecutionError{
			Code:    "invalid_image_result",
			Message: "image handler returned 2xx without an image result",
		}
	}
	return append([]byte(nil), result.body...), nil
}

func (a *openAIImagesTaskAdapter) loadTaskAPIKey(ctx context.Context, apiKeyID int64) (*service.APIKey, error) {
	if a != nil && a.loadAPIKey != nil {
		return a.loadAPIKey(ctx, apiKeyID)
	}
	if a == nil || a.handler == nil || a.handler.apiKeyService == nil {
		return nil, fmt.Errorf("api key service is not configured")
	}
	return a.handler.apiKeyService.GetByID(ctx, apiKeyID)
}

func (a *openAIImagesTaskAdapter) loadActiveSubscription(ctx context.Context, apiKey *service.APIKey) (*service.UserSubscription, error) {
	if apiKey == nil || apiKey.User == nil || apiKey.Group == nil || !apiKey.Group.IsSubscriptionType() {
		return nil, nil
	}
	userID := apiKey.User.ID
	groupID := apiKey.Group.ID
	if a != nil && a.loadSubscription != nil {
		sub, err := a.loadSubscription(ctx, userID, groupID)
		if err != nil {
			code := "subscription_load_failed"
			message := "failed to load active subscription"
			if errors.Is(err, service.ErrSubscriptionNotFound) {
				code = "subscription_not_found"
				message = "no active subscription found for this group"
			}
			return nil, service.ImagePlaygroundTaskExecutionError{
				Code:    code,
				Message: sanitizeImagePlaygroundErrorMessage(message + ": " + err.Error()),
			}
		}
		if err := a.authorizeActiveSubscription(apiKey, sub); err != nil {
			return nil, err
		}
		return sub, nil
	}
	return nil, service.ImagePlaygroundTaskExecutionError{
		Code:    "subscription_loader_missing",
		Message: "subscription loader is required for subscription image playground tasks",
	}
}

func (a *openAIImagesTaskAdapter) authorizeActiveSubscription(apiKey *service.APIKey, sub *service.UserSubscription) error {
	if a == nil || a.validateSubscription == nil {
		return service.ImagePlaygroundTaskExecutionError{
			Code:    "subscription_authorizer_missing",
			Message: "subscription authorizer is required for subscription image playground tasks",
		}
	}
	needsMaintenance, err := a.validateSubscription(sub, apiKey.Group)
	if err != nil {
		code := "subscription_invalid"
		if errors.Is(err, service.ErrDailyLimitExceeded) ||
			errors.Is(err, service.ErrWeeklyLimitExceeded) ||
			errors.Is(err, service.ErrMonthlyLimitExceeded) {
			code = "usage_limit_exceeded"
		}
		return service.ImagePlaygroundTaskExecutionError{
			Code:    code,
			Message: sanitizeImagePlaygroundErrorMessage(err.Error()),
		}
	}
	if needsMaintenance && a.maintainSubscription != nil {
		subCopy := *sub
		a.maintainSubscription(&subCopy)
	}
	return nil
}

func (a *openAIImagesTaskAdapter) touchTaskAPIKey(ctx context.Context, apiKeyID int64) {
	if a != nil && a.touchAPIKey != nil {
		_ = a.touchAPIKey(ctx, apiKeyID)
	}
}

func (a *openAIImagesTaskAdapter) invoke(ctx context.Context, task service.ImagePlaygroundTask, apiKey *service.APIKey, subscription *service.UserSubscription) (imagePlaygroundHandlerResult, error) {
	endpoint := strings.TrimSpace(task.Endpoint)
	switch endpoint {
	case service.ImagePlaygroundEndpointGenerations, service.ImagePlaygroundEndpointEdits:
	default:
		return imagePlaygroundHandlerResult{}, service.ImagePlaygroundTaskExecutionError{
			Code:    "invalid_endpoint",
			Message: "image playground endpoint is invalid",
		}
	}

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(task.RequestJSON))
	req.Header.Set("Content-Type", imagePlaygroundAdapterContentType)
	req.Header.Set("Accept", "application/json")
	if apiKey.Group != nil && service.IsGroupContextValid(apiKey.Group) {
		req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, apiKey.Group))
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	injectImagePlaygroundAuthContext(c, apiKey, subscription)

	invokeImages := a.invokeImages
	if invokeImages == nil {
		if a == nil || a.handler == nil {
			return imagePlaygroundHandlerResult{}, service.ImagePlaygroundTaskExecutionError{
				Code:    "handler_missing",
				Message: "openai images handler is not configured",
			}
		}
		invokeImages = a.handler.Images
	}
	invokeImages(c)

	status := rec.Code
	if status == 0 {
		status = http.StatusOK
	}
	return imagePlaygroundHandlerResult{
		status:  status,
		headers: rec.Result().Header.Clone(),
		body:    append([]byte(nil), rec.Body.Bytes()...),
	}, nil
}

func injectImagePlaygroundAuthContext(c *gin.Context, apiKey *service.APIKey, subscription *service.UserSubscription) {
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
		UserID:      apiKey.User.ID,
		Concurrency: apiKey.User.Concurrency,
	})
	c.Set(string(middleware.ContextKeyUserRole), apiKey.User.Role)
	if subscription != nil {
		c.Set(string(middleware.ContextKeySubscription), subscription)
	}
}

func validateImagePlaygroundTaskAPIKey(task service.ImagePlaygroundTask, apiKey *service.APIKey) error {
	if apiKey == nil {
		return service.ImagePlaygroundTaskExecutionError{Code: "api_key_not_found", Message: "api key not found"}
	}
	if apiKey.User == nil {
		return service.ImagePlaygroundTaskExecutionError{Code: "user_not_found", Message: "user associated with api key not found"}
	}
	if task.UserID > 0 && apiKey.User.ID != task.UserID {
		return service.ImagePlaygroundTaskExecutionError{Code: "api_key_owner_mismatch", Message: "api key does not belong to task user"}
	}
	if !apiKey.IsActive() {
		return service.ImagePlaygroundTaskExecutionError{Code: "api_key_disabled", Message: "api key is not active"}
	}
	if !apiKey.User.IsActive() {
		return service.ImagePlaygroundTaskExecutionError{Code: "user_inactive", Message: "user account is not active"}
	}
	if apiKey.Group == nil {
		return service.ImagePlaygroundTaskExecutionError{Code: "group_not_found", Message: "group associated with api key not found"}
	}
	if apiKey.Group.Platform != service.PlatformOpenAI {
		return service.ImagePlaygroundTaskExecutionError{Code: "unsupported_platform", Message: "images api is not supported for this platform"}
	}
	if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
		return service.ImagePlaygroundTaskExecutionError{Code: "ip_restricted_api_key", Message: "image playground async tasks do not support api keys with ip restrictions"}
	}
	if apiKey.IsExpired() {
		return service.ImagePlaygroundTaskExecutionError{Code: "api_key_expired", Message: "api key is expired"}
	}
	if apiKey.IsQuotaExhausted() {
		return service.ImagePlaygroundTaskExecutionError{Code: "api_key_quota_exhausted", Message: "api key quota is exhausted"}
	}
	return nil
}

func imagePlaygroundResponseContainsImage(body []byte) bool {
	if !gjson.ValidBytes(body) {
		return false
	}
	if strings.TrimSpace(gjson.GetBytes(body, "b64_json").String()) != "" || strings.TrimSpace(gjson.GetBytes(body, "url").String()) != "" {
		return true
	}
	found := false
	gjson.GetBytes(body, "data").ForEach(func(_, item gjson.Result) bool {
		if strings.TrimSpace(item.Get("b64_json").String()) != "" || strings.TrimSpace(item.Get("url").String()) != "" {
			found = true
			return false
		}
		return true
	})
	return found
}

func buildImagePlaygroundHandlerErrorMessage(result imagePlaygroundHandlerResult) string {
	message := extractImagePlaygroundErrorMessage(result.body)
	if message == "" {
		message = string(result.body)
	}
	if message == "" {
		message = http.StatusText(result.status)
	}
	headerSummary := imagePlaygroundSafeHeaderSummary(result.headers)
	if headerSummary != "" {
		message = message + " (" + headerSummary + ")"
	}
	return fmt.Sprintf("image handler returned status %d: %s", result.status, sanitizeImagePlaygroundErrorMessage(message))
}

func extractImagePlaygroundErrorMessage(body []byte) string {
	if !gjson.ValidBytes(body) {
		return ""
	}
	for _, path := range []string{"error.message", "message", "error"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func imagePlaygroundSafeHeaderSummary(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	allowed := map[string]string{}
	for _, key := range []string{"Content-Type", "X-Request-Id", "X-Upstream-Request-ID"} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			allowed[key] = sanitizeImagePlaygroundErrorMessage(value)
		}
	}
	if len(allowed) == 0 {
		return ""
	}
	raw, err := json.Marshal(allowed)
	if err != nil {
		return ""
	}
	return string(raw)
}

var (
	imagePlaygroundAuthorizationPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)[^\s,"'}]+`)
	imagePlaygroundAPIKeyPattern        = regexp.MustCompile(`(?i)(api[-_ ]?key\s*[:=]\s*)[^\s,"'}]+`)
	imagePlaygroundSecretKeyPattern     = regexp.MustCompile(`(?i)sk-[a-z0-9_-]{6,}`)
	imagePlaygroundLongTokenPattern     = regexp.MustCompile(`[A-Za-z0-9+/=_-]{80,}`)
)

func sanitizeImagePlaygroundErrorMessage(message string) string {
	message = strings.TrimSpace(message)
	message = imagePlaygroundAuthorizationPattern.ReplaceAllString(message, "${1}[redacted]")
	message = imagePlaygroundAPIKeyPattern.ReplaceAllString(message, "${1}[redacted]")
	message = imagePlaygroundSecretKeyPattern.ReplaceAllString(message, "[redacted]")
	message = imagePlaygroundLongTokenPattern.ReplaceAllString(message, "[redacted]")
	message = strings.Join(strings.Fields(message), " ")
	if len(message) <= imagePlaygroundAdapterMaxErrorMessage {
		return message
	}
	return strings.TrimSpace(message[:imagePlaygroundAdapterMaxErrorMessage]) + "..."
}

var _ service.ImagePlaygroundTaskExecutor = (*openAIImagesTaskAdapter)(nil)

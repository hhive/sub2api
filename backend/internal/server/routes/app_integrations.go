package routes

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type applicationIntegrationAuth interface {
	Launch(context.Context, string, int64) (*service.UserAppIntegrationLaunch, error)
	Exchange(context.Context, string, string, string) (*service.UserAppIntegrationExchange, error)
	Authorize(context.Context, string, string, string, string) (int64, error)
	Revoke(context.Context, string, string, string) error
}

type applicationIntegrationKeys interface {
	GetByID(context.Context, int64) (*service.APIKey, error)
	List(context.Context, int64, pagination.PaginationParams, service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error)
}

type applicationIntegrationRoutes struct {
	auth    applicationIntegrationAuth
	keys    applicationIntegrationKeys
	gateway http.Handler
}

func RegisterApplicationIntegrationRoutes(v1 *gin.RouterGroup, gateway http.Handler, auth *service.UserAppIntegrationService, keys *service.APIKeyService, jwt middleware.JWTAuthMiddleware, settings *service.SettingService) {
	if auth == nil {
		return
	}
	routes := applicationIntegrationRoutes{auth: auth, keys: keys, gateway: gateway}
	routes.register(v1, jwt, settings)
}

func (routes applicationIntegrationRoutes) register(v1 *gin.RouterGroup, jwt middleware.JWTAuthMiddleware, settings *service.SettingService) {
	app := v1.Group("/app-integrations/:app_id", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
	})
	app.POST("/launch", gin.HandlerFunc(jwt), middleware.BackendModeUserGuard(settings), routes.launch)
	app.POST("/exchange", routes.exchange)
	app.POST("/revoke", routes.revoke)
	app.GET("/keys", integrationDirectoryTimeout, routes.listKeys)
	app.GET("/keys/:key_id/models", integrationDirectoryTimeout, routes.models)
	app.POST("/keys/:key_id/credential", integrationDirectoryTimeout, routes.credential)
	app.POST("/chat/completions", routes.chat)
}

func integrationDirectoryTimeout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

func (a applicationIntegrationRoutes) launch(c *gin.Context) {
	user, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User authentication required")
		return
	}
	result, err := a.auth.Launch(c.Request.Context(), c.Param("app_id"), user.UserID)
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (a applicationIntegrationRoutes) exchange(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var body struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "A launch ticket is required")
		return
	}
	result, err := a.auth.Exchange(c.Request.Context(), c.Param("app_id"), c.GetHeader("X-Sub2API-App-Secret"), body.Ticket)
	if !response.ErrorFrom(c, err) {
		c.JSON(http.StatusOK, result)
	}
}

func integrationGrant(c *gin.Context) string {
	scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func (a applicationIntegrationRoutes) revoke(c *gin.Context) {
	err := a.auth.Revoke(c.Request.Context(), c.Param("app_id"), c.GetHeader("X-Sub2API-App-Secret"), integrationGrant(c))
	if !response.ErrorFrom(c, err) {
		c.Status(http.StatusNoContent)
	}
}

func (a applicationIntegrationRoutes) authorize(c *gin.Context, scope string) (int64, bool) {
	userID, err := a.auth.Authorize(c.Request.Context(), c.Param("app_id"), c.GetHeader("X-Sub2API-App-Secret"), integrationGrant(c), scope)
	return userID, !response.ErrorFrom(c, err)
}

type applicationIntegrationKey struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Mask       string `json:"mask"`
	GroupID    *int64 `json:"group_id"`
	GroupName  string `json:"group_name"`
	Platform   string `json:"platform"`
	Status     string `json:"status"`
	Selectable bool   `json:"selectable"`
}

func describeIntegrationKey(key *service.APIKey) applicationIntegrationKey {
	item := applicationIntegrationKey{ID: key.ID, Name: key.Name, Mask: "****", GroupID: key.GroupID, Status: key.Status}
	if len(key.Key) > 8 {
		item.Mask += key.Key[len(key.Key)-4:]
	}
	if key.Group != nil {
		item.GroupName = key.Group.Name
		item.Platform = key.Group.Platform
	}
	if !key.IsActive() {
		return item
	}
	switch {
	case key.IsExpired():
		item.Status = service.StatusAPIKeyExpired
	case key.IsQuotaExhausted():
		item.Status = service.StatusAPIKeyQuotaExhausted
	case key.Group == nil || !key.Group.IsActive():
		item.Status = "group_unavailable"
	case len(key.IPWhitelist) != 0 || len(key.IPBlacklist) != 0:
		item.Status = "ip_restricted"
	default:
		item.Selectable = true
	}
	return item
}

func (a applicationIntegrationRoutes) listKeys(c *gin.Context) {
	userID, ok := a.authorize(c, service.UserAppScopeKeysRead)
	if !ok {
		return
	}
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 32)
	if err != nil || page < 1 {
		response.BadRequest(c, "Invalid page")
		return
	}
	pageSize, err := strconv.ParseInt(c.DefaultQuery("page_size", "50"), 10, 32)
	if err != nil || pageSize < 1 {
		response.BadRequest(c, "Invalid page_size")
		return
	}
	params := pagination.PaginationParams{Page: int(page), PageSize: min(int(pageSize), 100), SortOrder: pagination.SortOrderDesc}
	keys, result, err := a.keys.List(c.Request.Context(), userID, params, service.APIKeyListFilters{})
	if response.ErrorFrom(c, err) {
		return
	}
	items := make([]applicationIntegrationKey, 0, len(keys))
	for i := range keys {
		items = append(items, describeIntegrationKey(&keys[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total, "page": result.Page, "page_size": result.PageSize})
}

func (a applicationIntegrationRoutes) models(c *gin.Context) {
	a.forward(c, service.UserAppScopeModelsRead, c.Param("key_id"), "/v1/models")
}

func (a applicationIntegrationRoutes) chat(c *gin.Context) {
	a.forward(c, service.UserAppScopeChatInvoke, c.GetHeader("X-Sub2API-Key-ID"), "/v1/chat/completions")
}

func (a applicationIntegrationRoutes) credential(c *gin.Context) {
	key := a.authorizedKey(c, service.UserAppScopeKeysRead, c.Param("key_id"))
	if key == nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": key.Key})
}

func (a applicationIntegrationRoutes) authorizedKey(c *gin.Context, scope, rawID string) *service.APIKey {
	userID, ok := a.authorize(c, scope)
	if !ok {
		return nil
	}
	keyID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || keyID < 1 {
		response.BadRequest(c, "Invalid Key ID")
		return nil
	}
	key, err := a.keys.GetByID(c.Request.Context(), keyID)
	if err != nil && !errors.Is(err, service.ErrAPIKeyNotFound) {
		response.ErrorFrom(c, err)
		return nil
	}
	if key == nil || key.UserID != userID {
		response.Forbidden(c, "Key unavailable")
		return nil
	}
	if item := describeIntegrationKey(key); !item.Selectable {
		slog.InfoContext(c.Request.Context(), "app_integration.key_denied", "key_id", key.ID, "platform", item.Platform, "reason", item.Status, "request_id", c.Writer.Header().Get("X-Request-Id"))
		response.ErrorWithDetails(c, http.StatusForbidden, "Key unavailable", item.Status, nil)
		return nil
	}
	return key
}

func (a applicationIntegrationRoutes) forward(c *gin.Context, scope, rawID, path string) {
	key := a.authorizedKey(c, scope, rawID)
	if key == nil {
		return
	}
	ctx := middleware.WithAppIntegrationKey(c.Request.Context(), key.UserID, key.ID)
	forwarded := integrationGatewayRequest(c.Request.WithContext(ctx), path, key.Key)
	if requestID := c.Writer.Header().Get("X-Request-Id"); requestID != "" {
		forwarded.Header.Set("X-Request-Id", requestID)
	}
	a.gateway.ServeHTTP(c.Writer, forwarded)
}

func integrationGatewayRequest(request *http.Request, path, key string) *http.Request {
	forwarded := request.Clone(request.Context())
	forwarded.URL = &url.URL{Path: path}
	forwarded.RequestURI = path
	forwarded.Header = make(http.Header)
	for _, header := range []string{"Accept", "Content-Type", "X-Request-Id"} {
		if value := request.Header.Get(header); value != "" {
			forwarded.Header.Set(header, value)
		}
	}
	forwarded.Header.Set("Authorization", "Bearer "+key)
	return forwarded
}

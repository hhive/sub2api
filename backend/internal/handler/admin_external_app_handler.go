package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const adminExternalAppSecretHeader = "X-Sub2API-External-App-Secret"

const adminExternalAppSecretValidatedContextKey = "admin_external_app_secret_validated"

type adminExternalAppService interface {
	List() []service.AdminExternalAppMetadata
	CreateLaunch(context.Context, string, int64) (*service.AdminExternalAppLaunchResult, error)
	ValidateExchangeSecret(string, string) error
	Exchange(context.Context, string, string) (*service.AdminExternalAppExchangePayload, error)
}

type AdminExternalAppHandler struct{ service adminExternalAppService }

func NewAdminExternalAppHandler(service adminExternalAppService) *AdminExternalAppHandler {
	return &AdminExternalAppHandler{service: service}
}

func ProvideAdminExternalAppService(service *service.AdminExternalAppService) adminExternalAppService {
	return service
}

func (h *AdminExternalAppHandler) List(c *gin.Context) {
	response.Success(c, h.service.List())
}

func (h *AdminExternalAppHandler) CreateLaunch(c *gin.Context) {
	middleware.SetAuditAction(c, service.AuditActionAdminExternalAppLaunch)
	appID := strings.TrimSpace(c.Param("app_id"))
	middleware.SetAuditExtra(c, map[string]any{"app_id": appID})
	if c.GetString("auth_method") != service.AuditAuthMethodJWT {
		response.Forbidden(c, "JWT administrator session required")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.service.CreateLaunch(c.Request.Context(), appID, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type adminExternalAppExchangeRequest struct {
	Token string `json:"token"`
}

func (h *AdminExternalAppHandler) RequireExchangeSecret(c *gin.Context) {
	appID := strings.TrimSpace(c.Param("app_id"))
	middleware.SetAuditAction(c, service.AuditActionExternalAppExchange)
	middleware.SetAuditExtra(c, map[string]any{"app_id": appID})
	if !service.IsValidAdminExternalAppID(appID) {
		response.ErrorFrom(c, invalidAdminExternalAppSecret())
		c.Abort()
		return
	}
	if err := h.service.ValidateExchangeSecret(appID, c.GetHeader(adminExternalAppSecretHeader)); err != nil {
		response.ErrorFrom(c, err)
		c.Abort()
		return
	}
	c.Set(adminExternalAppSecretValidatedContextKey, true)
	c.Next()
}

func (h *AdminExternalAppHandler) Exchange(c *gin.Context) {
	middleware.SetAuditAction(c, service.AuditActionExternalAppExchange)
	appID := strings.TrimSpace(c.Param("app_id"))
	middleware.SetAuditExtra(c, map[string]any{"app_id": appID})
	if !c.GetBool(adminExternalAppSecretValidatedContextKey) {
		if !service.IsValidAdminExternalAppID(appID) {
			response.ErrorFrom(c, invalidAdminExternalAppSecret())
			return
		}
		if err := h.service.ValidateExchangeSecret(appID, c.GetHeader(adminExternalAppSecretHeader)); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	var request adminExternalAppExchangeRequest
	if err := decodeStrictAdminExternalAppJSON(c, &request); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	request.Token = strings.TrimSpace(request.Token)
	if request.Token == "" {
		response.BadRequest(c, "launch token is required")
		return
	}
	payload, err := h.service.Exchange(c.Request.Context(), appID, request.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware.SetAuditActor(c, payload.UserID, payload.Email)
	response.Success(c, payload)
}

func invalidAdminExternalAppSecret() error {
	return infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_SECRET_INVALID", "external app secret invalid")
}

func decodeStrictAdminExternalAppJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}

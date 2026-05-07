package handler

import (
	"context"
	"crypto/subtle"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const onyxExchangeSecretHeader = "X-Sub2API-Onyx-Secret"

type onyxLaunchService interface {
	CreateLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	CreateImagePlaygroundLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	ConsumeLaunch(ctx context.Context, token string) (*service.OnyxLaunchPayload, error)
}

type OnyxHandler struct {
	launchService  onyxLaunchService
	settingService *service.SettingService
}

func NewOnyxHandler(launchService onyxLaunchService, settingService *service.SettingService) *OnyxHandler {
	return &OnyxHandler{launchService: launchService, settingService: settingService}
}

func (h *OnyxHandler) Launch(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.launchService.CreateLaunch(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OnyxHandler) ImagePlaygroundLaunch(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.launchService.CreateImagePlaygroundLaunch(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type onyxExchangeRequest struct {
	Token string `json:"token"`
}

func (h *OnyxHandler) Exchange(c *gin.Context) {
	if err := h.validateExchangeSecret(c); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req onyxExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		response.BadRequest(c, "launch token is required")
		return
	}

	payload, err := h.launchService.ConsumeLaunch(c.Request.Context(), req.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *OnyxHandler) validateExchangeSecret(c *gin.Context) error {
	if h.settingService == nil {
		return infraerrors.ServiceUnavailable("ONYX_SETTINGS_UNAVAILABLE", "onyx settings unavailable")
	}
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		return err
	}
	expected := strings.TrimSpace(settings.OnyxExchangeSecret)
	if expected == "" {
		return infraerrors.ServiceUnavailable("ONYX_EXCHANGE_SECRET_MISSING", "onyx exchange secret not configured")
	}
	actual := strings.TrimSpace(c.GetHeader(onyxExchangeSecretHeader))
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return infraerrors.Unauthorized("ONYX_EXCHANGE_SECRET_INVALID", "onyx exchange secret invalid")
	}
	return nil
}

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

const lobeHubExchangeSecretHeader = "X-Sub2API-LobeHub-Secret"
const mediaPlaygroundExchangeSecretHeader = "X-Sub2API-Media-Playground-Secret"

type launchService interface {
	CreateMediaPlaygroundLaunch(ctx context.Context, userID int64) (*service.LaunchResult, error)
	CreateLobeHubLaunch(ctx context.Context, userID int64) (*service.LaunchResult, error)
	ConsumeLobeHubLaunch(ctx context.Context, token string) (*service.LobeHubLaunchPayload, error)
	ConsumeMediaPlaygroundLaunch(ctx context.Context, token string) (*service.MediaPlaygroundLaunchPayload, error)
}

type LaunchHandler struct {
	launchService  launchService
	settingService *service.SettingService
}

func NewLaunchHandler(launchService launchService, settingService *service.SettingService) *LaunchHandler {
	return &LaunchHandler{launchService: launchService, settingService: settingService}
}

func ProvideLaunchService(s *service.LaunchService) launchService {
	return s
}

func (h *LaunchHandler) MediaPlaygroundLaunch(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.launchService.CreateMediaPlaygroundLaunch(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *LaunchHandler) LobeHubLaunch(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.launchService.CreateLobeHubLaunch(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type launchExchangeRequest struct {
	Token string `json:"token"`
}

func (h *LaunchHandler) MediaPlaygroundExchange(c *gin.Context) {
	if err := h.validateMediaPlaygroundExchangeSecret(c); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req launchExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		response.BadRequest(c, "launch token is required")
		return
	}

	payload, err := h.launchService.ConsumeMediaPlaygroundLaunch(c.Request.Context(), req.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *LaunchHandler) LobeHubExchange(c *gin.Context) {
	if err := h.validateLobeHubExchangeSecret(c); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req launchExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		response.BadRequest(c, "launch token is required")
		return
	}

	payload, err := h.launchService.ConsumeLobeHubLaunch(c.Request.Context(), req.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *LaunchHandler) validateLobeHubExchangeSecret(c *gin.Context) error {
	if h.settingService == nil {
		return infraerrors.ServiceUnavailable("LOBEHUB_SETTINGS_UNAVAILABLE", "lobehub settings unavailable")
	}
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		return err
	}
	expected := strings.TrimSpace(settings.LobeHubExchangeSecret)
	if expected == "" {
		return infraerrors.ServiceUnavailable("LOBEHUB_EXCHANGE_SECRET_MISSING", "lobehub exchange secret not configured")
	}
	actual := strings.TrimSpace(c.GetHeader(lobeHubExchangeSecretHeader))
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return infraerrors.Unauthorized("LOBEHUB_EXCHANGE_SECRET_INVALID", "lobehub exchange secret invalid")
	}
	return nil
}

func (h *LaunchHandler) validateMediaPlaygroundExchangeSecret(c *gin.Context) error {
	expected := service.MediaPlaygroundExchangeSecret()
	if expected == "" {
		return infraerrors.ServiceUnavailable("MEDIA_PLAYGROUND_EXCHANGE_SECRET_MISSING", "media playground exchange secret not configured")
	}
	actual := strings.TrimSpace(c.GetHeader(mediaPlaygroundExchangeSecretHeader))
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return infraerrors.Unauthorized("MEDIA_PLAYGROUND_EXCHANGE_SECRET_INVALID", "media playground exchange secret invalid")
	}
	return nil
}

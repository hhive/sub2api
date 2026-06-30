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
const lobeHubExchangeSecretHeader = "X-Sub2API-LobeHub-Secret"
const imagePlaygroundExchangeSecretHeader = "X-Sub2API-Image-Playground-Secret"
const videoPlaygroundExchangeSecretHeader = "X-Sub2API-Video-Playground-Secret"

type onyxLaunchService interface {
	CreateLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	CreateImagePlaygroundLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	CreateLobeHubLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	CreateVideoPlaygroundLaunch(ctx context.Context, userID int64) (*service.OnyxLaunchResult, error)
	ConsumeLaunch(ctx context.Context, token string) (*service.OnyxLaunchPayload, error)
	ConsumeImagePlaygroundLaunch(ctx context.Context, token string) (*service.VideoPlaygroundLaunchPayload, error)
	ConsumeVideoPlaygroundLaunch(ctx context.Context, token string) (*service.VideoPlaygroundLaunchPayload, error)
}

type OnyxHandler struct {
	launchService  onyxLaunchService
	settingService *service.SettingService
}

func NewOnyxHandler(launchService onyxLaunchService, settingService *service.SettingService) *OnyxHandler {
	return &OnyxHandler{launchService: launchService, settingService: settingService}
}

func ProvideOnyxLaunchService(s *service.OnyxLaunchService) onyxLaunchService {
	return s
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

func (h *OnyxHandler) LobeHubLaunch(c *gin.Context) {
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

func (h *OnyxHandler) VideoPlaygroundLaunch(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.launchService.CreateVideoPlaygroundLaunch(c.Request.Context(), subject.UserID)
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

func (h *OnyxHandler) ImagePlaygroundExchange(c *gin.Context) {
	if err := h.validateImagePlaygroundExchangeSecret(c); err != nil {
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

	payload, err := h.launchService.ConsumeImagePlaygroundLaunch(c.Request.Context(), req.Token)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, payload)
}

func (h *OnyxHandler) LobeHubExchange(c *gin.Context) {
	if err := h.validateLobeHubExchangeSecret(c); err != nil {
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

func (h *OnyxHandler) VideoPlaygroundExchange(c *gin.Context) {
	if err := h.validateVideoPlaygroundExchangeSecret(c); err != nil {
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

	payload, err := h.launchService.ConsumeVideoPlaygroundLaunch(c.Request.Context(), req.Token)
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

func (h *OnyxHandler) validateLobeHubExchangeSecret(c *gin.Context) error {
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

func (h *OnyxHandler) validateImagePlaygroundExchangeSecret(c *gin.Context) error {
	expected := service.ImagePlaygroundExchangeSecret()
	if expected == "" {
		return infraerrors.ServiceUnavailable("IMAGE_PLAYGROUND_EXCHANGE_SECRET_MISSING", "image playground exchange secret not configured")
	}
	actual := strings.TrimSpace(c.GetHeader(imagePlaygroundExchangeSecretHeader))
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return infraerrors.Unauthorized("IMAGE_PLAYGROUND_EXCHANGE_SECRET_INVALID", "image playground exchange secret invalid")
	}
	return nil
}

func (h *OnyxHandler) validateVideoPlaygroundExchangeSecret(c *gin.Context) error {
	if h.settingService == nil {
		return infraerrors.ServiceUnavailable("VIDEO_PLAYGROUND_SETTINGS_UNAVAILABLE", "video playground settings unavailable")
	}
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		return err
	}
	expected := service.VideoPlaygroundExchangeSecret(settings.VideoPlaygroundExchangeSecret)
	if expected == "" {
		return infraerrors.ServiceUnavailable("VIDEO_PLAYGROUND_EXCHANGE_SECRET_MISSING", "video playground exchange secret not configured")
	}
	actual := strings.TrimSpace(c.GetHeader(videoPlaygroundExchangeSecretHeader))
	if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return infraerrors.Unauthorized("VIDEO_PLAYGROUND_EXCHANGE_SECRET_INVALID", "video playground exchange secret invalid")
	}
	return nil
}

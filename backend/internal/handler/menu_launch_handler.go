package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MenuLaunchHandler struct {
	settingService      *service.SettingService
	chatService         *service.ChatService
	subscriptionService *service.SubscriptionService
}

func NewMenuLaunchHandler(settingService *service.SettingService, chatService *service.ChatService, subscriptionService *service.SubscriptionService) *MenuLaunchHandler {
	return &MenuLaunchHandler{settingService: settingService, chatService: chatService, subscriptionService: subscriptionService}
}

type victoryMenuLaunchRequest struct {
	MenuID string `json:"menu_id"`
}

func (h *MenuLaunchHandler) Victory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req victoryMenuLaunchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	menuID := strings.TrimSpace(req.MenuID)
	if menuID == "" {
		response.BadRequest(c, "menu_id is required")
		return
	}

	item, ok := h.findEnabledVictoryMenuItem(c, menuID)
	if !ok {
		response.NotFound(c, "Victory menu item not found")
		return
	}

	redirectURL := strings.TrimSpace(item.URL)
	if item.CarryAPIKey {
		apiKey, err := h.chatService.SelectDefaultAPIKey(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if err := h.validateLaunchAPIKey(c, apiKey); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		withKey, err := appendAPIKeyQueryParam(redirectURL, apiKey.Key)
		if err != nil {
			response.BadRequest(c, "Victory menu item URL is invalid")
			return
		}
		redirectURL = withKey
	}

	response.Success(c, gin.H{"redirect_url": redirectURL})
}

func (h *MenuLaunchHandler) validateLaunchAPIKey(c *gin.Context, apiKey *service.APIKey) error {
	if apiKey == nil || apiKey.User == nil || apiKey.Group == nil {
		return service.ErrChatAPIKeyNotFound
	}
	if apiKey.Group.IsSubscriptionType() {
		if h.subscriptionService == nil {
			return service.ErrChatAPIKeyNotFound
		}
		subscription, err := h.subscriptionService.GetActiveSubscription(c.Request.Context(), apiKey.User.ID, apiKey.Group.ID)
		if err != nil {
			return service.ErrChatAPIKeyNotFound
		}
		if _, err := h.subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group); err != nil {
			return service.ErrChatAPIKeyNotFound
		}
		return nil
	}
	if apiKey.User.Balance <= 0 {
		return service.ErrChatAPIKeyNotFound
	}
	return nil
}

func (h *MenuLaunchHandler) findEnabledVictoryMenuItem(c *gin.Context, menuID string) (dto.VictoryMenuItem, bool) {
	raw := h.settingService.GetVictoryMenuItemsRaw(c.Request.Context())
	items := dto.ParseVictoryMenuItems(raw)
	for _, item := range items {
		if item.ID == menuID && item.Enabled {
			return item, true
		}
	}
	return dto.VictoryMenuItem{}, false
}

func appendAPIKeyQueryParam(rawURL, apiKey string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if err != nil {
			return "", err
		}
		return "", errors.New("url must be absolute")
	}
	values := parsed.Query()
	values.Set("apikey", apiKey)
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

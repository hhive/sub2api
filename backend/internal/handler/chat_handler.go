package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chatService         *service.ChatService
	subscriptionService *service.SubscriptionService
	openAIGateway       *OpenAIGatewayHandler
}

func NewChatHandler(chatService *service.ChatService, subscriptionService *service.SubscriptionService, openAIGateway *OpenAIGatewayHandler) *ChatHandler {
	return &ChatHandler{chatService: chatService, subscriptionService: subscriptionService, openAIGateway: openAIGateway}
}

func (h *ChatHandler) ListModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	models, err := h.chatService.ListModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"models": dto.ChatModelsFromService(models)})
}

func (h *ChatHandler) CreateCompletion(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.prepareGatewayContext(c, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.openAIGateway.ChatCompletions(c)
}

func (h *ChatHandler) prepareGatewayContext(c *gin.Context, userID int64) error {
	apiKey, err := h.chatService.SelectDefaultAPIKey(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: apiKey.User.ID, Concurrency: apiKey.User.Concurrency})
	c.Set(string(middleware2.ContextKeyUserRole), apiKey.User.Role)
	if service.IsGroupContextValid(apiKey.Group) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.Group, apiKey.Group)
		c.Request = c.Request.WithContext(ctx)
	}
	if apiKey.Group != nil && apiKey.Group.IsSubscriptionType() {
		subscription, err := h.subscriptionService.GetActiveSubscription(c.Request.Context(), apiKey.User.ID, apiKey.Group.ID)
		if err != nil {
			return err
		}
		c.Set(string(middleware2.ContextKeySubscription), subscription)
	}

	return nil
}

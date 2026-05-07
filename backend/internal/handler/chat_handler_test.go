package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatHandlerPrepareGatewayContextInjectsSubscriptionForSubscriptionGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(7)
	subscription := &service.UserSubscription{
		ID:        11,
		UserID:    42,
		GroupID:   groupID,
		Status:    service.SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	apiKeyRepo := &chatHandlerAPIKeyRepoStub{
		list: []service.APIKey{{ID: 3, UserID: 42, Key: "sk-chat", Status: service.StatusAPIKeyActive, GroupID: &groupID}},
		byKey: map[string]*service.APIKey{
			"sk-chat": {
				ID:      3,
				UserID:  42,
				Key:     "sk-chat",
				Status:  service.StatusAPIKeyActive,
				GroupID: &groupID,
				User:    &service.User{ID: 42, Role: service.RoleUser, Status: service.StatusActive, Concurrency: 2},
				Group: &service.Group{
					ID:               groupID,
					Name:             "subscription",
					Platform:         service.PlatformOpenAI,
					Status:           service.StatusActive,
					Hydrated:         true,
					SubscriptionType: service.SubscriptionTypeSubscription,
				},
			},
		},
	}
	subRepo := &chatHandlerUserSubscriptionRepoStub{subscription: subscription}
	h := NewChatHandler(
		service.NewChatService(service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, nil)),
		service.NewSubscriptionService(nil, subRepo, nil, nil, nil),
		nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", nil)

	err := h.prepareGatewayContext(c, 42)

	require.NoError(t, err)
	loadedSubscription, ok := middleware2.GetSubscriptionFromContext(c)
	require.True(t, ok)
	require.Equal(t, subscription.ID, loadedSubscription.ID)
	require.Equal(t, int64(42), subRepo.userID)
	require.Equal(t, groupID, subRepo.groupID)

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	require.True(t, ok)
	require.Equal(t, int64(3), apiKey.ID)

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	require.True(t, ok)
	require.Equal(t, int64(42), subject.UserID)
	require.Equal(t, 2, subject.Concurrency)

	group, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
	require.True(t, ok)
	require.Equal(t, groupID, group.ID)
}

type chatHandlerAPIKeyRepoStub struct {
	service.APIKeyRepository
	list  []service.APIKey
	byKey map[string]*service.APIKey
}

func (r *chatHandlerAPIKeyRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	return r.list, &pagination.PaginationResult{Total: int64(len(r.list)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *chatHandlerAPIKeyRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	if r.byKey == nil || r.byKey[key] == nil {
		return nil, service.ErrAPIKeyNotFound
	}
	return r.byKey[key], nil
}

type chatHandlerUserSubscriptionRepoStub struct {
	service.UserSubscriptionRepository
	subscription *service.UserSubscription
	userID       int64
	groupID      int64
}

func (r *chatHandlerUserSubscriptionRepoStub) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	r.userID = userID
	r.groupID = groupID
	if r.subscription == nil {
		return nil, service.ErrSubscriptionNotFound
	}
	return r.subscription, nil
}

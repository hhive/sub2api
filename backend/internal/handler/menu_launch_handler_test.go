//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMenuLaunchVictoryAppendsAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(9)
	handler := NewMenuLaunchHandler(
		service.NewSettingService(&menuLaunchSettingRepoStub{values: map[string]string{
			service.SettingKeyVictoryMenuItems: `[{"id":"offer","label":"小逆Offer","url":"https://offer.xiaoni-ai.top/path?from=sub2api","carry_api_key":true,"enabled":true,"sort_order":0}]`,
		}}, &config.Config{}),
		service.NewChatService(service.NewAPIKeyService(&menuLaunchAPIKeyRepoStub{
			list: []service.APIKey{{ID: 3, UserID: 42, Key: "sk-live", Status: service.StatusAPIKeyActive, GroupID: &groupID}},
			byKey: map[string]*service.APIKey{
				"sk-live": {ID: 3, UserID: 42, Key: "sk-live", Status: service.StatusAPIKeyActive, GroupID: &groupID, User: &service.User{ID: 42, Status: service.StatusActive, Balance: 1}, Group: &service.Group{ID: groupID, Status: service.StatusActive}},
			},
		}, nil, nil, nil, nil, nil, nil)),
		nil,
	)
	router := gin.New()
	router.POST("/launch", withMenuLaunchSubject(42), handler.Victory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/launch", strings.NewReader(`{"menu_id":"offer"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"redirect_url":"https://offer.xiaoni-ai.top/path?apikey=sk-live\u0026from=sub2api"`)
}

func TestMenuLaunchVictoryRejectsDisabledMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewMenuLaunchHandler(
		service.NewSettingService(&menuLaunchSettingRepoStub{values: map[string]string{
			service.SettingKeyVictoryMenuItems: `[{"id":"offer","label":"小逆Offer","url":"https://offer.xiaoni-ai.top","carry_api_key":false,"enabled":false,"sort_order":0}]`,
		}}, &config.Config{}),
		service.NewChatService(service.NewAPIKeyService(&menuLaunchAPIKeyRepoStub{}, nil, nil, nil, nil, nil, nil)),
		nil,
	)
	router := gin.New()
	router.POST("/launch", withMenuLaunchSubject(42), handler.Victory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/launch", strings.NewReader(`{"menu_id":"offer"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestMenuLaunchVictoryReturnsNoKeyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewMenuLaunchHandler(
		service.NewSettingService(&menuLaunchSettingRepoStub{values: map[string]string{
			service.SettingKeyVictoryMenuItems: `[{"id":"offer","label":"小逆Offer","url":"https://offer.xiaoni-ai.top","carry_api_key":true,"enabled":true,"sort_order":0}]`,
		}}, &config.Config{}),
		service.NewChatService(service.NewAPIKeyService(&menuLaunchAPIKeyRepoStub{
			list: []service.APIKey{{ID: 3, UserID: 42, Key: "sk-no-group", Status: service.StatusAPIKeyActive}},
		}, nil, nil, nil, nil, nil, nil)),
		nil,
	)
	router := gin.New()
	router.POST("/launch", withMenuLaunchSubject(42), handler.Victory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/launch", strings.NewReader(`{"menu_id":"offer"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "CHAT_API_KEY_NOT_FOUND")
}

func withMenuLaunchSubject(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
		c.Next()
	}
}

type menuLaunchSettingRepoStub struct {
	values map[string]string
}

func (s *menuLaunchSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *menuLaunchSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *menuLaunchSettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *menuLaunchSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *menuLaunchSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *menuLaunchSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *menuLaunchSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type menuLaunchAPIKeyRepoStub struct {
	service.APIKeyRepository
	list  []service.APIKey
	byKey map[string]*service.APIKey
}

func (r *menuLaunchAPIKeyRepoStub) ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	return r.list, &pagination.PaginationResult{Total: int64(len(r.list)), Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (r *menuLaunchAPIKeyRepoStub) GetByKeyForAuth(ctx context.Context, key string) (*service.APIKey, error) {
	if r.byKey == nil || r.byKey[key] == nil {
		return nil, service.ErrAPIKeyNotFound
	}
	return r.byKey[key], nil
}

//go:build unit

package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIntegrationGatewayRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/app-integrations/app/chat/completions?key=foreign", nil)
	r.Header.Set("Authorization", "Bearer synthetic-grant")
	r.Header.Set("X-Sub2API-App-Secret", "synthetic-secret")
	r.Header.Set("X-Forwarded-For", "203.0.113.12")
	r.Header.Set("Cookie", "session=synthetic-cookie")
	r.Header.Set("X-API-Key", "foreign")
	forwarded := integrationGatewayRequest(r, "/v1/chat/completions", "synthetic-key")
	require.Equal(t, "/v1/chat/completions", forwarded.RequestURI)
	require.Equal(t, "Bearer synthetic-key", forwarded.Header.Get("Authorization"))
	for _, header := range []string{"X-Sub2API-App-Secret", "X-Forwarded-For", "Cookie", "X-API-Key"} {
		require.Empty(t, forwarded.Header.Get(header))
	}
	require.Equal(t, "Bearer synthetic-grant", r.Header.Get("Authorization"))
}

func TestIntegrationKeyFilter(t *testing.T) {
	groupID := int64(11)
	key := service.APIKey{ID: 2, UserID: 3, Key: "synthetic-secret", Status: service.StatusAPIKeyActive, GroupID: &groupID,
		Group: &service.Group{ID: groupID, Status: service.StatusActive, Platform: service.PlatformComposite}}
	require.True(t, describeIntegrationKey(&key).Selectable)
	key.IPBlacklist = []string{"198.51.100.0/24"}
	require.Equal(t, "ip_restricted", describeIntegrationKey(&key).Status)
	key.IPBlacklist = nil
	key.Quota, key.QuotaUsed = 1, 1
	require.False(t, describeIntegrationKey(&key).Selectable)
	require.Equal(t, service.StatusAPIKeyQuotaExhausted, describeIntegrationKey(&key).Status)
}

type integrationTestAuthorization struct {
	applicationIntegrationAuth
	userID int64
}

func (s integrationTestAuthorization) Authorize(_ context.Context, _, _, _, _ string) (int64, error) {
	return s.userID, nil
}

type integrationTestDirectory struct {
	applicationIntegrationKeys
	key  *service.APIKey
	list func(int64, pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error)
}

func (s integrationTestDirectory) List(_ context.Context, userID int64, params pagination.PaginationParams, _ service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	return s.list(userID, params)
}

func (s integrationTestDirectory) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	if id != s.key.ID {
		return nil, service.ErrAPIKeyNotFound
	}
	return s.key, nil
}

type integrationTestGatewayKeys struct {
	service.APIKeyRepository
	key *service.APIKey
}

func (s integrationTestGatewayKeys) GetByKeyForAuth(_ context.Context, credential string) (*service.APIKey, error) {
	if credential != s.key.Key {
		return nil, service.ErrAPIKeyNotFound
	}
	return s.key, nil
}

func (s integrationTestGatewayKeys) UpdateLastUsed(context.Context, int64, time.Time) error {
	return nil
}

func TestIntegrationChatNativePolicyAndStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(8)
	key := service.APIKey{
		ID: 2, UserID: 3, Key: "synthetic-gateway-key", Status: service.StatusAPIKeyActive, GroupID: &groupID,
		User: &service.User{ID: 3, Status: service.StatusActive, Balance: 1},
		Group: &service.Group{ID: groupID, Status: service.StatusActive, Platform: service.PlatformOpenAI, Hydrated: true,
			ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"allowed-model"}}},
	}
	gatewayKey := key
	cfg := &config.Config{}
	keys := service.NewAPIKeyService(integrationTestGatewayKeys{key: &gatewayKey}, nil, nil, nil, nil, nil, cfg)
	r := gin.New()
	r.POST("/v1/chat/completions", gin.HandlerFunc(middleware.NewAPIKeyAuthMiddleware(keys, nil, cfg)), middleware.GroupModelAllowlist(), func(c *gin.Context) {
		selected, ok := middleware.GetAPIKeyFromContext(c)
		require.True(t, ok)
		require.Equal(t, key.ID, selected.ID)
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: [DONE]\n\n")
		c.Writer.Flush()
	})
	r.POST("/app-integrations/:app_id/chat/completions", applicationIntegrationRoutes{
		auth: integrationTestAuthorization{userID: key.UserID}, keys: integrationTestDirectory{key: &key}, gateway: r,
	}.chat)
	call := func(id, model string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/app-integrations/app/chat/completions", strings.NewReader(`{"model":"`+model+`","stream":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Sub2API-Key-ID", id)
		w := httptest.NewRecorder()
		// The outer Gin writer must propagate native status and Flush calls.
		r.ServeHTTP(w, req)
		return w
	}
	allowed := call("2", "allowed-model")
	require.Equal(t, http.StatusOK, allowed.Code)
	require.True(t, allowed.Flushed)
	require.Equal(t, "data: [DONE]\n\n", allowed.Body.String())

	require.Equal(t, http.StatusNotFound, call("2", "unlisted-model").Code)
	missing := call("999", "allowed-model")
	require.Equal(t, http.StatusForbidden, missing.Code)
	key.UserID = 15
	foreign := call("2", "allowed-model")
	require.Equal(t, missing.Code, foreign.Code)
	require.JSONEq(t, missing.Body.String(), foreign.Body.String())
	key.UserID = 3
	gatewayKey.User.Balance = 0
	require.Equal(t, http.StatusForbidden, call("2", "allowed-model").Code)
	gatewayKey.User.Balance = 1
	gatewayKey.IPBlacklist = []string{"198.51.100.0/24"}
	ipDenied := call("2", "allowed-model")
	require.Equal(t, http.StatusForbidden, ipDenied.Code)
	require.Contains(t, ipDenied.Body.String(), "INTEGRATION_KEY_UNAVAILABLE")
}

func TestIntegrationDirectoryPaginationAndMask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	keys := integrationTestDirectory{list: func(userID int64, params pagination.PaginationParams) ([]service.APIKey, *pagination.PaginationResult, error) {
		require.EqualValues(t, 17, userID)
		result := &pagination.PaginationResult{Page: params.Page, PageSize: params.PageSize, Total: 101}
		if params.Page == 2 {
			require.Equal(t, 100, params.PageSize)
			return []service.APIKey{{ID: 5, Key: "synthetic-no-leak", UserID: userID}}, result, nil
		}
		require.Equal(t, 50, params.PageSize)
		result.Total = 0
		return nil, result, nil
	}}
	r.GET("/app-integrations/:app_id/keys", applicationIntegrationRoutes{auth: integrationTestAuthorization{userID: 17}, keys: keys}.listKeys)
	empty := httptest.NewRecorder()
	r.ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/app-integrations/app/keys", nil))
	require.JSONEq(t, `{"items":[],"page":1,"page_size":50,"total":0}`, empty.Body.String())
	page := httptest.NewRecorder()
	r.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/app-integrations/app/keys?page=2&page_size=123", nil))
	require.JSONEq(t, `{"items":[{"id":5,"name":"","mask":"****leak","group_id":null,"group_name":"","platform":"","status":"","selectable":false}],"page":2,"page_size":100,"total":101}`, page.Body.String())
}

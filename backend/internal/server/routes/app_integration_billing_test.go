//go:build unit

package routes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type integrationBillingUpstream struct {
	service.HTTPUpstream
	request *http.Request
}

func (u *integrationBillingUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.request = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/event-stream"}, "X-Request-Id": {"fixture-billing"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"id":"fixture-completion","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{"content":"fixture reply"},"finish_reason":null}]}`,
			`data: {"id":"fixture-completion","object":"chat.completion.chunk","model":"gpt-5.4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: {"id":"fixture-completion","object":"chat.completion.chunk","model":"gpt-5.4","choices":[],"usage":{"prompt_tokens":17,"completion_tokens":9,"total_tokens":26}}`,
			"data: [DONE]", "",
		}, "\n\n"))),
	}, nil
}

type integrationBillingUsers struct {
	service.UserRepository
	user *service.User
}

func (r integrationBillingUsers) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

type integrationBillingLedger struct {
	service.UsageBillingRepository
	commands []*service.UsageBillingCommand
}

func (r *integrationBillingLedger) Apply(_ context.Context, command *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	r.commands = append(r.commands, command)
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

func TestIntegrationChatBillsSelectedKeyThroughRegisteredGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(41)
	user := &service.User{ID: 43, Status: service.StatusActive, Balance: 10}
	key := &service.APIKey{
		ID: 79, UserID: user.ID, Key: "synthetic-selected-sub2api-key", Status: service.StatusAPIKeyActive, GroupID: &groupID,
		User: user, Group: &service.Group{ID: groupID, Status: service.StatusActive, Platform: service.PlatformOpenAI, Hydrated: true, RateMultiplier: 1},
	}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxBodySize: 1 << 20, TextMaxBodySize: 1 << 20}, Default: config.DefaultConfig{RateMultiplier: 1}}
	keys := service.NewAPIKeyService(integrationTestGatewayKeys{key: key}, nil, nil, nil, nil, nil, cfg)
	accounts := compatibleImagesAccounts{accounts: []service.Account{{
		ID: 95, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true,
		Credentials: map[string]any{"api_key": "synthetic-upstream-only-key", "base_url": "https://upstream.example"},
		Extra:       map[string]any{openai_compat.ExtraKeyResponsesSupported: false},
	}}}
	upstream, usage, ledger := &integrationBillingUpstream{}, &compatibleImagesUsage{}, &integrationBillingLedger{}
	users := integrationBillingUsers{user: user}
	concurrency := service.NewConcurrencyService(nil)
	billing := service.NewBillingCacheService(nil, users, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	gateway := service.NewOpenAIGatewayService(accounts, usage, ledger, users, nil, nil, nil, cfg, nil, concurrency, service.NewBillingService(cfg, nil), nil, billing, upstream, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil)

	r := gin.New()
	RegisterGatewayRoutes(r, &handler.Handlers{
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: handler.NewOpenAIGatewayHandler(gateway, concurrency, billing, keys, nil, nil, nil, nil, cfg),
	}, middleware.NewAPIKeyAuthMiddleware(keys, nil, cfg), keys, nil, nil, nil, nil, cfg)
	app := applicationIntegrationRoutes{auth: integrationTestAuthorization{userID: user.ID}, keys: integrationTestDirectory{key: key}, gateway: r}
	app.register(r.Group("/api/v1"), func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) }, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/app-integrations/chat-app/chat/completions", strings.NewReader(`{"model":"gpt-5.4","stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"fixture"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sub2API-Key-ID", "79")
	req.Header.Set("X-Sub2API-App-Secret", "synthetic-app-secret")
	req.Header.Set("Authorization", "Bearer synthetic-grant")
	req.Header.Set("X-API-Key", "synthetic-wrong-key")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, rec.Flushed)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, rec.Body.String(), "fixture reply")
	require.Contains(t, rec.Body.String(), "data: [DONE]")
	require.NotNil(t, upstream.request)
	require.Equal(t, "/v1/chat/completions", upstream.request.URL.Path)
	require.Equal(t, "Bearer synthetic-upstream-only-key", upstream.request.Header.Get("Authorization"))
	for _, header := range []string{"X-API-Key", "X-Sub2API-App-Secret", "X-Sub2API-Key-ID"} {
		require.Empty(t, upstream.request.Header.Get(header))
	}
	require.Len(t, ledger.commands, 1)
	require.Equal(t, key.ID, ledger.commands[0].APIKeyID)
	require.Equal(t, user.ID, ledger.commands[0].UserID)
	require.Positive(t, ledger.commands[0].BalanceCost)
	require.Len(t, usage.logs, 1)
	require.Equal(t, key.ID, usage.logs[0].APIKeyID)
	require.True(t, usage.logs[0].Stream)
	require.Equal(t, 17, usage.logs[0].InputTokens)
	require.Equal(t, 9, usage.logs[0].OutputTokens)
}

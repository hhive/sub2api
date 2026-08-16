//go:build unit

package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLaunchRoutesUseMediaPlaygroundContractOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	noop := func(c *gin.Context) { c.Next() }
	handlers := &handler.Handlers{Launch: &handler.LaunchHandler{}, AdminExternalApp: &handler.AdminExternalAppHandler{}, Admin: &handler.AdminHandlers{Account: &admin.AccountHandler{}}}

	RegisterLaunchRoutes(
		router.Group("/api/v1"),
		handlers,
		middleware.JWTAuthMiddleware(noop),
		middleware.APIKeyAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		nil,
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	require.Contains(t, routes, "POST /api/v1/media-playground/launch")
	require.Contains(t, routes, "POST /api/v1/media-playground/exchange")
	require.Contains(t, routes, "POST /api/v1/external-apps/:app_id/exchange")
	require.Contains(t, routes, "GET /api/v1/internal/relay-monitor/accounts/:id/priority")
	require.Contains(t, routes, "PUT /api/v1/internal/relay-monitor/accounts/:id/priority")
	require.Contains(t, routes, "POST /api/v1/internal/relay-monitor/accounts/:id/priority-cap-pause")
	require.NotContains(t, routes, "POST /api/v1/image-playground/launch")
	require.NotContains(t, routes, "POST /api/v1/image-playground/exchange")
	require.NotContains(t, routes, "POST /api/v1/video-playground/launch")
	require.NotContains(t, routes, "POST /api/v1/video-playground/exchange")
}

func TestRelayMonitorPriorityRoutesUseUpstreamMonitorSSOSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	noop := func(c *gin.Context) { c.Next() }
	svc := &adminExternalAppRouteServiceStub{}
	handlers := &handler.Handlers{
		Launch:           &handler.LaunchHandler{},
		AdminExternalApp: handler.NewAdminExternalAppHandler(svc),
		Admin:            &handler.AdminHandlers{Account: &admin.AccountHandler{}},
	}
	RegisterLaunchRoutes(
		router.Group("/api/v1"), handlers,
		middleware.JWTAuthMiddleware(noop), middleware.APIKeyAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop), nil, nil,
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/internal/relay-monitor/accounts/7/priority", nil)
	request.Header.Set("X-Sub2API-External-App-Secret", "shared-sso-secret")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, 1, svc.secretCalls)
	require.Equal(t, "upstream-monitor", svc.lastAppID)
	require.Equal(t, "shared-sso-secret", svc.lastSecret)
}

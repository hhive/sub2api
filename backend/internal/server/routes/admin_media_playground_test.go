//go:build unit

package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLegacyMediaPlaygroundAdminRoutesAreRemoved(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	noop := func(c *gin.Context) { c.Next() }
	handlers := &handler.Handlers{
		Admin:            &handler.AdminHandlers{},
		AdminExternalApp: handler.NewAdminExternalAppHandler(&adminExternalAppRouteServiceStub{}),
	}
	settingService := service.NewSettingService(&adminExternalAppSettingRepoStub{}, &config.Config{})

	RegisterAdminRoutes(
		router.Group("/api/v1"),
		handlers,
		middleware.AdminAuthMiddleware(noop),
		middleware.AuditLogMiddleware(noop),
		middleware.StepUpAuthMiddleware(noop),
		settingService,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	require.Contains(t, routes, "GET /api/v1/admin/external-apps")
	require.Contains(t, routes, "POST /api/v1/admin/external-apps/:app_id/launch")

	legacyPaths := []string{
		"/api/v1/admin/media-playground/image/models",
		"/api/v1/admin/media-playground/image/model-probe-runs",
		"/api/v1/admin/media-playground/image/tasks",
		"/api/v1/admin/media-playground/image/upstream-requests",
		"/api/v1/admin/media-playground/video/models",
		"/api/v1/admin/media-playground/video/tasks",
		"/api/v1/admin/media-playground/video/upstream-requests",
	}
	for _, path := range legacyPaths {
		require.NotContains(t, routes, http.MethodGet+" "+path)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, recorder.Code, path)
	}
}

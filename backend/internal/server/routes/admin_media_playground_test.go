package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMediaPlaygroundAdminRoutesReplaceLegacyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		MediaPlaygroundImage: adminhandler.NewMediaPlaygroundImageHandler(nil),
		MediaPlaygroundVideo: adminhandler.NewMediaPlaygroundVideoHandler(nil),
	}}
	registerMediaPlaygroundRoutes(router.Group("/api/admin"), h)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		"GET /api/admin/media-playground/image/models",
		"GET /api/admin/media-playground/image/model-probe-runs",
		"GET /api/admin/media-playground/video/models",
		"GET /api/admin/media-playground/video/tasks",
		"GET /api/admin/media-playground/video/tasks/:id",
		"GET /api/admin/media-playground/video/upstream-requests",
	} {
		require.Contains(t, routes, route)
	}
	for _, route := range []string{
		"GET /api/admin/image-playground/models",
		"GET /api/admin/image-playground/model-probe-runs",
		"GET /api/admin/video-playground/models",
		"GET /api/admin/video-playground/upstream-requests",
	} {
		require.NotContains(t, routes, route)
	}
}

func TestLegacyMediaPlaygroundAdminPathsReturnHTTP404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		MediaPlaygroundImage: adminhandler.NewMediaPlaygroundImageHandler(nil),
		MediaPlaygroundVideo: adminhandler.NewMediaPlaygroundVideoHandler(nil),
	}}
	registerMediaPlaygroundRoutes(router.Group("/api/admin"), h)
	for _, path := range []string{
		"/api/admin/media-playground/image/models",
		"/api/admin/media-playground/video/models",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
	for _, path := range []string{
		"/api/admin/image-playground/models", "/api/admin/image-playground/model-probe-runs",
		"/api/admin/video-playground/models", "/api/admin/video-playground/upstream-requests",
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, rec.Code, path)
	}
}

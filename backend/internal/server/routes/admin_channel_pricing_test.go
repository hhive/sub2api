package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDefaultModelPricingRouteIsAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{
		Channel: adminhandler.NewChannelHandler(nil, nil, service.NewPricingService(nil, nil)),
	}}
	registerChannelRoutes(admin, handlers)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	require.Contains(t, routes, "GET /api/v1/admin/channels/default-model-pricing")
	require.NotContains(t, routes, "GET /api/v1/channels/default-model-pricing")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/channels/default-model-pricing", nil))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

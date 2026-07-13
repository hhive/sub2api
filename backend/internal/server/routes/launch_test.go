//go:build unit

package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLaunchRoutesUseMediaPlaygroundContractOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	noop := func(c *gin.Context) { c.Next() }
	handlers := &handler.Handlers{Launch: &handler.LaunchHandler{}}

	RegisterLaunchRoutes(
		router.Group("/api/v1"),
		handlers,
		middleware.JWTAuthMiddleware(noop),
		middleware.APIKeyAuthMiddleware(noop),
		nil,
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	require.Contains(t, routes, "POST /api/v1/media-playground/launch")
	require.Contains(t, routes, "POST /api/v1/media-playground/exchange")
	require.NotContains(t, routes, "POST /api/v1/image-playground/launch")
	require.NotContains(t, routes, "POST /api/v1/image-playground/exchange")
	require.NotContains(t, routes, "POST /api/v1/video-playground/launch")
	require.NotContains(t, routes, "POST /api/v1/video-playground/exchange")
}

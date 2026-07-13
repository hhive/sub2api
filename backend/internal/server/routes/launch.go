package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterLaunchRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	settingService *service.SettingService,
) {
	lobeHubPublic := v1.Group("/lobehub")
	{
		lobeHubPublic.POST("/exchange", h.Launch.LobeHubExchange)
	}
	mediaPlaygroundPublic := v1.Group("/media-playground")
	{
		mediaPlaygroundPublic.POST("/exchange", h.Launch.MediaPlaygroundExchange)
	}

	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		mediaPlayground := authenticated.Group("/media-playground")
		{
			mediaPlayground.POST("/launch", h.Launch.MediaPlaygroundLaunch)
		}
		lobeHub := authenticated.Group("/lobehub")
		{
			lobeHub.POST("/launch", h.Launch.LobeHubLaunch)
		}
	}

	if h.ImagePlayground != nil {
		mediaPlaygroundTasks := v1.Group("/media-playground")
		mediaPlaygroundTasks.Use(gin.HandlerFunc(apiKeyAuth))
		mediaPlaygroundTasks.Use(middleware.BackendModeUserGuard(settingService))
		{
			mediaPlaygroundTasks.POST("/tasks", h.ImagePlayground.Create)
			mediaPlaygroundTasks.GET("/tasks/recent", h.ImagePlayground.Recent)
			mediaPlaygroundTasks.GET("/tasks/:id", h.ImagePlayground.Get)
			mediaPlaygroundTasks.POST("/tasks/:id/cancel", h.ImagePlayground.Cancel)
		}
	}
}

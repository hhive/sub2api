package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterOnyxRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	settingService *service.SettingService,
) {
	onyxPublic := v1.Group("/onyx")
	{
		onyxPublic.POST("/exchange", h.Onyx.Exchange)
	}
	lobeHubPublic := v1.Group("/lobehub")
	{
		lobeHubPublic.POST("/exchange", h.Onyx.LobeHubExchange)
	}
	videoPlaygroundPublic := v1.Group("/video-playground")
	{
		videoPlaygroundPublic.POST("/exchange", h.Onyx.VideoPlaygroundExchange)
	}

	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		onyx := authenticated.Group("/onyx")
		{
			onyx.POST("/launch", h.Onyx.Launch)
		}
		imagePlayground := authenticated.Group("/image-playground")
		{
			imagePlayground.POST("/launch", h.Onyx.ImagePlaygroundLaunch)
		}
		lobeHub := authenticated.Group("/lobehub")
		{
			lobeHub.POST("/launch", h.Onyx.LobeHubLaunch)
		}
		videoPlayground := authenticated.Group("/video-playground")
		{
			videoPlayground.POST("/launch", h.Onyx.VideoPlaygroundLaunch)
		}
	}

	if h.ImagePlayground != nil {
		imagePlaygroundTasks := v1.Group("/image-playground")
		imagePlaygroundTasks.Use(gin.HandlerFunc(apiKeyAuth))
		imagePlaygroundTasks.Use(middleware.BackendModeUserGuard(settingService))
		{
			imagePlaygroundTasks.POST("/tasks", h.ImagePlayground.Create)
			imagePlaygroundTasks.GET("/tasks/recent", h.ImagePlayground.Recent)
			imagePlaygroundTasks.GET("/tasks/:id", h.ImagePlayground.Get)
			imagePlaygroundTasks.POST("/tasks/:id/cancel", h.ImagePlayground.Cancel)
		}
	}
}

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
	settingService *service.SettingService,
) {
	onyxPublic := v1.Group("/onyx")
	{
		onyxPublic.POST("/exchange", h.Onyx.Exchange)
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
			if h.ImagePlayground != nil {
				imagePlayground.POST("/tasks", h.ImagePlayground.Create)
				imagePlayground.GET("/tasks/recent", h.ImagePlayground.Recent)
				imagePlayground.GET("/tasks/:id", h.ImagePlayground.Get)
				imagePlayground.POST("/tasks/:id/cancel", h.ImagePlayground.Cancel)
			}
		}
	}
}

package routes

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	basemiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterLaunchRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	redisClient *redis.Client,
	settingService *service.SettingService,
) {
	if h.Admin != nil && h.Admin.Account != nil {
		relayMonitor := v1.Group("/internal/relay-monitor/accounts")
		relayMonitor.GET("/:id/priority", h.Admin.Account.GetRelayMonitorPriority)
		relayMonitor.PUT("/:id/priority", h.Admin.Account.SetRelayMonitorPriority)
		relayMonitor.POST("/:id/priority-cap-pause", h.Admin.Account.PauseRelayMonitorPriorityCappedAccount)
	}

	exchangeLimiter := basemiddleware.NewRateLimiter(redisClient)
	exchangeRateLimitOptions := basemiddleware.RateLimitOptions{FailureMode: basemiddleware.RateLimitFailClose}
	exchangeIPRateLimit := exchangeLimiter.LimitWithOptions("admin-external-app-exchange", 30, time.Minute, exchangeRateLimitOptions)
	exchangeAppRateLimit := func(c *gin.Context) {
		appID := strings.TrimSpace(c.Param("app_id"))
		appScopedLimiter := exchangeLimiter.LimitWithOptions("admin-external-app-exchange-app:"+appID, 30, time.Minute, basemiddleware.RateLimitOptions{
			FailureMode: basemiddleware.RateLimitFailClose,
		})
		appScopedLimiter(c)
	}
	externalApps := v1.Group("/external-apps")
	externalApps.Use(gin.HandlerFunc(auditLog))
	{
		externalApps.POST("/:app_id/exchange", exchangeIPRateLimit, h.AdminExternalApp.RequireExchangeSecret, exchangeAppRateLimit, h.AdminExternalApp.Exchange)
	}

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

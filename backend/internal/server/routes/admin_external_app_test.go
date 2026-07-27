//go:build unit

package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type adminExternalAppRouteServiceStub struct {
	createCalls int
	secretCalls int
}

func (s *adminExternalAppRouteServiceStub) List() []service.AdminExternalAppMetadata {
	return []service.AdminExternalAppMetadata{{AppID: "media-management", Enabled: true}}
}
func (s *adminExternalAppRouteServiceStub) CreateLaunch(context.Context, string, int64) (*service.AdminExternalAppLaunchResult, error) {
	s.createCalls++
	return &service.AdminExternalAppLaunchResult{}, nil
}
func (s *adminExternalAppRouteServiceStub) ValidateExchangeSecret(string, string) error {
	s.secretCalls++
	return infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_SECRET_INVALID", "external app secret invalid")
}
func (s *adminExternalAppRouteServiceStub) Exchange(context.Context, string, string) (*service.AdminExternalAppExchangePayload, error) {
	return nil, infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_TOKEN_INVALID", "external app token invalid")
}

type adminExternalAppSettingRepoStub struct{}

func (*adminExternalAppSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}
func (*adminExternalAppSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", service.ErrSettingNotFound
}
func (*adminExternalAppSettingRepoStub) Set(context.Context, string, string) error { return nil }
func (*adminExternalAppSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (*adminExternalAppSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (*adminExternalAppSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (*adminExternalAppSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestAdminExternalAppRoutesAreInsideAdminAuthGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authCalls := 0
	adminAuth := middleware.AdminAuthMiddleware(func(c *gin.Context) {
		authCalls++
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	noop := func(c *gin.Context) { c.Next() }
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{}, AdminExternalApp: &handler.AdminExternalAppHandler{}}
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, middleware.AuditLogMiddleware(noop), middleware.StepUpAuthMiddleware(noop), nil, nil)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	require.Contains(t, routes, "GET /api/v1/admin/external-apps")
	require.Contains(t, routes, "POST /api/v1/admin/external-apps/:app_id/launch")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/external-apps", nil))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, 1, authCalls)
}

func TestAdminExternalAppLaunchRouteUsesAuthAuditAndComplianceGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authCalls := 0
	auditCalls := 0
	adminAuth := middleware.AdminAuthMiddleware(func(c *gin.Context) {
		authCalls++
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Set("auth_method", service.AuditAuthMethodJWT)
		c.Next()
	})
	audit := middleware.AuditLogMiddleware(func(c *gin.Context) {
		auditCalls++
		c.Next()
	})
	svc := &adminExternalAppRouteServiceStub{}
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{}, AdminExternalApp: handler.NewAdminExternalAppHandler(svc)}
	settingService := service.NewSettingService(&adminExternalAppSettingRepoStub{}, &config.Config{})
	RegisterAdminRoutes(router.Group("/api/v1"), handlers, adminAuth, audit, middleware.StepUpAuthMiddleware(func(c *gin.Context) { c.Next() }), settingService, nil)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/admin/external-apps/media-management/launch", nil))
	require.Equal(t, http.StatusLocked, recorder.Code)
	require.Equal(t, 1, authCalls)
	require.Equal(t, 1, auditCalls)
	require.Zero(t, svc.createCalls)
}

func TestAdminExternalAppExchangeRouteIsPublicButRequiresAppSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	router := gin.New()
	jwtCalls := 0
	auditCalls := 0
	jwtAuth := middleware.JWTAuthMiddleware(func(c *gin.Context) {
		jwtCalls++
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	audit := middleware.AuditLogMiddleware(func(c *gin.Context) {
		auditCalls++
		c.Next()
	})
	handlers := &handler.Handlers{
		Launch:           &handler.LaunchHandler{},
		AdminExternalApp: handler.NewAdminExternalAppHandler(&adminExternalAppRouteServiceStub{}),
	}
	RegisterLaunchRoutes(router.Group("/api/v1"), handlers, jwtAuth, middleware.APIKeyAuthMiddleware(func(c *gin.Context) { c.Next() }), audit, rdb, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/external-apps/media-management/exchange", strings.NewReader(`{"token":"ticket"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Sub2API-External-App-Secret", "wrong")
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, jwtCalls)
	require.Equal(t, 1, auditCalls)
}

func TestAdminExternalAppExchangeRouteUnknownAppRotationCannotBypassIPLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	miniRedis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	router := gin.New()
	svc := &adminExternalAppRouteServiceStub{}
	handlers := &handler.Handlers{
		Launch:           &handler.LaunchHandler{},
		AdminExternalApp: handler.NewAdminExternalAppHandler(svc),
	}
	noop := func(c *gin.Context) { c.Next() }
	RegisterLaunchRoutes(router.Group("/api/v1"), handlers, middleware.JWTAuthMiddleware(noop), middleware.APIKeyAuthMiddleware(noop), middleware.AuditLogMiddleware(noop), rdb, nil)

	for i := 0; i < 31; i++ {
		recorder := httptest.NewRecorder()
		path := fmt.Sprintf("/api/v1/external-apps/unknown-%02d/exchange", i)
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"token":"ticket"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Sub2API-External-App-Secret", "wrong")
		router.ServeHTTP(recorder, request)
		if i < 30 {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		} else {
			require.Equal(t, http.StatusTooManyRequests, recorder.Code)
		}
	}

	keys := miniRedis.Keys()
	require.Len(t, keys, 1)
	require.Contains(t, keys[0], "rate_limit:admin-external-app-exchange:")
	require.Equal(t, 30, svc.secretCalls)
}

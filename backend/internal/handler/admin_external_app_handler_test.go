//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminExternalAppServiceStub struct {
	list          []service.AdminExternalAppMetadata
	secretErr     error
	secretCalls   int
	exchangeErr   error
	exchange      *service.AdminExternalAppExchangePayload
	exchangeCalls int
}

func (s *adminExternalAppServiceStub) List() []service.AdminExternalAppMetadata { return s.list }
func (s *adminExternalAppServiceStub) CreateLaunch(context.Context, string, int64) (*service.AdminExternalAppLaunchResult, error) {
	return &service.AdminExternalAppLaunchResult{}, nil
}
func (s *adminExternalAppServiceStub) ValidateExchangeSecret(string, string) error {
	s.secretCalls++
	return s.secretErr
}
func (s *adminExternalAppServiceStub) Exchange(context.Context, string, string) (*service.AdminExternalAppExchangePayload, error) {
	s.exchangeCalls++
	if s.exchange != nil || s.exchangeErr != nil {
		return s.exchange, s.exchangeErr
	}
	return &service.AdminExternalAppExchangePayload{}, nil
}

func TestAdminExternalAppHandler_CreateRequiresJWTAuthMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminExternalAppHandler(&adminExternalAppServiceStub{})
	r := gin.New()
	r.POST("/apps/:app_id/launch", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Set("auth_method", service.AuditAuthMethodAdminAPIKey)
		c.Next()
	}, h.CreateLaunch)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/launch", nil)
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminExternalAppHandler_ListContainsOnlyGenericMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{list: []service.AdminExternalAppMetadata{{
		AppID: "media-management", LabelEN: "Media management", LabelZH: "Media management", Enabled: true, SortOrder: 20,
	}}}
	r := gin.New()
	r.GET("/apps", NewAdminExternalAppHandler(svc).List)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/apps", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.ElementsMatch(t, []string{"app_id", "label_en", "label_zh", "enabled", "sort_order"}, mapKeys(body.Data[0]))
	for _, mediaField := range []string{"model", "upstream_base_url", "upstream_api_key", "price_1k", "price_quota", "task_id"} {
		require.NotContains(t, body.Data[0], mediaField)
	}
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestAdminExternalAppHandler_SecretValidatedBeforeTokenConsumption(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{secretErr: infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_SECRET_INVALID", "external app secret invalid")}
	h := NewAdminExternalAppHandler(svc)
	r := gin.New()
	r.POST("/apps/:app_id/exchange", h.Exchange)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/exchange", bytes.NewBufferString(`{"token":"valid-token"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminExternalAppSecretHeader, "wrong")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Zero(t, svc.exchangeCalls)
}

func TestAdminExternalAppHandler_RejectsMalformedAppIDBeforeSecretLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, appID := range []string{"Bad_ID", strings.Repeat("a", 65)} {
		t.Run(appID[:6], func(t *testing.T) {
			svc := &adminExternalAppServiceStub{}
			h := NewAdminExternalAppHandler(svc)
			r := gin.New()
			r.POST("/apps/:app_id/exchange", h.Exchange)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/apps/"+appID+"/exchange", bytes.NewBufferString(`{"token":"ticket"}`))
			request.Header.Set(adminExternalAppSecretHeader, "secret")
			r.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Zero(t, svc.secretCalls)
			require.Zero(t, svc.exchangeCalls)
		})
	}
}

func TestAdminExternalAppHandler_RejectsUnknownExchangeField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{}
	h := NewAdminExternalAppHandler(svc)
	r := gin.New()
	r.POST("/apps/:app_id/exchange", h.Exchange)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/exchange", bytes.NewBufferString(`{"token":"valid-token","role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminExternalAppSecretHeader, "secret")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, svc.exchangeCalls)
}

func TestAdminExternalAppHandler_RejectsEmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{}
	h := NewAdminExternalAppHandler(svc)
	r := gin.New()
	r.POST("/apps/:app_id/exchange", h.Exchange)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/exchange", bytes.NewBufferString(`{"token":"  "}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminExternalAppSecretHeader, "secret")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, svc.exchangeCalls)
}

func TestAdminExternalAppHandler_WrongAudienceResponseDoesNotLeakToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{exchangeErr: infraerrors.Unauthorized("ADMIN_EXTERNAL_APP_TOKEN_INVALID", "external app token invalid")}
	h := NewAdminExternalAppHandler(svc)
	r := gin.New()
	r.POST("/apps/:app_id/exchange", h.Exchange)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/exchange", bytes.NewBufferString(`{"token":"sensitive-ticket"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminExternalAppSecretHeader, "secret")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.NotContains(t, rec.Body.String(), "sensitive-ticket")
}

func TestAdminExternalAppHandler_ExchangeResponseContainsOnlyMinimalClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &adminExternalAppServiceStub{exchange: &service.AdminExternalAppExchangePayload{
		UserID: 42, Email: "admin@example.com", Username: "admin", Role: service.RoleAdmin,
		AppID: "media-management", IssuedAt: 1_700_000_000, ExpiresAt: 1_700_000_060,
	}}
	h := NewAdminExternalAppHandler(svc)
	r := gin.New()
	r.POST("/apps/:app_id/exchange", h.Exchange)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/apps/media-management/exchange", bytes.NewBufferString(`{"token":"sensitive-ticket"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(adminExternalAppSecretHeader, "secret")
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	for _, forbidden := range []string{"access_token", "refresh_token", "api_key", "password", "sensitive-ticket"} {
		require.NotContains(t, rec.Body.String(), forbidden)
	}
}

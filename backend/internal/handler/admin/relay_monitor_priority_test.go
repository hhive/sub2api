package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func registerRelayMonitorAccountRoutes(router *gin.Engine, h *AccountHandler) {
	router.GET("/accounts/:id/priority", h.GetRelayMonitorPriority)
	router.PUT("/accounts/:id/priority", h.SetRelayMonitorPriority)
	router.POST("/accounts/:id/priority-cap-pause", h.PauseRelayMonitorPriorityCappedAccount)
}

func TestRelayMonitorPriorityEndpointRequiresSecretAndMinimumOne(t *testing.T) {
	t.Setenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET", "0123456789abcdef0123456789abcdef")
	adminSvc := newStubAdminService()
	adminSvc.getAccountResult = &service.Account{ID: 7, Name: "a", Priority: 5}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRelayMonitorAccountRoutes(router, h)

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/accounts/7/priority", nil))
	require.Equal(t, http.StatusUnauthorized, unauthorized.Code)
	unauthorizedPause := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedPause, httptest.NewRequest(http.MethodPost, "/accounts/7/priority-cap-pause", nil))
	require.Equal(t, http.StatusUnauthorized, unauthorizedPause.Code)

	invalidPause := httptest.NewRecorder()
	invalidPauseRequest := httptest.NewRequest(http.MethodPost, "/accounts/invalid/priority-cap-pause", nil)
	invalidPauseRequest.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	router.ServeHTTP(invalidPause, invalidPauseRequest)
	require.Equal(t, http.StatusBadRequest, invalidPause.Code)

	for _, target := range []int{0, -1} {
		body, _ := json.Marshal(map[string]int{"expected_priority": 5, "target_priority": target})
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPut, "/accounts/7/priority", bytes.NewReader(body))
		request.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestRelayMonitorPriorityEndpointReadsAndIdempotentlyUpdates(t *testing.T) {
	t.Setenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET", "0123456789abcdef0123456789abcdef")
	adminSvc := newStubAdminService()
	adminSvc.getAccountResult = &service.Account{ID: 7, Name: "a", Priority: 5}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRelayMonitorAccountRoutes(router, h)

	get := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/accounts/7/priority", nil)
	getRequest.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	router.ServeHTTP(get, getRequest)
	require.Equal(t, http.StatusOK, get.Code)
	require.Contains(t, get.Body.String(), `"priority":5`)

	body := []byte(`{"expected_priority":5,"target_priority":50}`)
	put := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/accounts/7/priority", bytes.NewReader(body))
	putRequest.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	putRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(put, putRequest)
	require.Equal(t, http.StatusOK, put.Code)
	require.Equal(t, 1, adminSvc.relayMonitorPriorityCASCalls)

	adminSvc.getAccountResult.Priority = 50
	idempotent := httptest.NewRecorder()
	idempotentRequest := httptest.NewRequest(http.MethodPut, "/accounts/7/priority", bytes.NewReader(body))
	idempotentRequest.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	idempotentRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(idempotent, idempotentRequest)
	require.Equal(t, http.StatusOK, idempotent.Code)
	require.Equal(t, 2, adminSvc.relayMonitorPriorityCASCalls)

	conflict := httptest.NewRecorder()
	conflictRequest := httptest.NewRequest(http.MethodPut, "/accounts/7/priority", bytes.NewReader([]byte(`{"expected_priority":5,"target_priority":60}`)))
	conflictRequest.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	conflictRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(conflict, conflictRequest)
	require.Equal(t, http.StatusConflict, conflict.Code)
}

func TestRelayMonitorPriorityCapPauseUsesNarrowServiceOperation(t *testing.T) {
	t.Setenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET", "0123456789abcdef0123456789abcdef")
	adminSvc := newStubAdminService()
	adminSvc.relayCapPauseUntil = time.Date(2026, 8, 9, 12, 1, 0, 0, time.UTC)
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRelayMonitorAccountRoutes(router, h)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts/7/priority-cap-pause", bytes.NewReader([]byte(`{"duration_seconds":720}`)))
	request.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, adminSvc.relayCapPauseCalls)
	require.Equal(t, int64(7), adminSvc.relayCapPauseAccountID)
	require.Equal(t, 12*time.Minute, adminSvc.relayCapPauseDuration)
	require.JSONEq(t, `{"account_id":7,"temp_unschedulable_until":"2026-08-09T12:01:00Z"}`, recorder.Body.String())
}

func TestRelayMonitorPriorityCapPauseRejectsInvalidDuration(t *testing.T) {
	t.Setenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET", "0123456789abcdef0123456789abcdef")
	adminSvc := newStubAdminService()
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRelayMonitorAccountRoutes(router, h)

	for _, body := range []string{`{}`, `{"duration_seconds":59}`, `{"duration_seconds":3601}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/accounts/7/priority-cap-pause", bytes.NewReader([]byte(body)))
		request.Header.Set("X-Sub2API-Relay-Monitor-Secret", "0123456789abcdef0123456789abcdef")
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code, body)
	}
	require.Zero(t, adminSvc.relayCapPauseCalls)
}

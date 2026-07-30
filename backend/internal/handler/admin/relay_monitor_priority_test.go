package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRelayMonitorPriorityEndpointRequiresSecretAndMinimumOne(t *testing.T) {
	t.Setenv("SUB2API_RELAY_MONITOR_PRIORITY_SECRET", "0123456789abcdef0123456789abcdef")
	adminSvc := newStubAdminService()
	adminSvc.getAccountResult = &service.Account{ID: 7, Name: "a", Priority: 5}
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/accounts/:id/priority", h.GetRelayMonitorPriority)
	router.PUT("/accounts/:id/priority", h.SetRelayMonitorPriority)

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/accounts/7/priority", nil))
	require.Equal(t, http.StatusUnauthorized, unauthorized.Code)

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
	router.GET("/accounts/:id/priority", h.GetRelayMonitorPriority)
	router.PUT("/accounts/:id/priority", h.SetRelayMonitorPriority)

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

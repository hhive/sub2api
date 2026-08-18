package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type vendorHallAdminServiceStub struct {
	*stubAdminService
	pauseCalls int
	pauseID    int64
	until      time.Time
}

func (s *vendorHallAdminServiceStub) PauseVendorHallAccount(_ context.Context, id int64) (time.Time, error) {
	s.pauseCalls++
	s.pauseID = id
	return s.until, nil
}

func TestPauseVendorHallAccountUsesFixedNarrowOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	until := time.Date(2026, 8, 17, 13, 0, 0, 0, time.UTC)
	adminService := &vendorHallAdminServiceStub{stubAdminService: newStubAdminService(), until: until}
	h := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/accounts/:id/pause-scheduling", h.PauseVendorHallAccount)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/accounts/7/pause-scheduling", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, adminService.pauseCalls)
	require.Equal(t, int64(7), adminService.pauseID)
	require.Contains(t, recorder.Body.String(), `"temp_unschedulable_until":"2026-08-17T13:00:00Z"`)
}

func TestPauseVendorHallAccountRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := &vendorHallAdminServiceStub{stubAdminService: newStubAdminService()}
	h := NewAccountHandler(adminService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.POST("/accounts/:id/pause-scheduling", h.PauseVendorHallAccount)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/accounts/nope/pause-scheduling", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, adminService.pauseCalls)
}

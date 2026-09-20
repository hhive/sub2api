package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/vendorhall"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type vendorHallListerStub struct {
	result *vendorhall.ListResult
	err    error
	params vendorhall.ListParams
}

func (s *vendorHallListerStub) List(_ context.Context, params vendorhall.ListParams) (*vendorhall.ListResult, error) {
	s.params = params
	return s.result, s.err
}

func TestListVendorHallAccountsValidatesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOpsHandler(nil)
	h.SetVendorHallLister(&vendorHallListerStub{})
	router := gin.New()
	router.GET("/vendor-hall", h.ListVendorHallAccounts)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vendor-hall?window=1h", nil))

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	for _, query := range []string{"account_id=0", "account_id=abc", "account_id=-3"} {
		recorder = httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vendor-hall?"+query, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code, query)
	}
}

func TestListVendorHallAccountsMapsUnavailableWithoutLeakingDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewOpsHandler(nil)
	h.SetVendorHallLister(&vendorHallListerStub{err: errors.Join(vendorhall.ErrUnavailable, errors.New("postgres://secret@host/db"))})
	router := gin.New()
	router.GET("/vendor-hall", h.ListVendorHallAccounts)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vendor-hall", nil))

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "postgres://")
}

func TestListVendorHallAccountsReturnsParsedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &vendorHallListerStub{result: &vendorhall.ListResult{Items: []vendorhall.Account{}, Total: 0, Page: 2, PageSize: 10}}
	h := NewOpsHandler(nil)
	h.SetVendorHallLister(stub)
	router := gin.New()
	router.GET("/vendor-hall", h.ListVendorHallAccounts)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vendor-hall?window=3d&page=2&page_size=10&sort_by=user_ttft&sort_order=asc&account_id=7&platform=openai&group=3", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "3d", stub.params.Window)
	require.Equal(t, "user_ttft", stub.params.SortBy)
	require.Equal(t, int64(7), stub.params.AccountID)
	require.Equal(t, "openai", stub.params.Platform)
	require.Equal(t, "3", stub.params.Group)
	require.Contains(t, recorder.Body.String(), `"page":2`)
}

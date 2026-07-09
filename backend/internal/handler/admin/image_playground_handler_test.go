package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestImagePlaygroundHandlerListProbeRunsProxiesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotQuery string
	t.Setenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL", "http://image-playground.test")

	handler := NewImagePlaygroundHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		require.Equal(t, http.MethodGet, r.Method)
		var body bytes.Buffer
		_ = json.NewEncoder(&body).Encode(map[string]any{
			"items":     []map[string]any{{"id": 9, "run_id": "run-1"}},
			"total":     1,
			"page":      2,
			"page_size": 20,
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body.Bytes())),
			Header:     make(http.Header),
		}, nil
	})}
	router := gin.New()
	router.GET("/model-probe-runs", handler.ListProbeRuns)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/model-probe-runs?page=2&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/model-probe-runs", gotPath)
	require.Equal(t, "page=2&page_size=20", gotQuery)
	require.Contains(t, rec.Body.String(), `"run_id":"run-1"`)
}

func TestImagePlaygroundHandlerListUpstreamRequestsProxiesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotQuery string
	t.Setenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL", "http://image-playground.test")

	handler := NewImagePlaygroundHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		require.Equal(t, http.MethodGet, r.Method)
		body := []byte(`{"items":[{"id":"task-1","model":"gpt-image-2"}],"total":1,"page":3,"page_size":20}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	router := gin.New()
	router.GET("/upstream-requests", handler.ListUpstreamRequests)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/upstream-requests?page=3&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/upstream-requests", gotPath)
	require.Equal(t, "page=3&page_size=20", gotQuery)
	require.Contains(t, rec.Body.String(), `"task-1"`)
}

func TestImagePlaygroundHandlerRunProbeProxiesPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath string
	t.Setenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL", "http://image-playground.test")

	handler := NewImagePlaygroundHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		require.Equal(t, http.MethodPost, r.Method)
		body := []byte(`{"ok":true,"running":true}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	router := gin.New()
	router.POST("/model-probe-runs/run", handler.RunProbe)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/model-probe-runs/run", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/model-probe-runs/run", gotPath)
	require.Contains(t, rec.Body.String(), `"running":true`)
}

func TestImagePlaygroundHandlerRunModelProbeProxiesPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath string
	t.Setenv("IMAGE_PLAYGROUND_ADMIN_BASE_URL", "http://image-playground.test")

	handler := NewImagePlaygroundHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		require.Equal(t, http.MethodPost, r.Method)
		body := []byte(`{"ok":true,"running":true}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	router := gin.New()
	router.POST("/models/:id/probe", handler.RunModelProbe)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/models/7/probe", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/models/7/probe", gotPath)
	require.Contains(t, rec.Body.String(), `"running":true`)
}

func TestImagePlaygroundHandlerRunModelProbeRejectsInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewImagePlaygroundHandler(nil)
	router := gin.New()
	router.POST("/models/:id/probe", handler.RunModelProbe)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/models/bad/probe", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

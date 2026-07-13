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

func TestMediaPlaygroundVideoHandlerListUpstreamRequestsProxiesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotQuery string
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://video-playground.test")

	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		require.Equal(t, http.MethodGet, r.Method)
		body := []byte(`{"items":[{"id":7,"task_id":"task-1","model":"seedance"}],"total":1,"page":2,"page_size":20}`)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	router := gin.New()
	router.GET("/upstream-requests", handler.ListUpstreamRequests)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/upstream-requests?page=2&page_size=20", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/media/video/upstream-requests", gotPath)
	require.Equal(t, "page=2&page_size=20", gotQuery)
	require.Contains(t, rec.Body.String(), `"task-1"`)
}

func TestMediaPlaygroundVideoHandlerListTasksProxiesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotQuery string
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://video-playground.test")
	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"items":[],"total":0,"page":1,"page_size":20}`)), Header: make(http.Header)}, nil
	})}
	router := gin.New()
	router.GET("/tasks", handler.ListTasks)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks?status=failed&model=seedance-2.0-mini&page=2", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/media/video/tasks", gotPath)
	require.Equal(t, "status=failed&model=seedance-2.0-mini&page=2", gotQuery)
}

func TestMediaPlaygroundVideoHandlerGetTaskEscapesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath string
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://video-playground.test")
	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.EscapedPath()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"task":{}}`)), Header: make(http.Header)}, nil
	})}
	router := gin.New()
	router.GET("/tasks/:id", handler.GetTask)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/task-1", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/media/video/tasks/task-1", gotPath)
}

func TestMediaPlaygroundVideoHandlerPrefersAdminBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "https://public-media.test")
	t.Setenv("MEDIA_PLAYGROUND_ADMIN_BASE_URL", "http://127.0.0.1:3304")
	var gotURL string
	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotURL = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`[]`)), Header: make(http.Header)}, nil
	})}
	router := gin.New()
	router.GET("/models", handler.ListModels)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/models", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "http://127.0.0.1:3304/api/admin/media/video/models", gotURL)
}

func TestMediaPlaygroundVideoHandlerCreateModelForwardsAPIMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath string
	var gotBody map[string]any
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://media-playground.test")
	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"id":1}`)), Header: make(http.Header)}, nil
	})}
	router := gin.New()
	router.POST("/models", handler.CreateModel)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/models", bytes.NewBufferString(`{"display_name":"Video","model":"v1","provider_name":"custom","api_mode":"openai_videos_v2","upstream_base_url":"https://example.com","upstream_api_key":"","price_quota":1,"billing_mode":"balance_prepaid","refund_enabled":true,"timeout_seconds":60,"enabled":true,"sort_order":0}`))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "/api/admin/media/video/models", gotPath)
	require.Equal(t, "openai_videos_v2", gotBody["api_mode"])
}

func TestMediaPlaygroundVideoHandlerUpdateModelForwardsAPIMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotMethod, gotPath string
	var gotBody map[string]any
	t.Setenv("MEDIA_PLAYGROUND_BASE_URL", "http://media-playground.test")
	handler := NewMediaPlaygroundVideoHandler(nil)
	handler.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`{"id":7}`)), Header: make(http.Header)}, nil
	})}
	router := gin.New()
	router.PATCH("/models/:id", handler.UpdateModel)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/models/7", bytes.NewBufferString(`{"display_name":"Video","model":"v1","provider_name":"custom","api_mode":"openai_videos_v2","upstream_base_url":"https://example.com","upstream_api_key":"","price_quota":1,"billing_mode":"balance_prepaid","refund_enabled":true,"timeout_seconds":60,"enabled":false,"sort_order":0}`))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, http.MethodPatch, gotMethod)
	require.Equal(t, "/api/admin/media/video/models/7", gotPath)
	require.Equal(t, "openai_videos_v2", gotBody["api_mode"])
	require.Equal(t, false, gotBody["enabled"])
}

func TestMediaPlaygroundVideoHandlerRejectsReadOnlyAndDeprecatedFields(t *testing.T) {
	for _, field := range []string{"media_type", "upstream_api_key_mask", "studio_model_id", "model_kind", "input_schema_json", "payload_mapping_json"} {
		t.Run(field, func(t *testing.T) {
			handler := NewMediaPlaygroundVideoHandler(nil)
			router := gin.New()
			router.POST("/models", handler.CreateModel)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/models", bytes.NewBufferString(`{"`+field+`":"value"}`))
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestMediaPlaygroundVideoHandlerRejectsMissingAPIMode(t *testing.T) {
	handler := NewMediaPlaygroundVideoHandler(nil)
	router := gin.New()
	router.POST("/models", handler.CreateModel)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/models", bytes.NewBufferString(`{"display_name":"Video"}`))
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "api_mode is required")
}

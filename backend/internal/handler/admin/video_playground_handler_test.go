package admin

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVideoPlaygroundHandlerListUpstreamRequestsProxiesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotPath, gotQuery string
	t.Setenv("VIDEO_PLAYGROUND_ADMIN_BASE_URL", "http://video-playground.test")

	handler := NewVideoPlaygroundHandler(nil)
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
	require.Equal(t, "/api/admin/upstream-requests", gotPath)
	require.Equal(t, "page=2&page_size=20", gotQuery)
	require.Contains(t, rec.Body.String(), `"task-1"`)
}

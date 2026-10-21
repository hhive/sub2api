//go:build embed

package web

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type landingSettingsStub struct{}

func (landingSettingsStub) GetPublicSettingsForInjection(context.Context) (any, error) {
	return map[string]any{"site_name": "landing-test"}, nil
}

func newLandingTestServer(t *testing.T, distFS fs.FS) *FrontendServer {
	t.Helper()

	cache := NewHTMLCache()
	baseHTML := []byte("<html><body>spa-index</body></html>")
	cache.SetBaseHTML(baseHTML)

	return &FrontendServer{
		distFS:     distFS,
		fileServer: http.FileServer(http.FS(distFS)),
		baseHTML:   baseHTML,
		cache:      cache,
		settings:   landingSettingsStub{},
	}
}

func landingTestRouter(server *FrontendServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// 模拟 SecurityHeaders 的全局默认头
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		c.Next()
	})
	router.Use(server.Middleware())
	router.GET("/home", func(c *gin.Context) {
		c.String(http.StatusOK, "home-route")
	})

	return router
}

func TestIsLandingPagePath(t *testing.T) {
	cases := map[string]bool{
		"/landing":                true,
		"/landing/":               true,
		"/landingx":               false,
		"/landing/app.css":        false,
		"/landing/index.html":     false,
		"/":                       false,
		"/home":                   false,
		"/api/v1/settings/public": false,
	}

	for path, expected := range cases {
		assert.Equal(t, expected, isLandingPagePath(path), "path %q", path)
	}
}

func TestRelaxFrameAncestorsToSelf(t *testing.T) {
	t.Run("rewrites_frame_ancestors_and_keeps_other_directives", func(t *testing.T) {
		policy := "default-src 'self'; script-src 'self' 'nonce-abc'; frame-ancestors 'none'; base-uri 'self'"
		got := relaxFrameAncestorsToSelf(policy)

		assert.Contains(t, got, "frame-ancestors 'self'")
		assert.NotContains(t, got, "frame-ancestors 'none'")
		assert.Equal(t,
			"default-src 'self'; script-src 'self' 'nonce-abc'; frame-ancestors 'self'; base-uri 'self'",
			got)
	})

	t.Run("keeps_other_frame_origins_out_of_landing_policy", func(t *testing.T) {
		policy := "frame-ancestors 'none' https://stripe.example"
		assert.Equal(t, "frame-ancestors 'self'", relaxFrameAncestorsToSelf(policy))
	})

	t.Run("returns_policy_unchanged_when_directive_missing", func(t *testing.T) {
		policy := "default-src 'self'; script-src 'self'"
		assert.Equal(t, policy, relaxFrameAncestorsToSelf(policy))
	})

	t.Run("handles_empty_policy", func(t *testing.T) {
		assert.Equal(t, "", relaxFrameAncestorsToSelf(""))
	})
}

func TestLandingPageResponseAllowsSameOriginFraming(t *testing.T) {
	for _, path := range []string{"/landing", "/landing/"} {
		distFS := fstest.MapFS{
			"landing/index.html": &fstest.MapFile{Data: []byte("<html><body>brand-landing</body></html>")},
		}
		router := landingTestRouter(newLandingTestServer(t, distFS))

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code, "path %q", path)
		assert.Contains(t, recorder.Body.String(), "brand-landing")
		assert.Equal(t, "SAMEORIGIN", recorder.Header().Get("X-Frame-Options"))
		assert.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))
		assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'self'")
		assert.NotContains(t, recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'")
		assert.Equal(t, "text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	}
}

func TestLandingPageMissingFallsBackToSpaIndex(t *testing.T) {
	distFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><body>spa-index</body></html>")},
	}
	router := landingTestRouter(newLandingTestServer(t, distFS))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/landing/", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "spa-index")
	// 缺失落地页时不得放宽嵌入头
	assert.Equal(t, "DENY", recorder.Header().Get("X-Frame-Options"))
	assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'")
}

func TestNonLandingPathKeepsDenyHeaders(t *testing.T) {
	distFS := fstest.MapFS{
		"landing/index.html": &fstest.MapFile{Data: []byte("<html><body>brand-landing</body></html>")},
	}
	router := landingTestRouter(newLandingTestServer(t, distFS))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/home", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "DENY", recorder.Header().Get("X-Frame-Options"))
	assert.Contains(t, recorder.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'")
}

func TestServeLandingPageFromFSReturnsFalseWithoutFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/landing/", nil)

	served := serveLandingPageFromFS(c, fstest.MapFS{})
	assert.False(t, served)
	assert.Empty(t, recorder.Body.String())
	assert.Empty(t, recorder.Header().Get("X-Frame-Options"))
}

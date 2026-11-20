//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIntegrationLaunchFormCSP(t *testing.T) {
	const launchURL = "https://app.example.org:8443/api/sub2api/launch"
	cases := []struct {
		name    string
		policy  string
		targets []string
		allowed bool
	}{
		{name: "enabled with default CSP", targets: []string{launchURL}, allowed: true},
		{name: "empty registry"},
		{name: "operator explicitly denies forms", policy: "default-src 'self'; form-action 'none'", targets: []string{launchURL}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SecurityHeaders(config.CSPConfig{Enabled: true, Policy: tc.policy}, nil, tc.targets...))
			r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			policy := w.Header().Get("Content-Security-Policy")
			require.Equal(t, tc.allowed, directiveHasValue(policy, "form-action", launchURL))
			require.False(t, directiveHasValue(policy, "form-action", "https://app.example.org:8443"))
			require.False(t, directiveHasValue(policy, "frame-src", launchURL))
			if tc.policy == "" {
				require.True(t, directiveHasValue(policy, "form-action", "'self'"))
			}
		})
	}
}

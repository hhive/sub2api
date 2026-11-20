//go:build unit

package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserAppIntegrationRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	t.Setenv("USER_APP_INTEGRATIONS_CONFIG_PATH", path)
	t.Setenv("CHAT_INTEGRATION_APP_ID", "")
	t.Setenv("APP_SECRET", "synthetic-secret-at-least-32-bytes")
	valid := `{"app_id":"example-app","enabled":true,"launch_url":"https://example.test/api/launch","secret_env":"APP_SECRET","scopes":["keys:read"]}`
	for name, value := range map[string]string{
		"unknown scope":           strings.Replace(valid, "keys:read", "admin:read", 1),
		"unknown field":           strings.Replace(valid, `"enabled"`, `"active"`, 1),
		"invalid URL":             strings.Replace(valid, "https:", "http:", 1),
		"CSP wildcard":            strings.Replace(valid, "example.test", "*.example.test", 1),
		"CSP directive":           strings.Replace(valid, "/launch", "/launch;form-action", 1),
		"CSP directory":           strings.Replace(valid, "/launch", "/launch/", 1),
		"URL query":               strings.Replace(valid, "/launch", "/launch?next=other", 1),
		"missing scopes":          strings.Replace(valid, `,"scopes":["keys:read"]`, "", 1),
		"missing secret":          strings.Replace(valid, "APP_SECRET", "UNCONFIGURED_TEST_APP_SECRET", 1),
		"duplicate app":           valid + "," + valid,
		"shared secret reference": valid + "," + strings.Replace(valid, "example-app", "other-app", 1),
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(path, []byte("["+value+"]"), 0600))
			_, err := NewUserAppIntegrationService(nil, nil)
			require.Error(t, err)
		})
	}
	require.NoError(t, os.WriteFile(path, []byte("["+valid+"]"), 0600))
	t.Setenv("CHAT_INTEGRATION_APP_ID", "unknown")
	_, err := NewUserAppIntegrationService(nil, nil)
	require.Error(t, err)
	t.Setenv("CHAT_INTEGRATION_APP_ID", "example-app")
	integration, err := NewUserAppIntegrationService(nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"https://example.test/api/launch"}, integration.LaunchFormActions())
}

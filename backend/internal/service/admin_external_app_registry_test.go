//go:build unit

package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAdminExternalAppRegistry_MissingIsEmpty(t *testing.T) {
	r, err := LoadAdminExternalAppRegistry(filepath.Join(t.TempDir(), "missing.json"))
	require.NoError(t, err)
	require.Empty(t, r.List())
}

func TestLoadAdminExternalAppRegistry_ValidAndSecretIsNotExposed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apps.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/api/sub2api/admin-launch","exchange_secret":"0123456789abcdef0123456789abcdef","enabled":true,"sort_order":10}]`), 0600))
	r, err := LoadAdminExternalAppRegistry(path)
	require.NoError(t, err)
	require.Len(t, r.List(), 1)
	require.NotContains(t, r.List()[0].LabelEN, "0123456789abcdef")
}

func TestLoadAdminExternalAppRegistry_InvalidDoesNotLeakSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apps.json")
	secret := "short-secret"
	require.NoError(t, os.WriteFile(path, []byte(`[{"app_id":"Bad_ID","label_en":"Media","label_zh":"媒体","launch_url":"http://evil.example.com/launch","exchange_secret":"`+secret+`","enabled":true}]`), 0600))
	_, err := LoadAdminExternalAppRegistry(path)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}

func TestLoadAdminExternalAppRegistry_RejectsUnsafeConfigurations(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	tests := map[string]string{
		"duplicate app id":  `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/launch","exchange_secret":"` + secret + `","enabled":true},{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/launch","exchange_secret":"` + secret + `","enabled":true}]`,
		"weak secret":       `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/launch","exchange_secret":"weak","enabled":true}]`,
		"non https":         `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"http://media.example.com/launch","exchange_secret":"` + secret + `","enabled":true}]`,
		"root path":         `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/","exchange_secret":"` + secret + `","enabled":true}]`,
		"protocol relative": `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"//media.example.com/launch","exchange_secret":"` + secret + `","enabled":true}]`,
		"userinfo":          `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://user@media.example.com/launch","exchange_secret":"` + secret + `","enabled":true}]`,
		"query":             `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/launch?next=evil","exchange_secret":"` + secret + `","enabled":true}]`,
		"fragment":          `[{"app_id":"media-management","label_en":"Media","label_zh":"媒体","launch_url":"https://media.example.com/launch#token","exchange_secret":"` + secret + `","enabled":true}]`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "apps.json")
			require.NoError(t, os.WriteFile(path, []byte(body), 0600))
			_, err := LoadAdminExternalAppRegistry(path)
			require.Error(t, err)
			require.NotContains(t, err.Error(), secret)
		})
	}
}

func TestLoadAdminExternalAppRegistry_RejectsBroadPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apps.json")
	require.NoError(t, os.WriteFile(path, []byte(`[]`), 0644))
	_, err := LoadAdminExternalAppRegistry(path)
	require.ErrorContains(t, err, "0640")
}

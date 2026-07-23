package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration186RemovesOnlyLobeHubAllowedEmailsSetting(t *testing.T) {
	content, err := FS.ReadFile("186_remove_lobehub_allowed_emails.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DELETE FROM settings WHERE key = 'lobehub_allowed_emails';")
	require.NotContains(t, sql, "DELETE FROM settings;")
}

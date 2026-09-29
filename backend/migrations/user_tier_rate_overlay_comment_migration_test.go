package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserTierRateOverlayCommentMigrationUpdatesComment 断言 245 只做「改注释」这一件事，
// 且新注释不再宣称「不写入 user_group_rate_multipliers」（那是已被取代的旧设计）。
func TestUserTierRateOverlayCommentMigrationUpdatesComment(t *testing.T) {
	content, err := FS.ReadFile("245_update_user_tier_rate_overlay_comment.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "COMMENT ON TABLE user_tier_rate_overlays IS")
	require.Contains(t, sql, "user_group_rate_multipliers")
	// 新注释正文（区别于文件头部的「变更前」说明）必须写明本表只是来源登记
	require.Contains(t, sql, "COMMENT ON TABLE user_tier_rate_overlays IS '等级倍率的来源登记")
	require.NotContains(t, sql, "COMMENT ON TABLE user_tier_rate_overlays IS '等级倍率覆盖层")
}

// TestUserTierRateOverlayCommentMigrationIsNonDestructive 断言本迁移不改结构、不改数据。
func TestUserTierRateOverlayCommentMigrationIsNonDestructive(t *testing.T) {
	content, err := FS.ReadFile("245_update_user_tier_rate_overlay_comment.sql")
	require.NoError(t, err)

	upper := strings.ToUpper(strings.Join(strings.Fields(string(content)), " "))

	for _, forbidden := range []string{
		"ALTER TABLE", "DROP TABLE", "DROP COLUMN", "CREATE TABLE", "CREATE INDEX",
		"INSERT INTO", "UPDATE ", "DELETE FROM", "TRUNCATE",
	} {
		require.NotContains(t, upper, forbidden, "245 只应改注释，不得出现 %s", forbidden)
	}
}

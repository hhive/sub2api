package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAddExpiredAmountMigrationIsAdditiveOnly 断言 242 只加列：
// 不动既有列、不做回填、不删列（升级路径必须对既有行逐位无影响）。
func TestAddExpiredAmountMigrationIsAdditiveOnly(t *testing.T) {
	content, err := FS.ReadFile("242_add_expired_amount_to_balance_credits.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS expired_amount NUMERIC(20, 8)")
	require.Contains(t, sql, "COMMENT ON COLUMN user_balance_credits.expired_amount IS")

	// 只加列：不得出现删列/改类型/回填
	require.NotContains(t, sql, "DROP COLUMN")
	require.NotContains(t, sql, "ALTER COLUMN")
	require.NotContains(t, sql, "UPDATE user_balance_credits")
	require.NotContains(t, sql, "DELETE FROM")

	// 发放状态的文档口径同步：现行实现是失败整体回滚，不再写 failed
	require.Contains(t, sql, "COMMENT ON COLUMN user_tier_effects.status IS")
}

// TestUserTiersMigrationStillDeclaresFailedStatus 242 不改约束取值，
// 只改文档口径 —— 状态机取值保持向后兼容（历史行可能仍带 failed）。
func TestUserTiersMigrationStillDeclaresFailedStatus(t *testing.T) {
	content, err := FS.ReadFile("241_add_user_tiers.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CHECK (status IN ('pending', 'applied', 'failed'))")
}

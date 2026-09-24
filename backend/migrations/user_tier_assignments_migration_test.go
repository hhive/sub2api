package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserTierAssignmentsMigrationCreatesTable 断言指派表、主键、来源约束与索引都在迁移里，
// 且 user_id 作主键即「一人一条、重指派覆盖」的幂等锚点。
func TestUserTierAssignmentsMigrationCreatesTable(t *testing.T) {
	content, err := FS.ReadFile("244_add_user_tier_assignments.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS user_tier_assignments")
	require.Contains(t, sql, "COMMENT ON TABLE user_tier_assignments")

	// 一人一条：user_id 主键
	require.Contains(t, sql, "user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE")

	// 目标档位可空（配置删除后保留快照），来源受约束
	require.Contains(t, sql, "tier_id BIGINT REFERENCES user_tiers(id) ON DELETE SET NULL")
	require.Contains(t, sql, "CHECK (source IN ('admin', 'historical_20260924'))")

	// 阶梯位置兜底与审计字段
	require.Contains(t, sql, "sort_order_snapshot INTEGER")
	require.Contains(t, sql, "tier_code VARCHAR(64) NOT NULL")
	require.Contains(t, sql, "assigned_by BIGINT")
	require.Contains(t, sql, "assigned_at TIMESTAMPTZ")

	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_user_tier_assignments_tier ON user_tier_assignments (tier_id)")
}

// TestUserTierAssignmentsMigrationDoesNotTouchExistingTables 断言本迁移不改既有表结构、
// 不做破坏性操作，也不导入任何数据（回滚策略是保留新表 + 关闭入口）。
func TestUserTierAssignmentsMigrationDoesNotTouchExistingTables(t *testing.T) {
	content, err := FS.ReadFile("244_add_user_tier_assignments.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upper := strings.ToUpper(sql)

	for _, existing := range []string{
		"ALTER TABLE USERS",
		"ALTER TABLE USER_TIERS",
		"ALTER TABLE USER_TIER_BENEFITS",
		"ALTER TABLE USER_TIER_AWARDS",
		"ALTER TABLE USER_TIER_EFFECTS",
		"ALTER TABLE USER_TIER_RATE_OVERLAYS",
		"ALTER TABLE USER_BALANCE_CREDITS",
		"ALTER TABLE USER_GROUP_RATE_MULTIPLIERS",
		"ALTER TABLE GROUPS",
		"ALTER TABLE USER_SUBSCRIPTIONS",
	} {
		require.NotContains(t, upper, existing)
	}

	require.NotContains(t, upper, "DROP TABLE")
	require.NotContains(t, upper, "DROP COLUMN")

	// 迁移只做 DDL：指派数据由管理端接口或一次性脚本写入
	require.NotContains(t, upper, "INSERT INTO")
	require.NotContains(t, upper, "UPDATE USER_TIER")
}

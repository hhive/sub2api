package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserTiersMigrationCreatesFiveTables 断言五张新表、三组幂等唯一约束、
// 状态/类型约束与索引都在迁移里，且不触碰任何既有表结构。
func TestUserTiersMigrationCreatesFiveTables(t *testing.T) {
	content, err := FS.ReadFile("241_add_user_tiers.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	for _, table := range []string{
		"user_tiers",
		"user_tier_benefits",
		"user_tier_awards",
		"user_tier_effects",
		"user_tier_rate_overlays",
	} {
		require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
		require.Contains(t, sql, "COMMENT ON TABLE "+table)
	}

	// 三组幂等锚点
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_awards_user_tier ON user_tier_awards (user_id, tier_code)")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_effects_award_benefit ON user_tier_effects (award_id, benefit_key)")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tier_rate_overlays_unique ON user_tier_rate_overlays (user_id, group_id, effect_id)")

	// 状态与类型值约束
	require.Contains(t, sql, "CHECK (trigger_type IN ('first_recharge', 'consumption'))")
	require.Contains(t, sql, "CHECK (benefit_type IN ('balance_credit', 'group_rate'))")
	require.Contains(t, sql, "CHECK (status IN ('pending', 'applied', 'failed'))")
	require.Contains(t, sql, "CHECK (status IN ('active', 'revoked'))")
	require.Contains(t, sql, "CHECK (rate_multiplier > 0)")
	require.Contains(t, sql, "CHECK (source IN ('claim', 'first_recharge', 'manual_20260924', 'backfill'))")

	// consumption 档必须有阈值，first_recharge 档必须没有阈值
	require.Contains(t, sql, "trigger_type = 'consumption' AND threshold_usd IS NOT NULL AND threshold_usd >= 0")
	require.Contains(t, sql, "trigger_type = 'first_recharge' AND threshold_usd IS NULL")

	// 唯一 code、覆盖层有效索引
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tiers_code ON user_tiers (code)")
	require.Contains(t, sql, "WHERE status = 'active'")
}

// TestUserTiersMigrationDoesNotTouchExistingTables 断言本迁移不改既有表结构，
// 也不做破坏性操作（回滚策略是保留新表 + 关闭入口）。
func TestUserTiersMigrationDoesNotTouchExistingTables(t *testing.T) {
	content, err := FS.ReadFile("241_add_user_tiers.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	upper := strings.ToUpper(sql)

	for _, existing := range []string{
		"ALTER TABLE USERS",
		"ALTER TABLE USER_BALANCE_CREDITS",
		"ALTER TABLE USER_GROUP_RATE_MULTIPLIERS",
		"ALTER TABLE GROUPS",
		"ALTER TABLE USER_SUBSCRIPTIONS",
	} {
		require.NotContains(t, upper, existing)
	}

	require.NotContains(t, upper, "DROP TABLE")
	require.NotContains(t, upper, "DROP COLUMN")

	// 等级倍率只写覆盖层：不向既有手工倍率表插入/更新
	require.NotContains(t, sql, "INSERT INTO user_group_rate_multipliers")
	require.NotContains(t, sql, "UPDATE user_group_rate_multipliers")

	// 注：本迁移不做数据导入。档位基线数据由上线脚本导入（见实施计划 C8），
	// 因此迁移文件只允许出现 DDL。
	require.NotContains(t, upper, "INSERT INTO USER_TIERS")
	require.NotContains(t, upper, "INSERT INTO USER_TIER_BENEFITS")
}

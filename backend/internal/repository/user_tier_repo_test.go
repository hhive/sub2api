package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func newUserTierRepoForTest(t *testing.T) (*userTierRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	return &userTierRepository{client: client, db: db}, mock
}

// TestUserTierRepoUpdateTierPersistsCode 守住一处真实缺陷：UPDATE 原先漏写 code，
// 服务层与前端都以为改名成功（响应体返回新 code），库里却仍是旧 code。
// 七个参数逐个钉住，漏写 code 会立刻让 WithArgs 数量不符而失败。
func TestUserTierRepoUpdateTierPersistsCode(t *testing.T) {
	repo, mock := newUserTierRepoForTest(t)

	threshold := 500.0
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)UPDATE user_tiers SET name = \$2, description = \$3, trigger_type = \$4, threshold_usd = \$5, enabled = \$6, code = \$7, updated_at = NOW\(\) WHERE id = \$1`).
		WithArgs(int64(7), "常客", "", service.UserTierTriggerConsumption, threshold, true, "consume_500_v2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// Benefits 为 nil：本次只验等级行的写入，不代表生产调用形状
	err := repo.UpdateTier(context.Background(), &service.UserTier{
		ID:           7,
		Code:         "consume_500_v2",
		Name:         "常客",
		TriggerType:  service.UserTierTriggerConsumption,
		ThresholdUSD: &threshold,
		Enabled:      true,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserTierRepoConsumedAmountExcludesExpiredAmount 守住消费口径：额度到期作废时
// remaining_amount 被置 0，若口径不减 expired_amount，未使用的部分会被整额算成「已消费」。
func TestUserTierRepoConsumedAmountExcludesExpiredAmount(t *testing.T) {
	repo, mock := newUserTierRepoForTest(t)

	mock.ExpectQuery(`(?s)SELECT COALESCE\(sum\(amount - remaining_amount - COALESCE\(expired_amount, 0\)\), 0\) FROM user_balance_credits WHERE user_id = \$1 AND source_type = 'redeem'`).
		WithArgs(int64(41)).
		WillReturnRows(sqlmock.NewRows([]string{"consumed"}).AddRow(120.5))

	consumed, err := repo.GetUserConsumedAmount(context.Background(), 41)
	require.NoError(t, err)
	require.InDelta(t, 120.5, consumed, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBalanceCreditExpiryPersistsExpiredAmount 守住修复的另一半：只加列不写值等于口径修正失效
// （expired_amount 恒为 NULL，等于没改）。到期任务必须把「到期那一刻仍未使用的余额」落库。
func TestBalanceCreditExpiryPersistsExpiredAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	expiresAt := time.Now().Add(-time.Hour)
	mock.ExpectQuery(`(?s)UPDATE user_balance_credits ubc SET status = 'expired', remaining_amount = 0, expired_amount = expired\.expired_amount, expired_at = \$1, updated_at = NOW\(\) FROM expired`).
		WithArgs(sqlmock.AnyArg(), 100, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expired_amount", "expires_at"}).
			AddRow(int64(9), int64(41), 300.0, expiresAt))

	repo := &balanceCreditRepository{db: db}
	expired, err := repo.ExpireDueCredits(context.Background(), time.Now(), time.Now(), 100)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.InDelta(t, 300.0, expired[0].Amount, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestUserTierRepoSetUserGroupRateMultiplierUpsertShape 守住倍率单一来源：等级倍率必须写进
// user_group_rate_multipliers（计费/用户端/管理端共用的唯一来源），且 SQL 形状与 main 的
// SyncUserGroupRates 单值 upsert 一致 —— SET 列表里只有 rate_multiplier 与 updated_at，
// 不得出现 rpm_override（写了就会把运营设的 RPM 上限抹掉）。
func TestUserTierRepoSetUserGroupRateMultiplierUpsertShape(t *testing.T) {
	repo, mock := newUserTierRepoForTest(t)

	mock.ExpectExec(`(?s)INSERT INTO user_group_rate_multipliers \(user_id, group_id, rate_multiplier, created_at, updated_at\)\s+VALUES \(\$1, \$2, \$3, NOW\(\), NOW\(\)\)\s+ON CONFLICT \(user_id, group_id\)\s+DO UPDATE SET rate_multiplier = EXCLUDED\.rate_multiplier, updated_at = EXCLUDED\.updated_at`).
		WithArgs(int64(612), int64(2), 1.08).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.SetUserGroupRateMultiplier(context.Background(), 612, 2, 1.08))
	require.NoError(t, mock.ExpectationsWereMet())
}

// 非正倍率直接拒绝且不写库（fail closed）：倍率表有 CHECK (rate_multiplier > 0) 的话
// 让它在 SQL 层报错会污染事务，不如在入口拦下。
func TestUserTierRepoSetUserGroupRateMultiplierRejectsNonPositive(t *testing.T) {
	repo, mock := newUserTierRepoForTest(t)

	require.Error(t, repo.SetUserGroupRateMultiplier(context.Background(), 612, 2, 0))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserTierRepoGetUserGroupRateMultiplier(t *testing.T) {
	repo, mock := newUserTierRepoForTest(t)

	mock.ExpectQuery(`(?s)SELECT rate_multiplier FROM user_group_rate_multipliers\s+WHERE user_id = \$1 AND group_id = \$2`).
		WithArgs(int64(612), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}).AddRow(1.08))

	rate, err := repo.GetUserGroupRateMultiplier(context.Background(), 612, 2)
	require.NoError(t, err)
	require.NotNil(t, rate)
	require.InDelta(t, 1.08, *rate, 1e-9)

	// 无行 → nil, nil（调用方据此判定「该组尚无倍率」）
	mock.ExpectQuery(`(?s)SELECT rate_multiplier FROM user_group_rate_multipliers`).
		WithArgs(int64(808), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}))

	rate, err = repo.GetUserGroupRateMultiplier(context.Background(), 808, 2)
	require.NoError(t, err)
	require.Nil(t, rate)

	// 有行但 rate 为 NULL（仅设了 rpm_override）→ nil, nil
	mock.ExpectQuery(`(?s)SELECT rate_multiplier FROM user_group_rate_multipliers`).
		WithArgs(int64(809), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"rate_multiplier"}).AddRow(nil))

	rate, err = repo.GetUserGroupRateMultiplier(context.Background(), 809, 2)
	require.NoError(t, err)
	require.Nil(t, rate)

	require.NoError(t, mock.ExpectationsWereMet())
}

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

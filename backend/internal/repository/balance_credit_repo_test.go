package repository

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBalanceCreditRepositoryCreateCreditSkipsNonPositive(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewBalanceCreditRepository(nil, db)
	err = repo.CreateCredit(context.Background(), service.BalanceCreditCreate{
		UserID: 1,
		Amount: 0,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositoryCreateCreditInsertsPositiveCredit(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expiresAt := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec("INSERT INTO user_balance_credits").
		WithArgs(int64(9), service.BalanceCreditSourceRedeem, "redeem-1", "CODE-1", 12.5, 12.5, expiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewBalanceCreditRepository(nil, db)
	err = repo.CreateCredit(context.Background(), service.BalanceCreditCreate{
		UserID:     9,
		SourceType: service.BalanceCreditSourceRedeem,
		SourceID:   "redeem-1",
		SourceCode: "CODE-1",
		Amount:     12.5,
		ExpiresAt:  &expiresAt,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositoryListDailyBalanceUsageAggregatesUsageLogs(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	start := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	mock.ExpectQuery("SELECT user_id, COALESCE").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "balance_cost"}).
			AddRow(int64(7), 6.5).
			AddRow(int64(9), 2.25))

	repo := NewBalanceCreditRepository(nil, db)
	usages, err := repo.ListDailyBalanceUsage(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, []service.DailyBalanceUsage{
		{UserID: 7, Amount: 6.5},
		{UserID: 9, Amount: 2.25},
	}, usages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositorySettleDailyUsageConsumesRowsInOrder(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	settlementDate := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	dayEnd := settlementDate.AddDate(0, 0, 1)
	rows := sqlmock.NewRows([]string{"id", "remaining_amount"}).
		AddRow(int64(1), 4.0).
		AddRow(int64(2), 10.0)
	mock.ExpectQuery("SELECT id, remaining_amount").
		WithArgs(int64(7), dayEnd, settlementDate).
		WillReturnRows(rows)
	mock.ExpectExec("UPDATE user_balance_credits").
		WithArgs(4.0, int64(1), settlementDate).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_balance_credits").
		WithArgs(2.5, int64(2), settlementDate).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_balance_credits").
		WithArgs(settlementDate, int64(1), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewBalanceCreditRepository(nil, db)
	err = repo.SettleDailyUsage(context.Background(), 7, 6.5, settlementDate, dayEnd)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositorySettleDailyUsageIgnoresMissingLedger(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	settlementDate := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	dayEnd := settlementDate.AddDate(0, 0, 1)
	mock.ExpectQuery("SELECT id, remaining_amount").
		WithArgs(int64(7), dayEnd, settlementDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "remaining_amount"}))

	repo := NewBalanceCreditRepository(nil, db)
	err = repo.SettleDailyUsage(context.Background(), 7, 6.5, settlementDate, dayEnd)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositoryExpireDueCreditsClaimsRows(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 5, 9, 1, 2, 3, 0, time.UTC)
	expiresAt := now.Add(-time.Minute)
	mock.ExpectQuery("UPDATE user_balance_credits").
		WithArgs(now, 50, now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "remaining_amount", "expires_at"}).
			AddRow(int64(3), int64(8), 2.75, expiresAt))

	repo := NewBalanceCreditRepository(nil, db)
	expired, err := repo.ExpireDueCredits(context.Background(), now, now, 50)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, int64(3), expired[0].ID)
	require.Equal(t, int64(8), expired[0].UserID)
	require.Equal(t, 2.75, expired[0].Amount)
	require.Equal(t, expiresAt, expired[0].ExpiresAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

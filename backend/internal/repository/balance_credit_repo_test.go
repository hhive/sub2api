package repository

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
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
		WithArgs(int64(9), "user@example.com", service.BalanceCreditSourceRedeem, "redeem-1", "CODE-1", 12.5, 12.5, expiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewBalanceCreditRepository(nil, db)
	err = repo.CreateCredit(context.Background(), service.BalanceCreditCreate{
		UserID:     9,
		Email:      "user@example.com",
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

func TestBalanceCreditRepositoryListUserCreditsReturnsPaginatedCredits(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	createdAt := time.Date(2026, 5, 9, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	expiresAt := createdAt.AddDate(0, 0, 30)
	settledUntil := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery("SELECT id,").
		WithArgs(int64(7), 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "source_type", "source_id", "source_code", "amount",
			"remaining_amount", "settled_until_date", "expires_at", "expired_at",
			"status", "created_at", "updated_at",
		}).AddRow(
			int64(3), int64(7), "user7@example.com", service.BalanceCreditSourceRedeem, "redeem-3", "CODE-3",
			12.5, 8.25, settledUntil, expiresAt, nil, service.BalanceCreditStatusActive,
			createdAt, updatedAt,
		))

	repo := NewBalanceCreditRepository(nil, db)
	credits, total, err := repo.ListUserCredits(context.Background(), 7, pagination.PaginationParams{Page: 2, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, credits, 1)
	require.Equal(t, int64(3), credits[0].ID)
	require.Equal(t, "user7@example.com", credits[0].Email)
	require.Equal(t, 12.5, credits[0].Amount)
	require.Equal(t, 8.25, credits[0].RemainingAmount)
	require.NotNil(t, credits[0].SettledUntilDate)
	require.NotNil(t, credits[0].ExpiresAt)
	require.Nil(t, credits[0].ExpiredAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBalanceCreditRepositoryListCreditsFiltersAndReturnsEmail(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	createdAt := time.Date(2026, 5, 9, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	expiresAt := createdAt.AddDate(0, 0, 180)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(7), service.BalanceCreditSourceRedeem, service.BalanceCreditStatusActive, "%alice%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery("SELECT COALESCE").
		WithArgs(int64(7), service.BalanceCreditSourceRedeem, service.BalanceCreditStatusActive, "%alice%").
		WillReturnRows(sqlmock.NewRows([]string{"total_amount", "today_amount", "total_remaining"}).AddRow(20.0, 8.0, 15.0))
	mock.ExpectQuery("SELECT id,").
		WithArgs(int64(7), service.BalanceCreditSourceRedeem, service.BalanceCreditStatusActive, "%alice%", 0, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "source_type", "source_id", "source_code", "amount",
			"remaining_amount", "settled_until_date", "expires_at", "expired_at",
			"status", "created_at", "updated_at",
		}).AddRow(
			int64(8), int64(7), "alice@example.com", service.BalanceCreditSourceRedeem, "88", "CODE-88",
			20.0, 15.0, nil, expiresAt, nil, service.BalanceCreditStatusActive,
			createdAt, updatedAt,
		))

	repo := NewBalanceCreditRepository(nil, db)
	credits, total, summary, err := repo.ListCredits(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, service.BalanceCreditListFilters{
		UserID:     7,
		Search:     "alice",
		SourceType: service.BalanceCreditSourceRedeem,
		Status:     service.BalanceCreditStatusActive,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, service.BalanceCreditListSummary{TotalAmount: 20.0, TodayAmount: 8.0, TotalRemaining: 15.0}, summary)
	require.Len(t, credits, 1)
	require.Equal(t, "alice@example.com", credits[0].Email)
	require.Equal(t, 15.0, credits[0].RemainingAmount)
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

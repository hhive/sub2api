package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetPublicUsageTotals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery("(?s)SUM\\(input_tokens \\+ output_tokens \\+ cache_creation_tokens \\+ cache_read_tokens\\)").
		WillReturnRows(sqlmock.NewRows([]string{"total_requests", "total_tokens"}).
			AddRow(int64(2853851), int64(318729031558)))

	requests, tokens, err := repo.GetPublicUsageTotals(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(2853851), requests)
	require.Equal(t, int64(318729031558), tokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPublicUsageTotalsErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &usageLogRepository{sql: db}
	mock.ExpectQuery("FROM usage_dashboard_daily").WillReturnError(errors.New("boom"))

	_, _, err = repo.GetPublicUsageTotals(context.Background())
	require.ErrorContains(t, err, "boom")

	var nilRepo *usageLogRepository
	_, _, err = nilRepo.GetPublicUsageTotals(context.Background())
	require.ErrorContains(t, err, "nil usage log repository")
}

func TestGetPublicStatsEmptyResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	usageRepo := &usageLogRepository{sql: db}
	mock.ExpectQuery("FROM usage_dashboard_daily").
		WillReturnRows(sqlmock.NewRows([]string{"total_requests", "total_tokens"}))
	_, _, err = usageRepo.GetPublicUsageTotals(context.Background())
	require.ErrorIs(t, err, sql.ErrNoRows)

	start := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	opsRepo := &opsRepository{db: db}
	mock.ExpectQuery("FROM ops_metrics_daily").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"success_count", "sla_total"}))
	_, _, err = opsRepo.GetPublicAvailability(context.Background(), start, end)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestGetPublicAvailability(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	start := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

	repo := &opsRepository{db: db}
	mock.ExpectQuery("(?s)platform IS NULL.*group_id IS NULL.*bucket_date >= \\$1::date.*bucket_date < \\$2::date").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"success_count", "sla_total"}).
			AddRow(int64(408394), int64(434281)))

	success, total, err := repo.GetPublicAvailability(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(408394), success)
	require.Equal(t, int64(434281), total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPublicAvailabilityErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &opsRepository{db: db}
	mock.ExpectQuery("FROM ops_metrics_daily").WillReturnError(errors.New("boom"))

	_, _, err = repo.GetPublicAvailability(context.Background(), time.Now(), time.Now())
	require.ErrorContains(t, err, "boom")

	var nilRepo *opsRepository
	_, _, err = nilRepo.GetPublicAvailability(context.Background(), time.Now(), time.Now())
	require.ErrorContains(t, err, "nil ops repository")
}

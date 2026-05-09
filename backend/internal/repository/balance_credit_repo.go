package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type balanceCreditSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type balanceCreditRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewBalanceCreditRepository(client *dbent.Client, db *sql.DB) service.BalanceCreditRepository {
	return &balanceCreditRepository{client: client, db: db}
}

func (r *balanceCreditRepository) execer(ctx context.Context) (balanceCreditSQL, error) {
	if r != nil && r.client != nil {
		if tx := dbent.TxFromContext(ctx); tx != nil {
			return tx.Client(), nil
		}
	}
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("balance credit repository db is nil")
	}
	return r.db, nil
}

func (r *balanceCreditRepository) CreateCredit(ctx context.Context, credit service.BalanceCreditCreate) error {
	if credit.Amount <= 0 {
		return nil
	}
	db, err := r.execer(ctx)
	if err != nil {
		return err
	}
	sourceType := strings.TrimSpace(credit.SourceType)
	if sourceType == "" {
		sourceType = service.BalanceCreditSourceRedeem
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO user_balance_credits (
	user_id, source_type, source_id, source_code, amount, remaining_amount, expires_at, status, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', NOW(), NOW())
`, credit.UserID, sourceType, strings.TrimSpace(credit.SourceID), strings.TrimSpace(credit.SourceCode), credit.Amount, credit.Amount, credit.ExpiresAt)
	return err
}

func (r *balanceCreditRepository) ListDailyBalanceUsage(ctx context.Context, start, end time.Time) ([]service.DailyBalanceUsage, error) {
	db, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT user_id, COALESCE(SUM(actual_cost), 0) AS balance_cost
FROM usage_logs
WHERE billing_type = 0
  AND actual_cost > 0
  AND created_at >= $1
  AND created_at < $2
GROUP BY user_id
HAVING COALESCE(SUM(actual_cost), 0) > 0
ORDER BY user_id
`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	usages := make([]service.DailyBalanceUsage, 0)
	for rows.Next() {
		var usage service.DailyBalanceUsage
		if err := rows.Scan(&usage.UserID, &usage.Amount); err != nil {
			return nil, err
		}
		usages = append(usages, usage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return usages, nil
}

func (r *balanceCreditRepository) SettleDailyUsage(ctx context.Context, userID int64, amount float64, settlementDate time.Time, dayEnd time.Time) error {
	if userID <= 0 || amount <= 0 {
		return nil
	}
	db, err := r.execer(ctx)
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, remaining_amount
FROM user_balance_credits
WHERE user_id = $1
  AND status = 'active'
  AND remaining_amount > 0
  AND created_at <= $2
  AND (settled_until_date IS NULL OR settled_until_date < $3::date)
ORDER BY expires_at ASC NULLS LAST, id ASC
FOR UPDATE
`, userID, dayEnd, settlementDate)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	remaining := amount
	eligibleIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		var available float64
		if err := rows.Scan(&id, &available); err != nil {
			return err
		}
		eligibleIDs = append(eligibleIDs, id)
		if remaining <= 0 {
			continue
		}
		delta := math.Min(available, remaining)
		if delta <= 0 {
			continue
		}
		if _, err := db.ExecContext(ctx, `
UPDATE user_balance_credits
SET remaining_amount = remaining_amount - $1,
    status = CASE WHEN remaining_amount - $1 <= 0 THEN 'consumed' ELSE status END,
    settled_until_date = $3::date,
    updated_at = NOW()
WHERE id = $2
`, delta, id, settlementDate); err != nil {
			return err
		}
		remaining -= delta
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(eligibleIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(eligibleIDs))
	args := make([]any, 0, len(eligibleIDs)+1)
	args = append(args, settlementDate)
	for i, id := range eligibleIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+2))
		args = append(args, id)
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf(`
UPDATE user_balance_credits
SET settled_until_date = $1::date,
    updated_at = NOW()
WHERE id IN (%s)
  AND status = 'active'
`, strings.Join(placeholders, ",")), args...)
	return err
}

func (r *balanceCreditRepository) ExpireDueCredits(ctx context.Context, now time.Time, settlementDate time.Time, limit int) ([]service.ExpiredBalanceCredit, error) {
	if limit <= 0 {
		limit = 100
	}
	db, err := r.execer(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
WITH due AS (
    SELECT id
    FROM user_balance_credits
    WHERE status = 'active'
      AND remaining_amount > 0
      AND expires_at IS NOT NULL
      AND expires_at <= $1
      AND (settled_until_date IS NULL OR settled_until_date <= $3::date)
    ORDER BY expires_at ASC, id ASC
    LIMIT $2
    FOR UPDATE SKIP LOCKED
),
expired AS (
    SELECT id, user_id, remaining_amount AS expired_amount, expires_at
    FROM user_balance_credits
    WHERE id IN (SELECT id FROM due)
)
UPDATE user_balance_credits ubc
SET status = 'expired',
    remaining_amount = 0,
    expired_at = $1,
    updated_at = NOW()
FROM expired
WHERE ubc.id = expired.id
RETURNING expired.id, expired.user_id, expired.expired_amount, expired.expires_at
`, now, limit, settlementDate)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ExpiredBalanceCredit, 0)
	for rows.Next() {
		var item service.ExpiredBalanceCredit
		if err := rows.Scan(&item.ID, &item.UserID, &item.Amount, &item.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

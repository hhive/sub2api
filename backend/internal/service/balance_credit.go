package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	BalanceCreditStatusActive   = "active"
	BalanceCreditStatusConsumed = "consumed"
	BalanceCreditStatusExpired  = "expired"

	BalanceCreditSourceRedeem    = "redeem"
	BalanceCreditSourceAdmin     = "admin"
	BalanceCreditSourcePromo     = "promo"
	BalanceCreditSourceAffiliate = "affiliate"
)

type BalanceCreditCreate struct {
	UserID     int64
	SourceType string
	SourceID   string
	SourceCode string
	Amount     float64
	ExpiresAt  *time.Time
}

type ExpiredBalanceCredit struct {
	ID        int64
	UserID    int64
	Amount    float64
	ExpiresAt time.Time
}

type DailyBalanceUsage struct {
	UserID int64
	Amount float64
}

type BalanceCredit struct {
	ID               int64
	UserID           int64
	SourceType       string
	SourceID         string
	SourceCode       string
	Amount           float64
	RemainingAmount  float64
	SettledUntilDate *time.Time
	ExpiresAt        *time.Time
	ExpiredAt        *time.Time
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type BalanceCreditRepository interface {
	CreateCredit(ctx context.Context, credit BalanceCreditCreate) error
	ListUserCredits(ctx context.Context, userID int64, params pagination.PaginationParams) ([]BalanceCredit, int64, error)
	ListDailyBalanceUsage(ctx context.Context, start, end time.Time) ([]DailyBalanceUsage, error)
	SettleDailyUsage(ctx context.Context, userID int64, amount float64, settlementDate time.Time, dayEnd time.Time) error
	ExpireDueCredits(ctx context.Context, now time.Time, settlementDate time.Time, limit int) ([]ExpiredBalanceCredit, error)
}

func balanceCreditExpiresAt(validityDays int, now time.Time) *time.Time {
	if validityDays <= 0 {
		return nil
	}
	expiresAt := now.AddDate(0, 0, validityDays)
	return &expiresAt
}

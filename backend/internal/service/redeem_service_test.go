//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemBalanceCreditRepoStub struct {
	created []BalanceCreditCreate
}

func (s *redeemBalanceCreditRepoStub) CreateCredit(ctx context.Context, credit BalanceCreditCreate) error {
	s.created = append(s.created, credit)
	return nil
}

func (s *redeemBalanceCreditRepoStub) CreateCreditIfAbsent(ctx context.Context, credit BalanceCreditCreate) (bool, error) {
	s.created = append(s.created, credit)
	return true, nil
}

func (s *redeemBalanceCreditRepoStub) ListUserCredits(ctx context.Context, userID int64, params pagination.PaginationParams) ([]BalanceCredit, int64, error) {
	panic("unexpected ListUserCredits call")
}

func (s *redeemBalanceCreditRepoStub) ListCredits(ctx context.Context, params pagination.PaginationParams, filters BalanceCreditListFilters) ([]BalanceCredit, int64, BalanceCreditListSummary, error) {
	panic("unexpected ListCredits call")
}

func (s *redeemBalanceCreditRepoStub) ListDailyBalanceUsage(ctx context.Context, start, end time.Time) ([]DailyBalanceUsage, error) {
	panic("unexpected ListDailyBalanceUsage call")
}

func (s *redeemBalanceCreditRepoStub) SettleDailyUsage(ctx context.Context, userID int64, amount float64, settlementDate time.Time, dayEnd time.Time) error {
	panic("unexpected SettleDailyUsage call")
}

func (s *redeemBalanceCreditRepoStub) ExpireDueCredits(ctx context.Context, now time.Time, settlementDate time.Time, limit int) ([]ExpiredBalanceCredit, error) {
	panic("unexpected ExpireDueCredits call")
}

func TestRedeemServiceCreateBalanceCreditPermanentWhenValidityZero(t *testing.T) {
	repo := &redeemBalanceCreditRepoStub{}
	service := &RedeemService{
		balanceCreditRepo: repo,
		settingService: NewSettingService(&settingRepoStub{
			values: map[string]string{
				SettingKeyBalanceCreditValidityDays: "0",
			},
		}, &config.Config{
			Default: config.DefaultConfig{
				UserBalance:     0,
				UserConcurrency: 1,
			},
		}),
	}

	err := service.createBalanceCredit(context.Background(), 42, &RedeemCode{
		ID:   1001,
		Code: "PERMANENT",
	}, 12.5)
	require.NoError(t, err)

	require.Len(t, repo.created, 1)
	require.Equal(t, int64(42), repo.created[0].UserID)
	require.Equal(t, BalanceCreditSourceRedeem, repo.created[0].SourceType)
	require.Equal(t, "1001", repo.created[0].SourceID)
	require.Equal(t, "PERMANENT", repo.created[0].SourceCode)
	require.Equal(t, 12.5, repo.created[0].Amount)
	require.Nil(t, repo.created[0].ExpiresAt)
}

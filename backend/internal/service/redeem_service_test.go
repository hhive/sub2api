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

func TestSubscriptionQuotaTotalForValidityDays(t *testing.T) {
	daily := 5.0
	weekly := 60.0
	monthly := 200.0

	got, ok := subscriptionQuotaTotalForValidityDays(30, &daily, &weekly, &monthly)
	require.True(t, ok)
	require.Equal(t, 150.0, got)

	got, ok = subscriptionQuotaTotalForValidityDays(30, nil, &weekly, &monthly)
	require.True(t, ok)
	require.Equal(t, 200.0, got)

	got, ok = subscriptionQuotaTotalForValidityDays(8, nil, &weekly, nil)
	require.True(t, ok)
	require.Equal(t, 120.0, got)

	got, ok = subscriptionQuotaTotalForValidityDays(31, nil, nil, &monthly)
	require.True(t, ok)
	require.Equal(t, 400.0, got)

	got, ok = subscriptionQuotaTotalForValidityDays(30, nil, nil, nil)
	require.False(t, ok)
	require.Zero(t, got)
}

func TestRedeemServiceSubscriptionRedeemRebateBaseAmountUsesGroupQuota(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group := client.Group.Create().
		SetName("月订阅Lite").
		SetPlatform("openai").
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetWeeklyLimitUsd(60).
		SetMonthlyLimitUsd(200).
		SaveX(ctx)

	svc := &RedeemService{entClient: client}
	baseAmount, ok, err := svc.subscriptionRedeemRebateBaseAmount(ctx, &RedeemCode{
		ID:      972,
		Type:    RedeemTypeSubscription,
		Value:   10,
		GroupID: &group.ID,
	}, 30)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 200.0, baseAmount)
}

func TestRedeemServiceSubscriptionRedeemRebateUsesGroupQuotaBaseAmount(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group := client.Group.Create().
		SetName("月订阅Lite").
		SetPlatform("openai").
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetWeeklyLimitUsd(60).
		SetMonthlyLimitUsd(200).
		SaveX(ctx)

	inviterID := int64(89)
	inviteeID := int64(989)
	rate := 10.0
	affiliateRepo := &affiliateDetailRepoStub{
		summaries: map[int64]*AffiliateSummary{
			inviteeID: {UserID: inviteeID, InviterID: &inviterID, CreatedAt: time.Now()},
			inviterID: {UserID: inviterID, AffRebateRatePercent: &rate},
		},
	}
	svc := &RedeemService{
		entClient: client,
		affiliateService: &AffiliateService{repo: affiliateRepo, settingService: affiliateSettingsService(map[string]string{
			SettingKeyAffiliateEnabled:                      "true",
			SettingKeyAffiliateSubscriptionRebateMultiplier: "80",
			SettingKeyAffiliateRebateDurationDays:           "0",
			SettingKeyAffiliateRebatePerInviteeCap:          "0",
			SettingKeyAffiliateTieredRebateEnabled:          "false",
			SettingKeyAffiliateRebateFreezeHours:            "0",
			SettingKeyAffiliateRebateRate:                   "10",
			SettingKeyAffiliateTier2MinPaidInvitees:         "10",
			SettingKeyAffiliateTier3MinPaidInvitees:         "30",
			SettingKeyAffiliateTier2MultiplierPercent:       "250",
			SettingKeyAffiliateTier3MultiplierPercent:       "500",
		})},
	}

	svc.tryAccrueAffiliateSubscriptionRebateForRedeem(ctx, inviteeID, &RedeemCode{
		ID:      972,
		Type:    RedeemTypeSubscription,
		Value:   10,
		GroupID: &group.ID,
	}, 30)

	require.Len(t, affiliateRepo.accrueCalls, 1)
	require.Equal(t, affiliateAccrueCall{
		inviterID:     inviterID,
		inviteeUserID: inviteeID,
		amount:        16,
		freezeHours:   0,
		sourceOrderID: nil,
	}, affiliateRepo.accrueCalls[0])
}

func TestRedeemServiceSubscriptionRedeemRebateSkipsNonPositiveValidityDays(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group := client.Group.Create().
		SetName("月订阅Lite").
		SetPlatform("openai").
		SetSubscriptionType(SubscriptionTypeSubscription).
		SetWeeklyLimitUsd(60).
		SetMonthlyLimitUsd(200).
		SaveX(ctx)

	inviterID := int64(89)
	inviteeID := int64(989)
	rate := 10.0
	affiliateRepo := &affiliateDetailRepoStub{
		summaries: map[int64]*AffiliateSummary{
			inviteeID: {UserID: inviteeID, InviterID: &inviterID, CreatedAt: time.Now()},
			inviterID: {UserID: inviterID, AffRebateRatePercent: &rate},
		},
	}
	svc := &RedeemService{
		entClient: client,
		affiliateService: &AffiliateService{repo: affiliateRepo, settingService: affiliateSettingsService(map[string]string{
			SettingKeyAffiliateEnabled: "true",
		})},
	}

	svc.tryAccrueAffiliateSubscriptionRebateForRedeem(ctx, inviteeID, &RedeemCode{
		ID:      972,
		Type:    RedeemTypeSubscription,
		Value:   10,
		GroupID: &group.ID,
	}, 0)
	svc.tryAccrueAffiliateSubscriptionRebateForRedeem(ctx, inviteeID, &RedeemCode{
		ID:      973,
		Type:    RedeemTypeSubscription,
		Value:   10,
		GroupID: &group.ID,
	}, -7)

	require.Empty(t, affiliateRepo.accrueCalls)
}

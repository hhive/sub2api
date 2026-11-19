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

type redeemBonusUserRepoStub struct {
	*userRepoStub
	balanceUpdates []float64
}

func (s *redeemBonusUserRepoStub) UpdateBalance(_ context.Context, _ int64, amount float64) error {
	s.balanceUpdates = append(s.balanceUpdates, amount)
	return nil
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

type generateBalanceRedeemRepoStub struct {
	RedeemCodeRepository
	codes []RedeemCode
}

func (r *generateBalanceRedeemRepoStub) Create(_ context.Context, code *RedeemCode) error {
	r.codes = append(r.codes, *code)
	return nil
}

func TestGenerateRedeemCodesBalanceCreditValidity(t *testing.T) {
	zero, custom, maxDays, negative, tooMany := 0, 7, 36500, -1, 36501
	for _, tt := range []struct {
		name  string
		kind  string
		value float64
		days  *int
		valid bool
	}{
		{"global", RedeemTypeBalance, 10, nil, true},
		{"permanent", RedeemTypeBalance, 10, &zero, true},
		{"custom", RedeemTypeBalance, 10, &custom, true},
		{"max", RedeemTypeBalance, 10, &maxDays, true},
		{"negative", RedeemTypeBalance, 10, &negative, false},
		{"too many days", RedeemTypeBalance, 10, &tooMany, false},
		{"concurrency code", RedeemTypeConcurrency, 10, &custom, false},
		{"zero amount", RedeemTypeBalance, 0, &custom, false},
		{"negative amount", RedeemTypeBalance, -10, &custom, false},
		{"legacy adjustment", RedeemTypeBalance, -10, nil, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &generateBalanceRedeemRepoStub{}
			svc := &adminServiceImpl{redeemCodeRepo: repo}
			codes, err := svc.GenerateRedeemCodes(context.Background(), &GenerateRedeemCodesInput{
				Count: 2, Type: tt.kind, Value: tt.value, BalanceCreditValidityDays: tt.days,
			})
			if !tt.valid {
				require.Error(t, err)
				require.Empty(t, repo.codes)
				return
			}
			require.NoError(t, err)
			require.Len(t, repo.codes, 2)
			require.Len(t, codes, 2)
			for i := range codes {
				require.Equal(t, tt.days, repo.codes[i].BalanceCreditValidityDays)
				require.Equal(t, tt.days, codes[i].BalanceCreditValidityDays)
			}
		})
	}
}

func TestRedeemServiceBalanceCreditValidityUsesOverride(t *testing.T) {
	zero, custom := 0, 7
	for _, tt := range []struct {
		name string
		days *int
		want int
	}{
		{"global", nil, 30},
		{"permanent", &zero, 0},
		{"custom", &custom, 7},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &redeemBalanceCreditRepoStub{}
			svc := &RedeemService{
				balanceCreditRepo: repo,
				settingService:    newAdminBalanceSettingService("30"),
			}
			start := time.Now()
			err := svc.createBalanceCredit(context.Background(), 42, &RedeemCode{
				ID: 1, Code: "CREDIT", BalanceCreditValidityDays: tt.days,
			}, 10)
			require.NoError(t, err)
			require.Len(t, repo.created, 1)
			expiresAt := repo.created[0].ExpiresAt
			if tt.want == 0 {
				require.Nil(t, expiresAt)
				return
			}
			require.NotNil(t, expiresAt)
			require.WithinDuration(t, start.AddDate(0, 0, tt.want), *expiresAt, time.Second)
		})
	}
}

func TestRedeemServiceFirstRechargeBonusSkipsAfterPriorPositiveSubscriptionRedeem(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := client.User.Create().
		SetEmail("prior-subscription@example.com").
		SetPasswordHash("test").
		SaveX(ctx)
	client.RedeemCode.Create().
		SetCode("PRIOR-SUBSCRIPTION").
		SetType(RedeemTypeSubscription).
		SetValue(10).
		SetStatus(StatusUsed).
		SetUsedBy(user.ID).
		SaveX(ctx)

	creditRepo := &redeemBalanceCreditRepoStub{}
	userRepo := &redeemBonusUserRepoStub{userRepoStub: &userRepoStub{}}
	svc := &RedeemService{
		entClient:         client,
		balanceCreditRepo: creditRepo,
		userRepo:          userRepo,
		settingService: firstRechargeBonusSettingService(map[string]string{
			SettingKeyFirstRechargeBonusEnabled:      "true",
			SettingKeyFirstRechargeBonusAmount:       "10",
			SettingKeyFirstRechargeBonusValidityDays: "3",
		}),
	}

	err := svc.applyFirstRechargeBonus(ctx, user.ID, &RedeemCode{
		ID:           2002,
		Code:         "CURRENT-BALANCE",
		Type:         RedeemTypeBalance,
		Value:        10,
		ValidityDays: 0,
	})

	require.NoError(t, err)
	require.Empty(t, creditRepo.created)
	require.Empty(t, userRepo.balanceUpdates)
}

func TestRedeemServiceFirstRechargeBonusAppliesToFirstPositiveSubscriptionRedeem(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	creditRepo := &redeemBalanceCreditRepoStub{}
	userRepo := &redeemBonusUserRepoStub{userRepoStub: &userRepoStub{}}
	svc := &RedeemService{
		entClient:         client,
		balanceCreditRepo: creditRepo,
		userRepo:          userRepo,
		settingService: firstRechargeBonusSettingService(map[string]string{
			SettingKeyFirstRechargeBonusEnabled:      "true",
			SettingKeyFirstRechargeBonusAmount:       "10",
			SettingKeyFirstRechargeBonusValidityDays: "3",
		}),
	}

	err := svc.applyFirstRechargeBonus(ctx, 42, &RedeemCode{
		ID:           2004,
		Code:         "FIRST-SUBSCRIPTION",
		Type:         RedeemTypeSubscription,
		Value:        10,
		ValidityDays: 30,
	})

	require.NoError(t, err)
	require.Len(t, creditRepo.created, 1)
	require.Equal(t, BalanceCreditSourceFirstRechargeBonus, creditRepo.created[0].SourceType)
	require.Equal(t, "2004", creditRepo.created[0].SourceID)
	require.Equal(t, 10.0, creditRepo.created[0].Amount)
	require.NotNil(t, creditRepo.created[0].ExpiresAt)
	require.Equal(t, []float64{10}, userRepo.balanceUpdates)
}

func TestRedeemServiceFirstRechargeBonusSkipsSubscriptionAfterPriorPositiveBalanceRedeem(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user := client.User.Create().
		SetEmail("prior-balance@example.com").
		SetPasswordHash("test").
		SaveX(ctx)
	client.RedeemCode.Create().
		SetCode("PRIOR-BALANCE").
		SetType(RedeemTypeBalance).
		SetValue(10).
		SetStatus(StatusUsed).
		SetUsedBy(user.ID).
		SaveX(ctx)

	creditRepo := &redeemBalanceCreditRepoStub{}
	userRepo := &redeemBonusUserRepoStub{userRepoStub: &userRepoStub{}}
	svc := &RedeemService{
		entClient:         client,
		balanceCreditRepo: creditRepo,
		userRepo:          userRepo,
		settingService: firstRechargeBonusSettingService(map[string]string{
			SettingKeyFirstRechargeBonusEnabled:      "true",
			SettingKeyFirstRechargeBonusAmount:       "10",
			SettingKeyFirstRechargeBonusValidityDays: "3",
		}),
	}

	err := svc.applyFirstRechargeBonus(ctx, user.ID, &RedeemCode{
		ID:           2005,
		Code:         "CURRENT-SUBSCRIPTION",
		Type:         RedeemTypeSubscription,
		Value:        10,
		ValidityDays: 30,
	})

	require.NoError(t, err)
	require.Empty(t, creditRepo.created)
	require.Empty(t, userRepo.balanceUpdates)
}

func TestRedeemServiceFirstRechargeBonusSkipsNegativeSubscriptionAdjustment(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	creditRepo := &redeemBalanceCreditRepoStub{}
	userRepo := &redeemBonusUserRepoStub{userRepoStub: &userRepoStub{}}
	svc := &RedeemService{
		entClient:         client,
		balanceCreditRepo: creditRepo,
		userRepo:          userRepo,
		settingService: firstRechargeBonusSettingService(map[string]string{
			SettingKeyFirstRechargeBonusEnabled:      "true",
			SettingKeyFirstRechargeBonusAmount:       "10",
			SettingKeyFirstRechargeBonusValidityDays: "3",
		}),
	}

	err := svc.applyFirstRechargeBonus(ctx, 42, &RedeemCode{
		ID:           2003,
		Code:         "REDUCE-SUBSCRIPTION",
		Type:         RedeemTypeSubscription,
		Value:        10,
		ValidityDays: -3,
	})

	require.NoError(t, err)
	require.Empty(t, creditRepo.created)
	require.Empty(t, userRepo.balanceUpdates)
}

func firstRechargeBonusSettingService(values map[string]string) *SettingService {
	return NewSettingService(&settingRepoStub{values: values}, &config.Config{
		Default: config.DefaultConfig{
			UserBalance:     0,
			UserConcurrency: 1,
		},
	})
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

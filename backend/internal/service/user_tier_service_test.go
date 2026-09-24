//go:build unit

package service

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// tierRepoStub 内存实现：只覆写被测路径用到的方法（未覆写的零值接口方法一旦被调用会 panic）。
// 夹具里的档位数值只属于测试，不进入生产代码。
type tierRepoStub struct {
	UserTierRepository

	tiers         []UserTier
	nextTierID    int64
	awards        map[string]UserTierAward
	awardSeq      int64
	effects       map[string]UserTierEffect
	effectSeq     int64
	overlays      []UserTierRateOverlay
	manualRates   map[string]bool
	consumed      map[int64]float64
	firstRecharge map[int64]*UserTierFirstRechargeCredit
	balances      map[int64]float64
	groupNames    map[int64]string
	applyCalls    int
	failType      string
}

func newTierRepoStub() *tierRepoStub {
	return &tierRepoStub{
		awards:        map[string]UserTierAward{},
		effects:       map[string]UserTierEffect{},
		manualRates:   map[string]bool{},
		consumed:      map[int64]float64{},
		firstRecharge: map[int64]*UserTierFirstRechargeCredit{},
		balances:      map[int64]float64{},
		groupNames:    map[int64]string{2: "企业专线"},
	}
}

func (s *tierRepoStub) addTier(tier UserTier) UserTier {
	s.nextTierID++
	tier.ID = s.nextTierID
	for i := range tier.Benefits {
		tier.Benefits[i].ID = int64(i + 1)
		tier.Benefits[i].TierID = tier.ID
	}
	s.tiers = append(s.tiers, tier)
	sort.SliceStable(s.tiers, func(i, j int) bool { return s.tiers[i].SortOrder < s.tiers[j].SortOrder })
	return tier
}

func (s *tierRepoStub) ListTiers(_ context.Context, includeDisabled bool) ([]UserTier, error) {
	out := make([]UserTier, 0, len(s.tiers))
	for _, tier := range s.tiers {
		if !includeDisabled && !tier.Enabled {
			continue
		}
		out = append(out, tier)
	}
	return out, nil
}

func (s *tierRepoStub) GetTierByID(_ context.Context, id int64) (*UserTier, error) {
	for i := range s.tiers {
		if s.tiers[i].ID == id {
			tier := s.tiers[i]
			return &tier, nil
		}
	}
	return nil, ErrUserTierNotFound
}

func (s *tierRepoStub) GetTierByCode(_ context.Context, code string) (*UserTier, error) {
	for i := range s.tiers {
		if s.tiers[i].Code == code {
			tier := s.tiers[i]
			return &tier, nil
		}
	}
	return nil, ErrUserTierNotFound
}

func (s *tierRepoStub) CreateTier(_ context.Context, tier *UserTier) error {
	created := s.addTier(*tier)
	*tier = created
	return nil
}

func (s *tierRepoStub) UpdateTier(_ context.Context, tier *UserTier) error {
	for i := range s.tiers {
		if s.tiers[i].ID == tier.ID {
			s.tiers[i] = *tier
			return nil
		}
	}
	return ErrUserTierNotFound
}

func (s *tierRepoStub) ReorderTiers(_ context.Context, orderedIDs []int64) error {
	for order, id := range orderedIDs {
		for i := range s.tiers {
			if s.tiers[i].ID == id {
				s.tiers[i].SortOrder = order
			}
		}
	}
	sort.SliceStable(s.tiers, func(i, j int) bool { return s.tiers[i].SortOrder < s.tiers[j].SortOrder })
	return nil
}

func (s *tierRepoStub) DeleteTier(_ context.Context, id int64) error {
	count, err := s.CountAwardsByTierID(context.Background(), id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrUserTierHasAwards
	}
	for i := range s.tiers {
		if s.tiers[i].ID == id {
			s.tiers = append(s.tiers[:i], s.tiers[i+1:]...)
			return nil
		}
	}
	return ErrUserTierNotFound
}

func (s *tierRepoStub) CountAwardsByTierID(_ context.Context, tierID int64) (int64, error) {
	var code string
	for i := range s.tiers {
		if s.tiers[i].ID == tierID {
			code = s.tiers[i].Code
		}
	}
	if code == "" {
		return 0, ErrUserTierNotFound
	}
	var count int64
	for _, award := range s.awards {
		if award.TierCode == code {
			count++
		}
	}
	return count, nil
}

func (s *tierRepoStub) GroupExists(_ context.Context, groupID int64) (bool, error) {
	_, ok := s.groupNames[groupID]
	return ok, nil
}

func (s *tierRepoStub) GroupName(_ context.Context, groupID int64) (string, error) {
	return s.groupNames[groupID], nil
}

func (s *tierRepoStub) GetUserConsumedAmount(_ context.Context, userID int64) (float64, error) {
	return s.consumed[userID], nil
}

func (s *tierRepoStub) GetFirstRechargeCredit(_ context.Context, userID int64) (*UserTierFirstRechargeCredit, error) {
	return s.firstRecharge[userID], nil
}

func (s *tierRepoStub) ListUserAwards(_ context.Context, userID int64) ([]UserTierAward, error) {
	var out []UserTierAward
	for _, award := range s.awards {
		if award.UserID == userID {
			out = append(out, award)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *tierRepoStub) ListUserEffects(_ context.Context, userID int64) ([]UserTierEffect, error) {
	var out []UserTierEffect
	for _, effect := range s.effects {
		if effect.UserID == userID {
			out = append(out, effect)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *tierRepoStub) ListUserRateOverlays(_ context.Context, userID int64) ([]UserTierRateOverlay, error) {
	var out []UserTierRateOverlay
	for _, overlay := range s.overlays {
		if overlay.UserID == userID && overlay.Status == "active" {
			out = append(out, overlay)
		}
	}
	return out, nil
}

func awardKey(userID int64, code string) string {
	return fmt.Sprintf("%d:%s", userID, code)
}

func effectKey(awardID int64, benefitKey string) string {
	return fmt.Sprintf("%d:%s", awardID, benefitKey)
}

func (s *tierRepoStub) EnsureAward(_ context.Context, award *UserTierAward) (int64, bool, error) {
	key := awardKey(award.UserID, award.TierCode)
	if existing, ok := s.awards[key]; ok {
		return existing.ID, false, nil
	}
	s.awardSeq++
	stored := *award
	stored.ID = s.awardSeq
	stored.AchievedAt = time.Now()
	s.awards[key] = stored
	return stored.ID, true, nil
}

func (s *tierRepoStub) EnsureEffect(_ context.Context, effect *UserTierEffect) (*UserTierEffect, bool, error) {
	key := effectKey(effect.AwardID, effect.BenefitKey)
	if existing, ok := s.effects[key]; ok {
		return &existing, false, nil
	}
	s.effectSeq++
	stored := *effect
	stored.ID = s.effectSeq
	stored.Status = UserTierEffectPending
	s.effects[key] = stored
	return &stored, true, nil
}

func (s *tierRepoStub) MarkEffectApplied(_ context.Context, effectID int64, detail map[string]any) error {
	for key, effect := range s.effects {
		if effect.ID != effectID {
			continue
		}
		now := time.Now()
		effect.Status = UserTierEffectApplied
		effect.Detail = detail
		effect.AppliedAt = &now
		effect.Attempts++
		s.effects[key] = effect
	}
	return nil
}

func (s *tierRepoStub) MarkEffectFailed(_ context.Context, effectID int64, failure string) error {
	for key, effect := range s.effects {
		if effect.ID != effectID {
			continue
		}
		effect.Status = UserTierEffectFailed
		effect.LastError = failure
		effect.Attempts++
		s.effects[key] = effect
	}
	return nil
}

func (s *tierRepoStub) HasManualRateMultiplier(_ context.Context, userID, groupID int64) (bool, error) {
	return s.manualRates[fmt.Sprintf("%d:%d", userID, groupID)], nil
}

func (s *tierRepoStub) UpsertRateOverlay(_ context.Context, overlay UserTierRateOverlay) error {
	s.overlays = append(s.overlays, overlay)
	return nil
}

func (s *tierRepoStub) ApplyTierRewardBalance(_ context.Context, userID int64, amount float64) error {
	s.applyCalls++
	s.balances[userID] += amount
	return nil
}

// tierCreditRepoStub 只覆写 CreateCredit（等级奖励按 effect 唯一约束做幂等，不走 CreateCreditIfAbsent）
type tierCreditRepoStub struct {
	BalanceCreditRepository
	credits []BalanceCreditCreate
	fail    bool
}

func (s *tierCreditRepoStub) CreateCredit(_ context.Context, credit BalanceCreditCreate) error {
	if s.fail {
		return fmt.Errorf("injected credit failure")
	}
	s.credits = append(s.credits, credit)
	return nil
}

func newTierServiceForTest(repo *tierRepoStub, credit *tierCreditRepoStub) *UserTierService {
	if credit == nil {
		credit = &tierCreditRepoStub{}
	}
	return NewUserTierService(repo, credit, nil, nil, nil, nil)
}

func balanceTier(id int64, code string, order int, threshold float64, amount float64) UserTier {
	return UserTier{
		ID:           id,
		Code:         code,
		Name:         code,
		SortOrder:    order,
		TriggerType:  UserTierTriggerConsumption,
		ThresholdUSD: &threshold,
		Enabled:      true,
		Benefits: []UserTierBenefit{{
			ID:            id,
			BenefitType:   UserTierBenefitBalanceCredit,
			BalanceCredit: &UserTierBalanceCreditParams{Amount: amount, ValidityDays: 3},
			Enabled:       true,
		}},
	}
}

// ---------- 阈值边界与当前/下一档推导 ----------

func TestUserTierView_ThresholdBoundary(t *testing.T) {
	cases := []struct {
		consumed    float64
		wantCurrent string
		wantNext    string
		wantGap     float64
	}{
		{499.99, "", "consume_500", 0.01},
		{500, "consume_500", "consume_1000", 500},
		{999.99, "consume_500", "consume_1000", 0.01},
		{1000, "consume_1000", "consume_1500", 500},
		{1499.99, "consume_1000", "consume_1500", 0.01},
		{1500, "consume_1500", "", 0},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("consumed=%.2f", tc.consumed), func(t *testing.T) {
			repo := newTierRepoStub()
			repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
			repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
			repo.addTier(balanceTier(0, "consume_1500", 2, 1500, 150))
			repo.consumed[7] = tc.consumed
			// 恰好等于阈值视为达成（不是严格大于），先给该用户补上已达成的授予记录
			for _, tier := range repo.tiers {
				if tier.ThresholdUSD != nil && tc.consumed >= *tier.ThresholdUSD {
					_, _, err := repo.EnsureAward(context.Background(), &UserTierAward{
						UserID: 7, TierCode: tier.Code, TierNameSnapshot: tier.Name, Source: UserTierAwardSourceClaim,
					})
					require.NoError(t, err)
				}
			}

			svc := newTierServiceForTest(repo, nil)
			view, err := svc.GetUserTierView(context.Background(), 7)
			require.NoError(t, err)

			if tc.wantCurrent == "" {
				require.Nil(t, view.CurrentTier)
			} else {
				require.NotNil(t, view.CurrentTier)
				require.Equal(t, tc.wantCurrent, view.CurrentTier.Code)
			}
			if tc.wantNext == "" {
				require.Nil(t, view.NextTier)
				require.Nil(t, view.RemainingToNext)
			} else {
				require.NotNil(t, view.NextTier)
				require.Equal(t, tc.wantNext, view.NextTier.Code)
				require.NotNil(t, view.RemainingToNext)
				require.InDelta(t, tc.wantGap, *view.RemainingToNext, 1e-9)
			}
		})
	}
}

// ---------- 领取资格 ----------

func TestUserTierClaim_RejectsUnachievedTier(t *testing.T) {
	repo := newTierRepoStub()
	high := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.consumed[9] = 100

	svc := newTierServiceForTest(repo, nil)
	_, err := svc.ClaimTier(context.Background(), 9, high.ID)

	require.ErrorIs(t, err, ErrUserTierNotAchieved)
	require.Empty(t, repo.awards)
	require.Empty(t, repo.effects)
}

func TestUserTierClaim_RejectsDisabledTier(t *testing.T) {
	repo := newTierRepoStub()
	tier := balanceTier(0, "consume_500", 0, 500, 50)
	tier.Enabled = false
	tier = repo.addTier(tier)
	repo.consumed[9] = 500

	svc := newTierServiceForTest(repo, nil)
	_, err := svc.ClaimTier(context.Background(), 9, tier.ID)

	require.ErrorIs(t, err, ErrUserTierDisabled)
}

func TestUserTierClaim_RejectsFirstRechargeTierAutomatically(t *testing.T) {
	repo := newTierRepoStub()
	tier := repo.addTier(UserTier{
		Code: "first_recharge", Name: "首充档", TriggerType: UserTierTriggerFirstRecharge, Enabled: true,
	})

	svc := newTierServiceForTest(repo, nil)
	_, err := svc.ClaimTier(context.Background(), 9, tier.ID)

	require.ErrorIs(t, err, ErrUserTierNotClaimable)
}

// 一步冲到最高档的用户必须能逐档补领早期档位（校验不能写成「必须等于已达成的最高档」）
func TestUserTierClaim_BackfillsEarlierTiers(t *testing.T) {
	repo := newTierRepoStub()
	t1 := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	t2 := repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	t3 := repo.addTier(balanceTier(0, "consume_1500", 2, 1500, 150))
	repo.consumed[11] = 1500

	svc := newTierServiceForTest(repo, nil)
	for _, tierID := range []int64{t1.ID, t2.ID, t3.ID} {
		result, err := svc.ClaimTier(context.Background(), 11, tierID)
		require.NoError(t, err)
		require.False(t, result.AlreadyClaimed)
	}

	require.Equal(t, 3, len(repo.awards))
	require.InDelta(t, 300, repo.balances[11], 1e-9)
	require.Equal(t, 3, repo.applyCalls)
}

// ---------- 幂等与重试 ----------

func TestUserTierClaim_IsIdempotent(t *testing.T) {
	repo := newTierRepoStub()
	credit := &tierCreditRepoStub{}
	tier := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.consumed[13] = 500

	svc := newTierServiceForTest(repo, credit)
	first, err := svc.ClaimTier(context.Background(), 13, tier.ID)
	require.NoError(t, err)
	require.False(t, first.AlreadyClaimed)

	balanceAfterFirst := repo.balances[13]
	creditsAfterFirst := len(credit.credits)
	effectsAfterFirst := len(repo.effects)

	second, err := svc.ClaimTier(context.Background(), 13, tier.ID)
	require.NoError(t, err)
	require.True(t, second.AlreadyClaimed)
	require.Empty(t, second.Applied)

	require.InDelta(t, balanceAfterFirst, repo.balances[13], 1e-9)
	require.Equal(t, creditsAfterFirst, len(credit.credits))
	require.Equal(t, effectsAfterFirst, len(repo.effects))
	require.Equal(t, 1, repo.applyCalls)
}

// 首次发放中途失败（effect 停在 pending）后再次领取：只补齐失败项，不重复已成功项
func TestUserTierClaim_RetriesPendingEffectsOnly(t *testing.T) {
	repo := newTierRepoStub()
	credit := &tierCreditRepoStub{}
	tier := repo.addTier(UserTier{
		ID: 1, Code: "consume_500", Name: "常客", SortOrder: 0,
		TriggerType: UserTierTriggerConsumption, ThresholdUSD: tierFloatPtr(500), Enabled: true,
		Benefits: []UserTierBenefit{
			{ID: 1, BenefitType: UserTierBenefitBalanceCredit, BalanceCredit: &UserTierBalanceCreditParams{Amount: 50, ValidityDays: 3}, Enabled: true},
			{ID: 2, BenefitType: UserTierBenefitGroupRate, GroupRate: &UserTierGroupRateParams{GroupID: 2, RateMultiplier: 0.9}, Enabled: true},
		},
	})
	repo.tiers = append(repo.tiers, tier)
	repo.consumed[17] = 500

	svc := newTierServiceForTest(repo, credit)
	// 第一次：倍率权益注入失败（repo.failType 让该类型抛错）——此处直接构造「额度已发、倍率停在 pending」的半成品状态
	firstResult, err := svc.ClaimTier(context.Background(), 17, tier.ID)
	require.NoError(t, err)
	require.Len(t, firstResult.Applied, 2)
	creditsAfterFirst := len(credit.credits)

	// 模拟倍率权益未落盘：删除 overlay 并把 effect 打回 pending
	repo.overlays = nil
	for key, effect := range repo.effects {
		if effect.BenefitType == UserTierBenefitGroupRate {
			effect.Status = UserTierEffectPending
			repo.effects[key] = effect
		}
	}

	retry, err := svc.ClaimTier(context.Background(), 17, tier.ID)
	require.NoError(t, err)
	require.True(t, retry.AlreadyClaimed)
	// 只补齐倍率：额度账本不新增行，余额不重复增加
	require.Equal(t, creditsAfterFirst, len(credit.credits))
	require.InDelta(t, 50, repo.balances[17], 1e-9)
	require.Len(t, repo.overlays, 1)
}

// ---------- 冲突规则：等级不覆盖手工倍率 ----------

func TestUserTierClaim_SkipsGroupRateWhenManualRateExists(t *testing.T) {
	repo := newTierRepoStub()
	credit := &tierCreditRepoStub{}
	tier := repo.addTier(UserTier{
		ID: 1, Code: "consume_1500", Name: "企业专线", SortOrder: 0,
		TriggerType: UserTierTriggerConsumption, ThresholdUSD: tierFloatPtr(1500), Enabled: true,
		Benefits: []UserTierBenefit{
			{ID: 1, BenefitType: UserTierBenefitBalanceCredit, BalanceCredit: &UserTierBalanceCreditParams{Amount: 150, ValidityDays: 30}, Enabled: true},
			{ID: 2, BenefitType: UserTierBenefitGroupRate, GroupRate: &UserTierGroupRateParams{GroupID: 2, RateMultiplier: 0.9}, Enabled: true},
		},
	})
	repo.tiers = append(repo.tiers, tier)
	repo.consumed[451] = 2000
	repo.manualRates["451:2"] = true // 该用户已有手工倍率 0.16

	svc := newTierServiceForTest(repo, credit)
	result, err := svc.ClaimTier(context.Background(), 451, tier.ID)
	require.NoError(t, err)

	// 倍率不被覆盖：不写 overlay
	require.Empty(t, repo.overlays)
	// 只发额度：10 元照发
	require.InDelta(t, 150, repo.balances[451], 1e-9)
	require.Len(t, credit.credits, 1)

	// 跳过原因写入 effect detail 并在结果里可见
	var rateDetail map[string]any
	for _, effect := range repo.effects {
		if effect.BenefitType == UserTierBenefitGroupRate {
			require.Equal(t, UserTierEffectApplied, effect.Status)
			rateDetail = effect.Detail
		}
	}
	require.Equal(t, UserTierSkipManualRateMultiplier, rateDetail["reason"])

	require.Len(t, result.Applied, 2)
	for _, applied := range result.Applied {
		if applied.BenefitType == UserTierBenefitGroupRate {
			require.Equal(t, UserTierSkipManualRateMultiplier, applied.SkippedReason)
		}
	}
}

// ---------- 余额可用性（奖励不计 total_recharged） ----------

func TestUserTierClaim_BalanceUsesTierRewardSourceWithoutRechargeBump(t *testing.T) {
	repo := newTierRepoStub()
	credit := &tierCreditRepoStub{}
	tier := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.consumed[21] = 600

	svc := newTierServiceForTest(repo, credit)
	_, err := svc.ClaimTier(context.Background(), 21, tier.ID)
	require.NoError(t, err)

	require.Len(t, credit.credits, 1)
	require.Equal(t, BalanceCreditSourceTierReward, credit.credits[0].SourceType)
	require.InDelta(t, 50, credit.credits[0].Amount, 1e-9)
	require.NotNil(t, credit.credits[0].ExpiresAt)
	require.InDelta(t, 50, repo.balances[21], 1e-9)
	// 余额只经等级仓储的专用方法增加（该方法不累加 total_recharged），
	// 而不是会累加充值额的 UpdateBalance
	require.Equal(t, 1, repo.applyCalls)
}

// ---------- 半成品防护 ----------

func TestUserTierClaim_CreditFailurePropagatesForRollback(t *testing.T) {
	repo := newTierRepoStub()
	credit := &tierCreditRepoStub{fail: true}
	tier := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.consumed[23] = 600

	svc := newTierServiceForTest(repo, credit)
	_, err := svc.ClaimTier(context.Background(), 23, tier.ID)

	require.Error(t, err)
	// 余额未落、overlay 未写；调用方（事务路径）会整体回滚授予与发放记录
	require.Zero(t, repo.applyCalls)
	require.Empty(t, repo.overlays)
}

// ---------- 查看接口 ----------

func TestUserTierView_FirstRechargeTierWithoutBackfill(t *testing.T) {
	repo := newTierRepoStub()
	repo.addTier(UserTier{
		ID: 1, Code: UserTierCodeFirstRecharge, Name: "新朋友", SortOrder: 0,
		TriggerType: UserTierTriggerFirstRecharge, Enabled: true,
		Benefits: []UserTierBenefit{{
			ID: 1, BenefitType: UserTierBenefitBalanceCredit,
			BalanceCredit: &UserTierBalanceCreditParams{Amount: 5, ValidityDays: 3}, Enabled: true,
		}},
	})
	createdAt := time.Now().Add(-time.Hour)
	expiresAt := createdAt.AddDate(0, 0, 3)
	repo.firstRecharge[29] = &UserTierFirstRechargeCredit{Amount: 5, CreatedAt: createdAt, ExpiresAt: &expiresAt}
	repo.consumed[29] = 0

	svc := newTierServiceForTest(repo, nil)
	view, err := svc.GetUserTierView(context.Background(), 29)
	require.NoError(t, err)

	require.Len(t, view.Tiers, 1)
	state := view.Tiers[0]
	require.True(t, state.Claimed)
	require.False(t, state.Claimable)
	require.NotNil(t, state.ClaimedAt)
	require.Len(t, state.Benefits, 1)
	require.Equal(t, UserTierEffectApplied, state.Benefits[0].Status)
	require.InDelta(t, 5, state.Benefits[0].BalanceCredit.Amount, 1e-9)
}

func TestUserTierView_ClaimableCountAndClaimAll(t *testing.T) {
	repo := newTierRepoStub()
	t1 := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.addTier(balanceTier(0, "consume_1000", 1, 1000, 100))
	repo.consumed[31] = 1000

	svc := newTierServiceForTest(repo, nil)
	view, err := svc.GetUserTierView(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, 2, view.ClaimableCount)
	require.Nil(t, view.CurrentTier)

	// 一键领取由前端逐档调用（不新增批量语义）
	for _, id := range []int64{view.Tiers[0].TierID, view.Tiers[1].TierID} {
		_, err := svc.ClaimTier(context.Background(), 31, id)
		require.NoError(t, err)
	}
	require.InDelta(t, 150, repo.balances[31], 1e-9)
	require.Equal(t, t1.ID, view.Tiers[0].TierID)

	after, err := svc.GetUserTierView(context.Background(), 31)
	require.NoError(t, err)
	require.Equal(t, 0, after.ClaimableCount)
	require.NotNil(t, after.CurrentTier)
	require.Equal(t, "consume_1000", after.CurrentTier.Code)
}

// ---------- 配置校验 ----------

func TestValidateTierConfig_RejectsInvalidBenefitParams(t *testing.T) {
	repo := newTierRepoStub()
	svc := newTierServiceForTest(repo, nil)

	code := "consume_500"
	name := "常客"
	trigger := UserTierTriggerConsumption
	zero := 0.0

	cases := []struct {
		label   string
		benefit UserTierBenefitInput
		wantErr error
	}{
		{"amount <= 0", UserTierBenefitInput{BenefitType: UserTierBenefitBalanceCredit, Amount: 0, ValidityDays: 3}, ErrUserTierInvalidBenefit},
		{"validity_days < 0", UserTierBenefitInput{BenefitType: UserTierBenefitBalanceCredit, Amount: 10, ValidityDays: -1}, ErrUserTierInvalidBenefit},
		{"group_id 不存在", UserTierBenefitInput{BenefitType: UserTierBenefitGroupRate, GroupID: 999, RateMultiplier: 0.9}, ErrUserTierInvalidBenefit},
		{"rate_multiplier <= 0", UserTierBenefitInput{BenefitType: UserTierBenefitGroupRate, GroupID: 2, RateMultiplier: 0}, ErrUserTierInvalidBenefit},
		{"未知权益类型", UserTierBenefitInput{BenefitType: "rpm_boost", Amount: 1}, ErrUserTierInvalidBenefit},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			benefits := []UserTierBenefitInput{tc.benefit}
			if tc.benefit.Enabled == false {
				benefits[0].Enabled = true
			}
			_, err := svc.CreateTier(context.Background(), UserTierInput{
				Code: &code, Name: &name, TriggerType: &trigger, ThresholdUSD: &zero, Benefits: &benefits,
			})
			require.ErrorIs(t, err, tc.wantErr)
		})
	}

	// 一档同类权益重复也拒绝
	benefits := []UserTierBenefitInput{
		{BenefitType: UserTierBenefitBalanceCredit, Amount: 10, ValidityDays: 3, Enabled: true},
		{BenefitType: UserTierBenefitBalanceCredit, Amount: 20, ValidityDays: 3, Enabled: true},
	}
	_, err := svc.CreateTier(context.Background(), UserTierInput{
		Code: &code, Name: &name, TriggerType: &trigger, ThresholdUSD: &zero, Benefits: &benefits,
	})
	require.ErrorIs(t, err, ErrUserTierInvalidBenefit)
}

func TestCreateTier_RequiresValidCodeAndThreshold(t *testing.T) {
	repo := newTierRepoStub()
	svc := newTierServiceForTest(repo, nil)

	badCode := "Consume 500"
	name := "常客"
	trigger := UserTierTriggerConsumption
	threshold := 500.0
	_, err := svc.CreateTier(context.Background(), UserTierInput{
		Code: &badCode, Name: &name, TriggerType: &trigger, ThresholdUSD: &threshold,
	})
	require.ErrorIs(t, err, ErrUserTierInvalidConfig)

	goodCode := "consume_500"
	_, err = svc.CreateTier(context.Background(), UserTierInput{
		Code: &goodCode, Name: &name, TriggerType: &trigger,
	})
	require.ErrorIs(t, err, ErrUserTierInvalidConfig, "consumption 档缺阈值必须被拒")

	firstRecharge := UserTierTriggerFirstRecharge
	created, err := svc.CreateTier(context.Background(), UserTierInput{
		Code: &goodCode, Name: &name, TriggerType: &firstRecharge,
	})
	require.NoError(t, err)
	require.Nil(t, created.ThresholdUSD)
	require.Equal(t, 0, created.SortOrder)
}

func TestUpdateTier_KeepsExistingBenefitIDsAndBlocksCodeChangeAfterAwards(t *testing.T) {
	repo := newTierRepoStub()
	tier := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	_, _, err := repo.EnsureAward(context.Background(), &UserTierAward{
		UserID: 41, TierCode: "consume_500", Source: UserTierAwardSourceClaim,
	})
	require.NoError(t, err)

	svc := newTierServiceForTest(repo, nil)
	newCode := "consume_500_v2"
	_, err = svc.UpdateTier(context.Background(), tier.ID, UserTierInput{Code: &newCode})
	require.ErrorIs(t, err, ErrUserTierHasAwards)

	// 删除同样被拒
	require.ErrorIs(t, svc.DeleteTier(context.Background(), tier.ID), ErrUserTierHasAwards)

	// 停用允许
	enabled := false
	updated, err := svc.SetTierEnabled(context.Background(), tier.ID, enabled)
	require.NoError(t, err)
	require.False(t, updated.Enabled)
}

// ---------- C5：首充单源读取 ----------

func TestResolveFirstRechargeGrant(t *testing.T) {
	t.Run("配置缺失时返回错误供回退", func(t *testing.T) {
		repo := newTierRepoStub()
		svc := newTierServiceForTest(repo, nil)
		_, err := svc.ResolveFirstRechargeGrant(context.Background())
		require.ErrorIs(t, err, ErrUserTierNotFound)
	})

	t.Run("停用档位返回 enabled=false", func(t *testing.T) {
		repo := newTierRepoStub()
		repo.addTier(UserTier{
			ID: 1, Code: UserTierCodeFirstRecharge, Name: "新朋友", TriggerType: UserTierTriggerFirstRecharge, Enabled: false,
			Benefits: []UserTierBenefit{{ID: 1, BenefitType: UserTierBenefitBalanceCredit, BalanceCredit: &UserTierBalanceCreditParams{Amount: 5, ValidityDays: 3}, Enabled: true}},
		})
		svc := newTierServiceForTest(repo, nil)
		grant, err := svc.ResolveFirstRechargeGrant(context.Background())
		require.NoError(t, err)
		require.False(t, grant.Enabled)
		require.InDelta(t, 5, grant.Amount, 1e-9)
		require.Equal(t, 3, grant.ValidityDays)
	})

	t.Run("启用但金额非法时返回错误", func(t *testing.T) {
		repo := newTierRepoStub()
		repo.addTier(UserTier{
			ID: 1, Code: UserTierCodeFirstRecharge, Name: "新朋友", TriggerType: UserTierTriggerFirstRecharge, Enabled: true,
			Benefits: []UserTierBenefit{{ID: 1, BenefitType: UserTierBenefitBalanceCredit, BalanceCredit: &UserTierBalanceCreditParams{Amount: 0, ValidityDays: 3}, Enabled: true}},
		})
		svc := newTierServiceForTest(repo, nil)
		_, err := svc.ResolveFirstRechargeGrant(context.Background())
		require.ErrorIs(t, err, ErrUserTierInvalidConfig)
	})

	t.Run("触发类型不匹配时返回错误", func(t *testing.T) {
		repo := newTierRepoStub()
		repo.addTier(balanceTier(0, UserTierCodeFirstRecharge, 0, 500, 50))
		svc := newTierServiceForTest(repo, nil)
		_, err := svc.ResolveFirstRechargeGrant(context.Background())
		require.ErrorIs(t, err, ErrUserTierInvalidConfig)
	})

	t.Run("未装配仓储时返回 nil 表示沿用既有行为", func(t *testing.T) {
		var svc *UserTierService
		grant, err := svc.ResolveFirstRechargeGrant(context.Background())
		require.NoError(t, err)
		require.Nil(t, grant)
	})
}

func tierFloatPtr(v float64) *float64 { return &v }

// ---------- C5：首充参数决策（等价性与回退） ----------

func TestSelectFirstRechargeGrant(t *testing.T) {
	t.Run("等级配置与 settings 现值一致时行为等价", func(t *testing.T) {
		// 迁移当场：settings enabled=true/amount=5/days=3，等级配置导入同样的值
		grant := &FirstRechargeTierGrant{TierCode: UserTierCodeFirstRecharge, Enabled: true, Amount: 5, ValidityDays: 3}
		enabled, amount, days, err := selectFirstRechargeGrant(true, 5, 3, grant)
		require.NoError(t, err)
		require.True(t, enabled)
		require.InDelta(t, 5, amount, 1e-9)
		require.Equal(t, 3, days)

		// 迁移前（直接读 settings）的结果必须逐项一致
		beforeEnabled, beforeAmount, beforeDays, err := selectFirstRechargeGrant(true, 5, 3, nil)
		require.NoError(t, err)
		require.Equal(t, beforeEnabled, enabled)
		require.InDelta(t, beforeAmount, amount, 1e-9)
		require.Equal(t, beforeDays, days)
	})

	t.Run("等级配置停用则不发放", func(t *testing.T) {
		grant := &FirstRechargeTierGrant{Enabled: false, Amount: 5, ValidityDays: 3}
		enabled, amount, days, err := selectFirstRechargeGrant(true, 5, 3, grant)
		require.NoError(t, err)
		require.False(t, enabled)
		require.Zero(t, amount)
		require.Zero(t, days)
	})

	t.Run("等级配置缺失时保留既有行为", func(t *testing.T) {
		enabled, amount, days, err := selectFirstRechargeGrant(true, 7.5, 9, nil)
		require.NoError(t, err)
		require.True(t, enabled)
		require.InDelta(t, 7.5, amount, 1e-9)
		require.Equal(t, 9, days)
	})

	t.Run("等级配置金额非法时回退既有行为且不静默不发", func(t *testing.T) {
		grant := &FirstRechargeTierGrant{Enabled: true, Amount: 0, ValidityDays: 3}
		enabled, amount, days, err := selectFirstRechargeGrant(true, 7.5, 9, grant)
		require.NoError(t, err)
		require.True(t, enabled, "非法等级配置不得静默不发，必须回退既有行为照发")
		require.InDelta(t, 7.5, amount, 1e-9)
		require.Equal(t, 9, days)
	})

	t.Run("等级配置启用时以等级配置为准", func(t *testing.T) {
		grant := &FirstRechargeTierGrant{Enabled: true, Amount: 12.5, ValidityDays: 30}
		enabled, amount, days, err := selectFirstRechargeGrant(true, 5, 3, grant)
		require.NoError(t, err)
		require.True(t, enabled)
		require.InDelta(t, 12.5, amount, 1e-9)
		require.Equal(t, 30, days)
	})
}

// ---------- 总开关（user_tier_enabled） ----------

// userTierSettingRepoStub 只覆写被测路径用到的设置读写。
type userTierSettingRepoStub struct {
	SettingRepository
	values map[string]string
	getErr error
}

func (s *userTierSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return s.values[key], nil
}

func (s *userTierSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func newTierServiceWithSwitch(repo *tierRepoStub, values map[string]string) (*UserTierService, *userTierSettingRepoStub) {
	store := &userTierSettingRepoStub{values: values}
	// 同包测试，直接构造（生产装配走 wire）
	settings := &SettingService{settingRepo: store}
	return NewUserTierService(repo, &tierCreditRepoStub{}, nil, nil, nil, settings), store
}

func TestUserTierSwitch_DefaultsToEnabled(t *testing.T) {
	cases := map[string]map[string]string{
		"未配置":     {},
		"空值":      {"user_tier_enabled": ""},
		"显式 true": {"user_tier_enabled": "true"},
		"其它值":     {"user_tier_enabled": "yes"},
	}
	for label, values := range cases {
		t.Run(label, func(t *testing.T) {
			repo := newTierRepoStub()
			repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
			repo.consumed[7] = 600
			svc, _ := newTierServiceWithSwitch(repo, values)

			view, err := svc.GetUserTierView(context.Background(), 7)
			require.NoError(t, err)
			require.True(t, view.Enabled)
			require.Equal(t, 1, view.ClaimableCount, "开启时照常可领取")
		})
	}
}

func TestUserTierSwitch_DisabledStopsVisibilityAndGrants(t *testing.T) {
	for _, off := range []string{"false", "0", "off", "disabled", "FALSE"} {
		t.Run(off, func(t *testing.T) {
			repo := newTierRepoStub()
			tier := repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
			repo.addTier(UserTier{
				ID: 2, Code: UserTierCodeFirstRecharge, Name: "新客",
				TriggerType: UserTierTriggerFirstRecharge, Enabled: true,
				Benefits: []UserTierBenefit{{ID: 1, BenefitType: UserTierBenefitBalanceCredit,
					BalanceCredit: &UserTierBalanceCreditParams{Amount: 5, ValidityDays: 3}, Enabled: true}},
			})
			repo.consumed[7] = 600
			svc, _ := newTierServiceWithSwitch(repo, map[string]string{"user_tier_enabled": off})

			// 用户端看不到任何等级内容
			view, err := svc.GetUserTierView(context.Background(), 7)
			require.NoError(t, err)
			require.False(t, view.Enabled)
			require.Empty(t, view.Tiers)
			require.Nil(t, view.CurrentTier)
			require.Nil(t, view.NextTier)
			require.Zero(t, view.ClaimableCount)

			// 领取被服务端拒绝（即便已达标）
			_, err = svc.ClaimTier(context.Background(), 7, tier.ID)
			require.ErrorIs(t, err, ErrUserTierFeatureDisabled)
			require.Empty(t, repo.awards, "拒绝时不得落任何授予记录")

			// 首充不再由等级配置触发（发放入口回退既有 settings 行为）
			grant, err := svc.ResolveFirstRechargeGrant(context.Background())
			require.NoError(t, err)
			require.Nil(t, grant)
		})
	}
}

func TestUserTierSwitch_SetAndRead(t *testing.T) {
	repo := newTierRepoStub()
	svc, store := newTierServiceWithSwitch(repo, map[string]string{})

	require.True(t, svc.IsFeatureEnabled(context.Background()))
	require.NoError(t, svc.SetFeatureEnabled(context.Background(), false))
	require.Equal(t, "false", store.values["user_tier_enabled"])
	require.False(t, svc.IsFeatureEnabled(context.Background()))

	require.NoError(t, svc.SetFeatureEnabled(context.Background(), true))
	require.Equal(t, "true", store.values["user_tier_enabled"])
	require.True(t, svc.IsFeatureEnabled(context.Background()))
}

func TestUserTierSwitch_ReadErrorFailsOpen(t *testing.T) {
	repo := newTierRepoStub()
	repo.addTier(balanceTier(0, "consume_500", 0, 500, 50))
	repo.consumed[7] = 600
	store := &userTierSettingRepoStub{getErr: fmt.Errorf("db down")}
	svc := NewUserTierService(repo, &tierCreditRepoStub{}, nil, nil, nil, &SettingService{settingRepo: store})

	require.True(t, svc.IsFeatureEnabled(context.Background()), "读抖动不得把入口打掉")
	view, err := svc.GetUserTierView(context.Background(), 7)
	require.NoError(t, err)
	require.True(t, view.Enabled)
}

func TestUserTierSwitch_MissingDependenciesAssumeEnabled(t *testing.T) {
	var nilService *UserTierService
	require.True(t, nilService.IsFeatureEnabled(context.Background()), "nil 服务视为开启")

	repo := newTierRepoStub()
	svc := newTierServiceForTest(repo, nil) // 未注入设置服务
	require.True(t, svc.IsFeatureEnabled(context.Background()))

	require.Error(t, svc.SetFeatureEnabled(context.Background(), false), "未装配时写入应报错而不是静默成功")
}

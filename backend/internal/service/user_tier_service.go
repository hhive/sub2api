package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// tierCodePattern 等级标识：稳定、可读、可安全用作幂等键
var tierCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// UserTierService 用户等级与权益：配置读写、用户端视图、手动领取与首充单源读取。
//
// 口径（设计第 3 节）：累计消费只算兑换额度消费（user_balance_credits 中 source_type='redeem'
// 的 amount - remaining_amount），不计订阅消费，不用 users.total_recharged。
type UserTierService struct {
	repo                 UserTierRepository
	balanceCreditRepo    BalanceCreditRepository
	entClient            *dbent.Client
	authCacheInvalidator APIKeyAuthCacheInvalidator
	billingCacheService  *BillingCacheService
	// settingService 提供总开关（settings.user_tier_enabled）；nil 时视为开启，
	// 既保证未装配场景行为不变，也让单元测试无需构造设置服务。
	settingService *SettingService
}

// NewUserTierService 创建用户等级服务
func NewUserTierService(
	repo UserTierRepository,
	balanceCreditRepo BalanceCreditRepository,
	entClient *dbent.Client,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
	settingService *SettingService,
) *UserTierService {
	return &UserTierService{
		repo:                 repo,
		balanceCreditRepo:    balanceCreditRepo,
		entClient:            entClient,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
		settingService:       settingService,
	}
}

// IsFeatureEnabled 读取总开关。读失败时记日志并视为开启：一次读抖动不应把用户端入口打掉。
func (s *UserTierService) IsFeatureEnabled(ctx context.Context) bool {
	if s == nil || s.settingService == nil {
		return true
	}
	enabled, err := s.settingService.IsUserTierEnabled(ctx)
	if err != nil {
		logger.LegacyPrintf("service.user_tier", "read user tier switch failed, treat as enabled: err=%v", err)
		return true
	}
	return enabled
}

// SetFeatureEnabled 写总开关（管理端「等级与权益」页的唯一写入口）。
func (s *UserTierService) SetFeatureEnabled(ctx context.Context, enabled bool) error {
	if s == nil || s.settingService == nil {
		return fmt.Errorf("user tier service setting dependency is not configured")
	}
	return s.settingService.SetUserTierEnabled(ctx, enabled)
}

// ---------- 管理端配置读写 ----------

// ListTiers 列出等级配置（includeDisabled=true 时含停用档）
func (s *UserTierService) ListTiers(ctx context.Context, includeDisabled bool) ([]UserTier, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListTiers(ctx, includeDisabled)
}

// CreateTier 新建等级（等级与权益同事务写入；权益为整体提交）
func (s *UserTierService) CreateTier(ctx context.Context, input UserTierInput) (*UserTier, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user tier service is not configured")
	}
	tier := &UserTier{SortOrder: 0, Enabled: true}
	if input.Code != nil {
		tier.Code = strings.TrimSpace(*input.Code)
	}
	if input.Name != nil {
		tier.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		tier.Description = *input.Description
	}
	if input.TriggerType != nil {
		tier.TriggerType = *input.TriggerType
	}
	if input.ThresholdUSD != nil {
		v := *input.ThresholdUSD
		tier.ThresholdUSD = &v
	}
	if input.Enabled != nil {
		tier.Enabled = *input.Enabled
	}
	if input.Benefits != nil {
		tier.Benefits = benefitsFromInput(*input.Benefits)
	}
	// 新增档位排在最后，避免与既有 sort_order 冲突造成展示顺序抖动
	if existing, err := s.repo.ListTiers(ctx, true); err == nil {
		for _, t := range existing {
			if t.SortOrder >= tier.SortOrder {
				tier.SortOrder = t.SortOrder + 1
			}
		}
	}
	if err := s.validateTierConfig(ctx, tier, true); err != nil {
		return nil, err
	}
	if err := s.repo.CreateTier(ctx, tier); err != nil {
		return nil, err
	}
	return tier, nil
}

// UpdateTier 更新等级（指针字段表示不修改；Benefits 非 nil 时整体替换该档权益）
func (s *UserTierService) UpdateTier(ctx context.Context, id int64, input UserTierInput) (*UserTier, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user tier service is not configured")
	}
	tier, err := s.repo.GetTierByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		tier.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		tier.Description = *input.Description
	}
	if input.TriggerType != nil {
		tier.TriggerType = *input.TriggerType
	}
	if input.ThresholdUSD != nil {
		v := *input.ThresholdUSD
		tier.ThresholdUSD = &v
	}
	if input.Enabled != nil {
		tier.Enabled = *input.Enabled
	}
	// code 是幂等键：一旦有授予记录就不可改（改了会让既有 award 找不到配置）
	if input.Code != nil {
		next := strings.TrimSpace(*input.Code)
		if next != tier.Code {
			count, err := s.repo.CountAwardsByTierID(ctx, tier.ID)
			if err != nil {
				return nil, err
			}
			if count > 0 {
				return nil, ErrUserTierHasAwards
			}
			tier.Code = next
		}
	}
	if input.Benefits != nil {
		tier.Benefits = benefitsFromInput(*input.Benefits)
	}

	if err := s.validateTierConfig(ctx, tier, false); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateTier(ctx, tier); err != nil {
		return nil, err
	}
	return tier, nil
}

// ReorderTiers 按传入的 id 顺序重排展示顺序
func (s *UserTierService) ReorderTiers(ctx context.Context, orderedIDs []int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("user tier service is not configured")
	}
	if len(orderedIDs) == 0 {
		return nil
	}
	return s.repo.ReorderTiers(ctx, orderedIDs)
}

// DeleteTier 删除等级：只有从未产生授予记录的等级可以物理删除，其余只能停用
func (s *UserTierService) DeleteTier(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("user tier service is not configured")
	}
	return s.repo.DeleteTier(ctx, id)
}

// SetTierEnabled 启停等级（停用只是不再展示与发放，不影响历史授予）
func (s *UserTierService) SetTierEnabled(ctx context.Context, id int64, enabled bool) (*UserTier, error) {
	return s.UpdateTier(ctx, id, UserTierInput{Enabled: &enabled})
}

func benefitsFromInput(inputs []UserTierBenefitInput) []UserTierBenefit {
	out := make([]UserTierBenefit, 0, len(inputs))
	for i, in := range inputs {
		benefit := UserTierBenefit{
			ID:          in.ID,
			BenefitType: in.BenefitType,
			Enabled:     in.Enabled,
			SortOrder:   i,
		}
		switch in.BenefitType {
		case UserTierBenefitBalanceCredit:
			benefit.BalanceCredit = &UserTierBalanceCreditParams{Amount: in.Amount, ValidityDays: in.ValidityDays}
		case UserTierBenefitGroupRate:
			benefit.GroupRate = &UserTierGroupRateParams{GroupID: in.GroupID, RateMultiplier: in.RateMultiplier}
		}
		out = append(out, benefit)
	}
	return out
}

// validateTierConfig 权益参数按类型显式校验（jsonb 不能靠数据库约束兜底）。
// requireCode=true 时要求 code 非空（新增场景；更新场景沿用既有 code）。
func (s *UserTierService) validateTierConfig(ctx context.Context, tier *UserTier, requireCode bool) error {
	if tier == nil {
		return ErrUserTierInvalidConfig
	}
	if requireCode && !tierCodePattern.MatchString(tier.Code) {
		return ErrUserTierInvalidConfig
	}
	name := strings.TrimSpace(tier.Name)
	if name == "" || len([]rune(name)) > 128 {
		return ErrUserTierInvalidConfig
	}
	tier.Name = name

	switch tier.TriggerType {
	case UserTierTriggerConsumption:
		if tier.ThresholdUSD == nil || *tier.ThresholdUSD < 0 {
			return ErrUserTierInvalidConfig
		}
	case UserTierTriggerFirstRecharge:
		// 首充档没有阈值；显式清空避免残留旧值触发约束
		tier.ThresholdUSD = nil
	default:
		return ErrUserTierInvalidConfig
	}

	seenBenefitTypes := make(map[string]int, len(tier.Benefits))
	for i := range tier.Benefits {
		benefit := &tier.Benefits[i]
		if !benefit.Enabled {
			continue
		}
		switch benefit.BenefitType {
		case UserTierBenefitBalanceCredit:
			if benefit.BalanceCredit == nil {
				return ErrUserTierInvalidBenefit
			}
			if benefit.BalanceCredit.Amount <= 0 || benefit.BalanceCredit.ValidityDays < 0 {
				return ErrUserTierInvalidBenefit
			}
		case UserTierBenefitGroupRate:
			if benefit.GroupRate == nil {
				return ErrUserTierInvalidBenefit
			}
			if benefit.GroupRate.GroupID <= 0 || benefit.GroupRate.RateMultiplier <= 0 {
				return ErrUserTierInvalidBenefit
			}
			exists, err := s.repo.GroupExists(ctx, benefit.GroupRate.GroupID)
			if err != nil {
				return err
			}
			if !exists {
				return ErrUserTierInvalidBenefit
			}
		default:
			return ErrUserTierInvalidBenefit
		}
		seenBenefitTypes[benefit.BenefitType]++
	}
	// 一档同类权益只允许一项：额度与倍率都按「一项」发放，多项同类会让「领取一次」语义含糊
	for kind, count := range seenBenefitTypes {
		if count > 1 {
			logger.LegacyPrintf("service.user_tier", "reject duplicated benefit type=%s tier=%s", kind, tier.Code)
			return ErrUserTierInvalidBenefit
		}
	}
	return nil
}

// ---------- 用户端视图 ----------

// GetUserTierView 汇总当前档、下一档、各档领取状态与累计消费（用户端页面唯一取数入口）
func (s *UserTierService) GetUserTierView(ctx context.Context, userID int64) (*UserTierView, error) {
	enabled := s.IsFeatureEnabled(ctx)
	view := &UserTierView{Enabled: enabled, Tiers: []UserTierClaimState{}}
	if s == nil || s.repo == nil || userID <= 0 {
		return view, nil
	}
	// 总开关关闭：不暴露任何等级内容（前端据 Enabled 隐藏入口与页面）
	if !enabled {
		return view, nil
	}

	tiers, err := s.repo.ListTiers(ctx, false)
	if err != nil {
		return nil, err
	}
	consumed, err := s.repo.GetUserConsumedAmount(ctx, userID)
	if err != nil {
		return nil, err
	}
	awards, err := s.repo.ListUserAwards(ctx, userID)
	if err != nil {
		return nil, err
	}
	effects, err := s.repo.ListUserEffects(ctx, userID)
	if err != nil {
		return nil, err
	}
	view.ConsumedAmount = consumed

	awardByCode := make(map[string]UserTierAward, len(awards))
	for _, award := range awards {
		awardByCode[award.TierCode] = award
	}
	effectsByAward := make(map[int64][]UserTierEffect, len(awards))
	for _, effect := range effects {
		effectsByAward[effect.AwardID] = append(effectsByAward[effect.AwardID], effect)
	}

	// 首充档的「已发放」状态由既有额度账本判定（零回填，见设计 6、9 节）
	var firstRechargeCredit *UserTierFirstRechargeCredit
	for _, tier := range tiers {
		if tier.TriggerType == UserTierTriggerFirstRecharge {
			credit, err := s.repo.GetFirstRechargeCredit(ctx, userID)
			if err != nil {
				return nil, err
			}
			firstRechargeCredit = credit
			break
		}
	}

	groupNames := make(map[int64]string)
	states := make([]UserTierClaimState, 0, len(tiers))
	for _, tier := range tiers {
		state := s.buildClaimState(ctx, tier, consumed, awardByCode[tier.Code], effectsByAward, firstRechargeCredit, groupNames)
		states = append(states, state)
	}
	view.Tiers = states

	// 当前等级：按配置顺序取已获得的最高档（首充档也可以成为当前等级）。
	// 下一等级：尚未达成的第一个消费档 —— 已达成但未领取的档位由「可领取」入口承载，
	// 不占用「还差多少钱」的语义（否则会出现「还差 0 元」的下一等级）。
	var current *UserTierClaimState
	var next *UserTierClaimState
	for i := range states {
		if states[i].Claimed {
			current = &states[i]
		}
	}
	for i := range states {
		if states[i].TriggerType == UserTierTriggerConsumption && !states[i].Achieved {
			next = &states[i]
			break
		}
	}
	view.CurrentTier = current
	view.NextTier = next
	if next != nil && next.ThresholdUSD != nil {
		remaining := *next.ThresholdUSD - consumed
		if remaining < 0 {
			remaining = 0
		}
		view.RemainingToNext = &remaining
	}
	claimable := 0
	for i := range states {
		if states[i].Claimable {
			claimable++
		}
	}
	view.ClaimableCount = claimable
	return view, nil
}

func (s *UserTierService) buildClaimState(
	ctx context.Context,
	tier UserTier,
	consumed float64,
	award UserTierAward,
	effectsByAward map[int64][]UserTierEffect,
	firstRechargeCredit *UserTierFirstRechargeCredit,
	groupNames map[int64]string,
) UserTierClaimState {
	state := UserTierClaimState{
		TierID:       tier.ID,
		Code:         tier.Code,
		Name:         tier.Name,
		Description:  tier.Description,
		SortOrder:    tier.SortOrder,
		TriggerType:  tier.TriggerType,
		ThresholdUSD: tier.ThresholdUSD,
		Enabled:      tier.Enabled,
		Benefits:     []UserTierBenefitState{},
	}

	switch tier.TriggerType {
	case UserTierTriggerFirstRecharge:
		state.Achieved = firstRechargeCredit != nil
		state.Claimed = firstRechargeCredit != nil || award.ID > 0
		state.Source = UserTierAwardSourceFirstRecharge
		if firstRechargeCredit != nil {
			t := firstRechargeCredit.CreatedAt
			state.ClaimedAt = &t
		} else if !award.AchievedAt.IsZero() {
			t := award.AchievedAt
			state.ClaimedAt = &t
		}
	case UserTierTriggerConsumption:
		if tier.ThresholdUSD != nil {
			state.Achieved = consumed >= *tier.ThresholdUSD
		}
		state.Claimed = award.ID > 0
		state.Source = award.Source
		if !award.AchievedAt.IsZero() {
			t := award.AchievedAt
			state.ClaimedAt = &t
		}
		// 手动领取：已达标且未领取才可点
		state.Claimable = tier.Enabled && state.Achieved && !state.Claimed
	}

	applied := effectsByAward[award.ID]
	appliedByKey := make(map[string]UserTierEffect, len(applied))
	for _, effect := range applied {
		appliedByKey[effect.BenefitKey] = effect
	}

	for i := range tier.Benefits {
		benefit := tier.Benefits[i]
		entry := UserTierBenefitState{
			BenefitType:   benefit.BenefitType,
			BalanceCredit: benefit.BalanceCredit,
			GroupRate:     benefit.GroupRate,
			Status:        UserTierEffectPending,
		}
		if benefit.GroupRate != nil {
			entry.GroupName = s.groupName(ctx, benefit.GroupRate.GroupID, groupNames)
		}
		if effect, ok := appliedByKey[benefitKey(benefit)]; ok {
			entry.EffectID = effect.ID
			entry.Status = effect.Status
			entry.AppliedAt = effect.AppliedAt
			if reason, ok := effect.Detail["reason"].(string); ok {
				entry.SkippedReason = reason
			}
		}
		// 首充档：权益实发以既有账本为准（本档权益不产生新的 effect 记录）
		if tier.TriggerType == UserTierTriggerFirstRecharge && firstRechargeCredit != nil {
			entry.Status = UserTierEffectApplied
			if entry.BalanceCredit != nil {
				params := *entry.BalanceCredit
				params.Amount = firstRechargeCredit.Amount
				entry.BalanceCredit = &params
			}
			entry.AppliedAt = &firstRechargeCredit.CreatedAt
		}
		state.Benefits = append(state.Benefits, entry)
	}
	return state
}

func (s *UserTierService) groupName(ctx context.Context, groupID int64, cache map[int64]string) string {
	if groupID <= 0 {
		return ""
	}
	if name, ok := cache[groupID]; ok {
		return name
	}
	name, err := s.repo.GroupName(ctx, groupID)
	if err != nil {
		name = ""
	}
	cache[groupID] = name
	return name
}

func benefitKey(benefit UserTierBenefit) string {
	return fmt.Sprintf("benefit:%d", benefit.ID)
}

// ---------- 领取 ----------

// ClaimTier 手动领取消费档权益。幂等：重复请求返回既有状态而不是错误。
// 全程同一 Ent 事务：任一权益发放失败则整体回滚，不产生「等级已获得但权益永久缺失」的状态。
func (s *UserTierService) ClaimTier(ctx context.Context, userID, tierID int64) (*UserTierClaimResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("user tier service is not configured")
	}
	if userID <= 0 {
		return nil, ErrUserNotFound
	}
	// 未装配 ent client 时退化为无事务执行（仅用于未接入数据库的单元测试场景；
	// 生产装配始终有 client，走下面的同一事务路径）
	if s.entClient == nil {
		result, appliedAny, err := s.claim(ctx, userID, tierID)
		if err != nil {
			return nil, err
		}
		if appliedAny {
			s.invalidateUserCaches(ctx, userID)
		}
		return result, nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	result, appliedAny, err := s.claim(txCtx, userID, tierID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if appliedAny {
		// 缓存失效必须在提交之后：提交前失效会让并发请求把旧余额重新读回缓存
		s.invalidateUserCaches(ctx, userID)
	}
	return result, nil
}

// claim 领取的实际逻辑，运行在给定上下文（生产为事务上下文），
// 返回值第二项表示本次是否产生了实际到账（供提交后的缓存失效判定）。
// 任一权益发放失败即返回错误；调用方在事务路径上会整体回滚，
// 因此不会留下「等级已获得但权益永久缺失」的不可重试状态。
func (s *UserTierService) claim(ctx context.Context, userID, tierID int64) (*UserTierClaimResult, bool, error) {
	// ctx 在生产路径上已是事务上下文（见 ClaimTier），此处所有读写都在同一事务内
	// 总开关关闭时，任何档位都不可领取（服务端强制，避免直接打接口绕过前端）
	if !s.IsFeatureEnabled(ctx) {
		return nil, false, ErrUserTierFeatureDisabled
	}
	tier, err := s.repo.GetTierByID(ctx, tierID)
	if err != nil {
		return nil, false, err
	}
	if !tier.Enabled {
		return nil, false, ErrUserTierDisabled
	}
	if tier.TriggerType != UserTierTriggerConsumption {
		return nil, false, ErrUserTierNotClaimable
	}

	consumed, err := s.repo.GetUserConsumedAmount(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if tier.ThresholdUSD == nil || consumed < *tier.ThresholdUSD {
		return nil, false, ErrUserTierNotAchieved
	}

	code := tier.Code
	awardID, created, err := s.repo.EnsureAward(ctx, &UserTierAward{
		UserID:            userID,
		TierID:            &tier.ID,
		TierCode:          code,
		TierNameSnapshot:  tier.Name,
		ThresholdSnapshot: tier.ThresholdUSD,
		Source:            UserTierAwardSourceClaim,
	})
	if err != nil {
		return nil, false, err
	}

	result := &UserTierClaimResult{TierID: tier.ID, Code: code, Name: tier.Name, AlreadyClaimed: !created}
	appliedAny := false

	for i := range tier.Benefits {
		benefit := tier.Benefits[i]
		if !benefit.Enabled {
			continue
		}
		key := benefitKey(benefit)
		effect, effectCreated, err := s.repo.EnsureEffect(ctx, &UserTierEffect{
			AwardID:        awardID,
			UserID:         userID,
			BenefitKey:     key,
			BenefitType:    benefit.BenefitType,
			Amount:         amountOf(benefit),
			GroupID:        groupIDOf(benefit),
			RateMultiplier: groupRateOf(benefit),
		})
		if err != nil {
			return nil, false, err
		}
		if !effectCreated && effect.Status == UserTierEffectApplied {
			// 该权益此前已发放，不重复发放
			continue
		}

		applied, err := s.applyBenefit(ctx, userID, awardID, effect.ID, benefit, result)
		if err != nil {
			return nil, false, err
		}
		if applied {
			appliedAny = true
		}
	}

	return result, appliedAny, nil
}

// applyBenefit 发放单项权益并标记 effect 状态。返回是否产生了实际到账（供缓存失效判定）。
func (s *UserTierService) applyBenefit(
	ctx context.Context,
	userID, awardID, effectID int64,
	benefit UserTierBenefit,
	result *UserTierClaimResult,
) (bool, error) {
	switch benefit.BenefitType {
	case UserTierBenefitBalanceCredit:
		params := benefit.BalanceCredit
		if params == nil || params.Amount <= 0 {
			return false, ErrUserTierInvalidBenefit
		}
		appliedAt := time.Now()
		expiresAt := balanceCreditExpiresAt(params.ValidityDays, appliedAt)
		if s.balanceCreditRepo == nil {
			return false, fmt.Errorf("balance credit repository is nil")
		}
		if err := s.balanceCreditRepo.CreateCredit(ctx, BalanceCreditCreate{
			UserID:     userID,
			SourceType: BalanceCreditSourceTierReward,
			SourceID:   fmt.Sprintf("%d", effectID),
			SourceCode: fmt.Sprintf("tier_award_%d", awardID),
			Amount:     params.Amount,
			ExpiresAt:  expiresAt,
		}); err != nil {
			return false, err
		}
		if err := s.repo.ApplyTierRewardBalance(ctx, userID, params.Amount); err != nil {
			return false, err
		}
		if err := s.repo.MarkEffectApplied(ctx, effectID, map[string]any{
			"outcome":       "granted",
			"amount":        params.Amount,
			"validity_days": params.ValidityDays,
		}); err != nil {
			return false, err
		}
		result.Applied = append(result.Applied, UserTierAppliedBenefit{
			BenefitType:  UserTierBenefitBalanceCredit,
			Amount:       params.Amount,
			ValidityDays: params.ValidityDays,
			ExpiresAt:    expiresAt,
		})
		return true, nil

	case UserTierBenefitGroupRate:
		params := benefit.GroupRate
		if params == nil || params.GroupID <= 0 || params.RateMultiplier <= 0 {
			return false, ErrUserTierInvalidBenefit
		}
		// 冲突规则：手工倍率优先，等级倍率不覆盖手工值（仅在 detail 记录跳过原因）
		hasManual, err := s.repo.HasManualRateMultiplier(ctx, userID, params.GroupID)
		if err != nil {
			return false, err
		}
		if hasManual {
			if err := s.repo.MarkEffectApplied(ctx, effectID, map[string]any{
				"outcome":  "skipped",
				"reason":   UserTierSkipManualRateMultiplier,
				"group_id": params.GroupID,
			}); err != nil {
				return false, err
			}
			result.Applied = append(result.Applied, UserTierAppliedBenefit{
				BenefitType:    UserTierBenefitGroupRate,
				GroupID:        params.GroupID,
				RateMultiplier: params.RateMultiplier,
				SkippedReason:  UserTierSkipManualRateMultiplier,
			})
			return false, nil
		}
		if err := s.repo.UpsertRateOverlay(ctx, UserTierRateOverlay{
			UserID:         userID,
			GroupID:        params.GroupID,
			EffectID:       effectID,
			RateMultiplier: params.RateMultiplier,
			Status:         "active",
		}); err != nil {
			return false, err
		}
		if err := s.repo.MarkEffectApplied(ctx, effectID, map[string]any{
			"outcome":  "granted",
			"group_id": params.GroupID,
			"rate":     params.RateMultiplier,
		}); err != nil {
			return false, err
		}
		groupName, _ := s.repo.GroupName(ctx, params.GroupID)
		result.Applied = append(result.Applied, UserTierAppliedBenefit{
			BenefitType:    UserTierBenefitGroupRate,
			GroupID:        params.GroupID,
			GroupName:      groupName,
			RateMultiplier: params.RateMultiplier,
		})
		return true, nil

	default:
		return false, ErrUserTierInvalidBenefit
	}
}

// ---------- 管理端只读查看 ----------

// GetUserTierAdminView 查看某用户的累计消费、授予与发放记录（只读，供运营核对）
func (s *UserTierService) GetUserTierAdminView(ctx context.Context, userID int64) (*UserTierView, []UserTierAward, []UserTierEffect, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return &UserTierView{Tiers: []UserTierClaimState{}}, nil, nil, nil
	}
	view, err := s.GetUserTierView(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	awards, err := s.repo.ListUserAwards(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	effects, err := s.repo.ListUserEffects(ctx, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	return view, awards, effects, nil
}

// ---------- 首充单源读取（C5） ----------

// ResolveFirstRechargeGrant 读取首充档配置（单源化后首充发放入口的唯一配置来源）。
// 返回 (nil, nil) 表示服务未装配等级仓储（调用方应沿用既有 settings 行为）。
// 返回错误表示等级配置缺失或非法，调用方必须回退到既有行为并记录可见错误，不得静默不发。
func (s *UserTierService) ResolveFirstRechargeGrant(ctx context.Context) (*FirstRechargeTierGrant, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	// 总开关关闭 = 回到迁移前状态：首充发放沿用既有 settings 行为（C5 的回滚语义），
	// 不再由等级配置触发。
	if !s.IsFeatureEnabled(ctx) {
		return nil, nil
	}
	tier, err := s.repo.GetTierByCode(ctx, UserTierCodeFirstRecharge)
	if err != nil {
		return nil, err
	}
	if tier.TriggerType != UserTierTriggerFirstRecharge {
		return nil, fmt.Errorf("%w: tier %s has trigger_type=%s", ErrUserTierInvalidConfig, tier.Code, tier.TriggerType)
	}
	grant := &FirstRechargeTierGrant{
		TierCode: tier.Code,
		TierName: tier.Name,
		Enabled:  tier.Enabled,
	}
	for _, benefit := range tier.Benefits {
		if !benefit.Enabled || benefit.BenefitType != UserTierBenefitBalanceCredit || benefit.BalanceCredit == nil {
			continue
		}
		grant.Amount = benefit.BalanceCredit.Amount
		grant.ValidityDays = benefit.BalanceCredit.ValidityDays
	}
	if grant.Enabled && grant.Amount <= 0 {
		return nil, fmt.Errorf("%w: tier %s has no valid balance_credit benefit", ErrUserTierInvalidConfig, tier.Code)
	}
	return grant, nil
}

// ---------- 缓存失效 ----------

func (s *UserTierService) invalidateUserCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
			logger.LegacyPrintf("service.user_tier", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
		}
	}()
}

func amountOf(benefit UserTierBenefit) *float64 {
	if benefit.BalanceCredit == nil {
		return nil
	}
	v := benefit.BalanceCredit.Amount
	return &v
}

func groupIDOf(benefit UserTierBenefit) *int64 {
	if benefit.GroupRate == nil {
		return nil
	}
	v := benefit.GroupRate.GroupID
	return &v
}

func groupRateOf(benefit UserTierBenefit) *float64 {
	if benefit.GroupRate == nil {
		return nil
	}
	v := benefit.GroupRate.RateMultiplier
	return &v
}

package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// userTierCodeMaxRunes 等级标识的长度上限，与数据库列 VARCHAR(64) 一致
// （Postgres 的 VARCHAR(n) 按字符计数，故此处也按 rune 计）。
const userTierCodeMaxRunes = 64

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
	// userLookup 按邮箱解析用户（管理端手工指派的入口）；nil 时指派入口返回明确错误。
	userLookup UserTierUserLookup
}

// NewUserTierService 创建用户等级服务
func NewUserTierService(
	repo UserTierRepository,
	balanceCreditRepo BalanceCreditRepository,
	entClient *dbent.Client,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	billingCacheService *BillingCacheService,
	settingService *SettingService,
	userLookup UserTierUserLookup,
) *UserTierService {
	return &UserTierService{
		repo:                 repo,
		balanceCreditRepo:    balanceCreditRepo,
		entClient:            entClient,
		authCacheInvalidator: authCacheInvalidator,
		billingCacheService:  billingCacheService,
		settingService:       settingService,
		userLookup:           userLookup,
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
	if err := s.validateTierConfig(ctx, tier); err != nil {
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

	if err := s.validateTierConfig(ctx, tier); err != nil {
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

// validateUserTierCode 等级标识只保留两条硬约束：去空白后非空、长度不超过列宽。
//
// 刻意**不再限制字符集**（2026-09-24 用户要求「code 去掉限制，可以设置为任意值」，
// 原实现为 ^[a-z0-9_]{1,64}$）：标识不参与任何 URL 路由或文件路径，Go 侧查询全部参数化
// （如 GetTierByCode 的 `WHERE t.code = $1`），唯一索引保证唯一性，因此放开字符集不带来注入面。
func validateUserTierCode(code string) error {
	if code == "" || len([]rune(code)) > userTierCodeMaxRunes {
		return ErrUserTierInvalidConfig
	}
	return nil
}

// validateTierConfig 权益参数按类型显式校验（jsonb 不能靠数据库约束兜底）。
// 新增与更新走同一套校验：code 是幂等键（award 唯一约束与首充查找都用它），
// 更新路径同样不能写入非法 code（此前只在新增时校验，改名可绕过）。
func (s *UserTierService) validateTierConfig(ctx context.Context, tier *UserTier) error {
	if tier == nil {
		return ErrUserTierInvalidConfig
	}
	// code 与 name 一样就地去掉首尾空白后再校验，避免「合法但带空白」的标识落库
	tier.Code = strings.TrimSpace(tier.Code)
	if err := validateUserTierCode(tier.Code); err != nil {
		return err
	}
	// 首充档是被写死查找的档位（见 UserTierCodeFirstRecharge）：语义上全系统只有一个，
	// 且必须用固定 code。两种情况都在此拦下——
	//   ① 把首充档改名：首充发放只认 UserTierCodeFirstRecharge，改完会静默停掉首充赠送；
	//   ② 另建一个 first_recharge 档：GetUserTierView 只认排序最前的首充档，且首充档不可手动
	//      领取，新建的那个永远不可领取（死档）。
	if tier.TriggerType == UserTierTriggerFirstRecharge && tier.Code != UserTierCodeFirstRecharge {
		logger.LegacyPrintf("service.user_tier",
			"reject non-canonical first_recharge tier: code=%s", tier.Code)
		return ErrUserTierFirstRechargeCodeLocked
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

// ---------- 管理端手工指派（按邮箱） ----------

// tierLadderFloor 被指派档位在阶梯中的位置 (sort_order, id)。
// id 只用于 sort_order 相同时的稳定比较（ReorderTiers 写入互不相同的序号，此处是兜底）。
type tierLadderFloor struct {
	SortOrder int
	TierID    int64
}

// tierCoveredByFloor 该档位是否落在指派的覆盖范围内（阶梯位置不晚于被指派档位）。
// 「被指派档位及其之前的档位都视为已达成」这条语义只在这里实现一次。
func tierCoveredByFloor(tier UserTier, floor *tierLadderFloor) bool {
	if floor == nil {
		return false
	}
	if tier.SortOrder != floor.SortOrder {
		return tier.SortOrder < floor.SortOrder
	}
	return tier.ID <= floor.TierID
}

// consumptionTierAchieved 消费档是否达成：真实累计消费达标，或被管理员指派覆盖。
// 用户端视图与领取校验共用这一个实现——两处判定分叉就会重现「页面显示可领、领取被拒」这类缺陷。
func consumptionTierAchieved(tier UserTier, consumed float64, floor *tierLadderFloor) bool {
	if tierCoveredByFloor(tier, floor) {
		return true
	}
	return tier.ThresholdUSD != nil && consumed >= *tier.ThresholdUSD
}

// resolveAssignment 读取用户当前指派并解析其阶梯位置。
// liveTier 为 nil 表示档位配置已不存在（只剩快照）；三者均为 nil 表示该用户没有指派。
func (s *UserTierService) resolveAssignment(ctx context.Context, userID int64) (*UserTierAssignment, *UserTier, *tierLadderFloor, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, nil, nil, nil
	}
	assignment, err := s.repo.GetAssignment(ctx, userID)
	if err != nil || assignment == nil {
		return nil, nil, nil, err
	}
	if assignment.TierID != nil && *assignment.TierID > 0 {
		tier, err := s.repo.GetTierByID(ctx, *assignment.TierID)
		switch {
		case err == nil && tier != nil:
			return assignment, tier, &tierLadderFloor{SortOrder: tier.SortOrder, TierID: tier.ID}, nil
		case errors.Is(err, ErrUserTierNotFound):
			// 档位配置被直接改库删除（服务层另有删除守卫，正常路径不可达）：退回顺序快照，
			// 并留下可见记录，不静默
			logger.LegacyPrintf("service.user_tier",
				"assigned tier %d of user %d is missing, fall back to sort_order snapshot=%d",
				*assignment.TierID, userID, assignment.SortOrderSnapshot)
		default:
			// 读取真失败时不假装有 floor：宁可让本次请求失败，也不把未知状态当成已达成
			return nil, nil, nil, err
		}
	}
	return assignment, nil, &tierLadderFloor{SortOrder: assignment.SortOrderSnapshot, TierID: math.MaxInt64}, nil
}

// assignedTierState 构造「当前指派档位」的展示状态：优先复用已构建的档位状态
// （这样呈现的 Achieved/Claimable 就是真实判定），档位被停用或配置已删除时退回快照构造，
// 此时 Enabled=false 表示该档位当前不可领取。
func assignedTierState(states []UserTierClaimState, assignment *UserTierAssignment, liveTier *UserTier) *UserTierClaimState {
	if assignment == nil {
		return nil
	}
	for i := range states {
		if liveTier != nil && states[i].TierID == liveTier.ID {
			return &states[i]
		}
		if states[i].Code == assignment.TierCode {
			return &states[i]
		}
	}
	state := UserTierClaimState{
		Code:        assignment.TierCode,
		Name:        assignment.TierNameSnapshot,
		SortOrder:   assignment.SortOrderSnapshot,
		TriggerType: UserTierTriggerConsumption,
		Benefits:    []UserTierBenefitState{},
	}
	if liveTier != nil {
		state.TierID = liveTier.ID
		state.Name = liveTier.Name
		state.Description = liveTier.Description
		state.SortOrder = liveTier.SortOrder
		state.TriggerType = liveTier.TriggerType
		state.ThresholdUSD = liveTier.ThresholdUSD
		state.Enabled = liveTier.Enabled
	}
	return &state
}

// resolveUserByEmail 按邮箱解析用户：复用既有 userRepository.GetByEmail 的规范化口径
// （LOWER(TRIM(email))），不另写一套匹配规则，避免与登录/注册链路的口径分叉。
func (s *UserTierService) resolveUserByEmail(ctx context.Context, email string) (*User, error) {
	if s == nil || s.userLookup == nil {
		return nil, fmt.Errorf("user tier service user lookup is not configured")
	}
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return nil, ErrUserTierAssignUserNotFound
	}
	user, err := s.userLookup.GetByEmail(ctx, trimmed)
	if err == nil && user != nil {
		return user, nil
	}
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrUserTierAssignUserNotFound
	}
	if err != nil {
		// 既有的 GetByEmail 在「规范化后匹配到多行」时返回错误（本 schema 的规范化邮箱写入门禁
		// 下不应出现）。这里不猜、不静默挑一个：按邮箱写错人比写不进去严重得多，统一交人工核对；
		// 真实读失败也落这一分支，两者靠下面的日志区分。
		logger.LegacyPrintf("service.user_tier", "resolve user by email failed: err=%v", err)
		return nil, ErrUserTierAssignEmailAmbiguous
	}
	return nil, ErrUserTierAssignUserNotFound
}

// GetUserAssignmentByEmail 管理端按邮箱查询用户与其当前指派（无指派时 assignment 为 nil）。
func (s *UserTierService) GetUserAssignmentByEmail(ctx context.Context, email string) (*User, *UserTierAssignment, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("user tier service is not configured")
	}
	user, err := s.resolveUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	assignment, err := s.repo.GetAssignment(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, assignment, nil
}

// AssignUserTierByEmail 按邮箱把用户指派到指定消费档（重指派即覆盖）。
// 只写指派表：不发放任何权益，用户仍需在「我的等级」页手动领取。
func (s *UserTierService) AssignUserTierByEmail(ctx context.Context, email, tierCode, note string, actorUserID int64) (*User, *UserTierAssignment, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("user tier service is not configured")
	}
	// 总开关关闭时拒绝写入：否则会留下「配置成功但用户端完全不可见」的静默状态
	if !s.IsFeatureEnabled(ctx) {
		return nil, nil, ErrUserTierFeatureDisabled
	}
	user, err := s.resolveUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	tier, err := s.repo.GetTierByCode(ctx, strings.TrimSpace(tierCode))
	if err != nil {
		return nil, nil, err
	}
	// 只允许指派消费档：首充档的权益由额度账本驱动、本身不可手动领取，指派它只会得到一个
	// 「看得见却领不到」的死状态；停用档同理。
	if tier.TriggerType != UserTierTriggerConsumption {
		return nil, nil, ErrUserTierAssignmentTargetNotConsumption
	}
	if !tier.Enabled {
		return nil, nil, ErrUserTierAssignmentTargetDisabled
	}
	assignment := &UserTierAssignment{
		UserID:            user.ID,
		TierID:            &tier.ID,
		TierCode:          tier.Code,
		TierNameSnapshot:  tier.Name,
		SortOrderSnapshot: tier.SortOrder,
		Source:            UserTierAssignmentSourceAdmin,
		Note:              strings.TrimSpace(note),
	}
	if actorUserID > 0 {
		actor := actorUserID
		assignment.AssignedBy = &actor
	}
	stored, err := s.repo.UpsertAssignment(ctx, assignment)
	if err != nil {
		return nil, nil, err
	}
	return user, stored, nil
}

// UnassignUserTierByEmail 取消指派（幂等：本来就没有指派时 removed=false 且不报错）。
// 取消只影响「尚未领取的档位是否可领取」，已发放的余额与倍率覆盖按既有「只升不降」规则不回收。
func (s *UserTierService) UnassignUserTierByEmail(ctx context.Context, email string) (*User, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, fmt.Errorf("user tier service is not configured")
	}
	user, err := s.resolveUserByEmail(ctx, email)
	if err != nil {
		return nil, false, err
	}
	removed, err := s.repo.DeleteAssignment(ctx, user.ID)
	if err != nil {
		return nil, false, err
	}
	return user, removed, nil
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
	// 手工指派（若有）：只抬高「是否达成」的下限，不改变累计消费本身的数值
	assignment, assignmentTier, floor, err := s.resolveAssignment(ctx, userID)
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
		state := s.buildClaimState(ctx, tier, consumed, floor, awardByCode[tier.Code], effectsByAward, firstRechargeCredit, groupNames)
		states = append(states, state)
	}
	view.Tiers = states
	view.AssignedTier = assignedTierState(states, assignment, assignmentTier)

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
	floor *tierLadderFloor,
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
		// 达成判定与领取校验同一实现：消费达标，或被管理员指派覆盖（指派档位及其之前的档位）
		state.Achieved = consumptionTierAchieved(tier, consumed, floor)
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
	// 达成判定必须与用户端视图同源：被管理员指派的档位及其之前的档位同样可领取
	_, _, floor, err := s.resolveAssignment(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if !consumptionTierAchieved(*tier, consumed, floor) {
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
			// 失败即整事务回滚。刻意不写 effect.status='failed'：回滚后 effect 行本身不存在，
			// 没有可标记的对象（保留 failed 语义需要一个「等级已获得但权益缺失」的半成品状态，
			// 与失败整体回滚互斥）。故这里打一条带上下文的日志，作为发放失败的唯一可见面。
			logger.LegacyPrintf("service.user_tier",
				"claim benefit failed, transaction rolls back: user_id=%d tier=%s benefit_key=%s benefit_type=%s err=%v",
				userID, tier.Code, key, benefit.BenefitType, err)
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

// GetUserTierAdminView 查看某用户的累计消费、等级状态、当前指派、授予与发放记录（只读，供运营核对）
func (s *UserTierService) GetUserTierAdminView(ctx context.Context, userID int64) (*UserTierView, *UserTierAssignment, []UserTierAward, []UserTierEffect, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return &UserTierView{Tiers: []UserTierClaimState{}}, nil, nil, nil, nil
	}
	view, err := s.GetUserTierView(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	assignment, err := s.repo.GetAssignment(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	awards, err := s.repo.ListUserAwards(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	effects, err := s.repo.ListUserEffects(ctx, userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return view, assignment, awards, effects, nil
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

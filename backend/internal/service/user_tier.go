package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 等级触发类型：代码只保留这两层枚举，具体档位（称呼/阈值/权益数值）全部是管理端数据。
const (
	UserTierTriggerFirstRecharge = "first_recharge"
	UserTierTriggerConsumption   = "consumption"
)

// 权益类型：首期只做额度赠送与分组倍率，RPM/并发等留作后续扩展位（类型集合扩展不必改表）。
const (
	UserTierBenefitBalanceCredit = "balance_credit"
	UserTierBenefitGroupRate     = "group_rate"
)

// user_tier_effects.status
const (
	UserTierEffectPending = "pending"
	UserTierEffectApplied = "applied"
	UserTierEffectFailed  = "failed"
)

// user_tier_awards.source
const (
	UserTierAwardSourceClaim         = "claim"
	UserTierAwardSourceFirstRecharge = "first_recharge"
)

// 等级权益发放的固定档位 code：首充档由既有首充链路自动发放，等级页显示为已领取。
// 该 code 是「首充档」这一语义的稳定标识，不是具体称呼或阈值。
const UserTierCodeFirstRecharge = "first_recharge"

// 等级权益跳过倍率发放的原因（写入 effect.detail，供管理端核对）
const UserTierSkipManualRateMultiplier = "manual_rate_multiplier_present"

var (
	ErrUserTierNotFound       = infraerrors.NotFound("USER_TIER_NOT_FOUND", "等级配置不存在")
	ErrUserTierDisabled       = infraerrors.BadRequest("USER_TIER_DISABLED", "该等级已停用，无法领取")
	ErrUserTierNotClaimable   = infraerrors.BadRequest("USER_TIER_NOT_CLAIMABLE", "该等级不支持手动领取")
	ErrUserTierNotAchieved    = infraerrors.BadRequest("USER_TIER_NOT_ACHIEVED", "尚未达到该等级的消费门槛")
	ErrUserTierHasAwards      = infraerrors.Conflict("USER_TIER_HAS_AWARDS", "该等级已产生授予记录，只能停用，不能删除")
	ErrUserTierCodeExists     = infraerrors.Conflict("USER_TIER_CODE_EXISTS", "等级标识已存在")
	ErrUserTierInvalidBenefit = infraerrors.BadRequest("USER_TIER_INVALID_BENEFIT", "权益参数非法")
	ErrUserTierInvalidConfig  = infraerrors.BadRequest("USER_TIER_INVALID_CONFIG", "等级配置非法")
	// ErrUserTierFirstRechargeCodeLocked 首充档唯一且标识固定：既不可改名（首充发放按
	// UserTierCodeFirstRecharge 查找，改名会静默停发），也不可另建第二个首充档（永远不可领取）
	ErrUserTierFirstRechargeCodeLocked = infraerrors.BadRequest("USER_TIER_FIRST_RECHARGE_CODE_LOCKED", "首充档必须使用固定标识 first_recharge，且不可更改")
	// ErrUserTierFeatureDisabled 总开关关闭（服务端强制，不只靠前端隐藏入口）
	ErrUserTierFeatureDisabled = infraerrors.Forbidden("USER_TIER_FEATURE_DISABLED", "用户等级体系当前已关闭")
)

// UserTierBalanceCreditParams 额度赠送权益参数（user_tier_benefits.params）
type UserTierBalanceCreditParams struct {
	// Amount 赠送金额（美元），必须 > 0
	Amount float64 `json:"amount"`
	// ValidityDays 有效期天数；<= 0 表示不过期
	ValidityDays int `json:"validity_days"`
}

// UserTierGroupRateParams 分组专属倍率权益参数（user_tier_benefits.params）
type UserTierGroupRateParams struct {
	// GroupID 目标分组
	GroupID int64 `json:"group_id"`
	// RateMultiplier 专属倍率，必须 > 0；只写入等级覆盖层，不覆盖手工倍率
	RateMultiplier float64 `json:"rate_multiplier"`
}

// UserTierBenefit 一项权益（params 已按 benefit_type 解析为强类型，二者只会有其一非空）
type UserTierBenefit struct {
	ID            int64
	TierID        int64
	BenefitType   string
	BalanceCredit *UserTierBalanceCreditParams
	GroupRate     *UserTierGroupRateParams
	Enabled       bool
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserTier 等级定义（可无限增删；已产生授予的等级只能停用）
type UserTier struct {
	ID           int64
	Code         string
	Name         string
	Description  string
	SortOrder    int
	TriggerType  string
	ThresholdUSD *float64
	Enabled      bool
	Benefits     []UserTierBenefit
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// AwardCount 仅管理端列表使用：已产生授予记录数（> 0 时禁止物理删除）
	AwardCount int64
}

// UserTierAward 已获得等级快照
type UserTierAward struct {
	ID                int64
	UserID            int64
	TierID            *int64
	TierCode          string
	TierNameSnapshot  string
	ThresholdSnapshot *float64
	AchievedAt        time.Time
	Source            string
	Detail            map[string]any
}

// UserTierEffect 权益发放状态（幂等与重试锚点，实发金额/倍率留快照）
type UserTierEffect struct {
	ID             int64
	AwardID        int64
	UserID         int64
	BenefitKey     string
	BenefitType    string
	Amount         *float64
	GroupID        *int64
	RateMultiplier *float64
	Status         string
	Attempts       int
	LastError      string
	Detail         map[string]any
	AppliedAt      *time.Time
}

// UserTierRateOverlay 等级倍率覆盖层条目
type UserTierRateOverlay struct {
	ID             int64
	UserID         int64
	GroupID        int64
	EffectID       int64
	RateMultiplier float64
	Status         string
	CreatedAt      time.Time
}

// UserTierInput 管理端写入等级（指针字段表示「不修改」；Benefits 非 nil 表示整体替换权益列表）
type UserTierInput struct {
	Code         *string
	Name         *string
	Description  *string
	TriggerType  *string
	ThresholdUSD *float64
	Enabled      *bool
	Benefits     *[]UserTierBenefitInput
}

// UserTierBenefitInput 管理端写入权益（字段含义随 BenefitType 变化）
type UserTierBenefitInput struct {
	// ID 为 0 表示新增；非 0 表示更新既有权益（保持 benefit_key 稳定，已发 effect 不受影响）
	ID             int64
	BenefitType    string
	Amount         float64
	ValidityDays   int
	GroupID        int64
	RateMultiplier float64
	Enabled        bool
}

// UserTierBenefitState 用户端展示的单项权益状态
type UserTierBenefitState struct {
	BenefitType   string
	BalanceCredit *UserTierBalanceCreditParams
	GroupRate     *UserTierGroupRateParams
	GroupName     string
	Status        string
	AppliedAt     *time.Time
	SkippedReason string
	EffectID      int64
}

// UserTierClaimState 用户端展示的单个档位状态
type UserTierClaimState struct {
	TierID       int64
	Code         string
	Name         string
	Description  string
	SortOrder    int
	TriggerType  string
	ThresholdUSD *float64
	Enabled      bool
	Achieved     bool
	Claimed      bool
	Claimable    bool
	ClaimedAt    *time.Time
	Source       string
	Benefits     []UserTierBenefitState
}

// UserTierView 用户端「我的等级」聚合视图
type UserTierView struct {
	// Enabled 总开关状态：false 时 Tiers 为空，前端据此隐藏入口与内容
	Enabled         bool
	ConsumedAmount  float64
	CurrentTier     *UserTierClaimState
	NextTier        *UserTierClaimState
	RemainingToNext *float64
	Tiers           []UserTierClaimState
	ClaimableCount  int
}

// UserTierClaimResult 领取回执（重复领取返回既有状态而不是错误）
type UserTierClaimResult struct {
	TierID         int64
	Code           string
	Name           string
	AlreadyClaimed bool
	Applied        []UserTierAppliedBenefit
}

// UserTierAppliedBenefit 本次实际到账的权益（供前端即时提示）
type UserTierAppliedBenefit struct {
	BenefitType    string
	Amount         float64
	ValidityDays   int
	ExpiresAt      *time.Time
	GroupID        int64
	GroupName      string
	RateMultiplier float64
	SkippedReason  string
}

// UserTierFirstRechargeCredit 既有首充奖励的账本记录（首充档的「已发放」状态与实发金额由此判定，
// 因此 433 名已发用户零回填）
type UserTierFirstRechargeCredit struct {
	Amount    float64
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// FirstRechargeTierGrant 首充档解析结果（C5 单源化的读取结果）
type FirstRechargeTierGrant struct {
	TierCode     string
	TierName     string
	Enabled      bool
	Amount       float64
	ValidityDays int
}

// UserTierRepository 等级配置、授予与发放状态的持久化接口（raw SQL 实现）
type UserTierRepository interface {
	ListTiers(ctx context.Context, includeDisabled bool) ([]UserTier, error)
	GetTierByID(ctx context.Context, id int64) (*UserTier, error)
	GetTierByCode(ctx context.Context, code string) (*UserTier, error)
	// CreateTier 等级与权益同事务写入，成功回填 ID
	CreateTier(ctx context.Context, tier *UserTier) error
	// UpdateTier 等级与权益同事务写入（Benefits 非 nil 时整体替换）
	UpdateTier(ctx context.Context, tier *UserTier) error
	ReorderTiers(ctx context.Context, orderedIDs []int64) error
	// DeleteTier 仅在没有授予记录时删除，否则返回 ErrUserTierHasAwards
	DeleteTier(ctx context.Context, id int64) error
	CountAwardsByTierID(ctx context.Context, tierID int64) (int64, error)
	GroupExists(ctx context.Context, groupID int64) (bool, error)
	GroupName(ctx context.Context, groupID int64) (string, error)

	// GetUserConsumedAmount 累计消费额（只算兑换额度消费，见设计第 3 节口径）
	GetUserConsumedAmount(ctx context.Context, userID int64) (float64, error)
	// GetFirstRechargeCredit 读取既有首充奖励账本记录（无记录返回 nil, nil）
	GetFirstRechargeCredit(ctx context.Context, userID int64) (*UserTierFirstRechargeCredit, error)
	ListUserAwards(ctx context.Context, userID int64) ([]UserTierAward, error)
	ListUserEffects(ctx context.Context, userID int64) ([]UserTierEffect, error)
	ListUserRateOverlays(ctx context.Context, userID int64) ([]UserTierRateOverlay, error)

	// EnsureAward 幂等创建授予记录，返回 (id, 是否新建, err)
	EnsureAward(ctx context.Context, award *UserTierAward) (int64, bool, error)
	// EnsureEffect 幂等创建权益发放记录，返回 (当前库内状态, 是否新建, err)
	EnsureEffect(ctx context.Context, effect *UserTierEffect) (*UserTierEffect, bool, error)
	// MarkEffectApplied 标记单项权益已发放。
	// 刻意没有 MarkEffectFailed：任一权益失败即整事务回滚，回滚后 effect 行不存在，
	// 没有可标记为 failed 的对象；重试语义 = 用户重新发起领取（见 ClaimTier 注释）。
	MarkEffectApplied(ctx context.Context, effectID int64, detail map[string]any) error

	HasManualRateMultiplier(ctx context.Context, userID, groupID int64) (bool, error)
	UpsertRateOverlay(ctx context.Context, overlay UserTierRateOverlay) error

	// ApplyTierRewardBalance 只增加余额，不累加 users.total_recharged
	//（UpdateBalance 会累加 total_recharged，与「奖励不计入充值额」的口径冲突）
	ApplyTierRewardBalance(ctx context.Context, userID int64, amount float64) error
}

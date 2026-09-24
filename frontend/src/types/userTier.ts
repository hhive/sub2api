/**
 * 用户等级与权益类型
 *
 * 后端接口：
 * - 用户端 GET /user/tier、POST /user/tier/claim
 * - 管理端 /admin/tiers、/admin/users/:id/tier
 *
 * 口径：consumed_amount 只统计「兑换额度消费」，不含订阅消费。
 */

/** 档位达成方式：按累计消费达成 / 首次充值自动发放 */
export type UserTierTriggerType = 'first_recharge' | 'consumption'

/** 权益类型：发放余额 / 覆盖分组倍率 */
export type UserTierBenefitType = 'balance_credit' | 'group_rate'

/** 权益发放状态 */
export type UserTierBenefitStatus = 'pending' | 'applied' | 'failed'

/** 用户端某一档位下的权益状态（后端 omitempty，缺省字段按「未提供」处理） */
export interface UserTierBenefitState {
  benefit_type: UserTierBenefitType
  amount?: number
  validity_days?: number
  expires_at?: string
  group_id?: number
  group_name?: string
  rate_multiplier?: number
  status: UserTierBenefitStatus
  applied_at?: string
  /** 跳过原因，例如 manual_rate_multiplier_present（已有专属倍率） */
  skipped_reason?: string
}

/** 用户端单个档位的达成与领取状态 */
export interface UserTierState {
  tier_id: number
  code: string
  name: string
  description?: string
  trigger_type: UserTierTriggerType
  threshold_usd?: number | null
  achieved: boolean
  claimed: boolean
  claimable: boolean
  claimed_at?: string
  /** 授予来源：claim / first_recharge / manual_20260924 / backfill */
  source?: string
  benefits: UserTierBenefitState[]
}

/** GET /user/tier 响应体 */
export interface MyTierResponse {
  /**
   * 等级功能是否对当前用户开放。后端总开关关闭时为 false，
   * 同时 tiers 为空、claimable_count 为 0、current_tier/next_tier 为 null。
   */
  enabled: boolean
  consumed_amount: number
  current_tier: UserTierState | null
  next_tier: UserTierState | null
  remaining_to_next: number | null
  tiers: UserTierState[]
  claimable_count: number
}

/** POST /user/tier/claim 中单条已发放权益 */
export interface UserTierAppliedBenefit {
  benefit_type: UserTierBenefitType
  amount?: number
  validity_days?: number
  expires_at?: string
  group_id?: number
  group_name?: string
  rate_multiplier?: number
  skipped_reason?: string
}

/** POST /user/tier/claim 响应体 */
export interface UserTierClaimResponse {
  tier_id: number
  code: string
  name: string
  already_claimed: boolean
  applied: UserTierAppliedBenefit[]
  /** 需要延迟生效的权益标识 */
  pending?: string[]
}

/** GET/PUT /admin/tiers/switch 响应体：等级与权益总开关 */
export interface UserTierFeatureSwitch {
  enabled: boolean
}

/** 管理端权益配置行 */
export interface AdminTierBenefit {
  id: number
  benefit_type: UserTierBenefitType
  amount: number
  validity_days: number
  group_id: number
  group_name?: string
  rate_multiplier: number
  enabled: boolean
  sort_order: number
}

/** 管理端档位配置 */
export interface AdminTier {
  id: number
  code: string
  name: string
  description: string
  sort_order: number
  trigger_type: UserTierTriggerType
  threshold_usd: number | null
  enabled: boolean
  /** 已产生的授予记录数；大于 0 时只能停用，不能删除、不能改 code */
  award_count: number
  benefits: AdminTierBenefit[]
  created_at: string
  updated_at: string
}

/** 管理端提交的权益配置（带 id 表示更新既有行） */
export interface AdminTierBenefitInput {
  id?: number
  benefit_type: UserTierBenefitType
  amount: number
  validity_days: number
  group_id: number
  rate_multiplier: number
  enabled: boolean
}

/** 创建 / 更新档位的请求体（更新时字段可选） */
export interface AdminTierSaveRequest {
  code?: string
  name?: string
  description?: string
  trigger_type?: UserTierTriggerType
  threshold_usd?: number | null
  enabled?: boolean
  benefits?: AdminTierBenefitInput[]
}

/** 管理端只读查看某用户时的档位状态 */
export interface AdminUserTierState {
  tier_id: number
  code: string
  name: string
  achieved: boolean
  claimed: boolean
  claimable: boolean
}

/** 权益发放副作用记录 */
export interface AdminUserTierEffect {
  id: number
  benefit_key: string
  benefit_type: string
  amount?: number
  group_id?: number
  rate_multiplier?: number
  status: string
  attempts: number
  last_error?: string
  detail?: Record<string, unknown>
  applied_at?: string
}

/** 授予记录 */
export interface AdminUserTierAward {
  id: number
  tier_code: string
  tier_name_snapshot: string
  threshold_snapshot?: number
  achieved_at: string
  source: string
  effects: AdminUserTierEffect[]
}

/** GET /admin/users/:id/tier 响应体（只读） */
export interface AdminUserTierResponse {
  user_id: number
  consumed_amount: number
  tiers: AdminUserTierState[]
  awards: AdminUserTierAward[]
  effects: AdminUserTierEffect[]
}

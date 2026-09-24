import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import MembershipView from '../MembershipView.vue'
import router from '@/router'
import { formatCurrency } from '@/utils/format'
import type { MyTierResponse, UserTierClaimResponse, UserTierState } from '@/types'

const { claimTier, getMyTier, showError, showSuccess } = vi.hoisted(() => ({
  claimTier: vi.fn(),
  getMyTier: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: { getMyTier, claimTier },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

// 只保留参与断言的文案模板，便于对渲染结果做可读断言（缺失的键回落到键名）。
const MESSAGES: Record<string, string> = {
  'membership.threshold': '门槛 {amount}',
  'membership.notAchievedRemaining': '还差 {amount}',
  'membership.nextTierRemaining': '还差 {amount}',
  'membership.claimSuccessBalance': '{amount} 已到账，有效期至 {expires}',
  'membership.claimSuccessRate': '{group} 倍率已生效：{rate}x',
  'membership.claimSkippedRate': '{group} 倍率未变更：{reason}',
  'membership.claimAllSuccess': '已领取 {count} 个档位',
  'membership.benefitNeverExpires': '永不过期',
  'membership.benefitAmount': '金额 {amount}',
  'membership.benefitValidityDays': '有效期 {days} 天',
  'membership.benefitExpiresAt': '有效期 {days} 天，至 {time}',
  'membership.benefitGroup': '分组 {group}',
  'membership.benefitRate': '倍率 {rate}x',
  'membership.claimedAt': '领取于 {time}',
  'membership.skippedReason': '已跳过：{reason}',
  'membership.skippedManualRateMultiplier': '因已有专属倍率而跳过',
  'membership.firstRechargeAuto': '首充自动发放',
  'membership.notAchieved': '未达标',
  'membership.claimed': '已领取',
  'membership.claim': '领取',
  'membership.summaryTitle': '当前等级',
  'membership.consumedAmount': '累计消费',
  'membership.allTiersTitle': '各档权益',
  'membership.featureDisabledTitle': '该功能未开放',
  'membership.featureDisabledDesc': '等级与权益功能当前未开放，如有疑问请联系客服。',
}

function translate(key: string, params?: Record<string, unknown>): string {
  const template = MESSAGES[key] ?? key
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_match, name: string) => String(params[name] ?? `{${name}}`))
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: translate }),
  }
})

const CONSUMED = 300

function tierState(overrides: Partial<UserTierState> = {}): UserTierState {
  return {
    tier_id: 1,
    code: 'tier_1',
    name: '青铜',
    trigger_type: 'consumption',
    threshold_usd: 100,
    achieved: true,
    claimed: true,
    claimable: false,
    claimed_at: '2026-09-20T10:00:00Z',
    benefits: [],
    ...overrides,
  }
}

function tierResponse(overrides: Partial<MyTierResponse> = {}): MyTierResponse {
  return {
    enabled: true,
    consumed_amount: CONSUMED,
    current_tier: tierState(),
    next_tier: tierState({ tier_id: 2, code: 'tier_2', name: '白银', threshold_usd: 500, achieved: false, claimed: false }),
    remaining_to_next: 200,
    tiers: [tierState()],
    claimable_count: 0,
    ...overrides,
  }
}

function claimResponse(overrides: Partial<UserTierClaimResponse> = {}): UserTierClaimResponse {
  return {
    tier_id: 2,
    code: 'tier_2',
    name: '白银',
    already_claimed: false,
    applied: [],
    ...overrides,
  }
}

function mountView() {
  return mount(MembershipView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
      },
    },
  })
}

describe('membership route', () => {
  it('is an authenticated user route without a subscription gate', () => {
    const route = router.getRoutes().find((item) => item.path === '/membership')

    expect(route).toBeDefined()
    expect(route!.name).toBe('Membership')
    expect(route!.meta).toEqual(
      expect.objectContaining({
        requiresAuth: true,
        requiresAdmin: false,
        titleKey: 'membership.title',
        descriptionKey: 'membership.description',
      }),
    )
    // 等级页对所有登录用户开放，不抄 SubscriptionsView 的订阅门禁
    expect(route!.meta).not.toHaveProperty('requiresSubscription')
  })
})

describe('MembershipView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getMyTier.mockResolvedValue(tierResponse())
    claimTier.mockResolvedValue(claimResponse())
  })

  it('renders claimable, claimed and not-achieved tiers with their own action state', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [
          tierState({ tier_id: 1, name: '青铜', claimed: true, claimable: false }),
          tierState({ tier_id: 2, name: '白银', threshold_usd: 500, achieved: false, claimed: false, claimable: false }),
          tierState({ tier_id: 3, name: '黄金', threshold_usd: 200, achieved: true, claimed: false, claimable: true }),
        ],
        claimable_count: 1,
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('已领取')
    expect(text).toContain('未达标')
    // 未达标档位显示差额 threshold_usd - consumed_amount = 500 - 300
    expect(text).toContain(`还差 ${formatCurrency(500 - CONSUMED)}`)
    expect(text).not.toContain(`还差 ${formatCurrency(500)}`)

    const claimButtons = wrapper.findAll('button').filter((button) => button.text() === '领取')
    expect(claimButtons).toHaveLength(1)
  })

  it('hides the claim button for first-recharge tiers and shows the auto-grant copy', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [
          tierState({
            tier_id: 9,
            code: 'first_recharge',
            name: '首充礼',
            trigger_type: 'first_recharge',
            threshold_usd: null,
            achieved: true,
            claimed: false,
            claimable: true,
          }),
        ],
        claimable_count: 0,
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('首充自动发放')
    expect(wrapper.findAll('button').filter((button) => button.text() === '领取')).toHaveLength(0)
    // claimable 为 true 但首充档自动发放，因此不应出现一键领取入口
    expect(wrapper.findAll('button').some((button) => button.text().includes('claimAll'))).toBe(false)
  })

  it('explains a historically granted tier through the configured description, with no built-in note', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [
          tierState({
            tier_id: 4,
            name: '历史档',
            description: '2026-09-24 活动期授予，非现行消费口径达标',
            threshold_usd: 900,
            achieved: false,
            claimed: true,
            claimable: false,
            source: 'manual_20260924',
          }),
        ],
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    // 说明文案由档位备注承载（管理端可改），页面不再内置任何授予来源文案
    expect(text).toContain('2026-09-24 活动期授予，非现行消费口径达标')
    expect(text).toContain('历史档')
  })

  it('explains a rate benefit skipped because the group already has a dedicated multiplier', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [
          tierState({
            tier_id: 5,
            achieved: true,
            claimed: true,
            benefits: [
              {
                benefit_type: 'group_rate',
                group_id: 7,
                group_name: 'SVIP',
                rate_multiplier: 0.5,
                status: 'pending',
                skipped_reason: 'manual_rate_multiplier_present',
              },
            ],
          }),
        ],
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('SVIP')
    expect(text).toContain('倍率 0.5x')
    expect(text).toContain('已跳过：因已有专属倍率而跳过')
  })

  // 后端把「因已有手工倍率而跳过」的权益也放进 applied，若一律提示「倍率已生效」，
  // 就是对用户谎报到账（实际没有写任何覆盖层）。
  it('does not report a skipped rate benefit as applied in the claim toast', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [tierState({ tier_id: 5, achieved: true, claimed: false, claimable: true })],
        claimable_count: 1,
      }),
    )
    claimTier.mockResolvedValueOnce(
      claimResponse({
        tier_id: 5,
        applied: [
          {
            benefit_type: 'group_rate',
            group_id: 7,
            group_name: 'SVIP',
            rate_multiplier: 0.5,
            skipped_reason: 'manual_rate_multiplier_present',
          },
        ],
      }),
    )

    const wrapper = mountView()
    await flushPromises()
    await wrapper
      .findAll('button')
      .find((node) => node.text().includes('领取'))
      ?.trigger('click')
    await flushPromises()

    expect(showSuccess).toHaveBeenCalledTimes(1)
    const message = showSuccess.mock.calls[0][0] as string
    expect(message).toContain('SVIP 倍率未变更')
    expect(message).toContain('因已有专属倍率而跳过')
    expect(message).not.toContain('已生效')
  })

  it('still reports a granted rate benefit as applied', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [tierState({ tier_id: 5, achieved: true, claimed: false, claimable: true })],
        claimable_count: 1,
      }),
    )
    claimTier.mockResolvedValueOnce(
      claimResponse({
        tier_id: 5,
        applied: [
          { benefit_type: 'group_rate', group_id: 7, group_name: 'SVIP', rate_multiplier: 0.5 },
        ],
      }),
    )

    const wrapper = mountView()
    await flushPromises()
    await wrapper
      .findAll('button')
      .find((node) => node.text().includes('领取'))
      ?.trigger('click')
    await flushPromises()

    const message = showSuccess.mock.calls[0][0] as string
    expect(message).toContain('SVIP 倍率已生效')
    expect(message).toContain('0.5x')
  })

  it('claims every unclaimed tier serially and reports the credited amount with its expiry', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [
          tierState({ tier_id: 1, name: '青铜', claimed: false, claimable: true }),
          tierState({ tier_id: 2, name: '白银', threshold_usd: 500, achieved: false, claimable: false }),
          tierState({ tier_id: 3, name: '黄金', threshold_usd: 200, claimed: false, claimable: true }),
        ],
        claimable_count: 2,
      }),
    )
    claimTier
      .mockResolvedValueOnce(
        claimResponse({
          tier_id: 1,
          applied: [
            {
              benefit_type: 'balance_credit',
              amount: 10,
              validity_days: 3,
              expires_at: '2026-10-01T00:00:00Z',
            },
          ],
        }),
      )
      .mockResolvedValueOnce(
        claimResponse({
          tier_id: 3,
          applied: [
            {
              benefit_type: 'balance_credit',
              amount: 20,
              validity_days: 7,
              expires_at: '2026-10-05T00:00:00Z',
            },
          ],
        }),
      )

    const wrapper = mountView()
    await flushPromises()

    const claimAllButton = wrapper.findAll('button').find((button) => button.text().includes('claimAll'))
    expect(claimAllButton).toBeDefined()
    await claimAllButton!.trigger('click')
    await flushPromises()

    // 逐档串行：只对 claimable 的两个档位各调用一次
    expect(claimTier).toHaveBeenCalledTimes(2)
    expect(claimTier).toHaveBeenNthCalledWith(1, 1)
    expect(claimTier).toHaveBeenNthCalledWith(2, 3)

    expect(showSuccess).toHaveBeenCalledTimes(1)
    const message = showSuccess.mock.calls[0][0] as string
    expect(message).toContain('已到账，有效期至')
    expect(message).toContain(formatCurrency(10))
    expect(message).toContain(formatCurrency(20))

    // 领取后刷新视图
    expect(getMyTier).toHaveBeenCalledTimes(2)
  })

  it('renders only the not-available state when the backend reports enabled=false', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        enabled: false,
        current_tier: null,
        next_tier: null,
        remaining_to_next: null,
        tiers: [],
        claimable_count: 0,
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('该功能未开放')
    expect(text).toContain('等级与权益功能当前未开放，如有疑问请联系客服。')

    // 等级内容整体不渲染：概览、累计消费、各档权益与领取入口都不出现
    expect(text).not.toContain('当前等级')
    expect(text).not.toContain('累计消费')
    expect(text).not.toContain('各档权益')
    expect(text).not.toContain('青铜')
    expect(text).not.toContain('尚未达成任何等级')
    expect(wrapper.findAll('button').filter((button) => button.text() === '领取')).toHaveLength(0)
    // 关闭不是错误，不弹错误提示
    expect(showError).not.toHaveBeenCalled()
  })

  it('falls into the not-available state when the tier endpoint replies 403 USER_TIER_FEATURE_DISABLED', async () => {
    getMyTier.mockRejectedValue({ status: 403, reason: 'USER_TIER_FEATURE_DISABLED' })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('该功能未开放')
    expect(text).not.toContain('各档权益')
    expect(showError).not.toHaveBeenCalled()
  })

  it('falls into the not-available state when a claim is rejected with USER_TIER_FEATURE_DISABLED', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [tierState({ tier_id: 3, name: '黄金', threshold_usd: 200, achieved: true, claimed: false, claimable: true })],
        claimable_count: 1,
      }),
    )
    claimTier.mockRejectedValueOnce({ status: 403, reason: 'USER_TIER_FEATURE_DISABLED' })

    const wrapper = mountView()
    await flushPromises()

    const claimButton = wrapper.findAll('button').find((button) => button.text() === '领取')!
    await claimButton.trigger('click')
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('该功能未开放')
    expect(text).not.toContain('黄金')
    expect(wrapper.findAll('button').filter((button) => button.text() === '领取')).toHaveLength(0)
    expect(showSuccess).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
  })

  it('keeps the tier content when enabled is missing from the payload', async () => {
    const response = tierResponse()
    delete (response as Partial<MyTierResponse>).enabled
    getMyTier.mockResolvedValue(response)

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('当前等级')
    expect(wrapper.text()).not.toContain('该功能未开放')
  })

  it('surfaces the claim error and keeps the view loaded when the serial claim fails', async () => {
    getMyTier.mockResolvedValue(
      tierResponse({
        tiers: [tierState({ tier_id: 1, claimed: false, claimable: true })],
        claimable_count: 1,
      }),
    )
    claimTier.mockRejectedValueOnce({ reason: 'USER_TIER_NOT_ACHIEVED', message: 'not achieved' })

    const wrapper = mountView()
    await flushPromises()

    const claimAllButton = wrapper.findAll('button').find((button) => button.text().includes('claimAll'))
    await claimAllButton!.trigger('click')
    await flushPromises()

    expect(showSuccess).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledTimes(1)
    // 失败时不再刷新视图
    expect(getMyTier).toHaveBeenCalledTimes(1)
  })
})

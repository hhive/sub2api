import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueDraggable } from 'vue-draggable-plus'

import TierConfigView from '../TierConfigView.vue'
import Select from '@/components/common/Select.vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import type { AdminTier } from '@/types'

const {
  createTier,
  deleteTier,
  getFeatureSwitch,
  getGroups,
  getUserTier,
  listTiers,
  reorderTiers,
  showError,
  showSuccess,
  updateFeatureSwitch,
  updateTier,
} = vi.hoisted(() => ({
  createTier: vi.fn(),
  deleteTier: vi.fn(),
  getFeatureSwitch: vi.fn(),
  getGroups: vi.fn(),
  getUserTier: vi.fn(),
  listTiers: vi.fn(),
  reorderTiers: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  updateFeatureSwitch: vi.fn(),
  updateTier: vi.fn(),
}))

vi.mock('@/api/admin/userTiers', () => ({
  default: {
    listTiers,
    createTier,
    updateTier,
    reorderTiers,
    deleteTier,
    getUserTier,
    getFeatureSwitch,
    updateFeatureSwitch,
  },
}))

vi.mock('@/api/admin', () => ({
  default: { groups: { getAll: getGroups } },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

// 只保留参与断言的文案模板，缺失的键回落到键名。
const MESSAGES: Record<string, string> = {
  'admin.tierConfig.nameRequired': '请填写等级称呼',
  'admin.tierConfig.codeRequired': '请填写等级标识',
  'admin.tierConfig.thresholdRequired': '按消费达成的等级必须填写消费门槛',
  'admin.tierConfig.amountInvalid': '权益金额必须大于 0',
  'admin.tierConfig.rateMultiplierInvalid': '倍率必须大于 0',
  'admin.tierConfig.groupRequired': '请选择倍率权益的分组',
  'admin.tierConfig.orderSaved': '排序已保存',
  'admin.tierConfig.createSuccess': '等级已创建',
  'admin.tierConfig.updateSuccess': '等级已更新',
  'admin.tierConfig.deleteSuccess': '等级已删除',
  'admin.tierConfig.USER_TIER_HAS_AWARDS': '有授予记录的等级只能停用',
  'admin.tierConfig.switchEnabled': '已开启',
  'admin.tierConfig.switchDisabled': '已关闭',
  'admin.tierConfig.switchDisableConfirm': '关闭后对用户立即生效',
  'admin.tierConfig.switchEnableSuccess': '等级与权益已开启',
  'admin.tierConfig.switchDisableSuccess': '等级与权益已关闭',
  'admin.tierConfig.switchLoadFailed': '加载等级与权益总开关失败',
  'admin.tierConfig.switchSaveFailed': '保存等级与权益总开关失败',
  // 说明文案内容另由下方的多语言用例校验真实词条，这里只验证卡片接线
  'admin.tierConfig.switchHint': '关闭后用户端不再显示等级入口与等级页面',
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

function tier(overrides: Partial<AdminTier> = {}): AdminTier {
  return {
    id: 1,
    code: 'tier_500',
    name: '青铜',
    description: '累计消费达标',
    sort_order: 0,
    trigger_type: 'consumption',
    threshold_usd: 500,
    enabled: true,
    award_count: 0,
    benefits: [],
    created_at: '2026-09-24T00:00:00Z',
    updated_at: '2026-09-24T00:00:00Z',
    ...overrides,
  }
}

function mountView() {
  return mount(TierConfigView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: true,
        // BaseDialog 使用 Teleport，这里用直通实现让弹窗内容留在 wrapper 内便于断言
        BaseDialog: {
          name: 'BaseDialog',
          props: ['show', 'title', 'width'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        ConfirmDialog: {
          name: 'ConfirmDialog',
          props: ['show', 'title', 'message', 'confirmText', 'danger'],
          template:
            '<div v-if="show"><button class="confirm-ok" @click="$emit(\'confirm\')">ok</button><button class="confirm-cancel" @click="$emit(\'cancel\')">cancel</button></div>',
        },
      },
    },
  })
}

function findInputByLabel(wrapper: ReturnType<typeof mountView>, labelText: string): DOMWrapper<HTMLInputElement> {
  // 必填标签后面跟着一个 "*" 星号 span，因此用前缀匹配而不是全等。
  const label = wrapper.findAll('label').find((node) => node.text().trim().startsWith(labelText))
  expect(label, `label not found: ${labelText}`).toBeDefined()

  let container = label?.element.parentElement
  while (container && container !== wrapper.element) {
    const input = container.querySelector('input')
    if (input instanceof HTMLInputElement) return new DOMWrapper(input)
    container = container.parentElement
  }
  throw new Error(`Input not found for label: ${labelText}`)
}

function rowActionButton(wrapper: ReturnType<typeof mountView>, name: string, iconIndex = 0) {
  return wrapper.findAll('tbody button').filter((button) => button.text() === name)[iconIndex]
}

describe('TierConfigView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getGroups.mockResolvedValue([{ id: 7, name: 'SVIP', platform: 'openai', rate_multiplier: 0.5 }])
    listTiers.mockResolvedValue([
      tier({ id: 1, code: 'tier_500', name: '青铜', sort_order: 0 }),
      tier({ id: 2, code: 'tier_1000', name: '白银', sort_order: 1 }),
      tier({ id: 3, code: 'tier_1500', name: '黄金', sort_order: 2 }),
    ])
    reorderTiers.mockResolvedValue({ message: 'ok' })
    updateTier.mockResolvedValue(tier())
    createTier.mockResolvedValue(tier())
    deleteTier.mockResolvedValue({ message: 'ok' })
    getFeatureSwitch.mockResolvedValue({ enabled: true })
    updateFeatureSwitch.mockResolvedValue({ enabled: false })
  })

  it('renders the tier list with its benefit summary', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        code: 'tier_500',
        name: '青铜',
        benefits: [
          {
            id: 11,
            benefit_type: 'balance_credit',
            amount: 10,
            validity_days: 3,
            group_id: 0,
            rate_multiplier: 0,
            enabled: true,
            sort_order: 0,
          },
        ],
      }),
      tier({ id: 2, code: 'tier_1000', name: '白银' }),
    ])

    const wrapper = mountView()
    await flushPromises()

    expect(listTiers).toHaveBeenCalledTimes(1)
    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)
    expect(wrapper.text()).toContain('青铜')
    expect(wrapper.text()).toContain('白银')
    // 额度权益展示金额与有效期
    expect(wrapper.text()).toContain('$10.00 / 3')
  })

  it('toggles enabled through updateTier with only the enabled field', async () => {
    const wrapper = mountView()
    await flushPromises()

    const toggles = wrapper.findAll('tbody button').filter((button) => button.classes().includes('rounded-full'))
    expect(toggles).toHaveLength(3)
    await toggles[1].trigger('click')
    await flushPromises()

    expect(updateTier).toHaveBeenCalledTimes(1)
    expect(updateTier).toHaveBeenCalledWith(2, { enabled: false })
  })

  it('submits the reordered ids after a drag', async () => {
    const wrapper = mountView()
    await flushPromises()

    const list = wrapper.findAllComponents(VueDraggable)[0]
    const items = list.props('modelValue') as AdminTier[]
    expect(items.map((item) => item.id)).toEqual([1, 2, 3])

    list.vm.$emit('update:modelValue', [items[2], items[0], items[1]])
    list.vm.$emit('end')
    await flushPromises()

    expect(reorderTiers).toHaveBeenCalledTimes(1)
    expect(reorderTiers).toHaveBeenCalledWith([3, 1, 2])
    expect(showSuccess).toHaveBeenCalledWith('排序已保存')
  })

  it('creates a tier with the full payload including the benefits array', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.createTier')!.trigger('click')
    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.addBenefit')!.trigger('click')
    await flushPromises()

    await findInputByLabel(wrapper, 'admin.tierConfig.name').setValue('铂金')
    await findInputByLabel(wrapper, 'admin.tierConfig.code').setValue('tier_2000')
    await findInputByLabel(wrapper, 'admin.tierConfig.thresholdUsd').setValue('2000')
    await findInputByLabel(wrapper, 'admin.tierConfig.amount').setValue('25')
    await findInputByLabel(wrapper, 'admin.tierConfig.validityDays').setValue('7')

    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(createTier).toHaveBeenCalledTimes(1)
    expect(createTier).toHaveBeenCalledWith({
      name: '铂金',
      code: 'tier_2000',
      description: '',
      trigger_type: 'consumption',
      threshold_usd: 2000,
      enabled: true,
      benefits: [
        {
          id: undefined,
          benefit_type: 'balance_credit',
          amount: 25,
          validity_days: 7,
          group_id: 0,
          rate_multiplier: 0,
          enabled: true,
        },
      ],
    })
    expect(showSuccess).toHaveBeenCalledWith('等级已创建')
  })

  it('updates an existing tier and keeps its existing benefit ids', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        code: 'tier_500',
        name: '青铜',
        benefits: [
          {
            id: 11,
            benefit_type: 'group_rate',
            amount: 0,
            validity_days: 0,
            group_id: 7,
            rate_multiplier: 0.5,
            enabled: true,
            sort_order: 0,
          },
        ],
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()

    await findInputByLabel(wrapper, 'admin.tierConfig.name').setValue('青铜二档')
    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).toHaveBeenCalledTimes(1)
    expect(updateTier).toHaveBeenCalledWith(1, {
      name: '青铜二档',
      code: 'tier_500',
      description: '累计消费达标',
      trigger_type: 'consumption',
      threshold_usd: 500,
      enabled: true,
      benefits: [
        {
          id: 11,
          benefit_type: 'group_rate',
          amount: 0,
          validity_days: 0,
          group_id: 7,
          rate_multiplier: 0.5,
          enabled: true,
        },
      ],
    })
    expect(showSuccess).toHaveBeenCalledWith('等级已更新')
  })

  it('rejects a blank name before calling the API', async () => {
    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()
    await findInputByLabel(wrapper, 'admin.tierConfig.name').setValue('   ')
    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('请填写等级称呼')
  })

  it('rejects a consumption tier without a positive threshold', async () => {
    listTiers.mockResolvedValue([tier({ id: 1, threshold_usd: 0 })])

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()
    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('按消费达成的等级必须填写消费门槛')
  })

  it('rejects a balance credit benefit with a non-positive amount', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        benefits: [
          {
            id: 11,
            benefit_type: 'balance_credit',
            amount: 0,
            validity_days: 3,
            group_id: 0,
            rate_multiplier: 0,
            enabled: true,
            sort_order: 0,
          },
        ],
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()
    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('权益金额必须大于 0')
  })

  it('rejects a group rate benefit with a non-positive multiplier', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        benefits: [
          {
            id: 11,
            benefit_type: 'group_rate',
            amount: 0,
            validity_days: 0,
            group_id: 7,
            rate_multiplier: 0,
            enabled: true,
            sort_order: 0,
          },
        ],
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()
    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('倍率必须大于 0')
  })

  it('disables delete for tiers that already produced awards', async () => {
    listTiers.mockResolvedValue([
      tier({ id: 1, code: 'tier_500', name: '青铜', award_count: 3 }),
      tier({ id: 2, code: 'tier_1000', name: '白银', award_count: 0 }),
    ])

    const wrapper = mountView()
    await flushPromises()

    const deleteButtons = wrapper.findAll('tbody button').filter((button) => button.text() === 'common.delete')
    expect(deleteButtons).toHaveLength(2)
    expect(deleteButtons[0].attributes('disabled')).toBeDefined()
    expect(deleteButtons[1].attributes('disabled')).toBeUndefined()
  })

  it('surfaces the backend conflict message when a locked tier delete is rejected with 409', async () => {
    deleteTier.mockRejectedValueOnce({ status: 409, reason: 'USER_TIER_HAS_AWARDS' })

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.delete')!.trigger('click')
    await flushPromises()
    await wrapper.find('.confirm-ok').trigger('click')
    await flushPromises()

    expect(deleteTier).toHaveBeenCalledWith(1)
    expect(showError).toHaveBeenCalledWith('有授予记录的等级只能停用')
  })

  it('locks the tier code once awards exist and omits it from the update payload', async () => {
    listTiers.mockResolvedValue([tier({ id: 1, code: 'tier_500', award_count: 2 })])

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.edit')!.trigger('click')
    await flushPromises()

    expect(findInputByLabel(wrapper, 'admin.tierConfig.code').element.disabled).toBe(true)

    await wrapper.find('form#tier-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateTier).toHaveBeenCalledTimes(1)
    expect(updateTier.mock.calls[0][1]).not.toHaveProperty('code')
  })

  it('looks up a user tier read-only from the id input', async () => {
    getUserTier.mockResolvedValue({
      user_id: 42,
      consumed_amount: 640,
      tiers: [
        { tier_id: 1, code: 'tier_500', name: '青铜', achieved: true, claimed: true, claimable: false },
        { tier_id: 2, code: 'tier_1000', name: '白银', achieved: false, claimed: false, claimable: false },
      ],
      awards: [],
      effects: [],
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('input[type="number"]').setValue('42')
    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.lookup')!.trigger('click')
    await flushPromises()

    expect(getUserTier).toHaveBeenCalledWith(42)
    const text = wrapper.text()
    expect(text).toContain('$640.00')
    expect(text).toContain('青铜')
    expect(text).toContain('admin.tierConfig.lookupAchieved')
  })

  it('warns instead of calling the API when the lookup id is empty', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.lookup')!.trigger('click')
    await flushPromises()

    expect(getUserTier).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.tierConfig.lookupEmpty')
  })

  it('keeps the trigger type select wired to the form', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.createTier')!.trigger('click')
    await flushPromises()

    const selects = wrapper.findAllComponents(Select)
    expect(selects.length).toBeGreaterThan(0)
    expect(selects[0].props('options')).toEqual([
      { value: 'consumption', label: 'admin.tierConfig.triggerConsumption' },
      { value: 'first_recharge', label: 'admin.tierConfig.triggerFirstRecharge' },
    ])
  })

  describe('feature master switch', () => {
    function switchButton(wrapper: ReturnType<typeof mountView>) {
      return wrapper.find('[role="switch"]')
    }

    it('loads the current switch state on mount', async () => {
      getFeatureSwitch.mockResolvedValue({ enabled: false })

      const wrapper = mountView()
      await flushPromises()

      expect(getFeatureSwitch).toHaveBeenCalledTimes(1)
      expect(switchButton(wrapper).attributes('aria-checked')).toBe('false')
      expect(wrapper.text()).toContain('已关闭')
    })

    it('reflects an enabled switch and renders the explanatory copy', async () => {
      const wrapper = mountView()
      await flushPromises()

      expect(switchButton(wrapper).attributes('aria-checked')).toBe('true')
      expect(wrapper.text()).toContain('已开启')
      expect(wrapper.text()).toContain('关闭后用户端不再显示等级入口与等级页面')
    })

    // 文案是唯一向管理员解释关闭后果的地方，因此直接校验真实词条而不经过 mock
    it('documents all four consequences of switching off in Chinese', () => {
      const hint = zh.admin.tierConfig.switchHint
      expect(hint).toContain('用户端不再显示等级入口')
      expect(hint).toContain('不再发放新的等级权益')
      expect(hint).toContain('已发放的权益保留，不回收')
      expect(hint).toContain('首充赠送由等级配置里的首充档决定')
      expect(hint).toContain('回到迁移前的系统设置取值')
    })

    it('documents all four consequences of switching off in English', () => {
      const hint = en.admin.tierConfig.switchHint
      expect(hint).toContain('no longer see the tier entry')
      expect(hint).toContain('no new tier benefits are granted')
      expect(hint).toContain('already granted are kept and never clawed back')
      expect(hint).toContain('First-recharge gifting is driven by the first-recharge tier')
      expect(hint).toContain('pre-migration system-settings value')
    })

    it('asks for confirmation when switching off and does not submit on cancel', async () => {
      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()

      // 关闭前先确认，此时不应发起请求
      expect(updateFeatureSwitch).not.toHaveBeenCalled()
      expect(wrapper.find('.confirm-ok').exists()).toBe(true)

      await wrapper.find('.confirm-cancel').trigger('click')
      await flushPromises()

      expect(updateFeatureSwitch).not.toHaveBeenCalled()
      expect(switchButton(wrapper).attributes('aria-checked')).toBe('true')
    })

    it('submits enabled=false once the operator confirms the switch-off dialog', async () => {
      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()
      await wrapper.find('.confirm-ok').trigger('click')
      await flushPromises()

      expect(updateFeatureSwitch).toHaveBeenCalledTimes(1)
      expect(updateFeatureSwitch).toHaveBeenCalledWith(false)
      expect(switchButton(wrapper).attributes('aria-checked')).toBe('false')
      expect(showSuccess).toHaveBeenCalledWith('等级与权益已关闭')
    })

    it('enables the feature without a confirmation dialog', async () => {
      getFeatureSwitch.mockResolvedValue({ enabled: false })
      updateFeatureSwitch.mockResolvedValue({ enabled: true })

      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()

      expect(updateFeatureSwitch).toHaveBeenCalledTimes(1)
      expect(updateFeatureSwitch).toHaveBeenCalledWith(true)
      expect(switchButton(wrapper).attributes('aria-checked')).toBe('true')
      expect(showSuccess).toHaveBeenCalledWith('等级与权益已开启')
    })

    it('surfaces the save error and keeps the previous switch state', async () => {
      updateFeatureSwitch.mockRejectedValueOnce({ status: 500, reason: 'USER_TIER_SWITCH_SAVE_FAILED' })

      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()
      await wrapper.find('.confirm-ok').trigger('click')
      await flushPromises()

      expect(showError).toHaveBeenCalledWith('保存等级与权益总开关失败')
      expect(showSuccess).not.toHaveBeenCalled()
      // 后端未确认成功，开关不翻转
      expect(switchButton(wrapper).attributes('aria-checked')).toBe('true')
    })

    it('surfaces a load failure for the switch without breaking the tier list', async () => {
      getFeatureSwitch.mockRejectedValueOnce({ status: 500, reason: 'USER_TIER_SWITCH_LOAD_FAILED' })

      const wrapper = mountView()
      await flushPromises()

      expect(showError).toHaveBeenCalledWith('加载等级与权益总开关失败')
      expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    })
  })
})

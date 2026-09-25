import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { VueDraggable } from 'vue-draggable-plus'

import TierConfigView from '../TierConfigView.vue'
import Select from '@/components/common/Select.vue'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import type { AdminTier, AdminTierAssignment } from '@/types'

const {
  createTier,
  deleteTier,
  fetchPublicSettings,
  getFeatureSwitch,
  getGroups,
  getTierAssignment,
  getUserTier,
  listTiers,
  removeTierAssignment,
  reorderTiers,
  saveTierAssignment,
  showError,
  showSuccess,
  updateFeatureSwitch,
  updateTier,
} = vi.hoisted(() => ({
  createTier: vi.fn(),
  deleteTier: vi.fn(),
  fetchPublicSettings: vi.fn(),
  getFeatureSwitch: vi.fn(),
  getGroups: vi.fn(),
  getTierAssignment: vi.fn(),
  getUserTier: vi.fn(),
  listTiers: vi.fn(),
  removeTierAssignment: vi.fn(),
  reorderTiers: vi.fn(),
  saveTierAssignment: vi.fn(),
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
    getTierAssignment,
    saveTierAssignment,
    removeTierAssignment,
  },
}))

vi.mock('@/api/admin', () => ({
  default: { groups: { getAll: getGroups } },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, fetchPublicSettings }),
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
  // 按邮箱配置等级
  'admin.tierConfig.assignEmpty': '该用户暂无等级指派',
  'admin.tierConfig.assignSourceAdmin': '管理端配置',
  'admin.tierConfig.assignTierOptionLabel': '{name}（{code}）',
  'admin.tierConfig.assignRemove': '取消配置',
  'admin.tierConfig.assignEmailRequired': '请先输入用户邮箱',
  'admin.tierConfig.assignTierRequired': '请先选择要指派的档位',
  'admin.tierConfig.assignSaveSuccess': '等级指派已保存',
  'admin.tierConfig.assignRemoveSuccess': '已取消等级指派',
  'admin.tierConfig.USER_TIER_ASSIGN_USER_NOT_FOUND': '该邮箱未匹配到用户',
  'admin.tierConfig.USER_TIER_ASSIGNMENT_TARGET_DISABLED': '该等级已停用',
  'admin.tierConfig.codeHint': '等级标识可为任意字符，最长 64 个字符',
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

  // 双击时两次调用会读到同一个旧值，各自翻转本地状态后会与服务端相反。
  it('sends only one toggle request when clicked twice and disables the switch meanwhile', async () => {
    let resolveUpdate: ((value: unknown) => void) | undefined
    updateTier.mockImplementationOnce(
      () => new Promise((resolve) => { resolveUpdate = resolve }),
    )

    const wrapper = mountView()
    await flushPromises()

    const toggles = wrapper.findAll('tbody button').filter((button) => button.classes().includes('rounded-full'))
    await toggles[1].trigger('click')
    await toggles[1].trigger('click')
    expect(toggles[1].attributes('disabled')).toBeDefined()

    resolveUpdate?.(tier())
    await flushPromises()

    expect(updateTier).toHaveBeenCalledTimes(1)
  })

  // 后端对停用权益显式跳过校验与发放，前端若仍强制校验，合理配置会被挡住。
  it('saves a tier holding a disabled benefit with empty values', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        benefits: [
          {
            id: 11,
            benefit_type: 'group_rate',
            amount: 0,
            validity_days: 0,
            group_id: 0,
            rate_multiplier: 0,
            enabled: false,
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

    expect(showError).not.toHaveBeenCalled()
    expect(updateTier).toHaveBeenCalledTimes(1)
    expect(updateTier.mock.calls[0][1].benefits).toEqual([
      {
        id: 11,
        benefit_type: 'group_rate',
        amount: 0,
        validity_days: 0,
        group_id: 0,
        rate_multiplier: 0,
        enabled: false,
      },
    ])
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

  // 上面的断言依赖 mock 词条，这一条校验真实词条确实存在：否则 extractI18nErrorMessage
  // 查不到 `命名空间.错误码` 会回落到后端中文，英文语系管理员会看到中文。
  it('maps backend tier error codes to real localized messages', () => {
    expect(zh.admin.tierConfig.USER_TIER_HAS_AWARDS).toBe('该等级已产生授予记录，只能停用')
    expect(en.admin.tierConfig.USER_TIER_HAS_AWARDS).toBeTruthy()
    expect(zh.admin.tierConfig.USER_TIER_FIRST_RECHARGE_CODE_LOCKED).toContain('first_recharge')
    expect(en.admin.tierConfig.USER_TIER_FIRST_RECHARGE_CODE_LOCKED).toContain('first_recharge')
  })

  it('sends only one delete when the confirm button is clicked twice', async () => {
    let resolveDelete: ((value: unknown) => void) | undefined
    deleteTier.mockImplementationOnce(
      () => new Promise((resolve) => { resolveDelete = resolve }),
    )

    const wrapper = mountView()
    await flushPromises()

    await rowActionButton(wrapper, 'common.delete')!.trigger('click')
    await flushPromises()
    await wrapper.find('.confirm-ok').trigger('click')
    await wrapper.find('.confirm-ok').trigger('click')
    resolveDelete?.({ message: 'ok' })
    await flushPromises()

    expect(deleteTier).toHaveBeenCalledTimes(1)
  })

  it('marks disabled benefits in the list so they do not read as active', async () => {
    listTiers.mockResolvedValue([
      tier({
        id: 1,
        benefits: [
          { id: 11, benefit_type: 'balance_credit', amount: 10, validity_days: 3, group_id: 0, rate_multiplier: 0, enabled: false },
          { id: 12, benefit_type: 'balance_credit', amount: 20, validity_days: 3, group_id: 0, rate_multiplier: 0, enabled: true },
        ],
      }),
    ])

    const wrapper = mountView()
    await flushPromises()

    const rows = wrapper.findAll('tbody li')
    expect(rows[0].text()).toContain('admin.tierConfig.benefitDisabled')
    expect(rows[1].text()).not.toContain('admin.tierConfig.benefitDisabled')
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
      assignment: {
        user_id: 42,
        tier_id: 2,
        tier_code: 'tier_1000',
        tier_name_snapshot: '白银',
        sort_order_snapshot: 1,
        source: 'historical_20260924',
        note: '历史累计（订阅+兑换）一次性核算 2026-09-24',
        assigned_by: null,
        assigned_at: '2026-09-24T10:00:00Z',
        updated_at: '2026-09-24T10:00:00Z',
      },
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
    // 指派来自历史一次性批量：来源要显示为「历史批量」而不是「管理端配置」
    expect(text).toContain('admin.tierConfig.assignCurrent')
    expect(text).toContain('admin.tierConfig.assignSourceHistorical20260924')
    expect(text).toContain('tier_1000')
  })

  it('shows the empty assignment state in the read-only lookup when there is none', async () => {
    getUserTier.mockResolvedValue({
      user_id: 43,
      consumed_amount: 0,
      assignment: null,
      tiers: [],
      awards: [],
      effects: [],
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('input[type="number"]').setValue('43')
    await wrapper.findAll('button').find((button) => button.text() === 'admin.tierConfig.lookup')!.trigger('click')
    await flushPromises()

    // 该键在测试 i18n 桩里有真实文案（其余键渲染为键名）
    expect(wrapper.text()).toContain('该用户暂无等级指派')
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
    it('documents the consequences of switching off in Chinese', () => {
      const hint = zh.admin.tierConfig.switchHint
      expect(hint).toContain('用户端不再显示等级入口')
      expect(hint).toContain('不再发放新的等级权益')
      expect(hint).toContain('已发放的权益保留，不回收')
    })

    it('documents the consequences of switching off in English', () => {
      const hint = en.admin.tierConfig.switchHint
      expect(hint).toContain('no longer see the tier entry')
      expect(hint).toContain('no new tier benefits are granted')
      expect(hint).toContain('already granted are kept and never clawed back')
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

    // 不刷新缓存的公开设置，侧边栏「等级权益」入口会按旧值继续渲染：
    // 操作者自己关掉开关后入口不消失，「关闭后用户看不到」这条无法被观察到。
    it('refreshes the cached public settings after a successful switch write', async () => {
      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()
      await wrapper.find('.confirm-ok').trigger('click')
      await flushPromises()

      expect(updateFeatureSwitch).toHaveBeenCalledWith(false)
      expect(fetchPublicSettings).toHaveBeenCalledWith(true)
    })

    it('does not refresh public settings when the switch write fails', async () => {
      updateFeatureSwitch.mockRejectedValueOnce({ status: 500, reason: 'USER_TIER_SWITCH_SAVE_FAILED' })

      const wrapper = mountView()
      await flushPromises()

      await switchButton(wrapper).trigger('click')
      await flushPromises()
      await wrapper.find('.confirm-ok').trigger('click')
      await flushPromises()

      expect(fetchPublicSettings).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('保存等级与权益总开关失败')
    })
  })

  describe('assign a tier by email', () => {
    const ASSIGNED_USER = { id: 42, email: 'alice@example.com', username: 'alice' }

    function assignment(overrides: Partial<AdminTierAssignment> = {}): AdminTierAssignment {
      return {
        user_id: 42,
        tier_id: 2,
        tier_code: 'tier_1000',
        tier_name_snapshot: '白银',
        sort_order_snapshot: 1,
        source: 'admin',
        note: '运营手工配置',
        assigned_by: 1,
        assigned_at: '2026-09-24T10:00:00Z',
        updated_at: '2026-09-24T10:00:00Z',
        ...overrides,
      }
    }

    async function lookup(wrapper: ReturnType<typeof mountView>, email: string) {
      await wrapper.find('#tier-assign-email').setValue(email)
      await wrapper.find('#tier-assign-lookup').trigger('click')
      await flushPromises()
    }

    async function saveButton(wrapper: ReturnType<typeof mountView>) {
      return wrapper.findAll('button').find((button) => button.text() === 'common.save')
    }

    it('shows the resolved user and the current assignment after a lookup', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: assignment() })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      expect(getTierAssignment).toHaveBeenCalledWith('alice@example.com')
      const text = wrapper.text()
      expect(text).toContain('alice')
      expect(text).toContain('alice@example.com')
      expect(text).toContain('42')
      expect(text).toContain('白银')
      expect(text).toContain('tier_1000')
      expect(text).toContain('运营手工配置')
      // 来源显示为可读文案，而不是后端枚举值
      expect(text).toContain('管理端配置')
      expect(text).not.toContain('historical_20260924')
    })

    it('prefills the target tier and note from the current assignment', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: assignment() })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      expect((wrapper.find('#tier-assign-tier').element as HTMLSelectElement).value).toBe('tier_1000')
      expect((wrapper.find('#tier-assign-note').element as HTMLInputElement).value).toBe('运营手工配置')
    })

    it('shows the empty state and hides removal when the user has no assignment', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      expect(wrapper.text()).toContain('该用户暂无等级指派')
      expect(wrapper.findAll('button').some((button) => button.text() === '取消配置')).toBe(false)
    })

    // 首充档后端明确拒绝（只允许消费档），停用档指派后无法领取，因此都不进下拉
    it('offers only enabled consumption tiers in the assignment dropdown', async () => {
      listTiers.mockResolvedValue([
        tier({
          id: 4,
          code: 'first_recharge',
          name: '新客',
          trigger_type: 'first_recharge',
          threshold_usd: null,
          sort_order: 0,
        }),
        tier({ id: 1, code: 'tier_500', name: '青铜', sort_order: 1 }),
        tier({ id: 3, code: 'tier_1500', name: '领航', enabled: false, sort_order: 2 }),
      ])
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      const select = wrapper.find('#tier-assign-tier')
      const options = select.findAll('option')
      // 第 0 项是「请选择档位」占位项
      expect(options.map((option) => option.attributes('value'))).toEqual(['', 'tier_500'])
      expect(options[1].text()).toBe('青铜（tier_500）')
      // 表格里仍会渲染这两个档位，因此只在录入区的下拉范围内断言
      expect(select.text()).not.toContain('新客')
      expect(select.text()).not.toContain('领航')
    })

    it('leaves the target tier empty when the assigned tier is no longer assignable', async () => {
      listTiers.mockResolvedValue([tier({ id: 1, code: 'tier_500', name: '青铜', sort_order: 0 })])
      getTierAssignment.mockResolvedValue({
        user: ASSIGNED_USER,
        assignment: assignment({ tier_code: 'tier_1500', tier_name_snapshot: '领航', tier_id: null }),
      })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      // 保留既有指派的展示，但不预填一个后端会拒绝的档位
      expect(wrapper.text()).toContain('领航')
      expect((wrapper.find('#tier-assign-tier').element as HTMLSelectElement).value).toBe('')
    })

    it('saves the assignment and refreshes the shown value from the response', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })
      saveTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: assignment() })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      await wrapper.find('#tier-assign-tier').setValue('tier_1000')
      await wrapper.find('#tier-assign-note').setValue('运营手工配置')
      await (await saveButton(wrapper))!.trigger('click')
      await flushPromises()

      expect(saveTierAssignment).toHaveBeenCalledTimes(1)
      expect(saveTierAssignment).toHaveBeenCalledWith({
        email: 'alice@example.com',
        tier_code: 'tier_1000',
        note: '运营手工配置',
      })
      expect(wrapper.text()).toContain('白银')
      expect(showSuccess).toHaveBeenCalledWith('等级指派已保存')
    })

    // 查询后若又改了邮箱输入框，按输入框提交会把指派写到另一个用户上
    it('saves against the resolved email even when the input was edited afterwards', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })
      saveTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: assignment() })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      await wrapper.find('#tier-assign-email').setValue('bob@example.com')
      await wrapper.find('#tier-assign-tier').setValue('tier_1000')
      await (await saveButton(wrapper))!.trigger('click')
      await flushPromises()

      expect(saveTierAssignment).toHaveBeenCalledWith({
        email: 'alice@example.com',
        tier_code: 'tier_1000',
        note: '',
      })
    })

    it('removes the assignment and falls back to the empty state', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: assignment() })
      removeTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null, removed: true })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      await wrapper.findAll('button').find((button) => button.text() === '取消配置')!.trigger('click')
      await flushPromises()

      expect(removeTierAssignment).toHaveBeenCalledWith('alice@example.com')
      expect(showSuccess).toHaveBeenCalledWith('已取消等级指派')
      expect(wrapper.text()).toContain('该用户暂无等级指派')
      // 取消后不再有可取消的对象
      expect(wrapper.findAll('button').some((button) => button.text() === '取消配置')).toBe(false)
    })

    it('requires an email before looking up', async () => {
      const wrapper = mountView()
      await flushPromises()

      await wrapper.find('#tier-assign-lookup').trigger('click')
      await flushPromises()

      expect(getTierAssignment).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('请先输入用户邮箱')
    })

    it('requires a target tier before saving', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      await (await saveButton(wrapper))!.trigger('click')
      await flushPromises()

      expect(saveTierAssignment).not.toHaveBeenCalled()
      expect(showError).toHaveBeenCalledWith('请先选择要指派的档位')
    })

    it('maps the unknown-email lookup error to localized copy and clears the result', async () => {
      getTierAssignment.mockRejectedValueOnce({ status: 404, reason: 'USER_TIER_ASSIGN_USER_NOT_FOUND' })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'nobody@example.com')

      expect(showError).toHaveBeenCalledWith('该邮箱未匹配到用户')
      // 失败时不残留可操作的录入区
      expect(wrapper.find('#tier-assign-tier').exists()).toBe(false)
    })

    it('maps a rejected save error code to localized copy', async () => {
      getTierAssignment.mockResolvedValue({ user: ASSIGNED_USER, assignment: null })
      saveTierAssignment.mockRejectedValueOnce({
        status: 400,
        reason: 'USER_TIER_ASSIGNMENT_TARGET_DISABLED',
      })

      const wrapper = mountView()
      await flushPromises()
      await lookup(wrapper, 'alice@example.com')

      await wrapper.find('#tier-assign-tier').setValue('tier_1000')
      await (await saveButton(wrapper))!.trigger('click')
      await flushPromises()

      expect(showError).toHaveBeenCalledWith('该等级已停用')
      expect(showSuccess).not.toHaveBeenCalled()
    })

    // 上面的断言依赖 mock 词条，这一条校验真实词条确实存在：缺键会让英文语系管理员看到后端中文
    it('ships real localized copy for every assignment error code', () => {
      const codes = [
        'USER_TIER_ASSIGN_USER_NOT_FOUND',
        'USER_TIER_ASSIGN_EMAIL_AMBIGUOUS',
        'USER_TIER_ASSIGNMENT_TARGET_NOT_CONSUMPTION',
        'USER_TIER_ASSIGNMENT_TARGET_DISABLED',
        'USER_TIER_HAS_ASSIGNMENTS',
        'USER_TIER_NOT_FOUND',
        'USER_TIER_FEATURE_DISABLED',
      ]
      for (const code of codes) {
        expect((zh.admin.tierConfig as Record<string, string>)[code], `zh missing ${code}`).toBeTruthy()
        expect((en.admin.tierConfig as Record<string, string>)[code], `en missing ${code}`).toBeTruthy()
      }
    })

    // 文案是向管理员解释「指派生效范围」的唯一位置，因此直接校验真实词条
    it('documents the coverage rule of an assignment in both locales', () => {
      expect(zh.admin.tierConfig.assignHint).toContain('被指派档位及其之前的档位都变为可领取')
      expect(zh.admin.tierConfig.assignHint).toContain('权益仍需用户自行领取')
      expect(zh.admin.tierConfig.assignHint).toContain('调整档位顺序会同步改变覆盖范围')
      expect(en.admin.tierConfig.assignHint).toContain('become claimable')
      expect(en.admin.tierConfig.assignHint).toContain('Reordering tiers also changes the covered range')
    })

    // 等级标识已放开为任意字符，只剩 64 字符上限，提示文案必须写清上限，避免运营反复撞 400
    it('shows the relaxed code hint under the code input', async () => {
      const wrapper = mountView()
      await flushPromises()

      await rowActionButton(wrapper, 'common.edit')!.trigger('click')
      await flushPromises()

      expect(wrapper.text()).toContain('等级标识可为任意字符，最长 64 个字符')
      expect(zh.admin.tierConfig.codeHint).toContain('64')
      expect(en.admin.tierConfig.codeHint).toContain('64')
    })

    it('replaces the code hint with the locked hint once awards exist', async () => {
      listTiers.mockResolvedValue([tier({ id: 1, code: 'tier_500', award_count: 2 })])

      const wrapper = mountView()
      await flushPromises()

      await rowActionButton(wrapper, 'common.edit')!.trigger('click')
      await flushPromises()

      expect(wrapper.text()).not.toContain('等级标识可为任意字符，最长 64 个字符')
    })
  })
})

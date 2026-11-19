import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import RedeemView from '../RedeemView.vue'

const { post, listRedeemCodes, batchUpdateRedeemCodes, getAllGroups, showSuccess, showError, showInfo } =
  vi.hoisted(() => ({
    post: vi.fn(),
    listRedeemCodes: vi.fn(),
    batchUpdateRedeemCodes: vi.fn(),
    getAllGroups: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showInfo: vi.fn()
  }))

vi.mock('@/api/client', () => ({ apiClient: { post } }))

vi.mock('@/api/admin', async () => ({
  adminAPI: {
    redeem: {
      list: listRedeemCodes,
      generate: (await import('@/api/admin/redeem')).generate,
      delete: vi.fn(),
      batchDelete: vi.fn(),
      batchUpdate: batchUpdateRedeemCodes,
      exportCodes: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
    showInfo
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <table>
      <thead>
        <tr>
          <th v-for="column in columns" :key="column.key">
            <slot :name="'header-' + column.key" :column="column">{{ column.label }}</slot>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in data" :key="row.id">
          <td v-for="column in columns" :key="column.key">
            <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  `
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  setup(props: { options: Array<{ value: unknown; label: string }> }, { emit }: { emit: (event: string, ...args: unknown[]) => void }) {
    const onChange = (event: Event) => {
      const raw = (event.target as HTMLSelectElement).value
      const option = props.options.find((item) => String(item.value ?? '') === raw)
      const value = option ? option.value : raw
      emit('update:modelValue', value)
      emit('change', value, option ?? null)
    }
    return { onChange }
  },
  template: `
    <select v-bind="$attrs" :value="modelValue ?? ''" @change="onChange">
      <option v-for="option in options" :key="String(option.value ?? '')" :value="option.value ?? ''">
        {{ option.label }}
      </option>
    </select>
  `
}

const mountRedeemView = (stubSelect = true) => mount(RedeemView, {
  attachTo: document.body,
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: {
        template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
      },
      DataTable: DataTableStub,
      Pagination: true,
      ConfirmDialog: true,
      Select: stubSelect ? SelectStub : false,
      GroupBadge: true,
      GroupOptionItem: true,
      Icon: true,
      Teleport: true
    }
  }
})

const balanceValidity = '#generate-balance-credit-validity-days'

const openGenerateForm = (wrapper: ReturnType<typeof mountRedeemView>) =>
  wrapper.findAll('button').find((button) => button.text() === 'admin.redeem.generateCodes')!.trigger('click')

describe('admin RedeemView', () => {
  beforeEach(() => {
    localStorage.clear()
    document.body.innerHTML = ''

    post.mockReset().mockResolvedValue({ data: [] })
    listRedeemCodes.mockReset()
    batchUpdateRedeemCodes.mockReset()
    getAllGroups.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    showInfo.mockReset()

    listRedeemCodes.mockResolvedValue({
      items: [
        {
          id: 1,
          code: 'CODE-1',
          type: 'balance',
          value: 10,
          status: 'unused',
          used_by: null,
          used_at: null,
          created_at: '2026-01-01T00:00:00Z',
          expires_at: null
        },
        {
          id: 2,
          code: 'CODE-2',
          type: 'balance',
          value: 20,
          status: 'unused',
          used_by: null,
          used_at: null,
          created_at: '2026-01-01T00:00:00Z',
          expires_at: null
        }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    batchUpdateRedeemCodes.mockResolvedValue({ updated: 1, message: 'ok' })
    getAllGroups.mockResolvedValue([])
  })

  it('submits only checked fields for selected redeem codes', async () => {
    const wrapper = mountRedeemView()

    await flushPromises()
    await wrapper.findAll('[data-test="select-code"]')[0].setValue(true)
    await wrapper.get('[data-test="batch-update-open"]').trigger('click')
    await flushPromises()

    await wrapper.get('[data-test="batch-field-status"]').setValue(true)
    await wrapper.get('[data-test="batch-status-select"]').setValue('disabled')
    await wrapper.get('[data-test="batch-field-notes"]').setValue(true)
    await wrapper.get('[data-test="batch-notes-input"]').setValue('maintenance')
    await wrapper.get('[data-test="batch-update-form"]').trigger('submit')
    await flushPromises()

    expect(batchUpdateRedeemCodes).toHaveBeenCalledWith([1], {
      status: 'disabled',
      notes: 'maintenance'
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.redeem.batchUpdateSuccess')
    wrapper.unmount()
  })

  it.each([
    { input: '', type: 'balance', expected: {} },
    { input: '30', type: 'balance', expected: { balance_credit_validity_days: 30 } },
    { input: '0', type: 'balance', expected: { balance_credit_validity_days: 0 } },
    { input: '36500', type: 'balance', expected: { balance_credit_validity_days: 36500 } },
    { input: '30', type: 'concurrency', expected: {} }
  ])('generates $type codes with balance validity "$input"', async ({ input, type, expected }) => {
    const wrapper = mountRedeemView(false)
    await flushPromises()

    await openGenerateForm(wrapper)
    const inputField = wrapper.get<HTMLInputElement>(balanceValidity)
    expect(inputField.element.value).toBe('')
    await inputField.setValue(input)
    const selectType = async (selectedType: string) => {
      const currentForm = wrapper.get('form')
      await currentForm.get('button[aria-haspopup="true"]').trigger('click')
      const option = currentForm.findAll('[role="option"]')
        .find((item) => item.text() === `admin.redeem.${selectedType}`)
      await option!.trigger('click')
    }
    if (type !== 'balance') {
      await selectType(type)
    }
    const form = wrapper.get('form')
    expect(form.find(balanceValidity).exists()).toBe(type === 'balance')
    await form.findAll('button').find((button) => button.text() === 'admin.redeem.expiryPresetDays')!.trigger('click')
    await form.trigger('submit')
    await flushPromises()

    expect(post).toHaveBeenCalledOnce()
    expect(post).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 1,
      value: 10,
      type,
      expires_in_days: 1,
      ...expected
    })

    await openGenerateForm(wrapper)
    if (type !== 'balance') {
      await selectType('balance')
    }
    expect(wrapper.get<HTMLInputElement>(balanceValidity).element.value).toBe('')
    wrapper.unmount()
  })

  it.each(['-1', '1.5', '36501'])('rejects invalid balance validity %s', async (input) => {
    const wrapper = mountRedeemView()
    await flushPromises()

    await openGenerateForm(wrapper)
    const field = wrapper.get<HTMLInputElement>(balanceValidity)
    await field.setValue(input)
    expect(field.element.checkValidity()).toBe(false)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(post).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.redeem.invalidBalanceCreditValidityDays')
    wrapper.unmount()
  })
})

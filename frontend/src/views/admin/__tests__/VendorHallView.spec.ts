import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'

import VendorHallView from '../VendorHallView.vue'

const { list, pauseScheduling, setSchedulable, push, showSuccess, showError } = vi.hoisted(() => ({
  list: vi.fn(),
  pauseScheduling: vi.fn(),
  setSchedulable: vi.fn(),
  push: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    vendorHall: { list, pauseScheduling },
    accounts: { setSchedulable },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError }),
}))

vi.mock('vue-router', () => ({ useRouter: () => ({ push }) }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('en') }) }
})

const ConfirmDialogStub = defineComponent({
  props: ['show', 'title', 'message', 'danger'],
  emits: ['confirm', 'cancel'],
  template: '<button v-if="show" data-test="confirm-action" @click="$emit(\'confirm\')">confirm</button>',
})

const response = {
  summary: {
    total_accounts: 2,
    healthy_accounts: 1,
    paused_accounts: 1,
    average_availability: 0.975,
    updated_at: '2026-08-17T12:00:00Z',
  },
  items: [
    {
      account_id: 7,
      account_name: 'OpenAI East',
      platform: 'openai',
      group_name: 'premium',
      rate_multiplier: 1.2,
      balance_usd: 12.34,
      scheduling_status: 'schedulable',
      availability: 0.99,
      cache_hit_rate: 0.45,
      average_latency_ms: 820,
      p95_latency_ms: 1400,
      user_ttft_average_ms: 310,
      user_ttft_p95_ms: 560,
      request_count: 3021,
      collected_at: '2026-08-17T12:00:00Z',
      trend: [
        { timestamp: '2026-08-17T11:00:00Z', ttft_p95_ms: 700 },
        { timestamp: '2026-08-17T12:00:00Z', ttft_p95_ms: 560 },
      ],
    },
    {
      account_id: 8,
      account_name: 'Claude West',
      platform: 'anthropic',
      group_name: 'fallback',
      rate_multiplier: null,
      balance_usd: null,
      scheduling_status: 'paused',
      availability: null,
      cache_hit_rate: null,
      average_latency_ms: null,
      p95_latency_ms: null,
      user_ttft_average_ms: null,
      user_ttft_p95_ms: null,
      request_count: 0,
      collected_at: null,
      trend: [],
    },
  ],
  total: 2,
  page: 1,
  page_size: 20,
}

function mountView() {
  return mount(VendorHallView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        ConfirmDialog: ConfirmDialogStub,
      },
    },
  })
}

describe('VendorHallView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue(response)
    pauseScheduling.mockResolvedValue({ account_id: 7, temp_unschedulable_until: '2026-08-17T13:00:00Z' })
    setSchedulable.mockResolvedValue({ id: 7, schedulable: false })
  })

  it('renders monitor metrics and a clear empty metric state', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI East')
    expect(wrapper.text()).toContain('99.0%')
    expect(wrapper.text()).toContain('310 ms')
    expect(wrapper.text()).toContain('560 ms')
    expect(wrapper.text()).toContain('$12.34')
    expect(wrapper.text()).toContain('--')
    expect(wrapper.find('[data-test="ttft-trend-7"]').exists()).toBe(true)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ window: '3h' }))
    expect(wrapper.text()).toContain('3H')
    expect(wrapper.text()).toContain('3D')
    expect(wrapper.text()).not.toContain('6H')
  })

  it('pauses the account from its expanded action area after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('[aria-label="admin.vendorHall.details"]')[0].trigger('click')
    await wrapper.get('[data-test="pause-account-7"]').trigger('click')
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(pauseScheduling).toHaveBeenCalledTimes(1)
    expect(pauseScheduling).toHaveBeenCalledWith(7)
    expect(list).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-test="pause-account-7"]').element.closest('article')?.textContent).toContain('admin.vendorHall.status.paused')
  })

  it('closes scheduling after a danger confirmation without duplicate submission', async () => {
    let resolveClose!: (value: unknown) => void
    setSchedulable.mockReturnValue(new Promise((resolve) => { resolveClose = resolve }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('[aria-label="admin.vendorHall.details"]')[0].trigger('click')
    await wrapper.get('[data-test="disable-account-7"]').trigger('click')
    const confirm = wrapper.get('[data-test="confirm-action"]')
    await confirm.trigger('click')
    await confirm.trigger('click')

    expect(setSchedulable).toHaveBeenCalledTimes(1)
    expect(setSchedulable).toHaveBeenCalledWith(7, false)
    resolveClose({ id: 7, schedulable: false })
    await flushPromises()
  })

  it('enables scheduling for a disabled account after confirmation', async () => {
    let loadCount = 0
    list.mockImplementation(async () => ({
      ...response,
      items: [{ ...response.items[0], account_id: 9, account_name: 'Disabled account', scheduling_status: loadCount++ === 0 ? 'disabled' : 'schedulable' }],
      total: 1,
    }))
    setSchedulable.mockResolvedValue({ id: 9, schedulable: true })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('button.vendor-expand').trigger('click')
    expect(wrapper.get('[data-test="enable-account-9"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="disable-account-9"]').exists()).toBe(false)
    await wrapper.get('[data-test="enable-account-9"]').trigger('click')
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(setSchedulable).toHaveBeenCalledWith(9, true)
    expect(wrapper.text()).toContain('admin.vendorHall.status.schedulable')
  })

  it('keeps account actions in the expanded row instead of the toolbar', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="view-usage-selected"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="manage-account-selected"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="pause-selected"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="disable-selected"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="select-account-7"]').exists()).toBe(false)

    await wrapper.findAll('[aria-label="admin.vendorHall.details"]')[0].trigger('click')
    await wrapper.get('[data-test="view-usage-account-7"]').trigger('click')
    expect(push).toHaveBeenLastCalledWith({ path: '/admin/usage', query: { account_id: '7' } })
    await wrapper.get('[data-test="manage-account-7"]').trigger('click')
    expect(push).toHaveBeenLastCalledWith({ path: '/admin/accounts', query: { account_id: '7' } })
  })

  it('offers account-specific actions in the expanded row', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('[aria-label="admin.vendorHall.details"]')[0].trigger('click')
    expect(wrapper.get('[data-test="view-usage-account-7"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="manage-account-7"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="pause-account-7"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="disable-account-7"]').exists()).toBe(true)

    await wrapper.get('[data-test="view-usage-account-7"]').trigger('click')
    expect(push).toHaveBeenCalledWith({ path: '/admin/usage', query: { account_id: '7' } })

    await wrapper.get('[data-test="pause-account-7"]').trigger('click')
    expect(wrapper.get('[data-test="confirm-action"]').exists()).toBe(true)
  })

  it('shows structured API error messages', async () => {
    list.mockRejectedValue({ status: 503, message: 'Vendor hall data is unavailable' })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Vendor hall data is unavailable')
  })
})

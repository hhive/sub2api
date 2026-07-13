import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import DefaultModelPricingView from '../DefaultModelPricingView.vue'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/api/admin', () => ({
  adminAPI: { defaultModelPricing: { list } }
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false }
  },
  emits: ['sort'],
  setup(props, { emit, slots }) {
    return () => h('div', { 'data-testid': 'pricing-table' }, [
      h('button', { 'data-testid': 'sort-provider', onClick: () => emit('sort', 'provider', 'desc') }, 'sort'),
      ...(props.data as Record<string, unknown>[]).map(row =>
        h('div', { 'data-testid': `row-${row.model}` }, slots['cell-model']?.({ row, value: row.model }))
      ),
      !props.loading && props.data.length === 0 ? slots.empty?.() : null
    ])
  }
})

const SelectStub = defineComponent({
  inheritAttrs: false,
  props: { modelValue: { type: [String, Number, Boolean], default: '' }, options: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(props, { attrs, emit }) {
    return () => h('select', {
      ...attrs,
      value: props.modelValue,
      onChange: (event: Event) => {
        const value = (event.target as HTMLSelectElement).value
        emit('update:modelValue', value)
        emit('change', value, null)
      }
    }, (props.options as Array<{ value: string; label: string }>).map(option =>
      h('option', { value: option.value }, option.label)
    ))
  }
})

const responseItem = {
  model: 'gpt-test',
  provider: 'openai',
  mode: 'chat',
  input_cost_per_token: 0.000001,
  input_cost_per_token_priority: null,
  output_cost_per_token: 0.000002,
  output_cost_per_token_priority: null,
  cache_creation_input_token_cost: null,
  cache_creation_input_token_cost_priority: null,
  cache_creation_input_token_cost_above_1hr: null,
  cache_read_input_token_cost: null,
  cache_read_input_token_cost_priority: null,
  output_cost_per_image: null,
  output_cost_per_image_token: null,
  long_context_input_token_threshold: null,
  long_context_input_cost_multiplier: null,
  long_context_output_cost_multiplier: null,
  supports_service_tier: false,
  supports_prompt_caching: false,
  token_pricing_absent: false
}

function response(model = 'gpt-test') {
  return {
    items: [{ ...responseItem, model }],
    total: 1,
    page: 1,
    page_size: 20,
    providers: ['openai', 'anthropic'],
    modes: ['chat'],
    status: { model_count: 2, last_updated: '2026-07-13T12:00:00Z', local_hash: '12345678' }
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

function mountView() {
  return mount(DefaultModelPricingView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters"/><slot name="table"/><slot name="pagination"/></div>' },
        DataTable: DataTableStub,
        Pagination: { template: '<div data-testid="pagination" />' },
        Select: SelectStub,
        Icon: { template: '<span />' }
      }
    }
  })
}

describe('DefaultModelPricingView', () => {
  beforeEach(() => list.mockReset())

  it('loads the runtime catalog and sends provider and sort filters', async () => {
    list.mockResolvedValue(response())
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="row-gpt-test"]').exists()).toBe(true)
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({
      page: 1,
      page_size: 20,
      sort_by: 'model',
      sort_order: 'asc',
      signal: expect.any(AbortSignal)
    }))

    await wrapper.find('[data-testid="provider-filter"]').setValue('openai')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ provider: 'openai', page: 1 }))

    await wrapper.find('[data-testid="sort-provider"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ sort_by: 'provider', sort_order: 'desc', page: 1 }))
  })

  it('ignores a stale response when filters change during loading', async () => {
    const first = deferred<ReturnType<typeof response>>()
    const second = deferred<ReturnType<typeof response>>()
    list.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const wrapper = mountView()

    const providerSelect = wrapper.findAllComponents(SelectStub)[0]
    providerSelect.vm.$emit('update:modelValue', 'openai')
    providerSelect.vm.$emit('change', 'openai', null)
    second.resolve(response('new-result'))
    await flushPromises()
    first.resolve(response('stale-result'))
    await flushPromises()

    expect(wrapper.find('[data-testid="row-new-result"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="row-stale-result"]').exists()).toBe(false)
  })

  it('keeps the last successful rows when refresh fails', async () => {
    list.mockResolvedValueOnce(response()).mockRejectedValueOnce(new Error('offline'))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-testid="pricing-refresh"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="row-gpt-test"]').exists()).toBe(true)
  })
})

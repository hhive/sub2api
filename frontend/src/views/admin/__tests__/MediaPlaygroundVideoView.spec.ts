import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, h, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import MediaPlaygroundVideoView from '../MediaPlaygroundVideoView.vue'

const { listModels, listTasks, getTask, createModel, updateModel } = vi.hoisted(() => ({
  listModels: vi.fn(), listTasks: vi.fn(), getTask: vi.fn(), createModel: vi.fn(), updateModel: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: { mediaPlaygroundVideo: {
  listModels, listTasks, getTask, createModel, updateModel, deleteModel: vi.fn(),
} } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
vi.mock('@/utils/apiError', () => ({ extractApiErrorMessage: (_e: unknown, fallback: string) => fallback }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const translations: Record<string, string> = {
    'admin.videoPlayground.filters.searchPlaceholder': '筛选视频模型',
    'admin.videoPlayground.filters.clear': '清空筛选',
    'admin.videoPlayground.filters.resultCount': '{visible} / {total}',
    'admin.videoPlayground.sections.basic': '基础信息',
    'admin.videoPlayground.sections.upstream': '上游连接',
    'admin.videoPlayground.sections.billingRuntime': '计费与运行',
    'admin.videoPlayground.sections.status': '状态',
  }
  return { ...actual, useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`)),
  }) }
})

const DataTableStub = defineComponent({
  props: { columns: Array, data: Array },
  setup(props, { slots }) {
    const sortKey = ref('')
    const sortOrder = ref<'asc' | 'desc'>('asc')
    const sortedData = computed(() => {
      const rows = [...(props.data as any[] || [])]
      if (!sortKey.value) return rows
      return rows.sort((left, right) => {
        const result = String(left[sortKey.value] ?? '').localeCompare(String(right[sortKey.value] ?? ''), undefined, {
          numeric: true,
          sensitivity: 'base',
        })
        return sortOrder.value === 'asc' ? result : -result
      })
    })
    const sort = (column: any) => {
      if (!column.sortable) return
      if (sortKey.value === column.key) sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
      else {
        sortKey.value = column.key
        sortOrder.value = 'asc'
      }
    }
    return () => h('table', [
      h('thead', [h('tr', (props.columns as any[] || []).map((column) => h('th', {
        'data-column': column.key,
        'data-sortable': String(Boolean(column.sortable)),
        'aria-sort': column.sortable
          ? sortKey.value === column.key
            ? sortOrder.value === 'asc' ? 'ascending' : 'descending'
            : 'none'
          : undefined,
        onClick: () => sort(column),
      }, column.label)))]),
      h('tbody', sortedData.value.map(row => h('tr', { 'data-row': row.id ?? row.task_id },
        (props.columns as any[] || []).map(col => h(
          'td',
          { 'data-column': col.key, 'data-row': row.id ?? row.task_id },
          slots[`cell-${col.key}`]?.({ row, value: row[col.key] }) || row[col.key] || ''
        ))
      ))),
    ])
  },
})
const DialogStub = defineComponent({ props: { show: Boolean }, setup(props, { slots }) { return () => props.show ? h('section', [slots.default?.(), slots.footer?.()]) : null } })
const model = { id: 7, media_type: 'video', display_name: 'Video', model: 'video-1', provider_name: 'OpenAI', api_mode: 'openai_videos_v2', upstream_base_url: 'https://upstream.test', upstream_api_key: '', upstream_api_key_mask: '123456', price_quota: 2, billing_mode: 'balance_prepaid', refund_enabled: true, timeout_seconds: 90, enabled: true, sort_order: 4 }
const secondModel = { ...model, id: 8, display_name: 'Animation', model: 'seedance-1', provider_name: 'ByteDance', api_mode: 'seedance_content_generation', upstream_base_url: 'https://seedance.test', price_quota: 1, billing_mode: 'postpaid', refund_enabled: false, sort_order: 5 }

function mountView() { return mount(MediaPlaygroundVideoView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, DataTable: DataTableStub, BaseDialog: DialogStub, ConfirmDialog: true, Icon: true } } }) }
function button(wrapper: ReturnType<typeof mountView>, text: string) { return wrapper.findAll('button').find(item => item.text().includes(text))! }

describe('MediaPlaygroundVideoView interactions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listModels.mockResolvedValue([model])
    listTasks.mockResolvedValue({
      items: [
        { task_id: 'local-task-1', user_id: 3, model: 'video-1', status: 'completed', progress: 100, upstream_task_id: 'upstream-task-9', duration_ms: 1500, refund_status: 'none', error_message: '', created_at: '2026-07-13T10:00:00Z' },
        { task_id: 'local-task-2', user_id: 4, model: 'video-1', status: 'running', progress: 10, upstream_task_id: '', duration_ms: null, refund_status: 'none', error_message: '', created_at: '2026-07-13T10:01:00Z' },
      ],
      total: 2,
      page: 1,
      page_size: 20,
    })
    getTask.mockResolvedValue({
      task: { task_id: 'local-task-1', status: 'completed', progress: 100, upstream_task_id: 'upstream-task-9', duration_ms: 1500, refund_status: 'none', refund_reason: '', error_message: '', request: { prompt: 'safe prompt' }, upstream_response: { status: 'completed' } },
    })
    createModel.mockResolvedValue({})
    updateModel.mockResolvedValue({})
  })

  it('shows one task-record entry with upstream ID and duration, without call records', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).not.toContain('admin.videoPlayground.callRecords.button')
    await button(wrapper, 'admin.videoPlayground.taskRecords.button').trigger('click')
    await flushPromises()
    expect(listTasks).toHaveBeenCalledWith({ page: 1, page_size: 20, status: undefined })
    expect(wrapper.text()).toContain('upstream-task-9')
    expect(wrapper.text()).toContain('1.50s')
    expect(wrapper.get('[data-row="local-task-2"][data-column="duration_ms"]').text()).toBe('-')
    await button(wrapper, 'local-task-1').trigger('click')
    await flushPromises()
    expect(getTask).toHaveBeenCalledWith('local-task-1')
    expect(wrapper.text()).toContain('safe prompt')
    expect(wrapper.text()).toContain('completed')
  })

  it('groups the video model form into four semantic sections', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'admin.videoPlayground.createModel').trigger('click')
    expect(wrapper.findAll('fieldset').map((section) => section.attributes('data-testid'))).toEqual([
      'model-section-basic',
      'model-section-upstream',
      'model-section-billing-runtime',
      'model-section-status',
    ])
    expect(wrapper.findAll('legend').map((legend) => legend.text())).toEqual([
      '基础信息',
      '上游连接',
      '计费与运行',
      '状态',
    ])
  })

  it('uses the shared leading column order and only marks supported video columns sortable', async () => {
    const wrapper = mountView(); await flushPromises()
    const columns = wrapper.findComponent(DataTableStub).props('columns') as any[]
    expect(columns.slice(0, 4).map((column) => column.key)).toEqual([
      'display_name',
      'provider_name',
      'api_mode',
      'upstream_base_url',
    ])
    expect(columns.filter((column) => column.sortable).map((column) => column.key)).toEqual([
      'display_name',
      'provider_name',
      'api_mode',
      'upstream_base_url',
      'price_quota',
      'billing_mode',
      'refund_enabled',
      'enabled',
    ])
    expect(columns.find((column) => column.key === 'actions')?.sortable).not.toBe(true)
  })

  it('filters video models by provider and protocol, reports counts, and clears locally', async () => {
    listModels.mockResolvedValue([model, secondModel])
    const wrapper = mountView(); await flushPromises()
    const callsAfterLoad = listModels.mock.calls.length
    const search = wrapper.get('input[type="search"]')
    expect(search.attributes('aria-label')).toBe('筛选视频模型')
    expect(wrapper.get('[role="status"]').attributes('aria-live')).toBe('polite')

    await search.setValue('bytedance')
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('Animation')
    expect(wrapper.text()).toContain('1 / 2')

    await search.setValue('OPENAI_VIDEOS_V2')
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('Video')
    expect(wrapper.text()).not.toContain('Animation')
    expect(listModels).toHaveBeenCalledTimes(callsAfterLoad)

    await wrapper.findAll('button').find((item) => item.text() === '清空筛选')!.trigger('click')
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    expect(wrapper.text()).toContain('2 / 2')
  })

  it('sorts video rows ascending and descending from the same header', async () => {
    listModels.mockResolvedValue([
      model,
      secondModel,
      { ...model, id: 9, display_name: 'Zeta Local', upstream_base_url: 'https://local.invalid' },
    ])
    const wrapper = mountView(); await flushPromises()
    const rowIds = () => wrapper.findAll('tbody [data-column="display_name"]').map((cell) => cell.attributes('data-row'))
    const header = wrapper.get('thead [data-column="display_name"]')

    await wrapper.get('input[type="search"]').setValue('.test')
    await header.trigger('click')
    expect(header.attributes('aria-sort')).toBe('ascending')
    expect(rowIds()).toEqual(['8', '7'])
    await header.trigger('click')
    expect(header.attributes('aria-sort')).toBe('descending')
    expect(rowIds()).toEqual(['7', '8'])

    await wrapper.findAll('button').find((item) => item.text() === '清空筛选')!.trigger('click')
    expect(header.attributes('aria-sort')).toBe('descending')
    expect(rowIds()).toEqual(['9', '7', '8'])
  })

  it('creates a model with the OpenAI Videos API2 mode', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'admin.videoPlayground.createModel').trigger('click')
    const form = wrapper.get('form')
    const apiMode = form.findAll('select').find(select => select.find('option[value="openai_videos_v2"]').exists())!
    await apiMode.setValue('openai_videos_v2')
    await form.trigger('submit'); await flushPromises()
    expect(createModel).toHaveBeenCalledWith(expect.objectContaining({ api_mode: 'openai_videos_v2' }))
  })

  it('edits with an empty key and submits the selected required API mode', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'common.edit').trigger('click')
    const form = wrapper.get('form'); expect(form.get('select').attributes('required')).toBeDefined()
    await form.trigger('submit'); await flushPromises()
    expect(updateModel).toHaveBeenCalledWith(7, expect.objectContaining({ api_mode: 'openai_videos_v2', upstream_api_key: '' }))
  })

  it('blocks submission when API mode is empty', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'common.edit').trigger('click')
    const form = wrapper.get('form')
    const apiMode = form.findAll('select').find(select => select.find('option[value="openai_videos"]').exists())!
    expect(apiMode.find('option[value="openai_videos_v2"]').exists()).toBe(true)
    await apiMode.setValue('')
    expect((form.element as HTMLFormElement).checkValidity()).toBe(false)
    await button(wrapper, 'common.save').trigger('click'); await flushPromises()
    expect(updateModel).not.toHaveBeenCalled()
  })

  it('reuses a model without copying its configured key', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'admin.videoPlayground.reuse').trigger('click')
    const reuse = wrapper.findAll('select').find(select => select.find('option[value="7"]').exists())!
    await reuse.setValue('7'); await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(createModel).toHaveBeenCalledWith(expect.objectContaining({ upstream_api_key: '', api_mode: 'openai_videos_v2' }))
  })

  it('toggles with the complete writable payload and exposes no legacy controls', async () => {
    const wrapper = mountView(); await flushPromises(); await button(wrapper, 'admin.videoPlayground.createModel').trigger('click')
    for (const legacyLabel of [
      'admin.videoPlayground.fields.studioTemplate',
      'admin.videoPlayground.fields.modelKind',
      'admin.videoPlayground.fields.inputSchemaJSON',
      'admin.videoPlayground.fields.payloadMappingJSON',
    ]) expect(wrapper.text()).not.toContain(legacyLabel)
    await button(wrapper, 'common.cancel').trigger('click')
    await button(wrapper, 'common.enabled').trigger('click'); await flushPromises()
    expect(updateModel).toHaveBeenCalledWith(7, { display_name: 'Video', model: 'video-1', provider_name: 'OpenAI', api_mode: 'openai_videos_v2', upstream_base_url: 'https://upstream.test', upstream_api_key: '', price_quota: 2, billing_mode: 'balance_prepaid', refund_enabled: true, timeout_seconds: 90, enabled: false, sort_order: 4 })
  })

  it('enables an API2 model without changing its mode', async () => {
    listModels.mockResolvedValue([{ ...model, enabled: false }])
    const wrapper = mountView(); await flushPromises()
    await button(wrapper, 'common.disabled').trigger('click'); await flushPromises()
    expect(updateModel).toHaveBeenCalledWith(7, expect.objectContaining({ api_mode: 'openai_videos_v2', enabled: true }))
  })
})

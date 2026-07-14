import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
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
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const DataTableStub = defineComponent({
  props: { columns: Array, data: Array },
  setup(props, { slots }) { return () => h('div', (props.data as any[] || []).flatMap(row => (props.columns as any[] || []).map(col => h('div', slots[`cell-${col.key}`]?.({ row, value: row[col.key] }) || row[col.key] || '')))) },
})
const DialogStub = defineComponent({ props: { show: Boolean }, setup(props, { slots }) { return () => props.show ? h('section', [slots.default?.(), slots.footer?.()]) : null } })
const model = { id: 7, media_type: 'video', display_name: 'Video', model: 'video-1', provider_name: 'OpenAI', api_mode: 'openai_videos_v2', upstream_base_url: 'https://upstream.test', upstream_api_key: '', upstream_api_key_mask: '123456', price_quota: 2, billing_mode: 'balance_prepaid', refund_enabled: true, timeout_seconds: 90, enabled: true, sort_order: 4 }

function mountView() { return mount(MediaPlaygroundVideoView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, DataTable: DataTableStub, BaseDialog: DialogStub, ConfirmDialog: true, Icon: true } } }) }
function button(wrapper: ReturnType<typeof mountView>, text: string) { return wrapper.findAll('button').find(item => item.text().includes(text))! }

describe('MediaPlaygroundVideoView interactions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listModels.mockResolvedValue([model])
    listTasks.mockResolvedValue({
      items: [{ task_id: 'local-task-1', user_id: 3, model: 'video-1', status: 'completed', progress: 100, upstream_task_id: 'upstream-task-9', duration_ms: 1500, refund_status: 'none', error_message: '', created_at: '2026-07-13T10:00:00Z' }],
      total: 1,
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
    await button(wrapper, 'local-task-1').trigger('click')
    await flushPromises()
    expect(getTask).toHaveBeenCalledWith('local-task-1')
    expect(wrapper.text()).toContain('safe prompt')
    expect(wrapper.text()).toContain('completed')
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

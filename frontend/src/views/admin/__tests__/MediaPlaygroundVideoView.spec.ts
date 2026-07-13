import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import MediaPlaygroundVideoView from '../MediaPlaygroundVideoView.vue'

const { listModels, createModel, updateModel } = vi.hoisted(() => ({
  listModels: vi.fn(), createModel: vi.fn(), updateModel: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: { mediaPlaygroundVideo: {
  listModels, createModel, updateModel, deleteModel: vi.fn(), listUpstreamRequests: vi.fn().mockResolvedValue({ items: [], total: 0 }),
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
  beforeEach(() => { vi.clearAllMocks(); listModels.mockResolvedValue([model]); createModel.mockResolvedValue({}); updateModel.mockResolvedValue({}) })

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

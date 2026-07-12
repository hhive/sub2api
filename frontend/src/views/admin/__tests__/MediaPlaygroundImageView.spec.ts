import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import MediaPlaygroundImageView from '../MediaPlaygroundImageView.vue'

const { listModels, listProbeRuns, runModelProbe, runProbe, showError, showSuccess } = vi.hoisted(() => ({
  listModels: vi.fn(),
  listProbeRuns: vi.fn(),
  runModelProbe: vi.fn(),
  runProbe: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    mediaPlaygroundImage: {
      listModels,
      listProbeRuns,
      runModelProbe,
      runProbe,
      createModel: vi.fn(),
      updateModel: vi.fn(),
      deleteModel: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const translations: Record<string, string> = {
    'common.actions': '操作',
    'common.refresh': '刷新',
    'common.enabled': '启用',
    'common.disabled': '停用',
    'common.edit': '编辑',
    'common.delete': '删除',
    'admin.imagePlayground.title': '图片模型配置',
    'admin.imagePlayground.description': '描述',
    'admin.imagePlayground.createModel': '新建模型',
    'admin.imagePlayground.reuse': '复用',
    'admin.imagePlayground.probeRuns.button': '探测记录',
    'admin.imagePlayground.probeRuns.runButton': '主动探测',
    'admin.imagePlayground.probeRuns.singleRunButton': '探测',
    'admin.imagePlayground.probeRuns.title': '探测记录',
    'admin.imagePlayground.probeRuns.description': '探测描述',
    'admin.imagePlayground.probeRuns.runSuccess': '主动探测已开始',
    'admin.imagePlayground.probeRuns.singleRunSuccess': '{name} 探测已开始',
    'admin.imagePlayground.probeRuns.previous': '上一页',
    'admin.imagePlayground.probeRuns.next': '下一页',
    'admin.imagePlayground.probeRuns.pageInfo': '第 {page} 页，共 {total} 条',
    'admin.imagePlayground.columns.name': '模型',
    'admin.imagePlayground.columns.apiMode': 'API 模式',
    'admin.imagePlayground.columns.provider': '供应商',
    'admin.imagePlayground.columns.upstream': '上游域名',
    'admin.imagePlayground.columns.prices': '档位价格',
    'admin.imagePlayground.columns.sizes': '尺寸',
    'admin.imagePlayground.columns.sortOrder': '排序',
    'admin.imagePlayground.columns.health': '健康状态',
    'admin.imagePlayground.columns.enabled': '启用',
    'admin.imagePlayground.apiModes.images': 'Images API',
    'admin.imagePlayground.apiModes.responses': 'Responses API',
    'admin.imagePlayground.apiModes.geminiGenerateContent': 'Gemini GenerateContent API',
    'admin.imagePlayground.health.cooldownUntil': '冷却至',
    'admin.imagePlayground.health.failures': '失败',
    'admin.imagePlayground.health.cooldowns': '冷却',
    'admin.imagePlayground.health.halfOpenAttempts': '半开',
    'admin.imagePlayground.health.lastError': '最后错误：',
    'admin.imagePlayground.health.status.available': '可用',
    'admin.imagePlayground.health.status.temporaryUnavailable': '冷却中',
    'admin.imagePlayground.health.status.halfOpen': '半开探测',
    'admin.imagePlayground.health.status.disabled': '已禁用',
    'admin.imagePlayground.probeRuns.columns.createdAt': '时间',
    'admin.imagePlayground.probeRuns.columns.model': '模型',
    'admin.imagePlayground.probeRuns.columns.apiMode': 'API 模式',
    'admin.imagePlayground.probeRuns.columns.upstream': '上游域名',
    'admin.imagePlayground.probeRuns.columns.attempt': '轮次',
    'admin.imagePlayground.probeRuns.columns.status': '状态',
    'admin.imagePlayground.probeRuns.columns.httpStatus': 'HTTP',
    'admin.imagePlayground.probeRuns.columns.elapsed': '耗时',
    'admin.imagePlayground.probeRuns.columns.responseBytes': '响应大小',
    'admin.imagePlayground.probeRuns.columns.imageCount': '图片数',
    'admin.imagePlayground.probeRuns.columns.error': '错误',
    'admin.imagePlayground.probeRuns.status.success': '成功',
    'admin.imagePlayground.probeRuns.status.failed': '失败',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`)),
    }),
  }
})

const DataTableStub = defineComponent({
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, default: () => [] },
    loading: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    return () =>
      h('table', [
        h('thead', [h('tr', (props.columns as any[]).map((column) => h('th', column.label)))]),
        h(
          'tbody',
          (props.data as any[]).map((row) =>
            h(
              'tr',
              (props.columns as any[]).map((column) =>
                h('td', slots[`cell-${column.key}`]?.({ row, value: row[column.key] }) ?? row[column.key] ?? '')
              )
            )
          )
        ),
      ])
  },
})

const BaseDialogStub = defineComponent({
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => (props.show ? h('section', [h('h2', props.title), slots.default?.(), slots.footer?.()]) : null)
  },
})

const model = {
  id: 7,
  display_name: 'Image Fast',
  model: 'gpt-image-2',
  api_mode: 'images',
  provider_name: 'OpenAI',
  upstream_base_url: 'https://upstream.test',
  upstream_api_key: '',
  upstream_api_key_mask: 'secret',
  price_1k: 1,
  price_2k: 2,
  price_4k: 4,
  supported_sizes: ['1k', '2k', '4k'],
  timeout_seconds: 600,
  fallback_to_responses_enabled: true,
  health_status: 'temporary_unavailable',
  consecutive_failures: 2,
  half_open_attempts: 1,
  cooldown_count: 1,
  cooldown_until: '2026-07-02T15:00:00Z',
  last_health_error: 'upstream status 524',
  enabled: true,
  sort_order: 1,
}

const secondModel = {
  ...model,
  id: 8,
  display_name: 'Image Backup',
  model: 'gpt-image-backup',
  sort_order: 2,
}

function mountView() {
  return mount(MediaPlaygroundImageView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: { template: '<div />' },
        Icon: { template: '<span />' },
      },
    },
  })
}

describe('MediaPlaygroundImageView', () => {
  beforeEach(() => {
    listModels.mockReset()
    listProbeRuns.mockReset()
    runModelProbe.mockReset()
    runProbe.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    listModels.mockResolvedValue([model])
    listProbeRuns.mockResolvedValue({
      items: [
        {
          id: 1,
          run_id: 'run-1',
          model_config_id: 7,
          model: 'gpt-image-2',
          api_mode: 'images',
          upstream_base_url: 'https://upstream.test',
          attempt: 1,
          status: 'success',
          http_status_code: 200,
          error_message: '',
          elapsed_ms: 1234,
          response_bytes: 2048,
          image_count: 1,
          created_at: '2026-07-02T12:00:00Z',
        },
      ],
      total: 21,
      page: 1,
      page_size: 20,
    })
    runModelProbe.mockResolvedValue({ ok: true, running: true })
    runProbe.mockResolvedValue({ ok: true, running: true })
  })

  it('shows image model health and error state in the config table', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('健康状态')
    expect(wrapper.text()).toContain('冷却中')
    expect(wrapper.text()).toContain('失败 2')
    expect(wrapper.text()).toContain('冷却 1')
    expect(wrapper.text()).toContain('半开 1')
    expect(wrapper.text()).toContain('upstream status 524')
  })

  it('loads probe records, pages, and refreshes the current page', async () => {
    const wrapper = mountView()
    await flushPromises()

	await wrapper.findAll('button').find((button) => button.text() === '探测记录')!.trigger('click')
	await flushPromises()
	expect(listProbeRuns).toHaveBeenLastCalledWith({ page: 1, page_size: 20 })
	expect(wrapper.text()).toContain('#7')
	expect(wrapper.text()).toContain('gpt-image-2')
	expect(wrapper.text()).toContain('成功')

    listProbeRuns.mockResolvedValueOnce({ items: [], total: 21, page: 2, page_size: 20 })
    await wrapper.findAll('button').find((button) => button.text() === '下一页')!.trigger('click')
    await flushPromises()
    expect(listProbeRuns).toHaveBeenLastCalledWith({ page: 2, page_size: 20 })

    await wrapper.findAll('button').filter((button) => button.text() === '刷新').at(-1)!.trigger('click')
    await flushPromises()
    expect(listProbeRuns).toHaveBeenLastCalledWith({ page: 2, page_size: 20 })
  })

  it('runs a manual async probe, shows started toast, and reloads models', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === '主动探测')!.trigger('click')
    await flushPromises()

    expect(runProbe).toHaveBeenCalledTimes(1)
    expect(showSuccess).toHaveBeenCalledWith('主动探测已开始')
    expect(listModels).toHaveBeenCalledTimes(2)
  })

  it('runs a single model probe from the config row', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text() === '探测')!.trigger('click')
    await flushPromises()

    expect(runModelProbe).toHaveBeenCalledWith(7)
    expect(showSuccess).toHaveBeenCalledWith('Image Fast 探测已开始')
    expect(listModels).toHaveBeenCalledTimes(2)
  })

  it('tracks loading state independently for multiple row probes', async () => {
    listModels.mockResolvedValue([model, secondModel])
    let resolveFirst!: (value: { ok: boolean; running: boolean }) => void
    let resolveSecond!: (value: { ok: boolean; running: boolean }) => void
    runModelProbe.mockImplementation((id: number) => {
      return new Promise((resolve) => {
        if (id === 7) {
          resolveFirst = resolve
          return
        }
        resolveSecond = resolve
      })
    })

    const wrapper = mountView()
    await flushPromises()

    const probeButtons = () => wrapper.findAll('button').filter((button) => button.text() === '探测')
    await probeButtons()[0].trigger('click')
    await nextTick()
    await probeButtons()[1].trigger('click')
    await nextTick()

    expect(runModelProbe).toHaveBeenNthCalledWith(1, 7)
    expect(runModelProbe).toHaveBeenNthCalledWith(2, 8)
    expect(probeButtons()[0].attributes('disabled')).toBeDefined()
    expect(probeButtons()[1].attributes('disabled')).toBeDefined()

    resolveFirst({ ok: true, running: true })
    await flushPromises()
    expect(probeButtons()[0].attributes('disabled')).toBeUndefined()
    expect(probeButtons()[1].attributes('disabled')).toBeDefined()

    resolveSecond({ ok: true, running: true })
    await flushPromises()
    expect(probeButtons()[1].attributes('disabled')).toBeUndefined()
  })
})

import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, patch } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, post, patch } }))

import { createModel, getTask, listModels, listTasks, listUpstreamRequests, updateModel } from '@/api/admin/mediaPlaygroundVideo'
import type { MediaPlaygroundVideoModelPayload } from '@/api/admin/mediaPlaygroundVideo'

const payload: MediaPlaygroundVideoModelPayload = {
  display_name: 'Video',
  model: 'video-1',
  provider_name: 'custom',
  api_mode: 'openai_videos',
  upstream_base_url: 'https://example.com',
  upstream_api_key: '',
  price_quota: 1,
  billing_mode: 'balance_prepaid',
  refund_enabled: true,
  timeout_seconds: 60,
  enabled: true,
  sort_order: 0,
}

describe('admin media playground video api', () => {
  beforeEach(() => vi.clearAllMocks())

  it('uses the unified media management paths', async () => {
    get.mockResolvedValue({ data: [] })
    await listModels()
    expect(get).toHaveBeenCalledWith('/admin/media-playground/video/models')

    const page = { items: [], total: 0, page: 2, page_size: 20 }
    get.mockResolvedValue({ data: page })
    await listUpstreamRequests({ page: 2, page_size: 20 })
    expect(get).toHaveBeenCalledWith('/admin/media-playground/video/upstream-requests', { params: { page: 2, page_size: 20 } })

    await listTasks({ page: 1, page_size: 20, status: 'failed' })
    expect(get).toHaveBeenCalledWith('/admin/media-playground/video/tasks', { params: { page: 1, page_size: 20, status: 'failed' } })
    await getTask('task-1')
    expect(get).toHaveBeenCalledWith('/admin/media-playground/video/tasks/task-1')
  })

  it('forwards api_mode on create and update', async () => {
    post.mockResolvedValue({ data: { id: 1 } })
    patch.mockResolvedValue({ data: { id: 1 } })
    await createModel(payload)
    await updateModel(1, { ...payload, api_mode: 'seedance_content_generation' })
    expect(post).toHaveBeenCalledWith('/admin/media-playground/video/models', payload)
    expect(patch).toHaveBeenCalledWith('/admin/media-playground/video/models/1', expect.objectContaining({ api_mode: 'seedance_content_generation' }))
  })

  it('forwards the OpenAI Videos API2 mode', async () => {
    await createModel({ ...payload, api_mode: 'openai_videos_v2' })
    expect(post).toHaveBeenCalledWith('/admin/media-playground/video/models', expect.objectContaining({ api_mode: 'openai_videos_v2' }))
  })
})

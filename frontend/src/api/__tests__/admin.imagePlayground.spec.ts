import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { listProbeRuns, runModelProbe, runProbe } from '@/api/admin/imagePlayground'

describe('admin image playground api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('requests probe runs with pagination params', async () => {
    const page = { items: [], total: 0, page: 2, page_size: 20 }
    get.mockResolvedValue({ data: page })

    const result = await listProbeRuns({ page: 2, page_size: 20 })

    expect(get).toHaveBeenCalledWith('/admin/image-playground/model-probe-runs', {
      params: { page: 2, page_size: 20 },
    })
    expect(result).toEqual(page)
  })

  it('posts the manual probe endpoint', async () => {
    post.mockResolvedValue({ data: { ok: true, running: true } })

    const result = await runProbe()

    expect(post).toHaveBeenCalledWith('/admin/image-playground/model-probe-runs/run')
    expect(result).toEqual({ ok: true, running: true })
  })

  it('posts the single model probe endpoint', async () => {
    post.mockResolvedValue({ data: { ok: true, running: true } })

    const result = await runModelProbe(7)

    expect(post).toHaveBeenCalledWith('/admin/image-playground/models/7/probe')
    expect(result).toEqual({ ok: true, running: true })
  })
})

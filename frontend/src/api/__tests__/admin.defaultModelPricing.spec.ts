import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { get } }))

import { list } from '@/api/admin/defaultModelPricing'

describe('admin default model pricing api', () => {
  beforeEach(() => get.mockReset())

  it('passes typed filters, sorting, pagination, and abort signal', async () => {
    const controller = new AbortController()
    const response = {
      items: [], total: 0, page: 2, page_size: 50,
      providers: ['openai'], modes: ['chat'],
      status: { model_count: 196, last_updated: null, local_hash: '1234abcd' },
    }
    get.mockResolvedValue({ data: response })

    const result = await list({
      page: 2, page_size: 50, search: 'gpt', provider: 'openai', mode: 'chat',
      sort_by: 'provider', sort_order: 'desc', signal: controller.signal,
    })

    expect(get).toHaveBeenCalledWith('/admin/channels/default-model-pricing', {
      params: {
        page: 2, page_size: 50, search: 'gpt', provider: 'openai', mode: 'chat',
        sort_by: 'provider', sort_order: 'desc',
      },
      signal: controller.signal,
    })
    expect(result).toEqual(response)
  })
})

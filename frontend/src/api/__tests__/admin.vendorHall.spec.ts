import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiClient } from '@/api/client'
import vendorHallAPI from '@/api/admin/vendorHall'

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

describe('admin vendor hall API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('forwards list filters to the monitor-backed endpoint', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      data: {
        items: [{
          account_id: 7,
          account_name: 'OpenAI East',
          platform: 'openai',
          group_name: 'premium',
          rate_multiplier: 1.2,
          scheduling_status: 'schedulable',
          availability: 0.99,
          cache_hit_rate: 0.5,
          average_latency_ms: 800,
          p95_latency_ms: 1200,
          user_ttft_p95_ms: 500,
          request_count: 10,
          collected_at: '2026-08-17T12:00:00Z',
          temp_unschedulable_until: null,
          trend: [{ timestamp: '2026-08-17T12:00:00Z', ttft_p95_ms: 500 }],
        }],
        total: 1,
        page: 1,
        page_size: 20,
        summary: { total_accounts: 1, healthy_accounts: 1, paused_accounts: 0, average_availability: 0.99, updated_at: '2026-08-17T12:00:00Z' },
      },
    })

    const result = await vendorHallAPI.list({ window: '3h', search: 'openai', sort_by: 'user_ttft', sort_order: 'desc' })

    expect(apiClient.get).toHaveBeenCalledWith('/admin/ops/vendor-hall', {
      params: { window: '3h', search: 'openai', sort_by: 'user_ttft', sort_order: 'desc' },
    })
    expect(result.items[0]).toEqual(expect.objectContaining({ account_name: 'OpenAI East', group_name: 'premium', user_ttft_p95_ms: 500, scheduling_status: 'schedulable' }))
    expect(result.items[0].trend).toEqual([{ timestamp: '2026-08-17T12:00:00Z', ttft_p95_ms: 500 }])
  })

  it('pauses one account through the administrator endpoint', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      data: { account_id: 7, temp_unschedulable_until: '2026-08-17T13:00:00Z' },
    })

    const result = await vendorHallAPI.pauseScheduling(7)

    expect(apiClient.post).toHaveBeenCalledWith('/admin/accounts/7/pause-scheduling')
    expect(result.account_id).toBe(7)
  })
})

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PublicStatsStrip from '@/components/common/PublicStatsStrip.vue'

const { getPublicStatsMock, localeRef } = vi.hoisted(() => ({
  getPublicStatsMock: vi.fn(),
  localeRef: { value: 'zh-CN' }
}))

vi.mock('@/api/publicStats', () => ({
  getPublicStats: (...args: any[]) => getPublicStatsMock(...args)
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: localeRef })
}))

const statsFixture = {
  running_since: '2026-05-07',
  running_days: 168,
  total_requests: 2853851,
  total_tokens: 318729031558,
  availability_30d: 0.9404,
  availability_window_days: 30,
  updated_at: '2026-10-22T12:00:00Z'
}

function mountStrip() {
  return mount(PublicStatsStrip)
}

describe('PublicStatsStrip', () => {
  beforeEach(() => {
    getPublicStatsMock.mockReset()
    localeRef.value = 'zh-CN'
  })

  it('shows placeholders while loading', () => {
    getPublicStatsMock.mockReturnValue(new Promise(() => {}))
    const wrapper = mountStrip()

    expect(wrapper.find('[data-testid="public-stats"]').attributes('aria-busy')).toBe('true')
    expect(wrapper.text()).toContain('--')
  })

  it('renders zh compact values', async () => {
    getPublicStatsMock.mockResolvedValue(statsFixture)
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.text()).toContain('运行天数')
    expect(wrapper.text()).toContain('168')
    expect(wrapper.text()).toContain('285万+')
    expect(wrapper.text()).toContain('3187亿+')
    expect(wrapper.text()).toContain('94.0%')
    expect(wrapper.findAll('.public-stats-item')).toHaveLength(4)
    expect(wrapper.find('[data-testid="public-stats"]').attributes('aria-busy')).toBe('false')
    expect(wrapper.find('[aria-label="运行天数 168 天"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="累计调用量 285万+"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="近 30 天在线率 94.0%"]').exists()).toBe(true)
  })

  it('renders en compact values', async () => {
    localeRef.value = 'en-US'
    getPublicStatsMock.mockResolvedValue(statsFixture)
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.text()).toContain('Days in service')
    expect(wrapper.text()).toContain('2.9M+')
    expect(wrapper.text()).toContain('318.7B+')
    expect(wrapper.text()).toContain('94.0%')
  })

  it('hides the availability cell when the value is null', async () => {
    getPublicStatsMock.mockResolvedValue({ ...statsFixture, availability_30d: null })
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.findAll('.public-stats-item')).toHaveLength(3)
    expect(wrapper.text()).not.toContain('在线率')
  })

  it('hides the whole strip when the request fails', async () => {
    getPublicStatsMock.mockRejectedValue(new Error('network down'))
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(false)
  })

  it('hides the whole strip when the payload is invalid', async () => {
    getPublicStatsMock.mockResolvedValue({ running_days: 168 })
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(false)
  })

  it('hides the whole strip when both totals are zero', async () => {
    getPublicStatsMock.mockResolvedValue({
      ...statsFixture,
      total_requests: 0,
      total_tokens: 0
    })
    const wrapper = mountStrip()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(false)
  })
})

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AuthLayout from '@/components/layout/AuthLayout.vue'

const { getPublicStatsMock } = vi.hoisted(() => ({
  getPublicStatsMock: vi.fn()
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteName: '小逆AI',
    siteLogo: '',
    cachedPublicSettings: { site_subtitle: '稳定的大模型入口' },
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn()
  })
}))

vi.mock('@/api/publicStats', () => ({
  getPublicStats: (...args: any[]) => getPublicStatsMock(...args)
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: { value: 'zh-CN' } })
}))

const realMatchMedia = window.matchMedia

function stubMatchMedia() {
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn()
  })) as unknown as typeof window.matchMedia
}

function setPath(path: string) {
  window.history.pushState({}, '', path)
}

function mountLayout() {
  return mount(AuthLayout, {
    slots: {
      default: '<form data-testid="auth-form">form</form>'
    }
  })
}

describe('AuthLayout public stats routing', () => {
  beforeEach(() => {
    stubMatchMedia()
    getPublicStatsMock.mockReset()
    getPublicStatsMock.mockResolvedValue({
      running_since: '2026-05-07',
      running_days: 168,
      total_requests: 2853851,
      total_tokens: 318729031558,
      availability_30d: 0.9404,
      availability_window_days: 30,
      updated_at: '2026-10-22T12:00:00Z'
    })
  })

  afterEach(() => {
    window.matchMedia = realMatchMedia
    setPath('/')
  })

  it('renders the strip on the login page', async () => {
    setPath('/login')
    const wrapper = mountLayout()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('285万+')
  })

  it('renders the strip on the register page', async () => {
    setPath('/register')
    const wrapper = mountLayout()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(true)
  })

  it('does not render the strip on other auth pages', async () => {
    setPath('/forgot-password')
    const wrapper = mountLayout()
    await flushPromises()

    expect(wrapper.find('[data-testid="public-stats"]').exists()).toBe(false)
    expect(getPublicStatsMock).not.toHaveBeenCalled()
  })
})

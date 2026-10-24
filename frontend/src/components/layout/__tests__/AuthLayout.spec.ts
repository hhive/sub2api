import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import AuthLayout from '@/components/layout/AuthLayout.vue'

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteName: '蛋蛋下游中转',
    siteLogo: '',
    cachedPublicSettings: { site_subtitle: '大模型、图片、视频顶级上游' },
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn()
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const realMatchMedia = window.matchMedia

function stubMatchMedia(matches: boolean) {
  window.matchMedia = ((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn()
  })) as unknown as typeof window.matchMedia
}

function mountLayout() {
  return mount(AuthLayout, {
    slots: {
      default: '<form data-testid="auth-form">form</form>',
      footer: '<a data-testid="auth-footer-link">footer</a>'
    }
  })
}

afterEach(() => {
  window.matchMedia = realMatchMedia
})

describe('AuthLayout tech backdrop', () => {
  it('renders the dark tech layers and keeps slots reachable', () => {
    stubMatchMedia(false)
    const wrapper = mountLayout()

    expect(wrapper.find('[data-testid="auth-tech-bg"]').exists()).toBe(true)
    expect(wrapper.findAll('.auth-aurora').length).toBe(3)
    expect(wrapper.find('.auth-grid').exists()).toBe(true)
    expect(wrapper.find('.auth-scan').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-pointer-glow"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-footer-link"]').exists()).toBe(true)
  })

  it('keeps brand, subtitle and copyright sourced from settings', () => {
    stubMatchMedia(false)
    const wrapper = mountLayout()

    expect(wrapper.text()).toContain('蛋蛋下游中转')
    expect(wrapper.text()).toContain('大模型、图片、视频顶级上游')
    expect(wrapper.text()).toContain(String(new Date().getFullYear()))
    expect(wrapper.find('img').attributes('src')).toContain('logo.svg')
  })

  it('renders the immersive hero column alongside the form slot', () => {
    stubMatchMedia(false)
    const wrapper = mountLayout()

    expect(wrapper.find('[data-testid="auth-hero"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('auth.hero.titlePrimary')
    expect(wrapper.text()).toContain('auth.hero.titleAccent')
    expect(wrapper.find('[data-testid="hero-proof-strip"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="hero-proof-strip"]').findAll('li').length).toBe(3)
    expect(wrapper.find('[data-testid="auth-form"]').exists()).toBe(true)
  })

  it('tracks pointer movement into the glow variables', async () => {
    stubMatchMedia(false)
    const wrapper = mountLayout()
    const glow = wrapper.find('[data-testid="auth-pointer-glow"]')

    window.dispatchEvent(new MouseEvent('pointermove', { clientX: 512, clientY: 384 }))
    await wrapper.vm.$nextTick()

    const style = (glow.element as HTMLElement).style
    expect(style.getPropertyValue('--mx')).toMatch(/%$/)
    expect(style.getPropertyValue('--my')).toMatch(/%$/)
  })

  it('mounts safely when matchMedia is unavailable', () => {
    // @ts-expect-error 模拟不支持 matchMedia 的旧环境
    delete window.matchMedia

    const wrapper = mountLayout()
    expect(wrapper.find('[data-testid="auth-tech-bg"]').exists()).toBe(true)
  })
})

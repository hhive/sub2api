import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
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
    useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key })
  }
})

function mountLayout() {
  return mount(AuthLayout, {
    slots: {
      default: '<form data-testid="auth-form">form</form>',
      footer: '<a data-testid="auth-footer-link">footer</a>'
    },
    global: {
      stubs: {
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        OrbitStage: { template: '<div data-testid="orbit-stage" />' }
      }
    }
  })
}

describe('AuthLayout centered orbit shell', () => {
  it('renders the globe backdrop and a centered card holding the slot', () => {
    const wrapper = mountLayout()

    expect(wrapper.classes()).toContain('dark')
    expect(wrapper.find('[data-testid="orbit-stage"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-form"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="auth-footer-link"]').exists()).toBe(true)
  })

  it('keeps brand and copyright sourced from settings', () => {
    const wrapper = mountLayout()

    expect(wrapper.text()).toContain('蛋蛋下游中转')
    expect(wrapper.text()).toContain(String(new Date().getFullYear()))
    expect(wrapper.find('img').attributes('src')).toContain('logo.svg')
  })

  it('drops the legacy split hero column', () => {
    const wrapper = mountLayout()

    expect(wrapper.find('[data-testid="auth-hero"]').exists()).toBe(false)
    expect(wrapper.findAll('.auth-orbit-ring').length).toBe(0)
  })

  it('renders the locale switcher in the top bar', () => {
    const wrapper = mountLayout()

    expect(wrapper.find('[data-testid="locale-switcher"]').exists()).toBe(true)
  })
})

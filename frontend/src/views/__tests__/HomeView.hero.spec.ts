import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings,
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
      },
    },
  })
}

function ctaDestination(wrapper: ReturnType<typeof mountHome>, testId: string) {
  const link = wrapper.find(`[data-testid="${testId}"]`)
  if (!link.exists()) return undefined
  return link.findComponent(RouterLinkStub).props('to')
}

describe('HomeView immersive hero', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    localStorage.clear()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('renders the hero headline, subtitle and three proof items', () => {
    const wrapper = mountHome()

    expect(wrapper.text()).toContain('home.hero.titlePrimary')
    expect(wrapper.text()).toContain('home.hero.titleAccent')
    expect(wrapper.text()).toContain('home.hero.subtitle')
    expect(wrapper.get('[data-testid="hero-proof-strip"]').findAll('li').length).toBe(3)
  })

  it('renders the aurora backdrop instead of the terminal mock', () => {
    const wrapper = mountHome()

    expect(wrapper.find('[data-testid="auth-tech-bg"]').exists()).toBe(true)
    expect(wrapper.find('.terminal-container').exists()).toBe(false)
  })

  it('points the primary call to action at login for anonymous visitors', () => {
    expect(ctaDestination(mountHome(), 'home-primary-cta')).toBe('/login')
  })

  it('points the primary call to action at the user dashboard when signed in', () => {
    authStore.isAuthenticated = true

    expect(ctaDestination(mountHome(), 'home-primary-cta')).toBe('/dashboard')
  })

  it('points the primary call to action at the admin dashboard for administrators', () => {
    authStore.isAuthenticated = true
    authStore.isAdmin = true

    expect(ctaDestination(mountHome(), 'home-primary-cta')).toBe('/admin/dashboard')
  })

  it('shows the secondary call to action when the public model plaza is reachable', () => {
    const wrapper = mountHome({ model_plaza_enabled: true, model_plaza_require_auth: false })

    expect(ctaDestination(wrapper, 'home-secondary-cta')).toBe('/model-plaza')
  })

  it('hides the secondary call to action when the model plaza feature is disabled', () => {
    const wrapper = mountHome({ model_plaza_enabled: false, model_plaza_require_auth: false })

    expect(wrapper.find('[data-testid="home-secondary-cta"]').exists()).toBe(false)
  })

  it('keeps custom home content ahead of the immersive hero', () => {
    const wrapper = mountHome({ home_content: '<section id="custom-home">Custom home</section>' })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="home-primary-cta"]').exists()).toBe(false)
  })
})

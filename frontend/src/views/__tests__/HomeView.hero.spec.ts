import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'
import HomeOrbitHero from '@/components/home/HomeOrbitHero.vue'

enableAutoUnmount(afterEach)

const { appStore, authStore } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn()
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

const SLIDES = {
  s1: {
    prefix: 'One key',
    verb: 'to every AI',
    subtitle: 'first subtitle',
    left: { kicker: 'Access', title: 'Left one', desc: 'left desc one' },
    right: { kicker: 'Billing', title: 'Right one', desc: 'right desc one' }
  },
  s2: {
    prefix: 'Model plaza',
    verb: 'prices at a glance',
    subtitle: 'second subtitle',
    left: { kicker: 'Catalog', title: 'Left two', desc: 'left desc two' },
    right: { kicker: 'Pricing', title: 'Right two', desc: 'right desc two' }
  }
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en' },
      t: (key: string) => key,
      tm: (key: string) => (key === 'home.hero.slides' ? SLIDES : []),
      rt: (value: unknown) => value
    })
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' }
      }
    }
  })
}

function ctaDestination(wrapper: ReturnType<typeof mountHome>, testId: string) {
  const link = wrapper.find(`[data-testid="${testId}"]`)
  if (!link.exists()) return undefined
  return link.findComponent(RouterLinkStub).props('to')
}

function defaultMatchMedia(matches = false) {
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (query: string) =>
      ({
        matches,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn()
      }) as unknown as MediaQueryList
  )
}

function reducedMotionMatchMedia(): void {
  vi.restoreAllMocks()
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (query: string) =>
      ({
        matches: query.includes('prefers-reduced-motion'),
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn()
      }) as unknown as MediaQueryList
  )
  // jsdom 没有 WebGL/Canvas 上下文，OrbitStage 走静态降级
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
}

const HERO_SLIDES = Object.keys(SLIDES)
  .sort()
  .map((key) => SLIDES[key as keyof typeof SLIDES])

const FIVE_SLIDES = [1, 2, 3, 4, 5].map((n) => ({
  prefix: `Prefix ${n}`,
  verb: `Verb ${n}`,
  subtitle: `Subtitle ${n}`,
  left: { kicker: `Kicker ${n}`, title: `Left ${n}`, desc: `Left desc ${n}` },
  right: { kicker: `Kicker ${n}`, title: `Right ${n}`, desc: `Right desc ${n}` }
}))

function mountHero(intervalMs = 6200, slides = HERO_SLIDES, siteSubtitle = '') {
  return mount(HomeOrbitHero, {
    props: {
      slides,
      isAuthenticated: false,
      dashboardPath: '/dashboard',
      showModelPlazaEntry: false,
      siteSubtitle,
      intervalMs
    },
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        transition: { template: '<div><slot /></div>' }
      }
    }
  })
}

describe('HomeView immersive hero', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    appStore.showSuccess.mockClear()
    appStore.showWarning.mockClear()
    localStorage.clear()
    defaultMatchMedia(false)
    // jsdom 没有 WebGL/Canvas 上下文，OrbitStage 走静态降级
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the orbit hero stage instead of the legacy terminal hero', () => {
    const wrapper = mountHome()

    expect(wrapper.find('[data-testid="home-orbit-hero"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="orbit-stage"]').exists()).toBe(true)
    expect(wrapper.find('.terminal-container').exists()).toBe(false)
  })

  it('exposes the full first headline through the accessible label', () => {
    const wrapper = mountHome()

    expect(wrapper.get('[data-testid="orbit-title"]').attributes('aria-label')).toBe(
      'One key to every AI'
    )
  })

  it('numbers the carousel and follows the hero slide changes', async () => {
    const wrapper = mountHome()
    const counter = () => wrapper.get('[data-testid="home-slide-counter"]').text().replace(/\s+/g, ' ')

    expect(counter()).toBe('01/ 02')

    wrapper.findComponent(HomeOrbitHero).vm.$emit('slide-change', 1)
    await nextTick()

    expect(counter()).toBe('02/ 02')
  })

  it('rotates the carousel to the next slide and swaps the focus cards', async () => {
    vi.useFakeTimers()
    const wrapper = mountHero()
    await nextTick()

    expect(wrapper.get('[data-testid="orbit-focus-left"]').text()).toContain('Left one')
    expect(wrapper.get('[data-testid="orbit-focus-right"]').text()).toContain('Right one')

    await vi.advanceTimersByTimeAsync(8000)
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([1])
    expect(wrapper.get('[data-testid="orbit-focus-left"]').text()).toContain('Left two')
    expect(wrapper.get('[data-testid="orbit-focus-right"]').text()).toContain('Right two')

    vi.useRealTimers()
  })

  it('reveals the whole headline immediately when reduced motion is preferred', async () => {
    reducedMotionMatchMedia()

    const wrapper = mountHome()
    await nextTick()

    const titleText = wrapper.get('[data-testid="orbit-title"]').text()
    expect(titleText).toContain('One key')
    expect(titleText).toContain('to every AI')
  })

  it('advances the carousel after the hold duration', async () => {
    vi.useFakeTimers()
    const wrapper = mountHero()
    await nextTick()

    expect(wrapper.find('[data-testid="hero-carousel-prev"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="hero-carousel-next"]').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(8000)
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([1])

    vi.useRealTimers()
  })

  it('moves to the next headline manually when reduced motion is preferred', async () => {
    reducedMotionMatchMedia()
    vi.useFakeTimers()

    const wrapper = mountHero()
    await nextTick()

    expect(wrapper.get('[data-testid="orbit-title"]').attributes('aria-label')).toBe(
      'One key to every AI'
    )

    await wrapper.get('[data-testid="hero-carousel-next"]').trigger('click')
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([1])
    expect(wrapper.get('[data-testid="orbit-focus-left"]').text()).toContain('Left two')

    await vi.advanceTimersByTimeAsync(20000)
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([1])

    vi.useRealTimers()
  })

  it('wraps to the last headline when stepping back from the first', async () => {
    const wrapper = mountHero()
    await nextTick()

    await wrapper.get('[data-testid="hero-carousel-prev"]').trigger('click')
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([1])
    expect(wrapper.get('[data-testid="orbit-focus-left"]').text()).toContain('Left two')
  })

  it('reaches all five headlines with the manual controls when reduced motion is preferred', async () => {
    reducedMotionMatchMedia()
    vi.useFakeTimers()

    const wrapper = mountHero(6200, FIVE_SLIDES)
    await nextTick()

    for (let step = 0; step < 4; step++) {
      await wrapper.get('[data-testid="hero-carousel-next"]').trigger('click')
      await nextTick()
    }

    expect(wrapper.emitted('slide-change')?.map((event) => event[0])).toEqual([0, 1, 2, 3, 4])
    expect(wrapper.get('[data-testid="orbit-focus-left"]').text()).toContain('Left 5')

    await vi.advanceTimersByTimeAsync(20000)
    await nextTick()

    expect(wrapper.emitted('slide-change')?.at(-1)).toEqual([4])

    vi.useRealTimers()
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

  it('offers the contact entry only when a contact channel is configured', () => {
    expect(mountHome().find('[data-testid="home-contact-cta"]').exists()).toBe(false)

    const withContact = mountHome({ contact_info: 'support@example.com' })
    expect(withContact.find('[data-testid="home-contact-cta"]').exists()).toBe(true)
  })

  it('keeps custom home content ahead of the immersive hero', () => {
    const wrapper = mountHome({ home_content: '<section id="custom-home">Custom home</section>' })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="home-primary-cta"]').exists()).toBe(false)
  })

  it('renders the configured site subtitle under the hero headline', () => {
    const wrapper = mountHome()

    expect(wrapper.find('[data-testid="home-orbit-hero"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="orbit-site-subtitle"]').text()).toBe('Test subtitle')
  })

  it('hides the site subtitle when the setting is empty', () => {
    const wrapper = mountHome({ site_subtitle: '' })

    expect(wrapper.find('[data-testid="orbit-site-subtitle"]').exists()).toBe(false)
  })
})

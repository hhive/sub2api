import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'

import { useAppStore } from '@/stores/app'
import type { PublicSettings } from '@/types'
import { FeatureFlags, makeSidebarFlag } from '@/utils/featureFlags'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const appStoreSource = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../../stores/app.ts'), 'utf8')
const settingsViewSource = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../../views/admin/SettingsView.vue'), 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('lets the user collapse a group even while a child route is active', () => {
    // The expand state must come from the user's override first, falling back
    // to the active-route heuristic only when the user has not clicked yet.
    expect(componentSource).toContain('const groupExpandOverrides = ref<Map<string, boolean>>(new Map())')
    expect(componentSource).not.toContain('expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar chat menu wiring', () => {
  it('exposes only the LobeHub chat launch', () => {
    expect(componentSource).toContain('launchLobeHub,')
    expect(componentSource).toContain('launchLobeHub()')
    expect(componentSource).toContain('FeatureFlags.lobehub')
    expect(componentSource).toContain("const launchWindow = window.open('', '_blank')")
    expect(componentSource).toContain('launchWindow.opener = null')
    expect(componentSource).toContain('launchWindow.location.href = result.redirect_url')
    expect(componentSource).not.toContain("t('nav.chat')")
  })
})

describe('AppSidebar media playground menu wiring', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('resolves the media playground flag from missing and explicit public settings', () => {
    const appStore = useAppStore()
    const mediaPlaygroundEnabled = makeSidebarFlag(FeatureFlags.mediaPlayground)

    appStore.cachedPublicSettings = {} as PublicSettings
    expect(mediaPlaygroundEnabled()).toBe(true)

    appStore.cachedPublicSettings = { media_playground_enabled: false } as PublicSettings
    expect(mediaPlaygroundEnabled()).toBe(false)
  })

  it('adds an authenticated launch action next to the chat menu', () => {
    expect(componentSource).toContain("action?: 'lobehub' | 'mediaPlayground' | 'victoryMenu'")
    expect(componentSource).toContain("label: t('nav.mediaPlayground')")
    expect(componentSource).toContain("action: 'mediaPlayground'")
    expect(componentSource).toContain('handleMediaPlaygroundLaunch')
    expect(componentSource).toContain('launchMediaPlayground()')
    expect(componentSource).toContain("handleMenuItemClick('__media_playground__')")
  })

  it('pre-opens a window from the click event so browsers do not block the launch', () => {
    const launchBlock = componentSource.match(/async function handleMediaPlaygroundLaunch\(\) \{[\s\S]*?\n\}/)?.[0] ?? ''

    expect(launchBlock).toContain("const launchWindow = window.open('', '_blank')")
    expect(launchBlock).toContain('launchWindow.opener = null')
    expect(launchBlock).toContain('launchWindow.location.href = result.redirect_url')
    expect(launchBlock).toContain('launchWindow?.close()')
    expect(launchBlock).not.toContain("window.open(result.redirect_url, '_blank', 'noopener')")
  })

  it('labels the user launch entry without retaining legacy admin labels', () => {
    const zhLocale = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/zh/common.ts'), 'utf8')
    expect(zhLocale).toContain("mediaPlayground: '图片与视频'")
    expect(zhLocale).not.toContain('mediaPlaygroundImageConfig')
    expect(zhLocale).not.toContain('videoPlaygroundConfig')
  })
})

describe('AppSidebar video playground menu retirement', () => {
  it('removes the old internal media administration entries and standalone user video launch', () => {
    expect(componentSource).not.toContain("path: '/admin/media-playground/image'")
    expect(componentSource).not.toContain("path: '/admin/media-playground/video'")
    expect(componentSource).not.toContain("path: '__video_playground__'")
    expect(componentSource).not.toContain("action: 'videoPlayground'")
    expect(componentSource).not.toContain('launchVideoPlayground()')
  })
})

describe('AppSidebar other operations menu', () => {
  it('places generic external apps after the two built-in admin tools', () => {
    const adminNavBlock = componentSource.match(/const adminNavItems = computed[\s\S]*?return visible\n}\)/)?.[0] ?? ''
    const groupStart = adminNavBlock.indexOf('const otherOperationsItem')
    const groupEnd = adminNavBlock.indexOf('\n  }', groupStart)
    const groupBlock = adminNavBlock.slice(groupStart, groupEnd)

    expect(adminNavBlock.indexOf('visible.push(otherOperationsItem)')).toBeGreaterThan(adminNavBlock.indexOf("visible.push({ path: '/admin/settings'"))
    expect(adminNavBlock.lastIndexOf('for (const cm of customMenuItemsForAdmin.value)')).toBeGreaterThan(adminNavBlock.indexOf('visible.push(otherOperationsItem)'))
    expect(groupBlock.indexOf("path: '/admin/channels/default-pricing'")).toBeLessThan(groupBlock.indexOf("path: '/admin/balance-credits'"))
    expect(groupBlock.indexOf("path: '/admin/balance-credits'")).toBeLessThan(groupBlock.indexOf('...adminExternalAppNavItems.value'))
    expect(adminNavBlock.match(/path: '\/admin\/balance-credits'/g)).toHaveLength(1)
    expect(adminNavBlock).not.toContain("path: '/admin/media-playground/image'")
    expect(adminNavBlock).not.toContain("path: '/admin/media-playground/video'")
    expect(adminNavBlock.slice(adminNavBlock.indexOf('if (authStore.isSimpleMode)'), adminNavBlock.indexOf('visible.push(otherOperationsItem)'))).not.toContain('filtered.push(otherOperationsItem)')
  })

  it('loads enabled external apps only for administrators and maps labels by locale', () => {
    expect(componentSource).toContain('listAdminExternalApps()')
    expect(componentSource).toContain('if (!isAdmin.value)')
    expect(componentSource).toContain('.filter((app) => app.enabled)')
    expect(componentSource).toContain("locale.value.startsWith('zh') ? app.label_zh : app.label_en")
    expect(componentSource).toContain("action: 'adminExternalApp'")
    expect(componentSource).toContain('adminExternalAppID: app.app_id')
  })

  it('uses per-app loading guards and the secure pre-open helper', () => {
    expect(componentSource).toContain('adminExternalAppLaunching.value[appID]')
    expect(componentSource).toContain('openAdminExternalAppWindow(() => launchAdminExternalApp(appID))')
    expect(componentSource).toContain("return t('adminExternalApps.opening')")
    expect(componentSource).toContain("t('adminExternalApps.openFailed')")
  })
})

describe('AppSidebar LobeHub menu wiring', () => {
  it('places the sole chat entry before the unified media entry', () => {
    expect(componentSource).toContain('FeatureFlags.lobehub')
    expect(componentSource).toContain("action: 'lobehub'")
    expect(componentSource).toContain('handleLobeHubLaunch')
    expect(componentSource).toContain('launchLobeHub()')
    expect(componentSource).toContain("handleMenuItemClick('__lobehub__')")
    expect(componentSource).toContain("path: '__lobehub__'")

    const selfNavBlock = componentSource.match(/function buildSelfNavItems[\s\S]*?return items\n}/)?.[0] ?? ''
    const adminNavBlock = componentSource.match(/const adminNavItems = computed[\s\S]*?return visible\n}\)/)?.[0] ?? ''
    expect(selfNavBlock).toContain("path: '__lobehub__'")
    expect(selfNavBlock.indexOf("path: '__lobehub__'")).toBeLessThan(selfNavBlock.indexOf("path: '__media_playground__'"))
    expect(selfNavBlock.match(/path: '__lobehub__'/g)).toHaveLength(1)
    expect(adminNavBlock).not.toContain("path: '__lobehub__'")
  })
})

describe('AppSidebar purchase menu wiring', () => {
  it('keeps the recharge subscription shop entry behind a system setting', () => {
    expect(componentSource).toContain('FeatureFlags.purchaseSubscription')
    expect(componentSource).toContain('purchaseSubscriptionUrl')
    expect(componentSource).toContain('externalUrl?: string')
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain("path: '__purchase_external__'")
    expect(componentSource).toContain("label: t('nav.buySubscription')")
    expect(componentSource).toContain('featureFlag: flagRechargeSubscription')
    expect(componentSource).not.toContain("externalUrl: 'https://pay.ldxp.cn/shop/xiaoni-ai'")
  })
})

describe('AppSidebar victory menu wiring', () => {
  it('adds configurable jump menus as top-level items at the end of the user menu', () => {
    const selfNavBlock = componentSource.match(/function buildSelfNavItems[\s\S]*?return items\n}/)?.[0] ?? ''

    expect(componentSource).toContain("label: 'Vibe论坛'")
    expect(componentSource).toContain("externalUrl: 'https://vibe.xiaoni-ai.top'")
    expect(selfNavBlock).not.toContain("path: '__victory_menu__'")
    expect(selfNavBlock).not.toContain("label: '旗开得胜'")
    expect(selfNavBlock).not.toContain('children: victoryMenuItems.value.map')
    expect(componentSource).not.toContain('defaultVictoryMenuItems')
    expect(componentSource).not.toContain("label: '小逆Offer'")
    expect(componentSource).not.toContain("url: 'https://offer.xiaoni-ai.top'")
    expect(appStoreSource).not.toContain("id: 'xiaoni-offer'")
    expect(settingsViewSource).not.toContain('id: "xiaoni-offer"')
    expect(selfNavBlock).toContain('...victoryMenuItems.value.map((item): NavItem => ({')
    expect(selfNavBlock.indexOf('...victoryMenuItems.value.map')).toBeGreaterThan(selfNavBlock.indexOf('...customMenuItemsForUser.value.map'))
    expect(componentSource).toContain('launchVictoryMenu(menuID)')
    expect(componentSource).toContain("action: item.carry_api_key ? 'victoryMenu' : undefined")
    expect(componentSource.match(/item\.action === 'lobehub' \|\| item\.action === 'mediaPlayground' \|\| item\.action === 'victoryMenu'/g)).toHaveLength(3)
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain('rel="noopener noreferrer"')
  })
})

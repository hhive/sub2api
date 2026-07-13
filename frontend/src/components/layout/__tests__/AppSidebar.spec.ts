import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
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
    expect(componentSource).toContain("import { launchMediaPlayground, launchLobeHub, launchVictoryMenu } from '@/api/launch'")
    expect(componentSource).toContain('FeatureFlags.lobehub')
    expect(componentSource).toContain("const launchWindow = window.open('', '_blank')")
    expect(componentSource).toContain('launchWindow.opener = null')
    expect(componentSource).toContain('launchWindow.location.href = result.redirect_url')
    expect(componentSource).not.toContain("t('nav.chat')")
  })
})

describe('AppSidebar media playground menu wiring', () => {
  it('adds an authenticated launch action next to the chat menu', () => {
    expect(componentSource).toContain("action?: 'lobehub' | 'mediaPlayground' | 'victoryMenu'")
    expect(componentSource).toContain("label: t('nav.mediaPlayground')")
    expect(componentSource).toContain("action: 'mediaPlayground'")
    expect(componentSource).toContain('handleMediaPlaygroundLaunch')
    expect(componentSource).toContain('launchMediaPlayground()')
    expect(componentSource).toContain("handleMenuItemClick('__media_playground__')")
  })

  it('labels the user launch entry as Infinite Canvas while keeping the admin model config separate', () => {
    const zhLocale = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/zh/common.ts'), 'utf8')
    expect(zhLocale).toContain("mediaPlayground: '图片与视频'")
    expect(zhLocale).toContain("mediaPlaygroundImageConfig: '媒体站图片模型'")
  })
})

describe('AppSidebar video playground menu retirement', () => {
  it('keeps video administration but removes the standalone user launch', () => {
    expect(componentSource).toContain("path: '/admin/media-playground/video'")
    expect(componentSource).not.toContain("path: '__video_playground__'")
    expect(componentSource).not.toContain("action: 'videoPlayground'")
    expect(componentSource).not.toContain('launchVideoPlayground()')
  })
})

describe('AppSidebar other operations menu', () => {
  it('places the four admin tools after settings without duplicate top-level entries', () => {
    const adminNavBlock = componentSource.match(/const adminNavItems = computed[\s\S]*?return visible\n}\)/)?.[0] ?? ''
    const groupStart = adminNavBlock.indexOf('const otherOperationsItem')
    const groupEnd = adminNavBlock.indexOf('\n  }', groupStart)
    const groupBlock = adminNavBlock.slice(groupStart, groupEnd)

    expect(adminNavBlock.indexOf('visible.push(otherOperationsItem)')).toBeGreaterThan(adminNavBlock.indexOf("visible.push({ path: '/admin/settings'"))
    expect(adminNavBlock.lastIndexOf('for (const cm of customMenuItemsForAdmin.value)')).toBeGreaterThan(adminNavBlock.indexOf('visible.push(otherOperationsItem)'))
    expect(groupBlock.indexOf("path: '/admin/channels/default-pricing'")).toBeLessThan(groupBlock.indexOf("path: '/admin/balance-credits'"))
    expect(groupBlock.indexOf("path: '/admin/balance-credits'")).toBeLessThan(groupBlock.indexOf("path: '/admin/media-playground/image'"))
    expect(groupBlock.indexOf("path: '/admin/media-playground/image'")).toBeLessThan(groupBlock.indexOf("path: '/admin/media-playground/video'"))
    expect(adminNavBlock.match(/path: '\/admin\/balance-credits'/g)).toHaveLength(1)
    expect(adminNavBlock.match(/path: '\/admin\/media-playground\/image'/g)).toHaveLength(1)
    expect(adminNavBlock.match(/path: '\/admin\/media-playground\/video'/g)).toHaveLength(1)
    expect(groupBlock.match(/hideInSimpleMode: true/g)).toHaveLength(5)
    expect(adminNavBlock.slice(adminNavBlock.indexOf('if (authStore.isSimpleMode)'), adminNavBlock.indexOf('visible.push(otherOperationsItem)'))).not.toContain('filtered.push(otherOperationsItem)')
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
  it('adds Vibe forum and configurable victory child launches to the user menu', () => {
    expect(componentSource).toContain("label: 'Vibe论坛'")
    expect(componentSource).toContain("externalUrl: 'https://vibe.xiaoni-ai.top'")
    expect(componentSource).toContain("label: '旗开得胜'")
    expect(componentSource).toContain("label: '小逆Offer'")
    expect(componentSource).toContain("url: 'https://offer.xiaoni-ai.top'")
    expect(componentSource).toContain('launchVictoryMenu(menuID)')
    expect(componentSource).toContain("action: item.carry_api_key ? 'victoryMenu' : undefined")
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain('rel="noopener noreferrer"')
  })
})

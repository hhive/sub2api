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

describe('AppSidebar Onyx menu wiring', () => {
  it('uses the backend launch endpoint and pre-opens the chat window synchronously', () => {
    expect(componentSource).toContain("import { launchImagePlayground, launchLobeHub, launchOnyx } from '@/api/onyx'")
    expect(componentSource).toContain('FeatureFlags.onyx')
    expect(componentSource).toContain('handleOnyxLaunch')
    expect(componentSource).toContain("const launchWindow = window.open('', '_blank')")
    expect(componentSource).toContain('launchWindow.opener = null')
    expect(componentSource).toContain('launchWindow.location.href = result.redirect_url')
    expect(componentSource).not.toContain("{ path: '/chat', label: t('nav.chat')")
  })
})

describe('AppSidebar image playground menu wiring', () => {
  it('adds an authenticated launch action next to the chat menu', () => {
    expect(componentSource).toContain("action?: 'lobehub' | 'onyx' | 'imagePlayground'")
    expect(componentSource).toContain("label: t('nav.imagePlayground')")
    expect(componentSource).toContain("action: 'imagePlayground'")
    expect(componentSource).toContain('handleImagePlaygroundLaunch')
    expect(componentSource).toContain('launchImagePlayground()')
    expect(componentSource).toContain("handleMenuItemClick('__image_playground__')")
  })
})

describe('AppSidebar LobeHub menu wiring', () => {
  it('places LobeHub in the left user menu above the original chat launch item', () => {
    expect(componentSource).toContain('FeatureFlags.lobehub')
    expect(componentSource).toContain("action: 'lobehub'")
    expect(componentSource).toContain('handleLobeHubLaunch')
    expect(componentSource).toContain('launchLobeHub()')
    expect(componentSource).toContain("handleMenuItemClick('__lobehub__')")
    expect(componentSource).toContain("path: '__lobehub__'")

    const selfNavBlock = componentSource.match(/function buildSelfNavItems[\s\S]*?return items\n}/)?.[0] ?? ''
    const adminNavBlock = componentSource.match(/const adminNavItems = computed[\s\S]*?return visible\n}\)/)?.[0] ?? ''
    expect(selfNavBlock).toContain("path: '__lobehub__'")
    expect(selfNavBlock.indexOf("path: '__lobehub__'")).toBeLessThan(selfNavBlock.indexOf("path: '__onyx__'"))
    expect(adminNavBlock).not.toContain("path: '__lobehub__'")
  })
})

describe('AppSidebar purchase menu wiring', () => {
  it('opens the recharge subscription shop in a new tab above redeem', () => {
    expect(componentSource).toContain('externalUrl?: string')
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain("path: '__purchase_external__'")
    expect(componentSource).toContain("label: t('nav.buySubscription')")
    expect(componentSource).toContain("externalUrl: 'https://pay.ldxp.cn/shop/xiaoni-ai'")
    expect(componentSource.indexOf("path: '__purchase_external__'")).toBeLessThan(
      componentSource.indexOf("path: '/redeem'")
    )
  })
})

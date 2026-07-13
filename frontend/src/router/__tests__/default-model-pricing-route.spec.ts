import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('default model pricing admin routes', () => {
  it('keeps the parent redirect and admin-only page metadata', () => {
    const parentStart = routerSource.indexOf("path: '/admin/other-operations'")
    const parentBlock = routerSource.slice(parentStart, routerSource.indexOf('\n  },', parentStart))
    expect(parentBlock).toContain("redirect: '/admin/channels/default-pricing'")
    expect(routerSource).toContain("path: '/admin/channels/default-pricing'")
    expect(routerSource).toContain("name: 'AdminDefaultModelPricing'")
    expect(routerSource).toContain("component: () => import('@/views/admin/DefaultModelPricingView.vue')")

    const routeStart = routerSource.indexOf("path: '/admin/channels/default-pricing'")
    const routeBlock = routerSource.slice(routeStart, routerSource.indexOf('\n  },', routeStart))
    expect(routeBlock).toContain('requiresAuth: true')
    expect(routeBlock).toContain('requiresAdmin: true')
  })
})

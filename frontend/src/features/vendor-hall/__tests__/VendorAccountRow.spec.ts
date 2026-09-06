import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../VendorAccountRow.vue'),
  'utf8',
)

describe('VendorAccountRow responsive metrics', () => {
  it('keeps average and P95 TTFT values with the trend in the collapsed account row', () => {
    const collapsedContent = source.split('<div v-if="expanded"')[0]

    expect(collapsedContent).toContain('<VendorTtftSparkline')
    expect(collapsedContent).toContain('account.user_ttft_average_ms')
    expect(collapsedContent).toContain('account.user_ttft_p95_ms')
    expect(collapsedContent).toContain("admin.vendorHall.metrics.userTtftAvg")
    expect(collapsedContent).toContain('account.balance_usd')
    expect(source).toContain("admin.vendorHall.metrics.balance")
  })

  it('keeps collapsed latency and TTFT metrics out of the expanded details', () => {
    const detailsContent = source.split('<div v-if="expanded"')[1]

    expect(detailsContent).not.toContain('account.average_latency_ms')
    expect(detailsContent).not.toContain('account.user_ttft_average_ms')
    expect(detailsContent).not.toContain('account.user_ttft_p95_ms')
    expect(detailsContent).toContain('account.request_count')
    expect(detailsContent).toContain('account.collected_at')
  })

  it('offers a clickable definition for every displayed metric', () => {
    expect(source).toContain('VendorMetricHelp')
    expect(source.match(/:description="t\('admin\.vendorHall\.help\./g)?.length).toBe(7)
  })

  it('uses a two-row horizontally scrollable metrics layout on narrow screens', () => {
    expect(source).toContain('class="vendor-account__metrics-scroll"')
    expect(source).toContain('overflow-x: auto')
    expect(source).toContain('grid-template-areas: "identity actions" "metrics metrics"')
    expect(source).toContain('grid-template-areas: "ttft latency availability cache balance multiplier"')
  })

  it('never hides the TTFT trend at responsive breakpoints', () => {
    expect(source).not.toMatch(/\.vendor-trend\s*\{[^}]*display:\s*none/)
  })

  it('opens the upstream homepage in a new tab when available', () => {
    expect(source).toContain('v-if="homepageUrl"')
    expect(source).toContain(':href="homepageUrl"')
    expect(source).toContain('target="_blank"')
    expect(source).toContain('rel="noopener noreferrer"')
  })
})

import { describe, expect, it } from 'vitest'
import { formatImagePrice, formatTokenPrice } from '../defaultModelPricingFormat'

describe('default model pricing format', () => {
  it('distinguishes missing, zero, and very small non-zero token prices', () => {
    expect(formatTokenPrice(null)).toBe('-')
    expect(formatTokenPrice(0)).toBe('$0 / 1M')
    expect(formatTokenPrice(0.000000000001)).not.toContain('$0 /')
  })

  it('distinguishes missing and zero image prices', () => {
    expect(formatImagePrice(null)).toBe('-')
    expect(formatImagePrice(0)).toBe('$0 / image')
  })
})

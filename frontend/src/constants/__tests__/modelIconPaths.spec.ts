import { describe, expect, it } from 'vitest'
import { getModelIcon, MODEL_ICON_DATA, resolveModelIconKey } from '../modelIconPaths'

describe('modelIconPaths', () => {
  it('resolves every bundled icon key to itself', () => {
    for (const key of Object.keys(MODEL_ICON_DATA)) {
      expect(resolveModelIconKey(key)).toBe(key)
    }
  })

  it('resolves vendor model names to their icon key', () => {
    expect(resolveModelIconKey('gpt-4o')).toBe('openai')
    expect(resolveModelIconKey('claude-sonnet-4-20250514')).toBe('claude')
    expect(resolveModelIconKey('gemini-2.5-pro')).toBe('gemini')
    expect(resolveModelIconKey('glm-4.5')).toBe('zhipu')
    expect(resolveModelIconKey('kimi-k2')).toBe('moonshot')
    expect(resolveModelIconKey('grok-4')).toBe('xai')
    expect(resolveModelIconKey('@cf/meta/llama-3-8b-instruct')).toBe('cloudflare')
  })

  it('returns null for unknown models', () => {
    expect(resolveModelIconKey('totally-unknown-model')).toBeNull()
    expect(resolveModelIconKey('')).toBeNull()
    expect(getModelIcon('totally-unknown-model')).toBeNull()
  })

  it('returns icon geometry for known models', () => {
    const icon = getModelIcon('openai')

    expect(icon?.color).toBe('#000000')
    expect(icon?.paths.length).toBeGreaterThan(0)
  })
})

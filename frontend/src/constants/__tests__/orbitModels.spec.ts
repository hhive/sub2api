import { describe, expect, it } from 'vitest'

import {
  DEFAULT_ORBIT_MODELS,
  MAX_ORBIT_MODELS,
  MAX_ORBIT_MODEL_LABEL_LENGTH,
  normalizeOrbitModels
} from '@/constants/orbitModels'

const EXCLUDED_VENDORS = ['hunyuan', 'wenxin', 'spark', 'yi']

describe('orbit model list', () => {
  it('ships a default roster without the excluded vendors', () => {
    expect(DEFAULT_ORBIT_MODELS).toHaveLength(20)

    const vendors = new Set(DEFAULT_ORBIT_MODELS.map((model) => model.vendor))
    for (const excluded of EXCLUDED_VENDORS) {
      expect(vendors.has(excluded)).toBe(false)
    }
    for (const required of ['openai', 'claude', 'gemini', 'xai', 'deepseek', 'qwen', 'zhipu', 'moonshot', 'doubao', 'minimax']) {
      expect(vendors.has(required)).toBe(true)
    }
  })

  it('falls back to the default roster for missing or unusable input', () => {
    expect(normalizeOrbitModels(undefined)).toEqual(DEFAULT_ORBIT_MODELS)
    expect(normalizeOrbitModels(null)).toEqual(DEFAULT_ORBIT_MODELS)
    expect(normalizeOrbitModels([])).toEqual(DEFAULT_ORBIT_MODELS)
    expect(normalizeOrbitModels('gpt-6')).toEqual(DEFAULT_ORBIT_MODELS)
    expect(normalizeOrbitModels([{ vendor: 'not-a-vendor', label: 'Nope' }])).toEqual(
      DEFAULT_ORBIT_MODELS
    )
  })

  it('keeps configured entries, trims labels and drops unusable ones', () => {
    const models = normalizeOrbitModels([
      { vendor: ' openai ', label: '  GPT-6 Astra  ' },
      { vendor: 'unknown-vendor', label: 'Nope' },
      { vendor: 'deepseek', label: '' },
      { vendor: 'hunyuan', label: 'Hunyuan T1' }
    ])

    expect(models).toEqual([
      { vendor: 'openai', label: 'GPT-6 Astra' },
      { vendor: 'deepseek', label: 'deepseek' }
    ])
  })

  it('limits label length and the number of entries', () => {
    const longLabel = 'L'.repeat(MAX_ORBIT_MODEL_LABEL_LENGTH + 12)
    const models = normalizeOrbitModels([
      { vendor: 'openai', label: longLabel },
      ...Array.from({ length: MAX_ORBIT_MODELS + 5 }, () => ({ vendor: 'claude', label: 'Opus' }))
    ])

    expect(models).toHaveLength(MAX_ORBIT_MODELS)
    expect(models[0].label).toHaveLength(MAX_ORBIT_MODEL_LABEL_LENGTH)
  })
})

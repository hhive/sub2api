import { describe, expect, it } from 'vitest'

import {
  ORBIT_TRACKS,
  buildOrbitPlacements,
  orbitPlacementGap,
  orbitPlacementRestGap,
  orbitPointOn
} from '@/components/home/orbitLayout'

/**
 * 间距按「屏幕中心距 ÷ 两颗卡的平均屏幕宽度」度量：
 * 1 表示刚好相切，小于 1 即画面上的卡面范围相交。
 */
const MIN_REST_GAP = 1.1
const MIN_ANY_GAP = 0.15

describe('orbit layout', () => {
  it('places one card per model, alternating tracks', () => {
    for (let count = 1; count <= 24; count++) {
      const placements = buildOrbitPlacements(count)
      expect(placements).toHaveLength(count)
      expect(placements.map((p) => p.track).join('')).toBe(
        Array.from({ length: count }, (_, i) => String(i % 2)).join('')
      )
    }
  })

  it('keeps every phase on the ring and every scale bounded', () => {
    for (const placement of buildOrbitPlacements(24)) {
      expect(placement.phase).toBeGreaterThanOrEqual(0)
      expect(placement.phase).toBeLessThan(1)
      expect(placement.scale).toBeGreaterThanOrEqual(0.82)
      expect(placement.scale).toBeLessThanOrEqual(1)
    }
  })

  it('is deterministic for the same count', () => {
    for (const count of [1, 7, 20, 24]) {
      expect(buildOrbitPlacements(count)).toEqual(buildOrbitPlacements(count))
    }
  })

  it('leaves a visible gap between cards at rest for every roster size', () => {
    for (let count = 2; count <= 24; count++) {
      const restGap = orbitPlacementRestGap(buildOrbitPlacements(count))
      expect(restGap, `count=${count}`).toBeGreaterThan(MIN_REST_GAP)
    }
  })

  it('never lets two cards collapse onto each other while dragging', () => {
    for (let count = 2; count <= 24; count++) {
      const gap = orbitPlacementGap(buildOrbitPlacements(count))
      expect(gap, `count=${count}`).toBeGreaterThanOrEqual(MIN_ANY_GAP)
    }
  })

  it('spreads the dense counts that used to stack in projection', () => {
    // 旧「按轨道等分相位」在 19/20/21/23 档的静止间距低到 0.038 世界单位，
    // 折合归一化间距约 0.2，画面上两颗卡几乎完全重合。
    for (const count of [19, 20, 21, 23]) {
      expect(orbitPlacementRestGap(buildOrbitPlacements(count))).toBeGreaterThan(1.5)
    }
  })

  it('returns an empty layout for an empty roster', () => {
    expect(buildOrbitPlacements(0)).toEqual([])
  })

  it('keeps the two rings at the configured radii', () => {
    expect(ORBIT_TRACKS.map((track) => track.radius)).toEqual([1.52, 1.68])
    const [x, y, z] = orbitPointOn(ORBIT_TRACKS[0], 0)
    expect(Math.hypot(x, y, z)).toBeCloseTo(ORBIT_TRACKS[0].radius, 5)
  })
})

import { describe, expect, it } from 'vitest'

import {
  ORBIT_TRACKS,
  buildOrbitPlacements,
  orbitPlacementGap,
  orbitPointOn
} from '@/components/home/orbitLayout'

/** 卡面基准宽 0.22，留出叠压余量后要求的最小中心距 */
const MIN_GAP = 0.13

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

  it('keeps cards apart across the drag angles for every roster size', () => {
    for (let count = 2; count <= 24; count++) {
      const gap = orbitPlacementGap(buildOrbitPlacements(count))
      expect(gap, `count=${count}`).toBeGreaterThanOrEqual(MIN_GAP)
    }
  })

  it('spreads the dense counts that used to stack in projection', () => {
    // 19/20/21/23 是旧「按轨道等分相位」时投影最挤的几档
    for (const count of [19, 20, 21, 23]) {
      expect(orbitPlacementGap(buildOrbitPlacements(count))).toBeGreaterThan(0.12)
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

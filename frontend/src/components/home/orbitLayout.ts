/**
 * 主视觉卫星的轨道几何与布点。
 *
 * 场景与测试共用同一份轨道参数、相位求解与间距度量：轨道几何只在这里定义一次，
 * useOrbitScene 负责把相位换算成 three 对象，测试则在纯函数层面校验间距下界。
 */

export interface OrbitTrackSpec {
  /** 轨道平面绕 Z 轴的倾角 */
  angle: number
  /** 轨道半径（世界单位） */
  radius: number
  /** 轨道线的基准不透明度 */
  alpha: number
}

export interface OrbitPlacement {
  track: 0 | 1
  /** 沿轨道的相位，取值 [0, 1) */
  phase: number
  /** 卡面相对基准尺寸的系数 */
  scale: number
}

export const ORBIT_TRACKS: readonly OrbitTrackSpec[] = [
  { angle: -0.27, radius: 1.52, alpha: 0.5 },
  { angle: 0.54, radius: 1.68, alpha: 0.34 }
]

/** 整组根节点的静态倾斜，与场景里 root.rotation 的固定分量一致 */
export const ORBIT_ROOT_TILT = { x: 0.07, z: -0.17 } as const

/** 相位候选粒度：1/32 圈，足够容纳上限 24 颗卡面又不至于让求解变慢 */
const PHASE_STEPS = 32
/** 评分覆盖的目标偏航角数量，对应拖拽可达的角度范围 */
const YAW_SAMPLES = 8
/** 局部搜索轮数；实测 4 轮已收敛，继续增加只增加启动耗时 */
const REFINE_PASSES = 4
/** 轨道平面内的纵向压扁系数与前后伸展系数，与场景绘制一致 */
const TRACK_SQUASH = 0.36
const TRACK_DEPTH = 0.93

export function orbitPointOn(
  track: OrbitTrackSpec,
  phase: number
): [number, number, number] {
  const a = phase * Math.PI * 2
  const px = Math.cos(a)
  const py = Math.sin(a) * TRACK_SQUASH
  const pz = Math.sin(a) * TRACK_DEPTH
  const ca = Math.cos(track.angle)
  const sa = Math.sin(track.angle)
  return [
    (px * ca - py * sa) * track.radius,
    (px * sa + py * ca) * track.radius,
    pz * track.radius
  ]
}

/** 场景 root.rotation 的欧拉顺序为 XYZ，即 R = Rx(tiltX) * Ry(yaw) * Rz(tiltZ) */
function projectAtYaw(track: OrbitTrackSpec, phase: number, yaw: number): [number, number] {
  const [x, y, z] = orbitPointOn(track, phase)

  const cz = Math.cos(ORBIT_ROOT_TILT.z)
  const sz = Math.sin(ORBIT_ROOT_TILT.z)
  const rx = x * cz - y * sz
  const ry = x * sz + y * cz

  const cy = Math.cos(yaw)
  const sy = Math.sin(yaw)
  const x2 = rx * cy + z * sy
  const z2 = -rx * sy + z * cy

  const cx = Math.cos(ORBIT_ROOT_TILT.x)
  const sx = Math.sin(ORBIT_ROOT_TILT.x)
  return [x2, ry * cx - z2 * sx]
}

const YAW_VALUES = Array.from({ length: YAW_SAMPLES }, (_, i) => (i * Math.PI) / YAW_SAMPLES)

/** 每个「轨道 × 相位档」在所有目标视角下的投影坐标，形状为 [track][step][x0,y0,x1,y1,...] */
let stepProjections: number[][][] | null = null

function projections(): number[][][] {
  if (stepProjections) return stepProjections
  stepProjections = ORBIT_TRACKS.map((track) => {
    const perTrack: number[][] = []
    for (let step = 0; step < PHASE_STEPS; step++) {
      const coords: number[] = []
      for (const yaw of YAW_VALUES) {
        const [px, py] = projectAtYaw(track, step / PHASE_STEPS, yaw)
        coords.push(px, py)
      }
      perTrack.push(coords)
    }
    return perTrack
  })
  return stepProjections
}

interface Chip {
  track: 0 | 1
  step: number
}

function gapOfProjections(a: readonly number[], b: readonly number[]): number {
  let min = Number.POSITIVE_INFINITY
  for (let i = 0; i < a.length; i += 2) {
    const d = Math.hypot(a[i] - b[i], a[i + 1] - b[i + 1])
    if (d < min) min = d
  }
  return min
}

/** 一组布点在所有目标视角下的最小投影间距；条数少于 2 时无意义，返回 Infinity */
export function orbitPlacementGap(placements: readonly OrbitPlacement[]): number {
  const coords = placements.map((placement) => {
    const flat: number[] = []
    for (const yaw of YAW_VALUES) {
      const [px, py] = projectAtYaw(ORBIT_TRACKS[placement.track], placement.phase, yaw)
      flat.push(px, py)
    }
    return flat
  })

  let min = Number.POSITIVE_INFINITY
  for (let i = 0; i < coords.length; i++) {
    for (let j = i + 1; j < coords.length; j++) {
      const d = gapOfProjections(coords[i], coords[j])
      if (d < min) min = d
    }
  }
  return min
}

function gapOfChips(chips: readonly Chip[]): number {
  const table = projections()
  let min = Number.POSITIVE_INFINITY
  for (let i = 0; i < chips.length; i++) {
    const a = table[chips[i].track][chips[i].step]
    for (let j = i + 1; j < chips.length; j++) {
      const d = gapOfProjections(a, table[chips[j].track][chips[j].step])
      if (d < min) min = d
    }
  }
  return min
}

/**
 * 把 count 颗卡面铺到两条轨道上：依次插入并重排相位，使多视角下的最小投影间距最大。
 * 轨道按序号奇偶交替，保证名单顺序在环上仍是交替分布。
 */
export function buildOrbitPlacements(count: number): OrbitPlacement[] {
  if (count <= 0) return []

  const chips: Chip[] = []

  for (let index = 0; index < count; index++) {
    const track = (index % 2) as 0 | 1
    let bestStep = 0
    let bestGap = -1

    for (let step = 0; step < PHASE_STEPS; step++) {
      if (chips.some((chip) => chip.track === track && chip.step === step)) continue
      chips.push({ track, step })
      const gap = gapOfChips(chips)
      chips.pop()
      if (gap > bestGap) {
        bestGap = gap
        bestStep = step
      }
    }

    chips.push({ track, step: bestStep })
  }

  for (let pass = 0; pass < REFINE_PASSES; pass++) {
    let moved = false
    for (const chip of chips) {
      const original = chip.step
      let bestStep = original
      let bestGap = gapOfChips(chips)

      for (let step = 0; step < PHASE_STEPS; step++) {
        if (step === original) continue
        if (chips.some((other) => other !== chip && other.track === chip.track && other.step === step)) {
          continue
        }
        chip.step = step
        const gap = gapOfChips(chips)
        if (gap > bestGap + 1e-9) {
          bestGap = gap
          bestStep = step
        }
      }

      chip.step = bestStep
      if (bestStep !== original) moved = true
    }
    if (!moved) break
  }

  return chips.map((chip, index) => ({
    track: chip.track,
    phase: chip.step / PHASE_STEPS,
    scale: 1 - 0.06 * (index % 4)
  }))
}

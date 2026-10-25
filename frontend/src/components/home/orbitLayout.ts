/**
 * 主视觉卫星的轨道几何与布点。
 *
 * 场景与测试共用同一份轨道参数、相位求解与间距度量：轨道几何只在这里定义一次，
 * useOrbitScene 负责把相位换算成 three 对象，测试则在纯函数层面校验间距。
 *
 * 间距按真实透视投影度量：每颗卡面在屏幕上的宽度随景深变化，比较的是
 * 「两颗卡的屏幕中心距 ÷ 两颗卡的平均屏幕宽度」，1 表示刚好相切。
 *
 * 布点目标：静止视角下卡面之间留出明显间隙（用户实际看到的就是这一帧），
 * 同时保证拖到任意偏航角时也不会重合成一颗。
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

/** 相机到原点的距离，取首页模式的取值 */
const CAMERA_Z = 4.75
/** 卡面在世界坐标下的基准宽度 */
const CARD_WIDTH = 0.22

/** 相位候选粒度：1/32 圈，足够容纳上限 24 颗卡面又不至于让求解变慢 */
const PHASE_STEPS = 24
/** 拖拽可达角度的采样数量，覆盖 [0, π) */
const YAW_SAMPLES = 16
/** 局部搜索轮数；实测 3 轮已收敛，继续增加只增加启动耗时 */
const REFINE_PASSES = 3
/**
 * 任意偏航角下的归一化间距下限：小于 1 表示两颗卡的屏幕范围已经相交。
 * 静止构图优先，这里只守住「拖动时不会重合成一颗」这个底线。
 */
const MIN_ANY_GAP = 0.9
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

interface Projection {
  /** 透视投影后的屏幕 x */
  x: number
  /** 透视投影后的屏幕 y */
  y: number
  /** 该景深下卡面屏幕宽度相对基准宽度的倍数 */
  widthFactor: number
}

/** 场景 root.rotation 的欧拉顺序为 XYZ，即 R = Rx(tiltX) * Ry(yaw) * Rz(tiltZ) */
function projectAtYaw(track: OrbitTrackSpec, phase: number, yaw: number): Projection {
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
  const y2 = ry * cx - z2 * sx
  const z3 = ry * sx + z2 * cx

  const depth = CAMERA_Z - z3
  const widthFactor = CAMERA_Z / depth
  return { x: x2 * widthFactor, y: y2 * widthFactor, widthFactor }
}

/** 采样角排序后第一项即静止视角，其余覆盖拖拽可达角度 */
const YAW_VALUES = Array.from({ length: YAW_SAMPLES }, (_, i) => (i * Math.PI) / YAW_SAMPLES)
const REST_YAW_INDEX = 0

interface Chip {
  track: 0 | 1
  step: number
  /** 卡面尺寸系数，决定该卡的真实屏幕宽度 */
  scale: number
}

/** 每个「轨道 × 相位档」在所有采样角下的投影，形状为 [track][step][x0,y0,w0,...] */
let stepProjections: number[][][] | null = null

function projections(): number[][][] {
  if (stepProjections) return stepProjections
  stepProjections = ORBIT_TRACKS.map((track) => {
    const perTrack: number[][] = []
    for (let step = 0; step < PHASE_STEPS; step++) {
      const coords: number[] = []
      for (const yaw of YAW_VALUES) {
        const { x, y, widthFactor } = projectAtYaw(track, step / PHASE_STEPS, yaw)
        coords.push(x, y, widthFactor)
      }
      perTrack.push(coords)
    }
    return perTrack
  })
  return stepProjections
}

interface Gaps {
  /** 所有采样角下的最小归一化间距 */
  all: number
  /** 静止视角下的最小归一化间距 */
  rest: number
}

function pairGaps(a: readonly number[], aScale: number, b: readonly number[], bScale: number): Gaps {
  let all = Number.POSITIVE_INFINITY
  for (let i = 0; i < a.length; i += 3) {
    const dx = a[i] - b[i]
    const dy = a[i + 1] - b[i + 1]
    const distance = Math.hypot(dx, dy)
    const width = (CARD_WIDTH * (a[i + 2] * aScale + b[i + 2] * bScale)) / 2
    const normalized = distance / width
    if (normalized < all) all = normalized
  }
  const ri = REST_YAW_INDEX * 3
  const restDistance = Math.hypot(a[ri] - b[ri], a[ri + 1] - b[ri + 1])
  const restWidth = (CARD_WIDTH * (a[ri + 2] * aScale + b[ri + 2] * bScale)) / 2
  return { all, rest: restDistance / restWidth }
}

function gapsOfChips(chips: readonly Chip[]): Gaps {
  const table = projections()
  let all = Number.POSITIVE_INFINITY
  let rest = Number.POSITIVE_INFINITY
  for (let i = 0; i < chips.length; i++) {
    const a = table[chips[i].track][chips[i].step]
    for (let j = i + 1; j < chips.length; j++) {
      const gaps = pairGaps(a, chips[i].scale, table[chips[j].track][chips[j].step], chips[j].scale)
      if (gaps.all < all) all = gaps.all
      if (gaps.rest < rest) rest = gaps.rest
    }
  }
  return { all, rest }
}

function scoreOfChips(chips: readonly Chip[]): number {
  const { all, rest } = gapsOfChips(chips)
  // 静止视角是用户实际看到的画面，作为主目标；全角度间距只做软约束，
  // 条数超出两条轨道的容量时优先保证首屏不叠卡。
  return rest + 0.35 * Math.min(all - MIN_ANY_GAP, 0)
}

function gapsOfPlacements(placements: readonly OrbitPlacement[]): Gaps {
  const coords = placements.map((placement) => {
    const flat: number[] = []
    for (const yaw of YAW_VALUES) {
      const { x, y, widthFactor } = projectAtYaw(
        ORBIT_TRACKS[placement.track],
        placement.phase,
        yaw
      )
      flat.push(x, y, widthFactor)
    }
    return flat
  })

  let all = Number.POSITIVE_INFINITY
  let rest = Number.POSITIVE_INFINITY
  for (let i = 0; i < coords.length; i++) {
    for (let j = i + 1; j < coords.length; j++) {
      const gaps = pairGaps(coords[i], placements[i].scale, coords[j], placements[j].scale)
      if (gaps.all < all) all = gaps.all
      if (gaps.rest < rest) rest = gaps.rest
    }
  }
  return { all, rest }
}

/** 任意拖拽角度下的最小归一化间距；1 表示相邻卡面刚好相切 */
export function orbitPlacementGap(placements: readonly OrbitPlacement[]): number {
  return gapsOfPlacements(placements).all
}

/** 静止视角下的最小归一化间距，决定首屏观感 */
export function orbitPlacementRestGap(placements: readonly OrbitPlacement[]): number {
  return gapsOfPlacements(placements).rest
}

function scaleForIndex(index: number): number {
  return 1 - 0.06 * (index % 4)
}

/**
 * 把 count 颗卡面铺到两条轨道上：依次插入并重排相位，在守住全角度间距下限的前提下
 * 最大化静止视角的间距。轨道按序号奇偶交替，保证名单顺序在环上仍是交替分布。
 */
export function buildOrbitPlacements(count: number): OrbitPlacement[] {
  if (count <= 0) return []

  const chips: Chip[] = []

  for (let index = 0; index < count; index++) {
    const track = (index % 2) as 0 | 1
    const scale = scaleForIndex(index)
    let bestStep = 0
    let bestScore = -1

    for (let step = 0; step < PHASE_STEPS; step++) {
      if (chips.some((chip) => chip.track === track && chip.step === step)) continue
      chips.push({ track, step, scale })
      const score = scoreOfChips(chips)
      chips.pop()
      if (score > bestScore) {
        bestScore = score
        bestStep = step
      }
    }

    chips.push({ track, step: bestStep, scale })
  }

  for (let pass = 0; pass < REFINE_PASSES; pass++) {
    let moved = false
    for (const chip of chips) {
      const original = chip.step
      let bestStep = original
      let bestScore = scoreOfChips(chips)

      for (let step = 0; step < PHASE_STEPS; step++) {
        if (step === original) continue
        if (chips.some((other) => other !== chip && other.track === chip.track && other.step === step)) {
          continue
        }
        chip.step = step
        const score = scoreOfChips(chips)
        if (score > bestScore + 1e-9) {
          bestScore = score
          bestStep = step
        }
      }

      chip.step = bestStep
      if (bestStep !== original) moved = true
    }
    if (!moved) break
  }

  return chips.map((chip) => ({
    track: chip.track,
    phase: chip.step / PHASE_STEPS,
    scale: chip.scale
  }))
}




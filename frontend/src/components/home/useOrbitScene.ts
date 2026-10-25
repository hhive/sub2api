import { getModelIcon } from '@/constants/modelIconPaths'
import { normalizeOrbitModels, type OrbitModelSpec } from '@/constants/orbitModels'

export type { OrbitModelSpec } from '@/constants/orbitModels'

/**
 * 着陆页/认证页共用的轨道地球场景。
 * three 在 start() 内动态加载；场景资源在 dispose() 内释放。
 *
 * 结构：菲涅尔球壳 + 陆地加权点阵 + 极淡经纬线 + 两条轨道与卫星。
 */

export type OrbitSceneMode = 'hero' | 'auth'

export interface OrbitSceneOptions {
  mode?: OrbitSceneMode
  /** 允许指针拖拽旋转；认证页保持关闭，避免与表单输入抢事件 */
  interactive?: boolean
  /** 主视觉模型卡名单；为空或未传时使用默认名单 */
  models?: OrbitModelSpec[]
}

export interface OrbitSceneHandle {
  start: () => Promise<void>
  dispose: () => void
}

interface SatelliteSpec {
  model: string
  label: string
  track: 0 | 1
  phase: number
  scale: number
}

interface TrackSpec {
  angle: number
  radius: number
  alpha: number
}

const BALL_RADIUS = 1.284
const CAM_FOV = 42
const CAM_Z_BY_MODE: Record<OrbitSceneMode, number> = { hero: 4.75, auth: 5.4 }

const COOL = '#93aec7'
const SILVER = '#e9f1f8'
const WARM = '#cbb8a6'

const TRACKS: TrackSpec[] = [
  { angle: -0.27, radius: 1.52, alpha: 0.5 },
  { angle: 0.54, radius: 1.68, alpha: 0.34 }
]

/* 卫星按名单均匀铺在两条轨道上 */
function buildSatellites(models: OrbitModelSpec[]): SatelliteSpec[] {
  const trackTotals: [number, number] = [0, 0]
  for (let i = 0; i < models.length; i++) trackTotals[i % 2] += 1

  const trackSeen: [number, number] = [0, 0]
  return models.map((model, index) => {
    const track = (index % 2) as 0 | 1
    const ordinal = trackSeen[track]
    trackSeen[track] += 1

    return {
      model: model.vendor,
      label: model.label,
      track,
      phase: (ordinal + 0.5) / trackTotals[track],
      scale: 1 - 0.06 * (index % 4)
    }
  })
}

const MONO_LOGOS = new Set(['openai', 'moonshot', 'xai', 'ollama', 'jina', 'suno'])

const SPRITE_W = 176
const SPRITE_H = 224
const SPRITE_TILE = 132

const LAND_MASK_URL = '/globe/land-mask.png'

function bakeChipSprite(model: string, label: string): HTMLCanvasElement {
  const cv = document.createElement('canvas')
  cv.width = SPRITE_W
  cv.height = SPRITE_H
  const ctx = cv.getContext('2d')
  if (!ctx) return cv

  const pad = 12
  const tile = SPRITE_TILE
  const radius = tile * 0.28
  const icon = getModelIcon(model)
  const brand = icon?.color || '#7c8cff'

  const glow = ctx.createRadialGradient(
    SPRITE_W / 2, pad + tile / 2, tile * 0.32,
    SPRITE_W / 2, pad + tile / 2, tile * 0.86
  )
  glow.addColorStop(0, `${brand}66`)
  glow.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.fillStyle = glow
  ctx.fillRect(0, 0, SPRITE_W, SPRITE_H)

  const base = ctx.createLinearGradient(pad, pad, pad + tile, pad + tile)
  base.addColorStop(0, `${brand}59`)
  base.addColorStop(0.55, 'rgba(14,17,24,0.94)')
  base.addColorStop(1, 'rgba(10,12,18,0.98)')
  ctx.beginPath()
  ctx.roundRect(pad, pad, tile, tile, radius)
  ctx.fillStyle = base
  ctx.fill()

  ctx.lineWidth = 2.4
  ctx.strokeStyle = `${brand}aa`
  ctx.stroke()

  const sheen = ctx.createLinearGradient(pad, pad, pad, pad + tile * 0.66)
  sheen.addColorStop(0, 'rgba(255,255,255,0.22)')
  sheen.addColorStop(0.6, 'rgba(255,255,255,0.04)')
  sheen.addColorStop(1, 'rgba(255,255,255,0)')
  ctx.beginPath()
  ctx.roundRect(pad + 1.5, pad + 1.5, tile - 3, tile - 3, radius)
  ctx.fillStyle = sheen
  ctx.fill()

  if (icon) {
    const inner = tile * 0.62
    const scale = inner / 24
    ctx.save()
    ctx.translate(pad + (tile - inner) / 2, pad + (tile - inner) / 2)
    ctx.scale(scale, scale)

    const paths: Path2D[] = []
    for (const d of icon.paths) {
      try {
        paths.push(new Path2D(d))
      } catch {
        // 单条 path 非法时跳过
      }
    }

    ctx.fillStyle = MONO_LOGOS.has(model) ? '#ffffff' : brand
    for (const path of paths) ctx.fill(path)

    ctx.globalAlpha = 0.55
    ctx.fillStyle = '#ffffff'
    for (const path of paths) ctx.fill(path)
    ctx.globalAlpha = 1

    ctx.restore()
  }

  const labelText = label.trim()
  const labelFamily = 'Lexend, system-ui, -apple-system, sans-serif'
  const maxLabelWidth = SPRITE_W - pad * 2
  let labelSize = 22
  ctx.font = `600 ${labelSize}px ${labelFamily}`
  while (labelSize > 13 && ctx.measureText(labelText).width > maxLabelWidth) {
    labelSize -= 1
    ctx.font = `600 ${labelSize}px ${labelFamily}`
  }

  // 最小字号仍放不下时才截断加省略号
  let fittedLabel = labelText
  if (ctx.measureText(fittedLabel).width > maxLabelWidth) {
    while (fittedLabel.length > 1 && ctx.measureText(`${fittedLabel}…`).width > maxLabelWidth) {
      fittedLabel = fittedLabel.slice(0, -1)
    }
    fittedLabel = `${fittedLabel}…`
  }

  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.fillStyle = 'rgba(255,255,255,0.96)'
  ctx.fillText(fittedLabel, SPRITE_W / 2, pad + tile + 16)

  return cv
}

/** 读取陆地掩码，供点阵按陆海加权 */
interface LandMask {
  data: Uint8ClampedArray
  width: number
  height: number
}

async function loadLandMask(): Promise<LandMask | null> {
  if (typeof Image === 'undefined') return null

  return new Promise((resolve) => {
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = () => {
      const cv = document.createElement('canvas')
      cv.width = img.width
      cv.height = img.height
      const ctx = cv.getContext('2d', { willReadFrequently: true })
      if (!ctx) {
        resolve(null)
        return
      }
      ctx.drawImage(img, 0, 0)
      resolve({ data: ctx.getImageData(0, 0, img.width, img.height).data, width: img.width, height: img.height })
    }
    img.onerror = () => resolve(null)
    img.src = LAND_MASK_URL
  })
}

export function useOrbitScene(
  canvas: HTMLCanvasElement,
  options: OrbitSceneOptions = {}
): OrbitSceneHandle {
  const mode: OrbitSceneMode = options.mode ?? 'hero'
  const interactive = options.interactive ?? mode === 'hero'

  let disposed = false
  let cleanupFn: (() => void) | null = null
  let pendingRenderer: { forceContextLoss: () => void; dispose: () => void } | null = null

  async function start(): Promise<void> {
    if (disposed) return

    const THREE = await import('three')
    if (disposed) return

    const landMask = await loadLandMask()
    if (disposed) return

    const reduceMotion =
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches

    const renderer = new THREE.WebGLRenderer({
      canvas,
      antialias: true,
      alpha: true,
      powerPreference: 'high-performance'
    })
    renderer.setClearColor(0x000000, 0)
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))
    renderer.outputColorSpace = THREE.SRGBColorSpace
    pendingRenderer = renderer

    const scene = new THREE.Scene()
    const camera = new THREE.PerspectiveCamera(CAM_FOV, 1, 0.1, 200)
    camera.position.set(0, 0, CAM_Z_BY_MODE[mode])

    const root = new THREE.Group()
    scene.add(root)

    const globe = new THREE.Group()
    root.add(globe)

    const silver = new THREE.Color(SILVER)
    const cool = new THREE.Color(COOL)
    const warm = new THREE.Color(WARM)

    /* ---------- 菲涅尔球壳 ---------- */
    const shellGeo = new THREE.SphereGeometry(BALL_RADIUS, 64, 40)
    const shellMat = new THREE.ShaderMaterial({
      uniforms: {
        uCool: { value: cool },
        uSilver: { value: silver },
        uWarm: { value: warm }
      },
      transparent: true,
      depthWrite: false,
      depthTest: false,
      vertexShader: /* glsl */ `
        varying vec3 vNormal;
        varying vec3 vEye;
        void main() {
          vec4 p = modelViewMatrix * vec4(position, 1.0);
          vNormal = normalize(normalMatrix * normal);
          vEye = normalize(-p.xyz);
          gl_Position = projectionMatrix * p;
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying vec3 vNormal;
        varying vec3 vEye;
        uniform vec3 uCool, uSilver, uWarm;
        void main() {
          vec3 n = normalize(vNormal);
          float f = 1.0 - abs(dot(n, normalize(vEye)));
          float side = smoothstep(-0.3, 0.9, n.x - n.y * 0.7);
          vec3 c = mix(uCool, uWarm, side);
          c = mix(c, uSilver, pow(f, 10.0));
          float sculpt = pow(max(0.0, dot(n, normalize(vec3(-0.7, 0.8, 0.1)))), 5.0);
          float a = 0.68 * pow(f, 16.0) + 0.018 * pow(f, 3.0) + sculpt * 0.012;
          gl_FragColor = vec4(c, a);
        }
      `
    })
    const shell = new THREE.Mesh(shellGeo, shellMat)
    shell.renderOrder = 1
    globe.add(shell)

    /* ---------- 经纬线（极淡） ---------- */
    const wireMat = new THREE.ShaderMaterial({
      uniforms: { uCool: { value: cool }, uSilver: { value: silver } },
      transparent: true,
      depthWrite: false,
      depthTest: false,
      vertexShader: /* glsl */ `
        varying float vDepth;
        void main() {
          vDepth = normalize(normalMatrix * normalize(position)).z;
          gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying float vDepth;
        uniform vec3 uCool, uSilver;
        void main() {
          float a = mix(0.003, 0.04, smoothstep(-0.1, 0.9, vDepth));
          gl_FragColor = vec4(mix(uCool, uSilver, 0.7), a);
        }
      `
    })

    const wireGroup = new THREE.Group()
    const SEG = 128
    const wireRadius = BALL_RADIUS * 1.002

    // 5 条纬线：固定纬度，半径随纬度收缩
    for (let p = 1; p <= 5; p++) {
      const theta = (p / 6) * Math.PI
      const y = Math.cos(theta) * wireRadius
      const ring = Math.sin(theta) * wireRadius
      const pts: InstanceType<typeof THREE.Vector3>[] = []
      for (let i = 0; i < SEG; i++) {
        const a = (i / SEG) * Math.PI * 2
        pts.push(new THREE.Vector3(Math.cos(a) * ring, y, Math.sin(a) * ring))
      }
      const line = new THREE.Line(new THREE.BufferGeometry().setFromPoints(pts), wireMat)
      line.renderOrder = 2
      wireGroup.add(line)
    }

    // 6 条经线：固定经度，沿纬度方向跨两个极点的圆
    for (let m = 0; m < 6; m++) {
      const lon = (m / 6) * Math.PI
      const pts: InstanceType<typeof THREE.Vector3>[] = []
      for (let i = 0; i < SEG; i++) {
        const phi = (i / SEG) * Math.PI * 2
        pts.push(
          new THREE.Vector3(
            Math.cos(phi) * Math.cos(lon) * wireRadius,
            Math.sin(phi) * wireRadius,
            Math.cos(phi) * Math.sin(lon) * wireRadius
          )
        )
      }
      const line = new THREE.Line(new THREE.BufferGeometry().setFromPoints(pts), wireMat)
      line.renderOrder = 2
      wireGroup.add(line)
    }
    globe.add(wireGroup)

    /* ---------- 陆地加权点阵 ---------- */
    // 候选点数：陆地全量保留、海洋每 19 个候选留 1 个，实际渲染约 2.8 万点
    const POINT_COUNT = 80000
    const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5))
    const positions: number[] = []
    const weights: number[] = []


    function isLand(x: number, y: number, z: number): boolean {
      if (!landMask) return true
      const lat = Math.asin(Math.max(-1, Math.min(1, y))) + (Math.random() - 0.5) * 0.02
      const lon = Math.atan2(z, x) + (Math.random() - 0.5) * 0.02
      const row = Math.min(
        landMask.height - 1,
        Math.max(0, Math.floor((0.5 - lat / Math.PI) * landMask.height))
      )
      const u = ((lon / (Math.PI * 2) + 0.5) % 1 + 1) % 1
      const col = Math.min(landMask.width - 1, Math.floor(u * landMask.width))
      return landMask.data[(row * landMask.width + col) * 4] > 127
    }

    for (let i = 0; i < POINT_COUNT; i++) {
      const y = 1 - (2 * (i + 0.5)) / POINT_COUNT
      const ring = Math.sqrt(Math.max(0, 1 - y * y))
      const lon = GOLDEN_ANGLE * i
      const x = Math.cos(lon) * ring
      const z = Math.sin(lon) * ring
      const land = isLand(x, y, z)

      // 陆地全量保留，海洋每 19 个点保留一个
      if (!land && i % 19 !== 0) continue

      const r = BALL_RADIUS * 1.002
      positions.push(x * r, y * r, z * r)
      weights.push(land ? 1 : 0.13)
    }

    const pointGeo = new THREE.BufferGeometry()
    pointGeo.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
    pointGeo.setAttribute('weight', new THREE.Float32BufferAttribute(weights, 1))

    const pointMat = new THREE.ShaderMaterial({
      uniforms: {
        uPixelRatio: { value: Math.min(window.devicePixelRatio || 1, 2) },
        uPointSize: { value: 2.8 },
        uCool: { value: cool },
        uSilver: { value: silver }
      },
      transparent: true,
      depthWrite: false,
      depthTest: false,
      vertexShader: /* glsl */ `
        attribute float weight;
        uniform float uPixelRatio, uPointSize;
        varying float vDepth, vWeight;
        void main() {
          vec4 p = modelViewMatrix * vec4(position, 1.0);
          vDepth = normalize(normalMatrix * normalize(position)).z;
          vWeight = weight;
          gl_PointSize = uPointSize * uPixelRatio;
          gl_Position = projectionMatrix * p;
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying float vDepth, vWeight;
        uniform vec3 uCool, uSilver;
        void main() {
          float mask = 1.0 - smoothstep(0.1, 0.5, length(gl_PointCoord - 0.5));
          float a = mix(0.30, 0.95, smoothstep(-0.4, 0.85, vDepth)) * vWeight * mask;
          gl_FragColor = vec4(mix(uCool, uSilver, 0.64), a);
        }
      `
    })

    const points = new THREE.Points(pointGeo, pointMat)
    points.renderOrder = 3
    globe.add(points)

    /* ---------- 轨道与卫星 ---------- */
    function orbitPoint(track: TrackSpec, phase: number, out: InstanceType<typeof THREE.Vector3>) {
      const a = phase * Math.PI * 2
      const px = Math.cos(a)
      const py = Math.sin(a) * 0.36
      const pz = Math.sin(a) * 0.93
      const ca = Math.cos(track.angle)
      const sa = Math.sin(track.angle)
      const x = px * ca - py * sa
      const y = px * sa + py * ca
      return out.set(x * track.radius, y * track.radius, pz * track.radius)
    }

    const trackLines: InstanceType<typeof THREE.Line>[] = []
    const trackMats: InstanceType<typeof THREE.ShaderMaterial>[] = []
    const TRACK_SEG = 256

    for (const track of TRACKS) {
      const pos: number[] = []
      const phaseAttr: number[] = []
      const tmp = new THREE.Vector3()
      for (let i = 0; i <= TRACK_SEG; i++) {
        const phase = i / TRACK_SEG
        orbitPoint(track, phase, tmp)
        pos.push(tmp.x, tmp.y, tmp.z)
        phaseAttr.push(phase)
      }

      const geo = new THREE.BufferGeometry()
      geo.setAttribute('position', new THREE.Float32BufferAttribute(pos, 3))
      geo.setAttribute('aPhase', new THREE.Float32BufferAttribute(phaseAttr, 1))

      const mat = new THREE.ShaderMaterial({
        uniforms: {
          uTime: { value: 0 },
          uBase: { value: track.alpha },
          uCool: { value: cool },
          uSilver: { value: silver }
        },
        transparent: true,
        depthWrite: false,
        depthTest: false,
        vertexShader: /* glsl */ `
          attribute float aPhase;
          varying float vPhase;
          void main() {
            vPhase = aPhase;
            gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
          }
        `,
        fragmentShader: /* glsl */ `
          precision highp float;
          varying float vPhase;
          uniform float uTime, uBase;
          uniform vec3 uCool, uSilver;
          void main() {
            float d = fract(vPhase - uTime * 0.06);
            float trail = pow(1.0 - d, 18.0);
            float a = uBase * (0.35 + trail * 0.65);
            gl_FragColor = vec4(mix(uCool, uSilver, trail * 0.8), a);
          }
        `
      })

      const line = new THREE.Line(geo, mat)
      line.renderOrder = 4
      root.add(line)
      trackLines.push(line)
      trackMats.push(mat)
    }

    const satellites: {
      node: InstanceType<typeof THREE.Sprite>
      spec: SatelliteSpec
      baseScale: number
    }[] = []
    const spriteTextures: InstanceType<typeof THREE.Texture>[] = []

    const satelliteSpecs = buildSatellites(normalizeOrbitModels(options.models))

    for (const spec of satelliteSpecs) {
      const texCanvas = bakeChipSprite(spec.model, spec.label)
      const tex = new THREE.CanvasTexture(texCanvas)
      tex.colorSpace = THREE.SRGBColorSpace
      tex.anisotropy = renderer.capabilities.getMaxAnisotropy()
      spriteTextures.push(tex)

      const mat = new THREE.SpriteMaterial({
        map: tex,
        transparent: true,
        depthWrite: false,
        depthTest: false
      })

      const sprite = new THREE.Sprite(mat)
      const baseScale = 0.22 * spec.scale
      sprite.scale.set(baseScale, baseScale * (SPRITE_H / SPRITE_W), 1)
      sprite.renderOrder = 5

      satellites.push({ node: sprite, spec, baseScale })
      root.add(sprite)
    }

    /* ---------- 星空 ---------- */
    const STAR_COUNT = mode === 'hero' ? 5200 : 2400
    const starGeo = new THREE.BufferGeometry()
    const starPos = new Float32Array(STAR_COUNT * 3)
    const starAlpha = new Float32Array(STAR_COUNT)
    const starSize = new Float32Array(STAR_COUNT)

    for (let i = 0; i < STAR_COUNT; i++) {
      const layer = Math.random()
      const r = layer < 0.45 ? 6 + Math.random() * 4 : layer < 0.8 ? 11 + Math.random() * 6 : 18 + Math.random() * 10
      const u = Math.random() * 2 - 1
      const phi = Math.random() * Math.PI * 2
      const s = Math.sqrt(1 - u * u)

      starPos[i * 3] = Math.cos(phi) * s * r
      starPos[i * 3 + 1] = u * r
      starPos[i * 3 + 2] = Math.sin(phi) * s * r

      const isBright = i % 22 === 0
      starAlpha[i] = isBright ? 0.95 + Math.random() * 0.05 : 0.4 + Math.random() * 0.52
      starSize[i] = isBright ? 4.2 + Math.random() * 3.4 : 1.2 + Math.random() * 2.2
    }

    starGeo.setAttribute('position', new THREE.BufferAttribute(starPos, 3))
    starGeo.setAttribute('aAlpha', new THREE.BufferAttribute(starAlpha, 1))
    starGeo.setAttribute('aSize', new THREE.BufferAttribute(starSize, 1))

    const starMat = new THREE.ShaderMaterial({
      uniforms: { uPixelRatio: { value: Math.min(window.devicePixelRatio || 1, 2) } },
      vertexShader: /* glsl */ `
        attribute float aAlpha;
        attribute float aSize;
        varying float vAlpha;
        uniform float uPixelRatio;
        void main() {
          vAlpha = aAlpha;
          vec4 mv = modelViewMatrix * vec4(position, 1.0);
          gl_PointSize = aSize * uPixelRatio * 2.2;
          gl_Position = projectionMatrix * mv;
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying float vAlpha;
        void main() {
          vec2 dp = gl_PointCoord - 0.5;
          float rr = length(dp);
          float core = smoothstep(0.3, 0.0, rr);
          float haloMask = smoothstep(0.5, 0.12, rr) * 0.4;
          float mask = max(core, haloMask);
          vec3 col = mix(vec3(0.8, 0.88, 1.0), vec3(1.0), core);
          gl_FragColor = vec4(col, vAlpha * mask);
        }
      `,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })

    const stars = new THREE.Points(starGeo, starMat)
    scene.add(stars)

    /* ---------- 拖拽交互 ---------- */
    let yaw = 0
    let pitch = 0
    let tarYaw = 0
    let tarPitch = 0
    let velYaw = 0
    let velPitch = 0
    let dragging = false
    let lastX = 0
    let lastY = 0

    const BASE_TILT_X = 0.07
    const BASE_TILT_Z = -0.17

    const onPointerDown = (e: PointerEvent) => {
      dragging = true
      lastX = e.clientX
      lastY = e.clientY
      canvas.setPointerCapture(e.pointerId)
      canvas.classList.add('is-dragging')
    }

    const onPointerMove = (e: PointerEvent) => {
      if (!dragging) return
      const dx = e.clientX - lastX
      const dy = e.clientY - lastY
      lastX = e.clientX
      lastY = e.clientY
      tarYaw += dx * 0.005
      tarPitch = Math.max(-0.7, Math.min(0.7, tarPitch + dy * 0.004))
      velYaw = dx * 0.005
      velPitch = dy * 0.004
    }

    const onPointerUp = (e: PointerEvent) => {
      dragging = false
      canvas.classList.remove('is-dragging')
      if (canvas.hasPointerCapture(e.pointerId)) canvas.releasePointerCapture(e.pointerId)
    }

    if (interactive) {
      canvas.addEventListener('pointerdown', onPointerDown)
      canvas.addEventListener('pointermove', onPointerMove)
      canvas.addEventListener('pointerup', onPointerUp)
      canvas.addEventListener('pointercancel', onPointerUp)
    }

    /* ---------- 尺寸 ---------- */
    const applySize = () => {
      const rect = canvas.getBoundingClientRect()
      const w = Math.max(1, rect.width)
      const h = Math.max(1, rect.height)
      renderer.setSize(w, h, false)
      camera.aspect = w / h
      camera.updateProjectionMatrix()
    }

    applySize()
    let ro: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(applySize)
      ro.observe(canvas)
    } else {
      window.addEventListener('resize', applySize)
    }

    /* ---------- 渲染 ---------- */
    const clock = new THREE.Clock()
    const tmpPoint = new THREE.Vector3()
    const worldPos = new THREE.Vector3()
    const forward = new THREE.Vector3(0, 0, 1)
    let rafId = 0
    let running = false

    const tick = () => {
      rafId = requestAnimationFrame(tick)
      const elapsed = clock.getElapsedTime()

      if (!dragging && !reduceMotion) {
        tarYaw += velYaw
        tarPitch = Math.max(-0.7, Math.min(0.7, tarPitch + velPitch))
        velYaw *= 0.94
        velPitch *= 0.94
      }

      yaw += (tarYaw - yaw) * 0.08
      pitch += (tarPitch - pitch) * 0.08

      const spin = reduceMotion ? 0 : elapsed * 0.055
      // 拖拽旋转整组（球体 + 轨道 + 卫星），卫星才会跟着指针一起转
      root.rotation.set(BASE_TILT_X + pitch, yaw, BASE_TILT_Z)
      root.updateMatrixWorld(true)
      globe.rotation.set(0, spin, 0)
      globe.updateMatrixWorld()

      for (const mat of trackMats) mat.uniforms.uTime.value = elapsed

      // 卫星沿轨道运行，按朝向缩放与淡出
      for (const item of satellites) {
        const track = TRACKS[item.spec.track]
        const phase = (item.spec.phase + elapsed * (reduceMotion ? 0 : 0.012)) % 1
        orbitPoint(track, phase, tmpPoint)
        item.node.position.copy(tmpPoint)
        item.node.updateMatrixWorld()
        item.node.getWorldPosition(worldPos)
        worldPos.normalize()
        const depth = (worldPos.dot(forward) + 1) / 2
        const taper = 0.28 + 0.72 * Math.pow(0.5 + 0.5 * Math.sin(phase * Math.PI * 2 + item.spec.track * 2), 2)
        const s = item.baseScale * (0.66 + depth * 0.5)
        item.node.scale.set(s, s * (SPRITE_H / SPRITE_W), 1)
        ;(item.node.material as InstanceType<typeof THREE.SpriteMaterial>).opacity =
          (0.5 + depth * 0.5) * (0.62 + taper * 0.38)
      }

      stars.rotation.y = yaw * 0.06
      stars.rotation.x = pitch * 0.05

      renderer.render(scene, camera)
    }

    const startLoop = () => {
      if (running || disposed) return
      running = true
      tick()
    }

    const stopLoop = () => {
      if (!running) return
      running = false
      cancelAnimationFrame(rafId)
    }

    const onVisibilityChange = () => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        stopLoop()
        return
      }
      startLoop()
    }

    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', onVisibilityChange)
    }

    startLoop()

    const cleanup = () => {
      stopLoop()
      if (typeof document !== 'undefined') {
        document.removeEventListener('visibilitychange', onVisibilityChange)
      }
      ro?.disconnect()
      window.removeEventListener('resize', applySize)
      if (interactive) {
        canvas.removeEventListener('pointerdown', onPointerDown)
        canvas.removeEventListener('pointermove', onPointerMove)
        canvas.removeEventListener('pointerup', onPointerUp)
        canvas.removeEventListener('pointercancel', onPointerUp)
      }
      canvas.classList.remove('is-dragging')

      for (const txt of spriteTextures) txt.dispose()
      scene.traverse((obj) => {
        if (
          obj instanceof THREE.Mesh ||
          obj instanceof THREE.Points ||
          obj instanceof THREE.Line ||
          obj instanceof THREE.Sprite
        ) {
          const anyObj = obj as InstanceType<typeof THREE.Mesh>
          anyObj.geometry?.dispose()
          const mat = anyObj.material
          if (Array.isArray(mat)) mat.forEach((m) => m.dispose())
          else mat?.dispose()
        }
      })
      renderer.dispose()
      renderer.forceContextLoss()
      pendingRenderer = null
    }

    cleanupFn = cleanup
  }

  function dispose(): void {
    disposed = true
    cleanupFn?.()
    cleanupFn = null

    if (pendingRenderer) {
      pendingRenderer.forceContextLoss()
      pendingRenderer.dispose()
      pendingRenderer = null
    }
  }

  return { start, dispose }
}

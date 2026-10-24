import { getModelIcon } from '@/constants/modelIconPaths'

/**
 * 着陆页/认证页共用的轨道地球场景。
 * three 在 start() 内动态加载，避免进入首屏主包。
 */

export type OrbitSceneMode = 'hero' | 'auth'

export interface OrbitSceneOptions {
  mode?: OrbitSceneMode
  /** 允许指针拖拽旋转；认证页保持关闭，避免与表单输入抢事件 */
  interactive?: boolean
}

export interface OrbitSceneHandle {
  start: () => Promise<void>
  resize: () => void
  dispose: () => void
}

interface ChipSpec {
  model: string
  label: string
  ring: 0 | 1 | 2
  angle: number
  scale: number
}

interface RingSpec {
  rx: number
  ry: number
  tiltX: number
  tiltZ: number
  alpha: number
}

const BALL_RADIUS = 1.284
const CAM_FOV = 42
const CYCLE_SECONDS = 42

/* 相机距离决定球体在屏幕上的占比：hero 约 64% 视口高，auth 约 54% */
const CAM_Z_BY_MODE: Record<OrbitSceneMode, number> = { hero: 5.15, auth: 5.75 }
const STAR_COUNT_BY_MODE: Record<OrbitSceneMode, number> = { hero: 6000, auth: 2600 }

const INK = '#eef2f4'

const CHIPS: ChipSpec[] = [
  { model: 'gemini', label: 'GM-3.6 Flash', ring: 1, angle: -1.92, scale: 1 },
  { model: 'zhipu', label: 'GLM-5.3', ring: 0, angle: -2.3, scale: 0.82 },
  { model: 'qwen', label: 'Qwen3-Max', ring: 2, angle: -2.66, scale: 0.76 },
  { model: 'claude', label: 'CL-Fable 5.1', ring: 1, angle: -2.94, scale: 1.08 },
  { model: 'deepseek', label: 'DeepSeek V4', ring: 0, angle: 2.9, scale: 0.9 },
  { model: 'openai', label: 'GPT-5.6', ring: 1, angle: 2.4, scale: 1.12 },
  { model: 'mistral', label: 'Mistral L', ring: 2, angle: 2.06, scale: 0.72 },
  { model: 'minimax', label: 'MiniMax M2', ring: 0, angle: 1.74, scale: 0.84 },
  { model: 'doubao', label: 'Doubao Pro', ring: 1, angle: 1.4, scale: 0.88 },
  { model: 'moonshot', label: 'Kimi K3', ring: 2, angle: 1.04, scale: 0.74 },
  { model: 'xai', label: 'Grok 4', ring: 0, angle: 0.72, scale: 0.84 },
  { model: 'meta', label: 'Llama 5', ring: 1, angle: 0.36, scale: 0.7 },
  { model: 'qwen', label: 'Qwen3-VL', ring: 2, angle: 0.14, scale: 0.6 },
  { model: 'gemini', label: 'GM-3 Pro Image', ring: 0, angle: -0.3, scale: 0.64 },
  { model: 'yi', label: 'Yi-Large', ring: 1, angle: -0.7, scale: 0.6 },
  { model: 'cohere', label: 'Command R', ring: 2, angle: -1.1, scale: 0.55 },
  { model: 'openrouter', label: 'OpenRouter', ring: 0, angle: -1.46, scale: 0.52 },
  { model: 'ollama', label: 'Ollama', ring: 1, angle: -2.58, scale: 0.5 },
  { model: 'jina', label: 'Jina', ring: 2, angle: 2.58, scale: 0.46 },
  { model: 'suno', label: 'Suno', ring: 0, angle: 1.08, scale: 0.46 }
]

/* 只有少数厂商标是纯线条，其余保留品牌色 */
const MONO_LOGOS = new Set(['openai', 'moonshot', 'xai', 'ollama', 'jina', 'suno'])

const RINGS: RingSpec[] = [
  { rx: BALL_RADIUS * 1.34, ry: BALL_RADIUS * 0.7, tiltX: 0.42, tiltZ: -0.16, alpha: 0.26 },
  { rx: BALL_RADIUS * 1.5, ry: BALL_RADIUS * 0.86, tiltX: -0.32, tiltZ: 0.22, alpha: 0.18 },
  { rx: BALL_RADIUS * 1.14, ry: BALL_RADIUS * 0.52, tiltX: 0.08, tiltZ: -0.4, alpha: 0.14 }
]

const SPRITE_W = 160
const SPRITE_H = 200
const SPRITE_TILE = 116

function bakeChipSprite(model: string, label: string): HTMLCanvasElement {
  const cv = document.createElement('canvas')
  cv.width = SPRITE_W
  cv.height = SPRITE_H
  const ctx = cv.getContext('2d')
  if (!ctx) return cv

  const pad = 10
  const tile = SPRITE_TILE
  const radius = tile * 0.26

  // 底板：深色玻璃，保留图标原色所需的对比度
  ctx.beginPath()
  ctx.roundRect(pad, pad, tile, tile, radius)
  ctx.fillStyle = 'rgba(10,12,16,0.9)'
  ctx.fill()
  ctx.lineWidth = 2
  ctx.strokeStyle = 'rgba(255,255,255,0.16)'
  ctx.stroke()

  // 顶部高光，制造玻璃质感
  const sheen = ctx.createLinearGradient(pad, pad, pad, pad + tile * 0.62)
  sheen.addColorStop(0, 'rgba(255,255,255,0.16)')
  sheen.addColorStop(0.5, 'rgba(255,255,255,0.03)')
  sheen.addColorStop(1, 'rgba(255,255,255,0)')
  ctx.beginPath()
  ctx.roundRect(pad + 1, pad + 1, tile - 2, tile - 2, radius)
  ctx.fillStyle = sheen
  ctx.fill()

  const icon = getModelIcon(model)
  if (icon) {
    const inner = tile * 0.58
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

    ctx.fillStyle = MONO_LOGOS.has(model) ? '#e8ecf0' : icon.color
    for (const path of paths) ctx.fill(path)

    // 品牌色之上叠一层低透明度白，提亮边缘而不洗白整块
    ctx.globalAlpha = 0.18
    ctx.fillStyle = '#ffffff'
    for (const path of paths) ctx.fill(path)
    ctx.globalAlpha = 1

    ctx.restore()
  }

  ctx.font = '600 21px Lexend, system-ui, -apple-system, sans-serif'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'top'
  ctx.fillStyle = 'rgba(255,255,255,0.94)'
  ctx.fillText(label, SPRITE_W / 2, pad + tile + 14)

  return cv
}

export function useOrbitScene(
  canvas: HTMLCanvasElement,
  options: OrbitSceneOptions = {}
): OrbitSceneHandle {
  const mode: OrbitSceneMode = options.mode ?? 'hero'
  const interactive = options.interactive ?? mode === 'hero'

  let disposed = false
  let applySizeFn: (() => void) | null = null
  let cleanupFn: (() => void) | null = null
  /* 场景构建中途抛错时，dispose 仍能释放已创建的渲染器 */
  let pendingRenderer: { forceContextLoss: () => void; dispose: () => void } | null = null

  function resize(): void {
    applySizeFn?.()
  }

  async function start(): Promise<void> {
    if (disposed) return

    const THREE = await import('three')
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

    const globe = new THREE.Group()
    scene.add(globe)

    /* ---------- 经纬线 ---------- */
    const linePositions: number[] = []
    const MERIDIANS = 16
    const PARALLELS = 9
    const R_LINE = BALL_RADIUS * 1.004

    for (let m = 0; m < MERIDIANS; m++) {
      const lon = (m / MERIDIANS) * Math.PI * 2
      const SEG = 64
      for (let i = 0; i < SEG; i++) {
        const t0 = (i / SEG) * Math.PI
        const t1 = ((i + 1) / SEG) * Math.PI
        linePositions.push(
          Math.sin(t0) * Math.cos(lon) * R_LINE, Math.cos(t0) * R_LINE, Math.sin(t0) * Math.sin(lon) * R_LINE,
          Math.sin(t1) * Math.cos(lon) * R_LINE, Math.cos(t1) * R_LINE, Math.sin(t1) * Math.sin(lon) * R_LINE
        )
      }
    }

    for (let p = 1; p <= PARALLELS; p++) {
      const theta = (p / (PARALLELS + 1)) * Math.PI
      const y = Math.cos(theta) * R_LINE
      const r = Math.sin(theta) * R_LINE
      const SEG = 96
      for (let i = 0; i < SEG; i++) {
        const a0 = (i / SEG) * Math.PI * 2
        const a1 = ((i + 1) / SEG) * Math.PI * 2
        linePositions.push(
          Math.cos(a0) * r, y, Math.sin(a0) * r,
          Math.cos(a1) * r, y, Math.sin(a1) * r
        )
      }
    }

    const lineGeo = new THREE.BufferGeometry()
    lineGeo.setAttribute('position', new THREE.Float32BufferAttribute(linePositions, 3))
    const lineMat = new THREE.LineBasicMaterial({
      color: new THREE.Color(INK),
      transparent: true,
      opacity: 0.07,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })
    globe.add(new THREE.LineSegments(lineGeo, lineMat))

    /* ---------- 球面点阵 ---------- */
    const DOT_ROWS = 120
    const DOT_COLS = 240
    const dotPos: number[] = []
    const dotAlpha: number[] = []

    for (let r = 0; r < DOT_ROWS; r++) {
      const theta = ((r + 0.5) / DOT_ROWS) * Math.PI
      const y = Math.cos(theta)

      for (let c = 0; c < DOT_COLS; c++) {
        const lon = (c / DOT_COLS) * Math.PI * 2

        const radius = BALL_RADIUS * 1.008

        dotPos.push(
          Math.sin(theta) * Math.cos(lon) * radius,
          Math.cos(theta) * radius,
          Math.sin(theta) * Math.sin(lon) * radius
        )

        const belt = 1 - Math.abs(y) * 0.5
        dotAlpha.push((0.14 + Math.random() * 0.3) * belt)
      }
    }

    const dotGeo = new THREE.BufferGeometry()
    dotGeo.setAttribute('position', new THREE.Float32BufferAttribute(dotPos, 3))
    dotGeo.setAttribute('aAlpha', new THREE.Float32BufferAttribute(dotAlpha, 1))

    const dotMat = new THREE.ShaderMaterial({
      uniforms: { uPixelRatio: { value: Math.min(window.devicePixelRatio || 1, 2) } },
      vertexShader: /* glsl */ `
        attribute float aAlpha;
        varying float vAlpha;
        uniform float uPixelRatio;
        void main() {
          vAlpha = aAlpha;
          vec4 mv = modelViewMatrix * vec4(position, 1.0);
          // 视图空间里相机朝 -Z 看，正半球的 z 为负，取反后才得到"越近越亮"
          float facing = clamp(-normalize(mv.xyz).z * 0.5 + 0.5, 0.0, 1.0);
          vAlpha *= 1.0 + facing * 0.9;
          gl_PointSize = (1.0 + aAlpha * 1.7) * uPixelRatio;
          gl_Position = projectionMatrix * mv;
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying float vAlpha;
        void main() {
          vec2 dp = gl_PointCoord - 0.5;
          float mask = smoothstep(0.5, 0.12, length(dp));
          gl_FragColor = vec4(vec3(0.94, 0.96, 0.98), vAlpha * mask);
        }
      `,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending
    })

    globe.add(new THREE.Points(dotGeo, dotMat))

    /* ---------- 大气辉光 ---------- */
    const haloGeo = new THREE.SphereGeometry(BALL_RADIUS * 1.1, 48, 32)
    const haloMat = new THREE.ShaderMaterial({
      uniforms: { uColor: { value: new THREE.Color('#9fc6d8') } },
      vertexShader: /* glsl */ `
        varying vec3 vNormalW;
        varying vec3 vViewDir;
        void main() {
          vec4 world = modelMatrix * vec4(position, 1.0);
          vNormalW = normalize(mat3(modelMatrix) * normal);
          vViewDir = normalize(cameraPosition - world.xyz);
          gl_Position = projectionMatrix * viewMatrix * world;
        }
      `,
      fragmentShader: /* glsl */ `
        precision highp float;
        varying vec3 vNormalW;
        varying vec3 vViewDir;
        uniform vec3 uColor;
        void main() {
          float rim = 1.0 - abs(dot(normalize(vNormalW), normalize(vViewDir)));
          float a = pow(rim, 3.4) * 0.34;
          gl_FragColor = vec4(uColor, a);
        }
      `,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      side: THREE.BackSide
    })
    globe.add(new THREE.Mesh(haloGeo, haloMat))

    /* ---------- 深度遮罩 ----------
       只写深度不写颜色：球体后方的轨道线与图标被正确遮挡，球面点阵仍在球外绘制 */
    const depthMaskGeo = new THREE.SphereGeometry(BALL_RADIUS * 0.985, 48, 32)
    const depthMaskMat = new THREE.MeshBasicMaterial({
      colorWrite: false,
      depthWrite: true,
      depthTest: true
    })
    globe.add(new THREE.Mesh(depthMaskGeo, depthMaskMat))

    /* ---------- 轨道 ---------- */
    function pointOnRing(spec: RingSpec, tt: number): InstanceType<typeof THREE.Vector3> {
      const ex = Math.cos(tt) * spec.rx
      const ey = Math.sin(tt) * spec.ry

      const cx = Math.cos(spec.tiltX)
      const sx = Math.sin(spec.tiltX)
      const y1 = ey * cx
      const z1 = ey * sx

      const cz = Math.cos(spec.tiltZ)
      const sz = Math.sin(spec.tiltZ)

      return new THREE.Vector3(ex * cz - y1 * sz, ex * sz + y1 * cz, z1)
    }

    const ringMats: InstanceType<typeof THREE.ShaderMaterial>[] = []

    for (const spec of RINGS) {
      const SEGMENTS = 384
      const pos: number[] = []
      const phase: number[] = []

      for (let i = 0; i <= SEGMENTS; i++) {
        const tt = (i / SEGMENTS) * Math.PI * 2
        const p = pointOnRing(spec, tt)
        pos.push(p.x, p.y, p.z)
        phase.push(i / SEGMENTS)
      }

      const geo = new THREE.BufferGeometry()
      geo.setAttribute('position', new THREE.Float32BufferAttribute(pos, 3))
      geo.setAttribute('aPhase', new THREE.Float32BufferAttribute(phase, 1))

      const mat = new THREE.ShaderMaterial({
        uniforms: { uTime: { value: 0 }, uBase: { value: spec.alpha } },
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
          uniform float uTime;
          uniform float uBase;
          void main() {
            float a = (vPhase - uTime * 0.06) * 6.28318530718;
            float glow = pow(0.5 + 0.5 * cos(a), 9.0);
            gl_FragColor = vec4(vec3(1.0), uBase * 0.68 + glow * 1.0);
          }
        `,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending
      })

      ringMats.push(mat)
      globe.add(new THREE.Line(geo, mat))
    }

    /* ---------- 轨道图标 ---------- */
    const sprites: { node: InstanceType<typeof THREE.Sprite>; baseScale: number }[] = []
    const spriteTextures: InstanceType<typeof THREE.Texture>[] = []

    for (const spec of CHIPS) {
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
      sprite.position.copy(pointOnRing(RINGS[spec.ring], spec.angle))

      const scale = 0.196 * spec.scale
      sprite.scale.set(scale, scale * (SPRITE_H / SPRITE_W), 1)

      sprites.push({ node: sprite, baseScale: scale })
      globe.add(sprite)
    }

    /* ---------- 星空 ---------- */
    const starGeo = new THREE.BufferGeometry()
    const STAR_COUNT = STAR_COUNT_BY_MODE[mode]
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

      // 每 22 颗挑一颗做亮星，补高光像素
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
    const forward = new THREE.Vector3(0, 0, 1)
    const worldPos = new THREE.Vector3()
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

      const autoSpin = reduceMotion ? 0 : ((elapsed % CYCLE_SECONDS) / CYCLE_SECONDS) * Math.PI * 2
      globe.rotation.set(BASE_TILT_X + pitch, -0.55 + yaw + autoSpin, BASE_TILT_Z)
      globe.updateMatrixWorld()

      for (const m of ringMats) m.uniforms.uTime.value = elapsed

      // 图标：按朝向收放，背面缩小变淡
      for (const item of sprites) {
        worldPos.copy(item.node.position).applyMatrix4(globe.matrixWorld).normalize()
        const depth = (worldPos.dot(forward) + 1) / 2
        const s = item.baseScale * (0.62 + depth * 0.55)
        item.node.scale.set(s, s * (SPRITE_H / SPRITE_W), 1)
        ;(item.node.material as InstanceType<typeof THREE.SpriteMaterial>).opacity = 0.58 + depth * 0.42
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

    applySizeFn = applySize
    cleanupFn = cleanup
  }

  function dispose(): void {
    disposed = true
    cleanupFn?.()
    cleanupFn = null
    applySizeFn = null

    if (pendingRenderer) {
      pendingRenderer.forceContextLoss()
      pendingRenderer.dispose()
      pendingRenderer = null
    }
  }

  return { start, resize, dispose }
}


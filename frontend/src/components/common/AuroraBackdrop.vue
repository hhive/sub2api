<template>
  <!-- 沉浸式极光背景层：首页与认证页共用，颜色走 CSS 变量以支持深浅色 -->
  <div
    data-testid="auth-tech-bg"
    class="auth-tech"
    :class="variant === 'void' ? 'auth-tech-void' : 'auth-tech-auth'"
    aria-hidden="true"
  >
    <i class="auth-base"></i>
    <i class="auth-aurora auth-aurora-a"></i>
    <i class="auth-aurora auth-aurora-b"></i>
    <i class="auth-aurora auth-aurora-c"></i>
    <i class="auth-grid"></i>
    <i class="auth-scan"></i>
    <i v-if="pointerGlow" ref="glowRef" data-testid="auth-pointer-glow" class="auth-glow"></i>
    <i class="auth-vignette"></i>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    variant?: 'void' | 'auth'
    pointerGlow?: boolean
  }>(),
  { variant: 'void', pointerGlow: false }
)

const glowRef = ref<HTMLElement | null>(null)
let pointerBound = false

// 只在精细指针且未开启减弱动效时跟随指针；matchMedia 缺失的旧环境直接跳过。
function supportsPointerGlow(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  return (
    !window.matchMedia('(pointer: coarse)').matches &&
    !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

function paintGlow(clientX: number, clientY: number) {
  const node = glowRef.value
  if (!node) return

  const x = (clientX / Math.max(window.innerWidth, 1)) * 100
  const y = (clientY / Math.max(window.innerHeight, 1)) * 100
  node.style.setProperty('--mx', `${x.toFixed(2)}%`)
  node.style.setProperty('--my', `${y.toFixed(2)}%`)
}

function onPointerMove(event: PointerEvent) {
  paintGlow(event.clientX, event.clientY)
}

function onPointerLeave() {
  paintGlow(window.innerWidth / 2, window.innerHeight * 0.3)
}

onMounted(() => {
  if (!props.pointerGlow) return
  if (!supportsPointerGlow()) return

  pointerBound = true
  window.addEventListener('pointermove', onPointerMove, { passive: true })
  window.addEventListener('pointerleave', onPointerLeave, { passive: true })
})

onBeforeUnmount(() => {
  if (!pointerBound) return

  pointerBound = false
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerleave', onPointerLeave)
})
</script>

<style scoped>
.auth-tech {
  /* 目标站极光主色 #7B39ED 的深色口径 */
  --aurora-base: #000000;
  --aurora-band-a: rgba(123, 57, 237, 0.34);
  --aurora-band-b: rgba(123, 57, 237, 0.16);
  --aurora-band-c: rgba(90, 32, 190, 0.12);
  --aurora-grid-line: rgba(123, 57, 237, 0.07);
  --aurora-scan: rgba(160, 118, 255, 0.5);
  --aurora-glow: rgba(123, 57, 237, 0.2);
  --aurora-vignette: rgba(0, 0, 0, 0.55);

  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  background: var(--aurora-base);
}

/* 浅色模式：同一套极光降到浅底上，保持深浅色切换可用 */
html:not(.dark) .auth-tech {
  --aurora-base: #f5f4fb;
  --aurora-band-a: rgba(123, 57, 237, 0.2);
  --aurora-band-b: rgba(123, 57, 237, 0.12);
  --aurora-band-c: rgba(99, 102, 241, 0.1);
  --aurora-grid-line: rgba(123, 57, 237, 0.08);
  --aurora-scan: rgba(123, 57, 237, 0.35);
  --aurora-glow: rgba(123, 57, 237, 0.12);
  --aurora-vignette: rgba(245, 244, 251, 0.72);
}

.auth-base {
  position: absolute;
  inset: 0;
  display: block;
  background: var(--aurora-base);
}

.auth-aurora {
  position: absolute;
  display: block;
  border-radius: 50%;
  filter: blur(72px);
  opacity: 0.7;
  will-change: transform;
}

.auth-aurora-a {
  width: 54vw;
  height: 54vw;
  min-width: 320px;
  min-height: 320px;
  left: 50%;
  top: -18vh;
  margin-left: -27vw;
  background: radial-gradient(circle, var(--aurora-band-a), transparent 66%);
  animation: auth-drift-a 36s ease-in-out infinite alternate;
}

.auth-aurora-b {
  width: 46vw;
  height: 46vw;
  min-width: 280px;
  min-height: 280px;
  right: -10vw;
  top: 22vh;
  background: radial-gradient(circle, var(--aurora-band-b), transparent 68%);
  animation: auth-drift-b 48s ease-in-out infinite alternate;
}

.auth-aurora-c {
  width: 42vw;
  height: 42vw;
  min-width: 240px;
  min-height: 240px;
  left: -8vw;
  bottom: -14vh;
  background: radial-gradient(circle, var(--aurora-band-c), transparent 70%);
  animation: auth-drift-c 54s ease-in-out infinite alternate;
}

@keyframes auth-drift-a {
  from { transform: translate3d(0, 0, 0) scale(1); }
  to { transform: translate3d(6vw, 5vh, 0) scale(1.12); }
}

@keyframes auth-drift-b {
  from { transform: translate3d(0, 0, 0) scale(1.04); }
  to { transform: translate3d(-7vw, -5vh, 0) scale(0.94); }
}

@keyframes auth-drift-c {
  from { transform: translate3d(0, 0, 0) scale(0.98); }
  to { transform: translate3d(6vw, -6vh, 0) scale(1.1); }
}

.auth-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(to right, var(--aurora-grid-line) 1px, transparent 1px),
    linear-gradient(to bottom, var(--aurora-grid-line) 1px, transparent 1px);
  background-size: 64px 64px;
  -webkit-mask-image: radial-gradient(ellipse 82% 72% at 50% 16%, #000 6%, transparent 78%);
  mask-image: radial-gradient(ellipse 82% 72% at 50% 16%, #000 6%, transparent 78%);
}

.auth-scan {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--aurora-scan), transparent);
  box-shadow: 0 0 22px var(--aurora-scan);
  opacity: 0.5;
  animation: auth-scan 9s linear infinite;
}

@keyframes auth-scan {
  0% { transform: translateY(-5vh); opacity: 0; }
  12% { opacity: 0.5; }
  88% { opacity: 0.5; }
  100% { transform: translateY(105vh); opacity: 0; }
}

.auth-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(
    460px circle at var(--mx, 50%) var(--my, 30%),
    var(--aurora-glow),
    transparent 62%
  );
}

.auth-vignette {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 72% 62% at 50% 40%, transparent 32%, var(--aurora-vignette) 100%);
}

@media (prefers-reduced-motion: reduce) {
  .auth-aurora,
  .auth-scan {
    animation: none !important;
  }

  .auth-glow {
    display: none;
  }
}
</style>

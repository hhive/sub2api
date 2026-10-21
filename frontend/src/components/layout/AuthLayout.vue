<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- 深色科技背景：极光 / 网格 / 扫描线 / 指针光斑，所有认证页共用 -->
    <div data-testid="auth-tech-bg" class="auth-tech" aria-hidden="true">
      <i class="auth-aurora auth-aurora-a"></i>
      <i class="auth-aurora auth-aurora-b"></i>
      <i class="auth-aurora auth-aurora-c"></i>
      <i class="auth-grid"></i>
      <i class="auth-scan"></i>
      <i ref="glowRef" data-testid="auth-pointer-glow" class="auth-glow"></i>
      <i class="auth-vignette"></i>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div
            class="auth-logo mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="auth-brand-title mb-2 text-3xl font-bold">
            {{ siteName }}
          </h1>
          <p class="auth-subtitle text-sm">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="auth-card-shell">
        <div class="card-glass rounded-2xl p-8 shadow-glass">
          <slot />
        </div>
      </div>

      <!-- Footer Links -->
      <div class="auth-footer-links mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="auth-copyright mt-8 text-center text-xs">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

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

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-shell {
  background: #050814;
}

.auth-tech {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
}

.auth-tech::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 62% 52% at 16% -6%, rgba(34, 211, 238, 0.2), transparent 62%),
    radial-gradient(ellipse 58% 46% at 90% 28%, rgba(139, 92, 246, 0.2), transparent 68%),
    radial-gradient(ellipse 70% 42% at 52% 108%, rgba(244, 114, 182, 0.1), transparent 70%),
    #050814;
}

.auth-aurora {
  position: absolute;
  display: block;
  border-radius: 50%;
  filter: blur(72px);
  opacity: 0.55;
  will-change: transform;
}

.auth-aurora-a {
  width: 52vw;
  height: 52vw;
  min-width: 320px;
  min-height: 320px;
  left: -12vw;
  top: -6vh;
  background: radial-gradient(circle, rgba(34, 211, 238, 0.45), transparent 66%);
  animation: auth-drift-a 36s ease-in-out infinite alternate;
}

.auth-aurora-b {
  width: 46vw;
  height: 46vw;
  min-width: 280px;
  min-height: 280px;
  right: -10vw;
  top: 18vh;
  background: radial-gradient(circle, rgba(139, 92, 246, 0.42), transparent 68%);
  animation: auth-drift-b 48s ease-in-out infinite alternate;
}

.auth-aurora-c {
  width: 40vw;
  height: 40vw;
  min-width: 240px;
  min-height: 240px;
  left: 18vw;
  bottom: -18vh;
  background: radial-gradient(circle, rgba(244, 114, 182, 0.28), transparent 70%);
  animation: auth-drift-c 54s ease-in-out infinite alternate;
}

@keyframes auth-drift-a {
  from { transform: translate3d(0, 0, 0) scale(1); }
  to { transform: translate3d(8vw, 6vh, 0) scale(1.12); }
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
    linear-gradient(to right, rgba(34, 211, 238, 0.06) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(34, 211, 238, 0.06) 1px, transparent 1px);
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
  background: linear-gradient(90deg, transparent, rgba(34, 211, 238, 0.55), rgba(139, 92, 246, 0.5), transparent);
  box-shadow: 0 0 22px rgba(34, 211, 238, 0.35);
  opacity: 0.6;
  animation: auth-scan 9s linear infinite;
}

@keyframes auth-scan {
  0% { transform: translateY(-5vh); opacity: 0; }
  12% { opacity: 0.6; }
  88% { opacity: 0.6; }
  100% { transform: translateY(105vh); opacity: 0; }
}

.auth-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(460px circle at var(--mx, 50%) var(--my, 30%), rgba(34, 211, 238, 0.16), transparent 62%);
}

.auth-vignette {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse 72% 62% at 50% 40%, transparent 32%, rgba(3, 6, 18, 0.6) 100%);
}

.auth-logo {
  background: linear-gradient(150deg, rgba(34, 211, 238, 0.2), rgba(139, 92, 246, 0.18));
  border: 1px solid rgba(196, 181, 253, 0.35);
  box-shadow: 0 0 30px rgba(34, 211, 238, 0.25), 0 10px 30px rgba(139, 92, 246, 0.25);
}

.auth-brand-title {
  background: linear-gradient(100deg, #8df2ff, #22d3ee 34%, #c4b5fd 68%, #8b5cf6);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  filter: drop-shadow(0 0 24px rgba(139, 92, 246, 0.28));
}

.auth-subtitle {
  color: #a9b8d4;
}

.auth-card-shell {
  padding: 1px;
  border-radius: 18px;
  background: linear-gradient(
    150deg,
    rgba(34, 211, 238, 0.55),
    rgba(139, 92, 246, 0.45) 46%,
    rgba(255, 255, 255, 0.08)
  );
  box-shadow: 0 24px 70px rgba(3, 8, 24, 0.65), 0 0 40px rgba(34, 211, 238, 0.12);
  animation: auth-rise 0.55s cubic-bezier(0.2, 0.7, 0.2, 1) both;
}

.auth-card-shell > .card-glass {
  border-radius: 17px;
  background-color: rgba(255, 255, 255, 0.82);
}

/* 认证页统一强调色：卡片标题与主按钮跟随落地页的青紫口径 */
.auth-card-shell :deep(h2) {
  background-image: linear-gradient(100deg, #0891b2, #22d3ee 32%, #8b5cf6 88%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

html:not(.dark) .auth-card-shell :deep(h2) {
  background-image: linear-gradient(100deg, #0e7490, #0891b2 32%, #7c3aed 88%);
}

.auth-card-shell :deep(.btn.btn-primary) {
  background-image: linear-gradient(112deg, #67e8f9, #22d3ee 42%, #8b5cf6);
  border: 1px solid rgba(196, 181, 253, 0.4);
  color: #04121f;
  box-shadow: 0 10px 30px rgba(34, 211, 238, 0.28), 0 8px 28px rgba(139, 92, 246, 0.28);
}

.auth-card-shell :deep(.btn.btn-primary:hover:not(:disabled)) {
  background-image: linear-gradient(112deg, #8df2ff, #22d3ee 45%, #a78bfa);
  box-shadow: 0 14px 38px rgba(34, 211, 238, 0.36), 0 10px 34px rgba(139, 92, 246, 0.34);
}

.auth-card-shell :deep(.btn.btn-primary svg) {
  color: inherit;
}

html.dark .auth-card-shell > .card-glass {
  background-color: rgba(15, 23, 42, 0.8);
}

.auth-footer-links {
  color: #a9b8d4;
}

.auth-footer-links :deep(a) {
  color: #c4b5fd;
}

.auth-copyright {
  color: #7d8bab;
}

@keyframes auth-rise {
  from { opacity: 0; transform: translateY(12px); }
  to { opacity: 1; transform: none; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-aurora,
  .auth-scan,
  .auth-card-shell {
    animation: none !important;
  }

  .auth-glow {
    display: none;
  }
}
</style>

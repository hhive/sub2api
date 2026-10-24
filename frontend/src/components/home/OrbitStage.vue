<template>
  <!-- 轨道地球背景层：首页与认证页共用，负责 canvas、装饰层与降级 -->
  <div
    class="orbit-stage"
    :class="[`orbit-stage-${mode}`, { 'is-static': failed }]"
    data-testid="orbit-stage"
    aria-hidden="true"
  >
    <canvas ref="canvasRef" class="orbit-stage-canvas"></canvas>
    <div class="orbit-stage-grid"></div>
    <div class="orbit-stage-veil"></div>
    <div class="orbit-stage-grain"></div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

import { useOrbitScene, type OrbitSceneHandle, type OrbitSceneMode } from './useOrbitScene'

const props = withDefaults(
  defineProps<{
    mode?: OrbitSceneMode
    interactive?: boolean
  }>(),
  { mode: 'hero', interactive: undefined }
)

const canvasRef = ref<HTMLCanvasElement | null>(null)
const failed = ref(false)

let scene: OrbitSceneHandle | null = null

// WebGL 不可用时直接走静态降级，不触发 three 的上下文创建异常
function supportsWebgl(): boolean {
  if (typeof document === 'undefined') return false
  try {
    const probe = document.createElement('canvas')
    return Boolean(probe.getContext('webgl2') ?? probe.getContext('webgl'))
  } catch {
    return false
  }
}

onMounted(async () => {
  const canvas = canvasRef.value
  if (!canvas || !supportsWebgl()) {
    failed.value = true
    return
  }

  const interactive = props.interactive ?? props.mode === 'hero'
  scene = useOrbitScene(canvas, { mode: props.mode, interactive })

  try {
    await scene.start()
  } catch {
    scene.dispose()
    scene = null
    failed.value = true
  }
})

onBeforeUnmount(() => {
  scene?.dispose()
  scene = null
})
</script>

<style scoped>
.orbit-stage {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  background: radial-gradient(ellipse 74% 68% at 50% 46%, #16191c 0%, #101316 46%, #0b0d0e 100%);
}

.orbit-stage-auth {
  background: radial-gradient(ellipse 70% 62% at 50% 44%, #12161a 0%, #0c0e11 52%, #08090b 100%);
}

.orbit-stage-canvas {
  position: absolute;
  inset: 0;
  height: 100%;
  width: 100%;
  pointer-events: auto;
  touch-action: pan-y;
  cursor: grab;
}

.orbit-stage-auth .orbit-stage-canvas {
  pointer-events: none;
  cursor: default;
}

.orbit-stage-canvas.is-dragging {
  cursor: grabbing;
}

.orbit-stage.is-static .orbit-stage-canvas {
  display: none;
}

.orbit-stage-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(to right, rgba(122, 152, 172, 0.085) 1px, transparent 1px),
    linear-gradient(to bottom, rgba(122, 152, 172, 0.085) 1px, transparent 1px);
  background-size: 76px 76px;
  background-position: center center;
  -webkit-mask-image: radial-gradient(ellipse 78% 76% at 50% 50%, #000 30%, transparent 94%);
  mask-image: radial-gradient(ellipse 78% 76% at 50% 50%, #000 30%, transparent 94%);
}

.orbit-stage-auth .orbit-stage-grid {
  background-size: 92px 92px;
  opacity: 0.7;
}

.orbit-stage-veil {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 34% 30% at 50% 50%, rgba(8, 8, 11, 0.74) 0%, rgba(8, 8, 11, 0.3) 52%, transparent 78%),
    linear-gradient(180deg, rgba(8, 8, 11, 0.62) 0%, transparent 14%, transparent 86%, rgba(8, 8, 11, 0.66) 100%);
}

.orbit-stage-auth .orbit-stage-veil {
  background:
    radial-gradient(ellipse 46% 42% at 50% 50%, rgba(6, 7, 9, 0.82) 0%, rgba(6, 7, 9, 0.52) 54%, transparent 82%),
    linear-gradient(180deg, rgba(6, 7, 9, 0.72) 0%, rgba(6, 7, 9, 0.24) 22%, rgba(6, 7, 9, 0.32) 78%, rgba(6, 7, 9, 0.78) 100%);
}

.orbit-stage-grain {
  position: absolute;
  inset: 0;
  opacity: 0.08;
  mix-blend-mode: overlay;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='140' height='140'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='3'/%3E%3C/filter%3E%3Crect width='140' height='140' filter='url(%23n)' opacity='0.55'/%3E%3C/svg%3E");
}

.orbit-stage-auth .orbit-stage-grain {
  opacity: 0.05;
}
</style>

<template>
  <svg
    v-if="iconInfo"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    xmlns="http://www.w3.org/2000/svg"
    class="model-icon"
    fill="currentColor"
    fill-rule="evenodd"
  >
    <path v-for="(p, idx) in iconInfo.paths" :key="idx" :d="p" :fill="iconInfo.color" />
  </svg>
  <span v-else class="model-icon-fallback" :style="{ width: size, height: size, fontSize: `calc(${size} * 0.5)` }">
    {{ fallbackText }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getModelIcon } from '@/constants/modelIconPaths'

const props = withDefaults(
  defineProps<{
    model: string
    size?: string
  }>(),
  {
    size: '18px'
  }
)

// 图标数据集中在 constants/modelIconPaths.ts，Canvas 3D 场景共用同一份
const iconInfo = computed(() => getModelIcon(props.model))
const fallbackText = computed(() => props.model.charAt(0).toUpperCase())
</script>

<style scoped>
.model-icon {
  flex-shrink: 0;
}
.model-icon-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: white;
  font-weight: 600;
  flex-shrink: 0;
}
</style>

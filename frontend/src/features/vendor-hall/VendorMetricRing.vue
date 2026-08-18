<template>
  <div class="vendor-ring" :style="ringStyle" :aria-label="`${label} ${displayValue}`">
    <div class="vendor-ring__center">
      <strong>{{ displayValue }}</strong>
      <span>{{ label }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: number | null; label: string; tone?: 'green' | 'blue' }>()

const normalized = computed(() => {
  if (props.value == null || !Number.isFinite(props.value)) return 0
  return Math.min(100, Math.max(0, props.value * 100))
})
const displayValue = computed(() => props.value == null ? '--' : `${normalized.value.toFixed(1)}%`)
const ringStyle = computed(() => ({
  '--ring-value': `${normalized.value * 3.6}deg`,
  '--ring-color': props.tone === 'blue' ? '#3b82f6' : '#10b981',
}))
</script>

<style scoped>
.vendor-ring { width: 76px; height: 76px; flex: none; border-radius: 50%; display: grid; place-items: center; background: conic-gradient(var(--ring-color) var(--ring-value), #e5e7eb 0); }
.vendor-ring__center { width: 60px; height: 60px; border-radius: 50%; display: grid; place-content: center; text-align: center; background: white; }
.vendor-ring strong { font-size: 13px; line-height: 1.1; color: #111827; }
.vendor-ring span { margin-top: 3px; font-size: 10px; color: #6b7280; }
:global(.dark .vendor-ring) { background: conic-gradient(var(--ring-color) var(--ring-value), #374151 0); }
:global(.dark .vendor-ring__center) { background: #1f2937; }
:global(.dark .vendor-ring strong) { color: #f9fafb; }
</style>

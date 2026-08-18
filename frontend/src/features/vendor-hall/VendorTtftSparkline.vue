<template>
  <div class="h-12 w-32" :data-test="`ttft-trend-${accountId}`">
    <svg v-if="path" class="h-full w-full overflow-visible" viewBox="0 0 128 48" role="img" :aria-label="label">
      <path :d="areaPath" fill="rgba(59, 130, 246, .12)" />
      <path :d="path" fill="none" stroke="#3b82f6" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
      <circle :cx="lastPoint?.x" :cy="lastPoint?.y" r="2.5" fill="#2563eb" />
    </svg>
    <div v-else class="grid h-full place-items-center text-xs text-gray-400">--</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { VendorHallTrendPoint } from '@/api/admin/vendorHall'

const props = defineProps<{ accountId: number; points: VendorHallTrendPoint[]; label: string }>()
const coordinates = computed(() => {
  const values = props.points.filter((item) => item.ttft_p95_ms != null)
  if (values.length < 2) return []
  const numbers = values.map((item) => item.ttft_p95_ms as number)
  const min = Math.min(...numbers)
  const max = Math.max(...numbers)
  const range = Math.max(max - min, 1)
  return numbers.map((value, index) => ({
    x: (index / (numbers.length - 1)) * 124 + 2,
    y: 44 - ((value - min) / range) * 38,
  }))
})
const path = computed(() => coordinates.value.map((point, index) => `${index ? 'L' : 'M'} ${point.x} ${point.y}`).join(' '))
const areaPath = computed(() => path.value ? `${path.value} L 126 46 L 2 46 Z` : '')
const lastPoint = computed(() => coordinates.value.at(-1))
</script>

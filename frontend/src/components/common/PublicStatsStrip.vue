<template>
  <section
    v-if="!failed"
    class="public-stats"
    data-testid="public-stats"
    :aria-busy="loading"
  >
    <dl class="public-stats-grid">
      <div class="public-stats-item">
        <dt>{{ copy.runningDays }}</dt>
        <dd>
          <span class="public-stats-value">{{ runningDaysText }}</span>
          <span class="public-stats-unit">{{ copy.dayUnit }}</span>
        </dd>
      </div>

      <div class="public-stats-item">
        <dt>{{ copy.totalRequests }}</dt>
        <dd>
          <span class="public-stats-value">{{ requestsText }}</span>
        </dd>
      </div>

      <div class="public-stats-item">
        <dt>{{ copy.totalTokens }}</dt>
        <dd>
          <span class="public-stats-value">{{ tokensText }}</span>
        </dd>
      </div>

      <div v-if="availabilityText !== null" class="public-stats-item">
        <dt>{{ copy.availability }}</dt>
        <dd>
          <span class="public-stats-value">{{ availabilityText }}</span>
        </dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPublicStats, type PublicStats } from '@/api/publicStats'

const COPY = {
  zh: {
    runningDays: '运行天数',
    totalRequests: '累计调用量',
    totalTokens: '累计 Token',
    availability: '近 30 天在线率',
    dayUnit: '天'
  },
  en: {
    runningDays: 'Days in service',
    totalRequests: 'Total requests',
    totalTokens: 'Total tokens',
    availability: '30-day uptime',
    dayUnit: 'days'
  }
} as const

const { locale } = useI18n()

const stats = ref<PublicStats | null>(null)
const loading = ref(true)
const failed = ref(false)

const isZh = computed(() => String(locale.value || '').toLowerCase().startsWith('zh'))
const copy = computed(() => (isZh.value ? COPY.zh : COPY.en))

const runningDaysText = computed(() => {
  if (loading.value || !stats.value) return '--'
  return String(stats.value.running_days)
})

const requestsText = computed(() => {
  if (loading.value) return '--'
  return formatCompact(stats.value?.total_requests ?? 0, isZh.value ? 'zh-CN' : 'en-US', isZh.value ? 0 : 1)
})

const tokensText = computed(() => {
  if (loading.value) return '--'
  return formatCompact(stats.value?.total_tokens ?? 0, isZh.value ? 'zh-CN' : 'en-US', isZh.value ? 0 : 1)
})

const availabilityText = computed(() => {
  if (loading.value) return '--'
  const value = stats.value?.availability_30d
  if (value === null || value === undefined) return null
  return `${(value * 100).toFixed(1)}%`
})

/**
 * 中文使用万/亿，英文使用 K/M/B；取整后追加 +，与公开口径一致。
 */
function formatCompact(value: number, numberLocale: string, fractionDigits: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  const formatted = new Intl.NumberFormat(numberLocale, {
    notation: 'compact',
    maximumFractionDigits: fractionDigits
  }).format(value)
  return `${formatted}+`
}

let cancelled = false

onMounted(async () => {
  try {
    const data = await getPublicStats()
    if (cancelled) return
    if (!data || typeof data.total_requests !== 'number' || typeof data.total_tokens !== 'number') {
      throw new Error('invalid public stats payload')
    }
    stats.value = data
  } catch {
    if (!cancelled) failed.value = true
  } finally {
    if (!cancelled) loading.value = false
  }
})

// 组件卸载后不再写入状态，避免测试或路由切换时产生无意义更新。
onBeforeUnmount(() => {
  cancelled = true
})
</script>

<style scoped>
.public-stats {
  margin-top: 1.25rem;
}

.public-stats-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
  margin: 0;
}

.public-stats-item {
  padding: 0.7rem 0.8rem;
  border: 1px solid rgba(148, 163, 255, 0.16);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.035);
  text-align: center;
}

.public-stats-item dt {
  color: #94a5c4;
  font-size: 11px;
  letter-spacing: 0.4px;
}

.public-stats-item dd {
  margin: 0.35rem 0 0;
  color: #f2f6ff;
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.public-stats-unit {
  margin-left: 2px;
  color: #94a5c4;
  font-size: 11px;
  font-weight: 400;
}

@media (prefers-reduced-motion: no-preference) {
  .public-stats {
    animation: public-stats-rise 0.45s cubic-bezier(0.2, 0.7, 0.2, 1) both;
  }
}

@keyframes public-stats-rise {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>

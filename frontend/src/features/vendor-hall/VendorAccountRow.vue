<template>
  <article class="vendor-account" :class="{ 'vendor-account--selected': selected }">
    <div class="vendor-account__grid">
      <label class="flex min-w-0 items-center gap-3">
        <input
          type="radio"
          name="vendor-account"
          class="h-4 w-4 border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="selected"
          :data-test="`select-account-${account.account_id}`"
          @change="$emit('select', account.account_id)"
        />
        <span class="vendor-logo">{{ platformInitial }}</span>
        <span class="min-w-0">
          <strong class="block truncate text-sm text-gray-900 dark:text-white">{{ account.account_name }}</strong>
          <span class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ account.platform }}</span>
            <span v-if="account.group_name" class="vendor-tag">{{ account.group_name }}</span>
            <span>#{{ account.account_id }}</span>
          </span>
        </span>
      </label>

      <div class="vendor-stat">
        <span>{{ t('admin.vendorHall.metrics.rateMultiplier') }}</span>
        <strong>{{ formatMultiplier(account.rate_multiplier) }}</strong>
      </div>
      <div class="vendor-stat">
        <span>{{ t('admin.vendorHall.metrics.userLatency') }}</span>
        <strong>{{ formatMs(account.average_latency_ms) }}</strong>
        <small>P95 {{ formatMs(account.p95_latency_ms) }}</small>
      </div>
      <VendorMetricRing :value="account.cache_hit_rate" :label="t('admin.vendorHall.metrics.cache')" tone="blue" />
      <VendorMetricRing :value="account.availability" :label="t('admin.vendorHall.metrics.availability')" />
      <div class="vendor-trend">
        <VendorTtftSparkline
          :account-id="account.account_id"
          :points="account.trend"
          :label="t('admin.vendorHall.metrics.userTtft')"
        />
        <strong>{{ formatMs(account.user_ttft_p95_ms) }}</strong>
        <span>{{ t('admin.vendorHall.metrics.userTtft') }}</span>
      </div>
      <div class="flex items-center justify-end gap-3">
        <span class="vendor-status" :class="`vendor-status--${account.scheduling_status}`">
          {{ t(`admin.vendorHall.status.${account.scheduling_status}`) }}
        </span>
        <button
          type="button"
          class="vendor-expand"
          :aria-expanded="expanded"
          :aria-label="t('admin.vendorHall.details')"
          @click="$emit('toggle')"
        >
          <Icon :name="expanded ? 'chevronUp' : 'chevronDown'" size="sm" />
        </button>
      </div>
    </div>

    <div v-if="expanded" class="vendor-account__details">
      <div><span>{{ t('admin.vendorHall.metrics.requests') }}</span><strong>{{ formatNumber(account.request_count) }}</strong></div>
      <div><span>{{ t('admin.vendorHall.metrics.updated') }}</span><strong>{{ formatDate(account.collected_at) }}</strong></div>
      <div><span>{{ t('admin.vendorHall.metrics.averageLatency') }}</span><strong>{{ formatMs(account.average_latency_ms) }}</strong></div>
      <div><span>{{ t('admin.vendorHall.metrics.userTtft') }}</span><strong>{{ formatMs(account.user_ttft_p95_ms) }}</strong></div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { VendorHallAccount } from '@/api/admin/vendorHall'
import VendorMetricRing from './VendorMetricRing.vue'
import VendorTtftSparkline from './VendorTtftSparkline.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ account: VendorHallAccount; selected: boolean; expanded: boolean }>()
defineEmits<{ select: [accountId: number]; toggle: [] }>()
const { t, locale } = useI18n()
const platformInitial = computed(() => (props.account.platform || props.account.account_name || '?').charAt(0).toUpperCase())
const formatMs = (value: number | null) => value == null ? '--' : `${Math.round(value).toLocaleString(locale.value)} ms`
const formatMultiplier = (value: number | null) => value == null ? '--' : `${value.toFixed(2)}x`
const formatNumber = (value: number) => value.toLocaleString(locale.value)
const formatDate = (value: string | null) => value ? new Intl.DateTimeFormat(locale.value, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : '--'
</script>

<style scoped>
.vendor-account { border-bottom: 1px solid rgb(229 231 235); background: white; transition: background-color .16s ease; }
.vendor-account:last-child { border-bottom: 0; }
.vendor-account:hover, .vendor-account--selected { background: rgb(248 250 252); }
.vendor-account--selected { box-shadow: inset 3px 0 #2563eb; }
.vendor-account__grid { display: grid; grid-template-columns: minmax(210px, 1.7fr) .7fr 1fr 82px 82px minmax(140px, 1fr) minmax(120px, .8fr); gap: 18px; min-height: 112px; align-items: center; padding: 16px 20px; }
.vendor-logo { display: grid; width: 36px; height: 36px; flex: none; place-items: center; border-radius: 7px; background: #111827; color: white; font-weight: 700; }
.vendor-tag { border-radius: 4px; background: #eff6ff; padding: 2px 5px; color: #2563eb; }
.vendor-stat { min-width: 0; }
.vendor-stat span, .vendor-trend span { display: block; font-size: 11px; color: #6b7280; }
.vendor-stat strong { display: block; margin-top: 5px; font-size: 15px; color: #111827; }
.vendor-stat small { display: block; margin-top: 3px; font-size: 10px; color: #9ca3af; }
.vendor-trend strong { font-size: 13px; color: #111827; }
.vendor-status { border-radius: 999px; padding: 4px 8px; font-size: 11px; white-space: nowrap; background: #f3f4f6; color: #4b5563; }
.vendor-status--schedulable { background: #ecfdf5; color: #047857; }
.vendor-status--paused { background: #fffbeb; color: #b45309; }
.vendor-status--disabled { background: #fef2f2; color: #b91c1c; }
.vendor-expand { display: grid; width: 28px; height: 28px; place-items: center; border-radius: 5px; color: #6b7280; transition: background-color .16s; }
.vendor-expand:hover { background: #e5e7eb; color: #111827; }
.vendor-account__details { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; border-top: 1px dashed #e5e7eb; padding: 14px 72px 18px; }
.vendor-account__details div { display: flex; flex-direction: column; gap: 3px; }
.vendor-account__details span { font-size: 11px; color: #6b7280; }
.vendor-account__details strong { font-size: 13px; color: #111827; }
:global(.dark .vendor-account) { border-color: #374151; background: #1f2937; }
:global(.dark .vendor-account:hover), :global(.dark .vendor-account--selected) { background: #253247; }
:global(.dark .vendor-stat strong), :global(.dark .vendor-trend strong), :global(.dark .vendor-account__details strong) { color: #f9fafb; }
:global(.dark .vendor-account__details) { border-color: #374151; }
@media (max-width: 1100px) { .vendor-account__grid { grid-template-columns: minmax(210px, 1.6fr) .7fr 1fr 82px 82px auto; } .vendor-trend { display: none; } }
@media (max-width: 760px) { .vendor-account__grid { grid-template-columns: minmax(0, 1fr) 76px 76px auto; gap: 10px; padding: 14px; } .vendor-stat, .vendor-trend { display: none; } .vendor-account__details { grid-template-columns: repeat(2, minmax(0, 1fr)); padding: 14px; } }
@media (max-width: 520px) { .vendor-account__grid { grid-template-columns: minmax(0, 1fr) auto; } :deep(.vendor-ring) { display: none; } }
</style>

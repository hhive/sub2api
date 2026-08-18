<template>
  <article class="vendor-account">
    <div class="vendor-account__grid">
      <div class="vendor-account__identity flex min-w-0 items-center gap-3">
        <span class="vendor-logo">{{ platformInitial }}</span>
        <span class="min-w-0">
          <strong class="block truncate text-sm text-gray-900 dark:text-white">{{ account.account_name }}</strong>
          <span class="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ account.platform }}</span>
            <span v-if="account.group_name" class="vendor-tag">{{ account.group_name }}</span>
            <span>#{{ account.account_id }}</span>
          </span>
        </span>
      </div>

      <div
        class="vendor-account__metrics-scroll"
        tabindex="0"
        :aria-label="t('admin.vendorHall.metrics.userTtft')"
      >
        <div class="vendor-account__metrics">
          <div class="vendor-stat vendor-stat--multiplier">
            <span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.rateMultiplier') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.rateMultiplier')" :description="t('admin.vendorHall.help.rateMultiplier')" /></span>
            <strong>{{ formatMultiplier(account.rate_multiplier) }}</strong>
          </div>
          <div class="vendor-stat vendor-stat--latency">
            <span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.userLatency') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.userLatency')" :description="t('admin.vendorHall.help.userLatency')" /></span>
            <strong>{{ formatMs(account.average_latency_ms) }}</strong>
            <small>P95 {{ formatMs(account.p95_latency_ms) }}</small>
          </div>
          <div class="vendor-metric vendor-metric--cache">
            <VendorMetricRing :value="account.cache_hit_rate" :label="t('admin.vendorHall.metrics.cache')" :help="t('admin.vendorHall.help.cache')" tone="blue" />
          </div>
          <div class="vendor-metric vendor-metric--availability">
            <VendorMetricRing :value="account.availability" :label="t('admin.vendorHall.metrics.availability')" :help="t('admin.vendorHall.help.availability')" />
          </div>
          <div class="vendor-trend">
            <VendorTtftSparkline
              :account-id="account.account_id"
              :points="account.trend"
              :label="t('admin.vendorHall.metrics.userTtft')"
            />
            <strong>{{ formatMs(account.user_ttft_p95_ms) }}</strong>
            <span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.userTtft') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.userTtft')" :description="t('admin.vendorHall.help.userTtft')" /></span>
          </div>
        </div>
      </div>
      <div class="vendor-account__actions flex items-center justify-end gap-3">
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
      <div><span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.requests') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.requests')" :description="t('admin.vendorHall.help.requests')" /></span><strong>{{ formatNumber(account.request_count) }}</strong></div>
      <div><span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.updated') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.updated')" :description="t('admin.vendorHall.help.updated')" /></span><strong>{{ formatDate(account.collected_at) }}</strong></div>
      <div><span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.averageLatency') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.averageLatency')" :description="t('admin.vendorHall.help.averageLatency')" /></span><strong>{{ formatMs(account.average_latency_ms) }}</strong></div>
      <div><span class="vendor-metric-label">{{ t('admin.vendorHall.metrics.userTtft') }}<VendorMetricHelp :label="t('admin.vendorHall.metrics.userTtft')" :description="t('admin.vendorHall.help.userTtft')" /></span><strong>{{ formatMs(account.user_ttft_p95_ms) }}</strong></div>
      <div class="vendor-account__detail-actions">
        <button type="button" class="btn btn-secondary" :data-test="`view-usage-account-${account.account_id}`" @click="$emit('usage')">
          {{ t('admin.vendorHall.actions.usage') }}
        </button>
        <button type="button" class="btn btn-secondary" :data-test="`manage-account-${account.account_id}`" @click="$emit('manage')">
          {{ t('admin.vendorHall.actions.manage') }}
        </button>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="actionLoading || account.scheduling_status === 'disabled'"
          :data-test="`pause-account-${account.account_id}`"
          @click="$emit('pause')"
        >
          {{ t('admin.vendorHall.actions.pause') }}
        </button>
        <button
          type="button"
          class="btn btn-danger"
          :disabled="actionLoading || account.scheduling_status === 'disabled'"
          :data-test="`disable-account-${account.account_id}`"
          @click="$emit('disable')"
        >
          {{ t('admin.vendorHall.actions.disable') }}
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { VendorHallAccount } from '@/api/admin/vendorHall'
import VendorMetricRing from './VendorMetricRing.vue'
import VendorMetricHelp from './VendorMetricHelp.vue'
import VendorTtftSparkline from './VendorTtftSparkline.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ account: VendorHallAccount; expanded: boolean; actionLoading: boolean }>()
defineEmits<{ toggle: []; usage: []; manage: []; pause: []; disable: [] }>()
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
.vendor-account:hover { background: rgb(248 250 252); }
.vendor-account__grid { display: grid; grid-template-areas: "identity metrics actions"; grid-template-columns: minmax(210px, 1.7fr) minmax(520px, 4.4fr) minmax(120px, .8fr); gap: 18px; min-height: 112px; align-items: center; padding: 16px 20px; }
.vendor-account__identity { grid-area: identity; }
.vendor-account__actions { grid-area: actions; }
.vendor-account__metrics-scroll { grid-area: metrics; min-width: 0; }
.vendor-account__metrics { display: grid; grid-template-areas: "multiplier latency cache availability ttft"; grid-template-columns: .7fr 1fr 82px 82px minmax(140px, 1fr); gap: 18px; align-items: center; }
.vendor-logo { display: grid; width: 36px; height: 36px; flex: none; place-items: center; border-radius: 7px; background: #111827; color: white; font-weight: 700; }
.vendor-tag { border-radius: 4px; background: #eff6ff; padding: 2px 5px; color: #2563eb; }
.vendor-stat { min-width: 0; }
.vendor-stat--multiplier { grid-area: multiplier; }
.vendor-stat--latency { grid-area: latency; }
.vendor-metric--cache { grid-area: cache; }
.vendor-metric--availability { grid-area: availability; }
.vendor-stat span, .vendor-trend span { display: block; font-size: 11px; color: #6b7280; }
.vendor-metric-label { display: flex !important; align-items: center; gap: 2px; }
.vendor-stat strong { display: block; margin-top: 5px; font-size: 15px; color: #111827; }
.vendor-stat small { display: block; margin-top: 3px; font-size: 10px; color: #9ca3af; }
.vendor-trend { grid-area: ttft; min-width: 0; }
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
.vendor-account__detail-actions { grid-column: 1 / -1; display: flex !important; flex-flow: row wrap !important; justify-content: flex-end; gap: 8px !important; border-top: 1px solid #e5e7eb; margin-top: 4px; padding-top: 12px; }
.vendor-account__detail-actions .btn { min-height: 34px; padding: 6px 10px; font-size: 12px; }
:global(.dark .vendor-account) { border-color: #374151; background: #1f2937; }
:global(.dark .vendor-account:hover) { background: #253247; }
:global(.dark .vendor-stat strong), :global(.dark .vendor-trend strong), :global(.dark .vendor-account__details strong) { color: #f9fafb; }
:global(.dark .vendor-account__details) { border-color: #374151; }
:global(.dark .vendor-account__detail-actions) { border-color: #374151; }
@media (max-width: 1100px) {
  .vendor-account__grid { grid-template-areas: "identity actions" "metrics metrics"; grid-template-columns: minmax(0, 1fr) auto; gap: 12px 16px; padding: 16px 20px 10px; }
  .vendor-account__metrics-scroll { margin: 0 -20px; overflow-x: auto; overscroll-behavior-inline: contain; padding: 2px 20px 10px; scrollbar-gutter: stable; scrollbar-width: thin; }
  .vendor-account__metrics { grid-template-areas: "ttft latency availability cache multiplier"; grid-template-columns: 220px 130px 86px 86px 90px; min-width: 680px; gap: 16px; }
  .vendor-trend { display: grid; grid-template-columns: 128px minmax(72px, 1fr); grid-template-rows: auto auto; column-gap: 10px; align-items: center; }
  .vendor-trend > :deep(div) { grid-row: 1 / 3; }
  .vendor-trend strong { align-self: end; }
  .vendor-trend span { align-self: start; }
}
@media (max-width: 760px) {
  .vendor-account__grid { gap: 10px 12px; padding: 14px 14px 8px; }
  .vendor-account__metrics-scroll { margin: 0 -14px; padding: 2px 14px 10px; }
  .vendor-account__details { grid-template-columns: repeat(2, minmax(0, 1fr)); padding: 14px; }
  .vendor-account__detail-actions { justify-content: stretch; }
  .vendor-account__detail-actions .btn { flex: 1 1 calc(50% - 4px); }
}
</style>

<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-3 lg:flex-row lg:items-center">
          <div class="grid min-w-0 flex-1 grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-[minmax(240px,1fr)_190px_190px]">
            <div class="relative min-w-0">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="search"
                data-testid="pricing-search"
                type="search"
                class="input w-full pl-10"
                :placeholder="t('admin.defaultModelPricing.search')"
                @input="scheduleLoad"
              />
            </div>
            <Select
              v-model="provider"
              data-testid="provider-filter"
              :options="providerOptions"
              @change="onFilterChange"
            />
            <Select
              v-model="mode"
              data-testid="mode-filter"
              :options="modeOptions"
              @change="onFilterChange"
            />
          </div>
          <button
            type="button"
            data-testid="pricing-refresh"
            class="btn btn-secondary h-10 w-10 flex-shrink-0 p-0"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="load"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <div class="border-b border-gray-200 px-4 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400 sm:px-5">
          {{ statusText }}
        </div>
        <div
          v-if="error"
          class="mx-4 mt-3 flex items-center justify-between gap-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300 sm:mx-5"
          role="alert"
        >
          <span>{{ t('admin.defaultModelPricing.loadError') }}</span>
          <button type="button" class="font-medium underline" @click="load">
            {{ t('common.refresh') }}
          </button>
        </div>
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          row-key="model"
          :server-side-sort="true"
          default-sort-key="model"
          default-sort-order="asc"
          @sort="handleSort"
        >
          <template #cell-model="{ value }">
            <span class="block max-w-[320px] whitespace-normal break-all font-medium text-gray-900 dark:text-white" :title="value">
              {{ value }}
            </span>
          </template>
          <template #cell-input="{ row }">{{ formatTokenPrice(row.input_cost_per_token) }}</template>
          <template #cell-output="{ row }">{{ formatTokenPrice(row.output_cost_per_token) }}</template>
          <template #cell-cache_write="{ row }">{{ formatTokenPrice(row.cache_creation_input_token_cost) }}</template>
          <template #cell-cache_read="{ row }">{{ formatTokenPrice(row.cache_read_input_token_cost) }}</template>
          <template #cell-image="{ row }">{{ formatImagePrice(row.output_cost_per_image) }}</template>
          <template #cell-details="{ row }">
            <div class="min-w-[260px] whitespace-normal">
              <button
                type="button"
                class="inline-flex h-8 w-8 items-center justify-center rounded text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                :title="expanded.has(row.model) ? t('common.collapse') : t('common.expand')"
                @click="toggleDetails(row.model)"
              >
                <Icon :name="expanded.has(row.model) ? 'chevronUp' : 'chevronDown'" size="sm" />
              </button>
              <dl v-if="expanded.has(row.model)" class="mt-2 grid grid-cols-[minmax(120px,auto)_minmax(100px,1fr)] gap-x-3 gap-y-1 text-xs">
                <template v-for="detail in detailRows(row)" :key="detail.label">
                  <dt class="text-gray-500 dark:text-dark-400" :title="detail.hint">{{ detail.label }}</dt>
                  <dd class="break-all text-gray-800 dark:text-gray-200">{{ detail.value }}</dd>
                </template>
              </dl>
            </div>
          </template>
          <template #empty>
            <div class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">{{ emptyText }}</div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="total > 0"
          :page="page"
          :total="total"
          :page-size="pageSize"
          @update:page="onPage"
          @update:pageSize="onPageSize"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { DefaultModelPricingItem } from '@/api/admin/defaultModelPricing'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatImagePrice, formatTokenPrice } from './defaultModelPricingFormat'

const { t } = useI18n()
const items = ref<DefaultModelPricingItem[]>([])
const providers = ref<string[]>([])
const modes = ref<string[]>([])
const page = ref(1)
const pageSize = ref(getPersistedPageSize())
const total = ref(0)
const search = ref('')
const provider = ref('')
const mode = ref('')
const sortBy = ref<'model' | 'provider' | 'mode'>('model')
const sortOrder = ref<'asc' | 'desc'>('asc')
const loading = ref(false)
const error = ref(false)
const status = ref({ model_count: 0, last_updated: null as string | null, local_hash: '' })
const expanded = ref(new Set<string>())

let searchTimer: ReturnType<typeof setTimeout> | undefined
let activeController: AbortController | undefined
let requestID = 0

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.defaultModelPricing.columns.model'), sortable: true },
  { key: 'provider', label: t('admin.defaultModelPricing.columns.provider'), sortable: true },
  { key: 'mode', label: t('admin.defaultModelPricing.columns.mode'), sortable: true },
  { key: 'input', label: t('admin.defaultModelPricing.columns.input') },
  { key: 'output', label: t('admin.defaultModelPricing.columns.output') },
  { key: 'cache_write', label: t('admin.defaultModelPricing.columns.cacheWrite') },
  { key: 'cache_read', label: t('admin.defaultModelPricing.columns.cacheRead') },
  { key: 'image', label: t('admin.defaultModelPricing.columns.image') },
  { key: 'details', label: t('admin.defaultModelPricing.columns.details') }
])

const providerOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.defaultModelPricing.allProviders') },
  ...providers.value.map(value => ({ value, label: value }))
])

const modeOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.defaultModelPricing.allModes') },
  ...modes.value.map(value => ({ value, label: value }))
])

const emptyText = computed(() => {
  if (loading.value) return ''
  return search.value || provider.value || mode.value
    ? t('admin.defaultModelPricing.noResults')
    : t('admin.defaultModelPricing.empty')
})

const statusText = computed(() => t('admin.defaultModelPricing.status', {
  count: status.value.model_count,
  updated: formatDate(status.value.last_updated),
  hash: status.value.local_hash || '-'
}))

function formatDate(value: string | null): string {
  return value ? new Date(value).toLocaleString() : '-'
}

function detailRows(row: DefaultModelPricingItem): Array<{ label: string; value: string; hint: string }> {
  const priorityHint = t('admin.defaultModelPricing.hints.priority')
  const longContextHint = t('admin.defaultModelPricing.hints.longContext')
  return [
    { label: t('admin.defaultModelPricing.details.priorityInput'), value: formatTokenPrice(row.input_cost_per_token_priority), hint: priorityHint },
    { label: t('admin.defaultModelPricing.details.priorityOutput'), value: formatTokenPrice(row.output_cost_per_token_priority), hint: priorityHint },
    { label: t('admin.defaultModelPricing.details.priorityCacheWrite'), value: formatTokenPrice(row.cache_creation_input_token_cost_priority), hint: priorityHint },
    { label: t('admin.defaultModelPricing.details.priorityCacheRead'), value: formatTokenPrice(row.cache_read_input_token_cost_priority), hint: priorityHint },
    { label: t('admin.defaultModelPricing.details.cacheOneHour'), value: formatTokenPrice(row.cache_creation_input_token_cost_above_1hr), hint: t('admin.defaultModelPricing.hints.cacheOneHour') },
    { label: t('admin.defaultModelPricing.details.imageOutputToken'), value: formatTokenPrice(row.output_cost_per_image_token), hint: t('admin.defaultModelPricing.hints.imageOutputToken') },
    { label: t('admin.defaultModelPricing.details.longContextThreshold'), value: row.long_context_input_token_threshold?.toLocaleString() ?? '-', hint: longContextHint },
    { label: t('admin.defaultModelPricing.details.longContextInputMultiplier'), value: formatMultiplier(row.long_context_input_cost_multiplier), hint: longContextHint },
    { label: t('admin.defaultModelPricing.details.longContextOutputMultiplier'), value: formatMultiplier(row.long_context_output_cost_multiplier), hint: longContextHint },
    { label: t('admin.defaultModelPricing.details.serviceTier'), value: formatBoolean(row.supports_service_tier), hint: t('admin.defaultModelPricing.hints.capabilities') },
    { label: t('admin.defaultModelPricing.details.promptCaching'), value: formatBoolean(row.supports_prompt_caching), hint: t('admin.defaultModelPricing.hints.capabilities') },
    { label: t('admin.defaultModelPricing.details.tokenPricingAbsent'), value: formatBoolean(row.token_pricing_absent), hint: t('admin.defaultModelPricing.hints.tokenPricingAbsent') }
  ]
}

function formatMultiplier(value: number | null): string {
  return value === null ? '-' : `${value}x`
}

function formatBoolean(value: boolean): string {
  return value ? t('common.yes') : t('common.no')
}

async function load(): Promise<void> {
  const id = ++requestID
  activeController?.abort()
  const controller = new AbortController()
  activeController = controller
  loading.value = true
  error.value = false

  try {
    const result = await adminAPI.defaultModelPricing.list({
      page: page.value,
      page_size: pageSize.value,
      search: search.value || undefined,
      provider: provider.value || undefined,
      mode: mode.value || undefined,
      sort_by: sortBy.value,
      sort_order: sortOrder.value,
      signal: controller.signal
    })
    if (id !== requestID) return
    items.value = result.items
    total.value = result.total
    providers.value = result.providers
    modes.value = result.modes
    status.value = result.status
  } catch {
    if (id === requestID && !controller.signal.aborted) error.value = true
  } finally {
    if (id === requestID) loading.value = false
  }
}

function scheduleLoad(): void {
  page.value = 1
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(load, 300)
}

function onFilterChange(): void {
  page.value = 1
  load()
}

function toggleDetails(model: string): void {
  const next = new Set(expanded.value)
  next.has(model) ? next.delete(model) : next.add(model)
  expanded.value = next
}

function onPage(value: number): void {
  page.value = value
  load()
}

function onPageSize(value: number): void {
  pageSize.value = value
  page.value = 1
  load()
}

function handleSort(key: string, order: 'asc' | 'desc'): void {
  if (key !== 'model' && key !== 'provider' && key !== 'mode') return
  sortBy.value = key
  sortOrder.value = order
  page.value = 1
  load()
}

onMounted(load)
onUnmounted(() => {
  requestID++
  activeController?.abort()
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

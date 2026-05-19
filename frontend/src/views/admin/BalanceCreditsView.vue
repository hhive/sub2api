<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-80">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />
            <input
              v-model="filters.search"
              type="email"
              class="input pl-10"
              :placeholder="t('admin.balanceCredits.searchPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </div>
          <div class="w-full sm:w-44">
            <Select v-model="filters.source_type" :options="sourceOptions" @change="applyFilters" />
          </div>
          <div class="w-full sm:w-44">
            <Select v-model="filters.status" :options="statusOptions" @change="applyFilters" />
          </div>
          <div class="ml-auto flex items-center gap-2">
            <button class="btn btn-secondary" :disabled="loading || settling" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
            <button class="btn btn-secondary" :disabled="loading || settling" @click="runSettlement">
              <Icon name="refresh" size="sm" :class="settling ? 'animate-spin' : ''" />
              <span>{{ t('admin.balanceCredits.settleNow') }}</span>
            </button>
            <button class="btn btn-primary" :disabled="loading || settling" @click="loadCredits">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              <span>{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-3">
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.balanceCredits.totalAmount') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatCurrency(summary.total_amount) }}
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.balanceCredits.todayAmount') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-blue-600 dark:text-blue-400">
            {{ formatCurrency(summary.today_amount) }}
          </div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
            {{ t('admin.balanceCredits.totalRemaining') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
            {{ formatCurrency(summary.total_remaining) }}
          </div>
        </div>
      </div>

      <DataTable
        :columns="columns"
        :data="credits"
        :loading="loading"
        :server-side-sort="true"
        default-sort-key="created_at"
        default-sort-order="desc"
        :sort-storage-key="BALANCE_CREDITS_SORT_STORAGE_KEY"
        row-key="id"
        :sticky-first-column="false"
        :sticky-actions-column="false"
        @sort="handleSort"
      >
        <template #empty>
          <div class="flex flex-col items-center">
            <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('admin.balanceCredits.empty') }}</p>
          </div>
        </template>

        <template #cell-email="{ row }">
          <div class="whitespace-nowrap text-left">
            <div class="text-sm font-medium text-gray-900 dark:text-white">{{ row.email || '-' }}</div>
            <div class="text-xs text-gray-500 dark:text-dark-400">ID: {{ row.user_id }}</div>
          </div>
        </template>

        <template #cell-source_type="{ row }">
          <div class="whitespace-nowrap text-left">
            <div class="text-sm font-medium text-gray-900 dark:text-white">{{ sourceLabel(row.source_type) }}</div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ row.source_code || row.source_id || '-' }}</div>
          </div>
        </template>

        <template #cell-amount="{ value }">
          <span class="font-medium text-gray-900 dark:text-white">{{ formatCurrency(value) }}</span>
        </template>

        <template #cell-remaining_amount="{ value }">
          <span class="font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(value) }}</span>
        </template>

        <template #cell-status="{ value }">
          <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="statusClass(value)">
            {{ statusLabel(value) }}
          </span>
        </template>

        <template #cell-created_at="{ value }">
          <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(value) }}</span>
        </template>

        <template #cell-expires_at="{ value }">
          <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">
            {{ value ? formatDateTime(value) : t('admin.users.ledgerNeverExpires') }}
          </span>
        </template>

        <template #cell-settled_until_date="{ value }">
          <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">{{ formatDate(value) }}</span>
        </template>
      </DataTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { UserBalanceCredit } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const settling = ref(false)
const credits = ref<UserBalanceCredit[]>([])
const filters = reactive({
  search: '',
  source_type: '',
  status: ''
})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0
})
const summary = reactive({
  total_amount: 0,
  today_amount: 0,
  total_remaining: 0
})
const BALANCE_CREDITS_SORT_STORAGE_KEY = 'balance-credits-sort'
const sortableKeys = new Set(['email', 'source_type', 'amount', 'remaining_amount', 'status', 'created_at', 'expires_at', 'settled_until_date'])
const loadInitialSortState = (): { sort_by: string; sort_order: 'asc' | 'desc' } => {
  const fallback = { sort_by: 'created_at', sort_order: 'desc' as 'asc' | 'desc' }
  try {
    const raw = localStorage.getItem(BALANCE_CREDITS_SORT_STORAGE_KEY)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as { key?: string; order?: string }
    if (!parsed.key || !sortableKeys.has(parsed.key)) return fallback
    return {
      sort_by: parsed.key,
      sort_order: parsed.order === 'asc' ? 'asc' : 'desc'
    }
  } catch {
    return fallback
  }
}
const sortState = reactive(loadInitialSortState())

const columns = computed<Column[]>(() => [
  { key: 'email', label: t('admin.balanceCredits.user'), sortable: true },
  { key: 'source_type', label: t('admin.users.ledgerSource'), sortable: true },
  { key: 'amount', label: t('admin.users.ledgerAmount'), sortable: true, class: 'text-right' },
  { key: 'remaining_amount', label: t('admin.users.ledgerRemaining'), sortable: true, class: 'text-right' },
  { key: 'status', label: t('admin.users.ledgerStatus'), sortable: true },
  { key: 'created_at', label: t('admin.balanceCredits.createdAt'), sortable: true },
  { key: 'expires_at', label: t('admin.users.ledgerExpiry'), sortable: true },
  { key: 'settled_until_date', label: t('admin.users.ledgerSettlement'), sortable: true }
])

const sourceOptions = computed(() => [
  { value: '', label: t('admin.balanceCredits.allSources') },
  { value: 'redeem', label: t('admin.users.ledgerSourceRedeem') },
  { value: 'admin', label: t('admin.users.ledgerSourceAdmin') },
  { value: 'promo', label: t('admin.users.ledgerSourcePromo') },
  { value: 'affiliate', label: t('admin.users.ledgerSourceAffiliate') },
  { value: 'initial', label: t('admin.users.ledgerSourceInitial') }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.balanceCredits.allStatuses') },
  { value: 'active', label: t('admin.users.ledgerStatusActive') },
  { value: 'consumed', label: t('admin.users.ledgerStatusConsumed') },
  { value: 'expired', label: t('admin.users.ledgerStatusExpired') }
])

function sourceLabel(sourceType: string): string {
  const found = sourceOptions.value.find((item) => item.value === sourceType)
  return found?.label || sourceType
}

function statusLabel(status: string): string {
  const found = statusOptions.value.find((item) => item.value === status)
  return found?.label || status
}

function statusClass(status: string): string {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  if (status === 'consumed') return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
  if (status === 'expired') return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
}

function formatDate(value: string | null): string {
  if (!value) return '-'
  return value.slice(0, 10)
}

function formatCurrency(value: number): string {
  return `$${Number(value || 0).toFixed(2)}`
}

async function loadCredits(): Promise<void> {
  loading.value = true
  try {
    const res = await adminAPI.users.listBalanceCredits(pagination.page, pagination.page_size, {
      search: filters.search.trim() || undefined,
      source_type: filters.source_type || undefined,
      status: filters.status || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order
    })
    credits.value = res.items
    pagination.total = res.total
    pagination.page = res.page
    pagination.page_size = res.page_size
    summary.total_amount = res.total_amount
    summary.today_amount = res.today_amount
    summary.total_remaining = res.total_remaining
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || t('admin.balanceCredits.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function runSettlement(): Promise<void> {
  settling.value = true
  try {
    const result = await adminAPI.users.settleBalanceCredits()
    appStore.showSuccess(t('admin.balanceCredits.settleSuccess', { date: result.settlement_date }))
    await loadCredits()
  } catch (error: any) {
    appStore.showError(error?.response?.data?.message || t('admin.balanceCredits.settleFailed'))
  } finally {
    settling.value = false
  }
}

function applyFilters(): void {
  pagination.page = 1
  loadCredits()
}

function resetFilters(): void {
  filters.search = ''
  filters.source_type = ''
  filters.status = ''
  applyFilters()
}

function handlePageChange(page: number): void {
  pagination.page = page
  loadCredits()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  loadCredits()
}

function handleSort(key: string, order: 'asc' | 'desc'): void {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadCredits()
}

onMounted(loadCredits)
</script>

<template>
  <AppLayout>
    <div class="vendor-hall space-y-5 pb-12">
      <header class="vendor-hero">
        <div>
          <div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-[.08em] text-primary-600 dark:text-primary-400">
            <span class="h-2 w-2 rounded-full bg-emerald-500"></span>{{ t('admin.vendorHall.live') }}
          </div>
          <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">{{ t('admin.vendorHall.title') }}</h1>
          <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ t('admin.vendorHall.description') }}</p>
        </div>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadData">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />{{ t('common.refresh') }}
        </button>
      </header>

      <section class="grid grid-cols-2 gap-3 md:grid-cols-4" aria-label="summary">
        <div class="vendor-summary"><span>{{ t('admin.vendorHall.summary.total') }}</span><strong>{{ summary.total_accounts }}</strong><small>{{ t('admin.vendorHall.summary.accounts') }}</small></div>
        <div class="vendor-summary"><span>{{ t('admin.vendorHall.summary.healthy') }}</span><strong class="text-emerald-600">{{ summary.healthy_accounts }}</strong><small>{{ t('admin.vendorHall.summary.running') }}</small></div>
        <div class="vendor-summary"><span>{{ t('admin.vendorHall.summary.paused') }}</span><strong class="text-amber-600">{{ summary.paused_accounts }}</strong><small>{{ t('admin.vendorHall.summary.manual') }}</small></div>
        <div class="vendor-summary"><span>{{ t('admin.vendorHall.summary.availability') }}</span><strong>{{ formatPercent(summary.average_availability) }}</strong><small>{{ formatUpdated(summary.updated_at) }}</small></div>
      </section>

      <section class="vendor-toolbar">
        <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
          <div class="vendor-window" role="group" :aria-label="t('admin.vendorHall.window')">
            <button v-for="option in windows" :key="option.value" type="button" :class="{ active: window === option.value }" @click="window = option.value; applyFilters()">{{ option.label }}</button>
          </div>
          <input v-model="search" class="input vendor-search" :placeholder="t('admin.vendorHall.search')" @keyup.enter="applyFilters" />
          <select v-model="status" class="input vendor-select" @change="applyFilters"><option value="">{{ t('admin.vendorHall.allStatuses') }}</option><option value="schedulable">{{ t('admin.vendorHall.status.schedulable') }}</option><option value="paused">{{ t('admin.vendorHall.status.paused') }}</option><option value="disabled">{{ t('admin.vendorHall.status.disabled') }}</option></select>
        </div>
        <div class="flex items-center gap-2">
          <select v-model="sortBy" class="input vendor-select" @change="applyFilters"><option value="availability">{{ t('admin.vendorHall.sort.availability') }}</option><option value="cache_hit_rate">{{ t('admin.vendorHall.sort.cache') }}</option><option value="user_ttft">{{ t('admin.vendorHall.sort.ttft') }}</option><option value="requests">{{ t('admin.vendorHall.sort.requests') }}</option></select>
        </div>
      </section>

      <section class="vendor-table overflow-hidden rounded-xl border border-gray-200 shadow-sm dark:border-dark-700">
        <div class="vendor-table__head"><span>{{ t('admin.vendorHall.columns.account') }}</span><span>{{ t('admin.vendorHall.columns.multiplier') }}</span><span>{{ t('admin.vendorHall.columns.latency') }}</span><span>{{ t('admin.vendorHall.columns.cache') }}</span><span>{{ t('admin.vendorHall.columns.availability') }}</span><span>{{ t('admin.vendorHall.columns.ttft') }}</span><span>{{ t('admin.vendorHall.columns.status') }}</span></div>
        <div v-if="loading" class="p-12 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
        <div v-else-if="error" class="p-12 text-center text-sm text-red-600">{{ error }}</div>
        <div v-else-if="accounts.length === 0" class="p-12 text-center text-sm text-gray-500">{{ t('admin.vendorHall.empty') }}</div>
        <VendorAccountRow
          v-for="account in accounts"
          v-else
          :key="account.account_id"
          :account="account"
          :homepage-url="homepageUrls.get(account.account_id)"
          :expanded="expandedId === account.account_id"
          :action-loading="actionLoading"
          @toggle="expandedId = expandedId === account.account_id ? null : account.account_id"
          @usage="goToUsage(account)"
          @manage="goToAccount(account)"
          @pause="openConfirmFor(account, 'pause')"
          @disable="openConfirmFor(account, 'disable')"
          @enable="openConfirmFor(account, 'enable')"
        />
        <Pagination v-if="!loading && !error && total > pageSize" :page="page" :total="total" :page-size="pageSize" :show-page-size-selector="false" @update:page="page = $event; loadData()" />
      </section>

      <ConfirmDialog :show="Boolean(confirmAction)" :title="confirmTitle" :message="confirmMessage" :danger="confirmAction === 'disable'" :confirm-text="t('common.confirm')" @confirm="performAction" @cancel="confirmAction = null" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import VendorAccountRow from '@/features/vendor-hall/VendorAccountRow.vue'
import type { VendorHallAccount, VendorHallSummary, VendorHallWindow, VendorHallSort } from '@/api/admin/vendorHall'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const windows: Array<{ value: VendorHallWindow; label: string }> = [{ value: '3h', label: '3H' }, { value: '24h', label: '24H' }, { value: '3d', label: '3D' }]
const window = ref<VendorHallWindow>('3h')
const search = ref('')
const status = ref('')
const sortBy = ref<VendorHallSort>('availability')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const accounts = ref<VendorHallAccount[]>([])
const summary = ref<VendorHallSummary>({ total_accounts: 0, healthy_accounts: 0, paused_accounts: 0, average_availability: null, updated_at: null })
const loading = ref(false)
const actionLoading = ref(false)
const error = ref('')
const expandedId = ref<number | null>(null)
const confirmAction = ref<'pause' | 'disable' | 'enable' | null>(null)
const actionAccount = ref<VendorHallAccount | null>(null)
const schedulingOverrides = new Map<number, VendorHallAccount['scheduling_status']>()
const homepageUrls = ref(new Map<number, string>())
const confirmTitle = computed(() => confirmAction.value === 'disable' ? t('admin.vendorHall.confirm.disableTitle') : confirmAction.value === 'enable' ? t('admin.vendorHall.confirm.enableTitle') : t('admin.vendorHall.confirm.pauseTitle'))
const confirmMessage = computed(() => confirmAction.value === 'disable' ? t('admin.vendorHall.confirm.disableMessage') : confirmAction.value === 'enable' ? t('admin.vendorHall.confirm.enableMessage') : t('admin.vendorHall.confirm.pauseMessage'))

const loadData = async () => {
  loading.value = true; error.value = ''
  try {
    const [vendorResult, accountResult] = await Promise.allSettled([
      adminAPI.vendorHall.list({ window: window.value, search: search.value.trim() || undefined, status: status.value || undefined, sort_by: sortBy.value, sort_order: 'desc', page: page.value, page_size: pageSize }),
      typeof adminAPI.accounts?.list === 'function'
        ? adminAPI.accounts.list(1, 1000, { lite: 'true' })
        : Promise.reject(new Error('Account metadata unavailable')),
    ])
    if (vendorResult.status === 'rejected') throw vendorResult.reason
    const result = vendorResult.value
    const nextHomepageUrls = new Map<number, string>()
    if (accountResult.status === 'fulfilled') {
      for (const account of accountResult.value.items || []) {
        if (account.type !== 'apikey' || typeof account.credentials?.base_url !== 'string') continue
        const baseUrl = sanitizeUrl(account.credentials.base_url)
        if (!baseUrl) continue
        try { nextHomepageUrls.set(account.id, new URL(baseUrl).origin) } catch { /* Ignore malformed URLs. */ }
      }
    }
    homepageUrls.value = nextHomepageUrls
    const rawItems = result.items || []
    const mergedItems = rawItems.map((item) => {
      const overriddenStatus = schedulingOverrides.get(item.account_id)
      if (!overriddenStatus) return item
      if (item.scheduling_status === overriddenStatus) schedulingOverrides.delete(item.account_id)
      return { ...item, scheduling_status: overriddenStatus }
    })
    accounts.value = status.value ? mergedItems.filter((item) => item.scheduling_status === status.value) : mergedItems
    total.value = Math.max(0, (result.total || 0) - (rawItems.length - accounts.value.length))
    const nextSummary = { ...(result.summary || summary.value) }
    for (const item of rawItems) {
      const overriddenStatus = schedulingOverrides.get(item.account_id)
      if (!overriddenStatus || overriddenStatus === item.scheduling_status) continue
      if (item.scheduling_status === 'schedulable') nextSummary.healthy_accounts = Math.max(0, nextSummary.healthy_accounts - 1)
      if (item.scheduling_status === 'paused') nextSummary.paused_accounts = Math.max(0, nextSummary.paused_accounts - 1)
      if (overriddenStatus === 'schedulable') nextSummary.healthy_accounts += 1
      if (overriddenStatus === 'paused') nextSummary.paused_accounts += 1
    }
    summary.value = nextSummary
  } catch (err) {
    error.value = extractApiErrorMessage(err, t('admin.vendorHall.failed'))
  } finally { loading.value = false }
}
const applyFilters = () => { page.value = 1; void loadData() }
const openConfirmFor = (account: VendorHallAccount, action: 'pause' | 'disable' | 'enable') => {
  if (actionLoading.value) return
  actionAccount.value = account
  confirmAction.value = action
}
const performAction = async () => {
  const account = actionAccount.value
  const action = confirmAction.value
  if (!account || !action || actionLoading.value) return
  actionLoading.value = true; confirmAction.value = null
  try {
    if (action === 'pause') {
      await adminAPI.vendorHall.pauseScheduling(account.account_id)
      schedulingOverrides.set(account.account_id, 'paused')
    } else {
      await adminAPI.accounts.setSchedulable(account.account_id, action === 'enable')
      schedulingOverrides.set(account.account_id, action === 'enable' ? 'schedulable' : 'disabled')
    }
    appStore.showSuccess(t(action === 'pause' ? 'admin.vendorHall.success.paused' : action === 'enable' ? 'admin.vendorHall.success.enabled' : 'admin.vendorHall.success.disabled'))
    await loadData()
  } catch (err) { appStore.showError(extractApiErrorMessage(err, t('admin.vendorHall.failed'))) }
  finally { actionLoading.value = false }
}
const goToUsage = (account: VendorHallAccount) => { void router.push({ path: '/admin/usage', query: { account_id: String(account.account_id) } }) }
const goToAccount = (account: VendorHallAccount) => { void router.push({ path: '/admin/accounts', query: { account_id: String(account.account_id) } }) }
const formatPercent = (value: number | null) => value == null ? '--' : `${(value * 100).toFixed(1)}%`
const formatUpdated = (value: string | null) => value ? new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : '--'
onMounted(loadData)
</script>

<style scoped>
.vendor-hero { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; border-bottom: 1px solid rgb(229 231 235); padding-bottom: 20px; }
.vendor-summary { min-height: 108px; border: 1px solid rgb(229 231 235); border-radius: 10px; background: white; padding: 17px 18px; }
.vendor-summary span, .vendor-summary small { display: block; font-size: 12px; color: #6b7280; }
.vendor-summary strong { display: block; margin-top: 9px; font-size: 24px; line-height: 1; color: #111827; }
.vendor-summary small { margin-top: 8px; font-size: 11px; }
.vendor-toolbar { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 12px; border: 1px solid rgb(229 231 235); border-radius: 10px; background: white; padding: 12px; }
.vendor-window { display: inline-flex; border: 1px solid #d1d5db; border-radius: 6px; overflow: hidden; }
.vendor-window button { min-width: 46px; padding: 7px 10px; font-size: 12px; color: #6b7280; }
.vendor-window button + button { border-left: 1px solid #e5e7eb; }
.vendor-window button.active { background: #eff6ff; color: #2563eb; font-weight: 600; }
.vendor-search { width: 180px; }
.vendor-select { width: 150px; min-width: 145px; flex: none; }
.vendor-table__head { display: grid; grid-template-columns: minmax(210px, 1.7fr) .7fr 1fr 82px 82px minmax(140px, 1fr) minmax(120px, .8fr); gap: 18px; padding: 11px 20px; background: #f8fafc; color: #6b7280; font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: .05em; }
:global(.dark .vendor-hero) { border-color: #374151; }
:global(.dark .vendor-summary), :global(.dark .vendor-toolbar) { border-color: #374151; background: #1f2937; }
:global(.dark .vendor-summary strong) { color: #f9fafb; }
:global(.dark .vendor-table__head) { background: #111827; }
@media (max-width: 1100px) { .vendor-table__head { display: none; } }
@media (max-width: 760px) { .vendor-hero { align-items: flex-start; flex-direction: column; } .vendor-toolbar > div:last-child { width: 100%; } .vendor-toolbar > div:last-child .vendor-select { width: 100%; } .vendor-search { width: 100%; } .vendor-select { width: auto; min-width: 0; flex: 1; } }
</style>

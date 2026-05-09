<template>
  <BaseDialog :show="show" :title="t('admin.users.balanceLedgerTitle')" width="wide" :close-on-click-outside="true" :z-index="40" @close="$emit('close')">
    <div v-if="user" class="space-y-4">
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
            <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
              {{ user.email.charAt(0).toUpperCase() }}
            </span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
              <span
                v-if="user.username"
                class="flex-shrink-0 rounded bg-primary-50 px-1.5 py-0.5 text-xs text-primary-600 dark:bg-primary-900/20 dark:text-primary-400"
              >
                {{ user.username }}
              </span>
            </div>
            <p class="text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.users.createdAt') }}: {{ formatDateTime(user.created_at) }}
            </p>
          </div>
          <div class="flex-shrink-0 text-right">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.currentBalance') }}</p>
            <p class="text-xl font-bold text-gray-900 dark:text-white">
              ${{ user.balance?.toFixed(2) || '0.00' }}
            </p>
          </div>
        </div>
      </div>

      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <div v-else-if="credits.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('admin.users.noBalanceLedger') }}</p>
      </div>

      <div v-else class="max-h-[32rem] overflow-y-auto rounded-xl border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-600">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerSource') }}</th>
              <th class="px-4 py-3 text-right font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerAmount') }}</th>
              <th class="px-4 py-3 text-right font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerRemaining') }}</th>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerStatus') }}</th>
              <th v-if="showSettlementColumn" class="px-4 py-3 text-left font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerSettlement') }}</th>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-dark-300">{{ t('admin.users.ledgerExpiry') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
            <tr v-for="credit in credits" :key="credit.id">
              <td class="px-4 py-3">
                <div class="font-medium text-gray-900 dark:text-white">{{ sourceLabel(credit.source_type) }}</div>
                <div class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                  {{ formatDateTime(credit.created_at) }}
                </div>
              </td>
              <td class="px-4 py-3 text-right font-semibold text-emerald-600 dark:text-emerald-400">
                ${{ credit.amount.toFixed(2) }}
              </td>
              <td class="px-4 py-3 text-right">
                <div class="font-semibold text-gray-900 dark:text-white">${{ credit.remaining_amount.toFixed(2) }}</div>
                <div class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('admin.users.ledgerConsumed') }} ${{ consumedAmount(credit).toFixed(2) }}
                </div>
              </td>
              <td class="px-4 py-3">
                <span :class="['inline-flex rounded-md px-2 py-1 text-xs font-medium', statusClass(credit.status)]">
                  {{ statusLabel(credit.status) }}
                </span>
              </td>
              <td v-if="showSettlementColumn" class="px-4 py-3 text-gray-500 dark:text-dark-300">
                {{ formatSettledDate(credit.settled_until_date) }}
              </td>
              <td class="px-4 py-3 text-gray-500 dark:text-dark-300">
                <div>{{ formatNullableDate(credit.expires_at) }}</div>
                <div v-if="credit.expired_at" class="mt-0.5 text-xs text-red-500 dark:text-red-400">
                  {{ t('admin.users.ledgerExpiredAt') }}: {{ formatDateTime(credit.expired_at) }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 pt-2">
        <button
          :disabled="currentPage <= 1"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadCredits(currentPage - 1)"
        >
          {{ t('pagination.previous') }}
        </button>
        <span class="text-sm text-gray-500 dark:text-dark-400">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          :disabled="currentPage >= totalPages"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadCredits(currentPage + 1)"
        >
          {{ t('pagination.next') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { userAPI, type UserBalanceCredit } from '@/api/user'
import { formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'

interface BalanceLedgerUser {
  id: number
  email: string
  username?: string
  balance?: number
  created_at: string
}

const props = withDefaults(defineProps<{
  show: boolean
  user: BalanceLedgerUser | null
  scope?: 'admin' | 'self'
}>(), {
  scope: 'admin'
})
defineEmits(['close'])
const { t } = useI18n()

const credits = ref<UserBalanceCredit[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 15

const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)
const showSettlementColumn = computed(() => props.scope === 'admin')

watch(() => props.show, (v) => {
  if (v && props.user) {
    loadCredits(1)
  }
})

const loadCredits = async (page: number) => {
  if (!props.user) return
  loading.value = true
  currentPage.value = page
  try {
    const res = props.scope === 'self'
      ? await userAPI.getBalanceCredits(page, pageSize)
      : await adminAPI.users.getUserBalanceCredits(props.user.id, page, pageSize)
    credits.value = res.items || []
    total.value = res.total || 0
  } catch (error) {
    console.error('Failed to load balance ledger:', error)
  } finally {
    loading.value = false
  }
}

const consumedAmount = (credit: UserBalanceCredit) => Math.max(credit.amount - credit.remaining_amount, 0)

const formatNullableDate = (value: string | null, dateOnly = false) => {
  if (!value) return t('admin.users.ledgerNeverExpires')
  if (!dateOnly) return formatDateTime(value)
  return value.slice(0, 10)
}

const formatSettledDate = (value: string | null) => value ? value.slice(0, 10) : '-'

const sourceLabel = (source: string) => {
  switch (source) {
    case 'redeem':
      return t('admin.users.ledgerSourceRedeem')
    case 'admin':
      return t('admin.users.ledgerSourceAdmin')
    case 'promo':
      return t('admin.users.ledgerSourcePromo')
    case 'affiliate':
      return t('admin.users.ledgerSourceAffiliate')
    case 'initial':
      return t('admin.users.ledgerSourceInitial')
    default:
      return source || t('common.unknown')
  }
}

const statusLabel = (status: string) => {
  switch (status) {
    case 'active':
      return t('admin.users.ledgerStatusActive')
    case 'consumed':
      return t('admin.users.ledgerStatusConsumed')
    case 'expired':
      return t('admin.users.ledgerStatusExpired')
    default:
      return status || t('common.unknown')
  }
}

const statusClass = (status: string) => {
  switch (status) {
    case 'active':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
    case 'consumed':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
    case 'expired':
      return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-dark-300'
  }
}
</script>

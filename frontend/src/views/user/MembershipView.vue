<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <!-- 总开关关闭：入口已从导航隐藏，这里兜住直接访问 URL 的情况 -->
      <div v-else-if="featureDisabled" class="card p-6 text-center">
        <p class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('membership.featureDisabledTitle') }}
        </p>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('membership.featureDisabledDesc') }}
        </p>
      </div>

      <template v-else-if="tier">
        <!-- 口径说明：等级只统计兑换额度消费，不含订阅消费 -->
        <div
          class="rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20"
        >
          <p class="text-sm text-primary-700 dark:text-primary-300">
            {{ t('membership.subscriptionDisclaimer') }}
          </p>
        </div>

        <!-- 概览：当前等级 / 累计消费 / 下一等级 -->
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.summaryTitle') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ tier.current_tier ? tier.current_tier.name : t('membership.noTier') }}
            </p>
            <p
              v-if="tier.current_tier && tier.current_tier.claimed_at"
              class="mt-1 text-xs text-gray-400 dark:text-dark-500"
            >
              {{ t('membership.claimedAt', { time: formatDateTime(tier.current_tier.claimed_at) }) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.consumedAmount') }}</p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formatCurrency(tier.consumed_amount) }}
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('membership.consumedAmountHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.nextTierTitle') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ tier.next_tier ? tier.next_tier.name : '-' }}
            </p>
            <p
              v-if="tier.next_tier && remainingToNext !== null"
              class="mt-1 text-xs text-primary-600 dark:text-primary-400"
            >
              {{ t('membership.nextTierRemaining', { amount: formatCurrency(remainingToNext) }) }}
            </p>
          </div>
        </div>

        <!-- 一键领取全部未领档位 -->
        <div v-if="claimableTiers.length > 0" class="card flex items-center justify-between gap-4 p-5">
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('membership.claimAll') }}</p>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="claimingAll"
            @click="claimAll"
          >
            {{ claimingAll ? t('membership.claiming') : t('membership.claimAll') }}
          </button>
        </div>

        <!-- 各档权益 -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('membership.allTiersTitle') }}
          </h3>

          <p v-if="tier.tiers.length === 0" class="mt-4 text-sm text-gray-500 dark:text-dark-400">
            {{ t('membership.empty') }}
          </p>

          <div
            v-for="item in tier.tiers"
            :key="item.tier_id"
            class="mt-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.name }}</p>
                <p v-if="item.description" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ item.description }}
                </p>
                <p v-if="item.threshold_usd != null" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
                  {{ t('membership.threshold', { amount: formatCurrency(item.threshold_usd) }) }}
                </p>
              </div>

              <!-- 领取三态；首充档自动发放，不显示领取按钮 -->
              <div class="flex shrink-0 items-center gap-2">
                <span
                  v-if="item.trigger_type === 'first_recharge'"
                  class="text-xs text-gray-500 dark:text-dark-400"
                >
                  {{ t('membership.firstRechargeAuto') }}
                </span>
                <template v-else-if="item.claimed">
                  <span class="badge badge-success">{{ t('membership.claimed') }}</span>
                  <span v-if="item.claimed_at" class="text-xs text-gray-400 dark:text-dark-500">
                    {{ t('membership.claimedAt', { time: formatDateTime(item.claimed_at) }) }}
                  </span>
                </template>
                <button
                  v-else-if="item.claimable"
                  type="button"
                  class="btn btn-primary btn-sm"
                  :disabled="claimingTierId !== null"
                  @click="claimOne(item)"
                >
                  {{ claimingTierId === item.tier_id ? t('membership.claiming') : t('membership.claim') }}
                </button>
                <template v-else>
                  <span class="badge badge-warning">{{ t('membership.notAchieved') }}</span>
                  <span class="text-xs text-gray-500 dark:text-dark-400">
                    {{ t('membership.notAchievedRemaining', { amount: formatCurrency(remainingFor(item)) }) }}
                  </span>
                </template>
              </div>
            </div>

            <!-- 历史授予说明：活动期手工授予且现行口径下未达标 -->
            <p
              v-if="item.source === HISTORICAL_GRANT_SOURCE && !item.achieved"
              class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
            >
              {{ t('membership.historicalGrantNote') }}
            </p>

            <!-- 权益清单 -->
            <ul v-if="item.benefits.length" class="mt-3 space-y-1.5">
              <li
                v-for="(benefit, index) in item.benefits"
                :key="`${item.tier_id}-${benefit.benefit_type}-${index}`"
                class="text-xs text-gray-600 dark:text-dark-300"
              >
                <template v-if="benefit.benefit_type === 'balance_credit'">
                  <span>{{ t('membership.benefitAmount', { amount: formatCurrency(benefit.amount ?? 0) }) }}</span>
                  <span class="ml-2 text-gray-400 dark:text-dark-500">{{ benefitExpiryText(benefit) }}</span>
                </template>
                <template v-else-if="benefit.benefit_type === 'group_rate'">
                  <span>{{ t('membership.benefitRate', { rate: benefit.rate_multiplier ?? 0 }) }}</span>
                  <span v-if="benefitGroupText(benefit)" class="ml-2 text-gray-400 dark:text-dark-500">
                    {{ t('membership.benefitGroup', { group: benefitGroupText(benefit) }) }}
                  </span>
                </template>

                <span v-if="benefit.status === 'pending'" class="ml-2 text-amber-600 dark:text-amber-400">
                  {{ t('membership.benefitPending') }}
                </span>
                <span v-else-if="benefit.status === 'failed'" class="ml-2 text-red-600 dark:text-red-400">
                  {{ t('membership.benefitFailed') }}
                </span>

                <span v-if="benefit.skipped_reason" class="ml-2 text-amber-600 dark:text-amber-400">
                  {{ t('membership.skippedReason', { reason: skippedReasonText(benefit.skipped_reason) }) }}
                </span>
              </li>
            </ul>
          </div>
        </div>
      </template>

      <div v-else class="card p-6 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('membership.empty') }}
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import userAPI from '@/api/user'
import type { MyTierResponse, UserTierBenefitState, UserTierClaimResponse, UserTierState } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorCode, extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'

const { t } = useI18n()
const appStore = useAppStore()

/** 2026-09-24 活动期手工授予的档位来源标识 */
const HISTORICAL_GRANT_SOURCE = 'manual_20260924'
/** 已有专属倍率时后端跳过倍率权益的原因 */
const SKIPPED_MANUAL_RATE_MULTIPLIER = 'manual_rate_multiplier_present'
/** 管理员关闭等级总开关后，领取接口返回的 403 原因 */
const USER_TIER_FEATURE_DISABLED = 'USER_TIER_FEATURE_DISABLED'

const loading = ref(false)
const tier = ref<MyTierResponse | null>(null)
const claimingTierId = ref<number | null>(null)
const claimingAll = ref(false)
/**
 * 等级功能未开放：接口返回 enabled === false，或领取接口返回
 * USER_TIER_FEATURE_DISABLED（403）。两种途径都只渲染中性提示，不展示等级内容。
 */
const featureDisabled = ref(false)

function isFeatureDisabledError(err: unknown): boolean {
  return extractApiErrorCode(err) === USER_TIER_FEATURE_DISABLED
}

/**
 * 可领取的档位。
 * 首充档由系统自动发放（不显示领取按钮），因此即便后端标记 claimable 也排除在一键领取之外，
 * 避免对一个没有手动领取入口的档位发起请求。
 */
const claimableTiers = computed(() =>
  (tier.value?.tiers ?? []).filter(
    (item) => item.claimable && item.trigger_type !== 'first_recharge',
  ),
)

const remainingToNext = computed(() =>
  tier.value?.remaining_to_next != null ? Math.max(0, tier.value.remaining_to_next) : null,
)

/** 未达标档位还差的消费额（门槛 - 累计消费） */
function remainingFor(item: UserTierState): number {
  return Math.max(0, (item.threshold_usd ?? 0) - (tier.value?.consumed_amount ?? 0))
}

function benefitGroupText(benefit: UserTierBenefitState): string {
  if (benefit.group_name) return benefit.group_name
  return benefit.group_id ? `#${benefit.group_id}` : ''
}

function benefitExpiryText(benefit: UserTierBenefitState): string {
  if (benefit.expires_at) {
    return t('membership.benefitExpiresAt', {
      days: benefit.validity_days ?? 0,
      time: formatDateTime(benefit.expires_at),
    })
  }
  if (benefit.validity_days) {
    return t('membership.benefitValidityDays', { days: benefit.validity_days })
  }
  return t('membership.benefitNeverExpires')
}

function skippedReasonText(reason: string): string {
  if (reason === SKIPPED_MANUAL_RATE_MULTIPLIER) {
    return t('membership.skippedManualRateMultiplier')
  }
  return reason
}

/** 把一次领取结果翻译成面向用户的到账提示（金额 + 到期时间 / 分组倍率） */
function describeClaim(result: UserTierClaimResponse): string[] {
  const messages = result.applied.flatMap((benefit) => {
    if (benefit.benefit_type === 'balance_credit') {
      return [
        t('membership.claimSuccessBalance', {
          amount: formatCurrency(benefit.amount ?? 0),
          expires: benefit.expires_at
            ? formatDateTime(benefit.expires_at)
            : t('membership.benefitNeverExpires'),
        }),
      ]
    }
    if (benefit.benefit_type === 'group_rate') {
      return [
        t('membership.claimSuccessRate', {
          group: benefit.group_name || `#${benefit.group_id ?? 0}`,
          rate: benefit.rate_multiplier ?? 0,
        }),
      ]
    }
    return []
  })
  if (messages.length === 0 && result.already_claimed) {
    messages.push(t('membership.claimed'))
  }
  return messages
}

async function load() {
  loading.value = true
  try {
    const result = await userAPI.getMyTier()
    tier.value = result
    // 未配置/缺省视为开启，只有显式 false 才算关闭
    featureDisabled.value = result.enabled === false
  } catch (err: unknown) {
    // 总开关关闭时后端返回 403 USER_TIER_FEATURE_DISABLED，落到与 enabled=false 相同的状态
    if (isFeatureDisabledError(err)) {
      tier.value = null
      featureDisabled.value = true
    } else {
      appStore.showError(extractI18nErrorMessage(err, t, 'membership', t('membership.loadFailed')))
    }
  } finally {
    loading.value = false
  }
}

async function claimOne(item: UserTierState) {
  claimingTierId.value = item.tier_id
  try {
    const result = await userAPI.claimTier(item.tier_id)
    const messages = describeClaim(result)
    if (messages.length > 0) {
      appStore.showSuccess(messages.join('; '))
    }
    await load()
  } catch (err: unknown) {
    // 关闭总开关后后端拒绝领取，页面切到未开放状态而不是报错
    if (isFeatureDisabledError(err)) {
      featureDisabled.value = true
    } else {
      appStore.showError(extractI18nErrorMessage(err, t, 'membership', t('membership.claimFailed')))
    }
  } finally {
    claimingTierId.value = null
  }
}

/**
 * 一键领取：不新增批量接口，逐档串行调用 claimTier，成功后再刷新视图。
 */
async function claimAll() {
  const targets = claimableTiers.value
  if (targets.length === 0) {
    appStore.showError(t('membership.claimAllNothing'))
    return
  }

  claimingAll.value = true
  const messages: string[] = []
  let claimedCount = 0
  try {
    for (const target of targets) {
      const result = await userAPI.claimTier(target.tier_id)
      claimedCount += 1
      messages.push(...describeClaim(result))
    }
    appStore.showSuccess(
      messages.length > 0
        ? messages.join('; ')
        : t('membership.claimAllSuccess', { count: claimedCount }),
    )
    await load()
  } catch (err: unknown) {
    if (isFeatureDisabledError(err)) {
      featureDisabled.value = true
    } else {
      appStore.showError(extractI18nErrorMessage(err, t, 'membership', t('membership.claimFailed')))
    }
  } finally {
    claimingAll.value = false
  }
}

onMounted(load)
</script>

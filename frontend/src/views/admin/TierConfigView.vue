<template>
  <AppLayout>
    <div class="space-y-4">
      <!-- 总开关：关闭后用户端入口与页面不可见，且不再发放新的等级权益（已发权益不回收） -->
      <div class="card p-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.tierConfig.switchTitle') }}
            </h3>
            <p class="mt-0.5 max-w-3xl text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.tierConfig.switchHint') }}
            </p>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <span class="text-xs text-gray-500 dark:text-dark-400">
              {{ featureEnabled ? t('admin.tierConfig.switchEnabled') : t('admin.tierConfig.switchDisabled') }}
            </span>
            <button
              type="button"
              role="switch"
              :aria-checked="featureEnabled"
              :disabled="switchLoading || switchSaving"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
                featureEnabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
              ]"
              @click="requestToggleFeatureSwitch"
            >
              <span :class="[
                'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                featureEnabled ? 'translate-x-4' : 'translate-x-0'
              ]" />
            </button>
          </div>
        </div>
      </div>

      <!-- 只读查询：管理端用户详情没有合适的挂载位置，收敛在配置页顶部 -->
      <div class="card p-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.tierConfig.lookupTitle') }}
            </h3>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.tierConfig.lookupHint') }}
            </p>
          </div>
          <div class="flex items-end gap-2">
            <input
              v-model="lookupUserId"
              type="number"
              min="1"
              class="input w-36"
              :placeholder="t('admin.tierConfig.userIdPlaceholder')"
              @keyup.enter="lookupUserTier"
            />
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="lookupLoading"
              @click="lookupUserTier"
            >
              {{ t('admin.tierConfig.lookup') }}
            </button>
          </div>
        </div>

        <div v-if="lookupResult" class="mt-4 space-y-4 border-t border-gray-200 pt-4 dark:border-dark-700">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.userIdPlaceholder') }}</p>
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ lookupResult.user_id }}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.lookupConsumedAmount') }}</p>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ formatCurrency(lookupResult.consumed_amount) }}
              </p>
            </div>
          </div>

          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.lookupTiers') }}</p>
            <ul class="mt-2 space-y-1">
              <li
                v-for="item in lookupResult.tiers"
                :key="item.tier_id"
                class="flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-300"
              >
                <span class="font-medium">{{ item.name }}</span>
                <span class="text-xs text-gray-400">{{ item.code }}</span>
                <span class="badge" :class="item.achieved ? 'badge-success' : 'badge-warning'">
                  {{ item.achieved ? t('admin.tierConfig.lookupAchieved') : t('admin.tierConfig.lookupNotAchieved') }}
                </span>
                <span v-if="item.claimed" class="badge badge-success">{{ t('admin.tierConfig.lookupClaimed') }}</span>
                <span v-if="item.claimable" class="badge badge-primary">{{ t('admin.tierConfig.lookupClaimable') }}</span>
              </li>
            </ul>
          </div>

          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.lookupAwards') }}</p>
            <p v-if="lookupResult.awards.length === 0" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.tierConfig.lookupNoAwards') }}
            </p>
            <ul v-else class="mt-2 space-y-1">
              <li
                v-for="award in lookupResult.awards"
                :key="award.id"
                class="text-sm text-gray-700 dark:text-gray-300"
              >
                <span class="font-medium">{{ award.tier_name_snapshot }}</span>
                <span class="ml-2 text-xs text-gray-400">{{ award.tier_code }}</span>
                <span class="ml-2 text-xs text-gray-400">
                  {{ t('admin.tierConfig.lookupAwardSource') }}: {{ award.source }}
                </span>
                <span class="ml-2 text-xs text-gray-400">
                  {{ t('admin.tierConfig.lookupAwardAchievedAt') }}: {{ formatDateTime(award.achieved_at) }}
                </span>
              </li>
            </ul>
          </div>

          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.lookupEffects') }}</p>
            <p v-if="lookupResult.effects.length === 0" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.tierConfig.lookupNoEffects') }}
            </p>
            <ul v-else class="mt-2 space-y-1">
              <li
                v-for="effect in lookupResult.effects"
                :key="effect.id"
                class="text-sm text-gray-700 dark:text-gray-300"
              >
                <span class="font-medium">{{ effect.benefit_type }}</span>
                <span class="ml-2 text-xs text-gray-400">
                  {{ t('admin.tierConfig.lookupEffectStatus') }}: {{ effect.status }}
                </span>
                <span class="ml-2 text-xs text-gray-400">
                  {{ t('admin.tierConfig.lookupEffectAttempts') }}: {{ effect.attempts }}
                </span>
                <span v-if="effect.last_error" class="ml-2 text-xs text-red-500">
                  {{ t('admin.tierConfig.lookupEffectLastError') }}: {{ effect.last_error }}
                </span>
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-end gap-2">
        <button @click="loadTiers" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>
        <button @click="openTierEdit(null)" class="btn btn-primary">{{ t('admin.tierConfig.createTier') }}</button>
      </div>

      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.tierConfig.deleteLockedHint') }}</p>

      <!-- Tiers table (draggable rows) -->
      <div class="card overflow-x-auto">
        <table class="w-full min-w-[900px] text-left text-sm">
          <thead class="border-b border-gray-200 text-xs uppercase text-gray-500 dark:border-dark-700 dark:text-dark-400">
            <tr>
              <th class="px-3 py-3">{{ t('admin.tierConfig.sortOrder') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.name') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.code') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.triggerType') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.thresholdUsd') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.benefits') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.awardCount') }}</th>
              <th class="px-3 py-3">{{ t('admin.tierConfig.status') }}</th>
              <th class="px-3 py-3">{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <VueDraggable
            v-if="localTiers.length"
            tag="tbody"
            v-model="localTiers"
            :animation="150"
            handle=".drag-handle"
            @end="applyOrder"
          >
            <tr
              v-for="item in localTiers"
              :key="item.id"
              class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
            >
              <td class="px-3 py-3">
                <span
                  class="drag-handle inline-flex cursor-grab items-center text-gray-300 hover:text-gray-500 active:cursor-grabbing dark:text-dark-600 dark:hover:text-dark-400"
                  :title="t('admin.tierConfig.dragHint')"
                >
                  <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M7 2a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 2a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM7 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM7 14a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 14a2 2 0 1 0 0 4 2 2 0 0 0 0-4z"/>
                  </svg>
                </span>
              </td>
              <td class="px-3 py-3">
                <div class="font-medium text-gray-900 dark:text-white">{{ item.name }}</div>
                <div v-if="item.description" class="text-xs text-gray-500 dark:text-dark-400">{{ item.description }}</div>
              </td>
              <td class="px-3 py-3 text-xs text-gray-500 dark:text-dark-400">{{ item.code }}</td>
              <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ triggerTypeLabel(item.trigger_type) }}</td>
              <td class="px-3 py-3 text-gray-700 dark:text-gray-300">
                {{ item.threshold_usd != null ? formatCurrency(item.threshold_usd) : '-' }}
              </td>
              <td class="px-3 py-3 text-gray-700 dark:text-gray-300">
                <span v-if="item.benefits.length === 0" class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('admin.tierConfig.benefitsEmpty') }}
                </span>
                <ul v-else class="space-y-0.5 text-xs">
                  <li v-for="benefit in item.benefits" :key="benefit.id">
                    <span>{{ benefitTypeLabel(benefit.benefit_type) }}</span>
                    <span v-if="benefit.benefit_type === 'balance_credit'" class="ml-1 text-gray-500 dark:text-dark-400">
                      {{ formatCurrency(benefit.amount) }} / {{ benefit.validity_days }}
                    </span>
                    <span v-else class="ml-1 text-gray-500 dark:text-dark-400">
                      {{ groupName(benefit.group_id) }} / {{ benefit.rate_multiplier }}x
                    </span>
                  </li>
                </ul>
              </td>
              <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.award_count }}</td>
              <td class="px-3 py-3">
                <button
                  type="button"
                  :class="[
                    'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                    item.enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
                  ]"
                  @click="toggleEnabled(item)"
                >
                  <span :class="[
                    'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                    item.enabled ? 'translate-x-4' : 'translate-x-0'
                  ]" />
                </button>
              </td>
              <td class="px-3 py-3">
                <div class="flex items-center gap-2">
                  <button
                    @click="openTierEdit(item)"
                    class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                  >
                    <Icon name="edit" size="sm" />
                    <span class="text-xs">{{ t('common.edit') }}</span>
                  </button>
                  <button
                    @click="confirmDeleteTier(item)"
                    :disabled="item.award_count > 0"
                    :title="item.award_count > 0 ? t('admin.tierConfig.deleteLocked') : t('common.delete')"
                    class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                  >
                    <Icon name="trash" size="sm" />
                    <span class="text-xs">{{ t('common.delete') }}</span>
                  </button>
                </div>
              </td>
            </tr>
          </VueDraggable>
        </table>

        <p v-if="!loading && localTiers.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('admin.tierConfig.empty') }}
        </p>
      </div>
    </div>

    <!-- Tier Edit Dialog -->
    <BaseDialog
      :show="showTierDialog"
      :title="editingTierId === null ? t('admin.tierConfig.createTier') : t('admin.tierConfig.editTier')"
      width="wide"
      @close="showTierDialog = false"
    >
      <form id="tier-form" class="space-y-4" @submit.prevent="saveTier">
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.tierConfig.name') }} <span class="text-red-500">*</span></label>
            <input v-model="form.name" type="text" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.tierConfig.code') }} <span class="text-red-500">*</span></label>
            <input v-model="form.code" type="text" class="input" :disabled="tierCodeLocked" />
            <p v-if="tierCodeLocked" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.tierConfig.codeLockedHint') }}
            </p>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.tierConfig.descriptionField') }}</label>
          <textarea v-model="form.description" rows="2" class="input"></textarea>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.tierConfig.triggerType') }} <span class="text-red-500">*</span></label>
            <Select v-model="form.trigger_type" :options="triggerTypeOptions" />
          </div>
          <div v-if="form.trigger_type === 'consumption'">
            <label class="input-label">{{ t('admin.tierConfig.thresholdUsd') }} <span class="text-red-500">*</span></label>
            <input v-model.number="form.threshold_usd" type="number" step="0.01" min="0" class="input" />
          </div>
        </div>

        <div class="flex items-center gap-3">
          <label class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.tierConfig.enabled') }}</label>
          <Toggle v-model="form.enabled" />
        </div>

        <!-- 权益子表 -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-700">
          <div class="flex items-center justify-between">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.tierConfig.benefits') }}</p>
            <button type="button" class="btn btn-secondary btn-sm" @click="addBenefit">
              {{ t('admin.tierConfig.addBenefit') }}
            </button>
          </div>

          <p v-if="form.benefits.length === 0" class="mt-2 text-xs text-gray-400 dark:text-dark-500">
            {{ t('admin.tierConfig.benefitsEmpty') }}
          </p>

          <VueDraggable
            v-if="form.benefits.length"
            v-model="form.benefits"
            :animation="150"
            handle=".benefit-drag-handle"
            class="mt-3 space-y-3"
          >
            <div
              v-for="(benefit, index) in form.benefits"
              :key="index"
              class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
            >
              <div class="flex items-start gap-2">
                <span
                  class="benefit-drag-handle mt-2 cursor-grab text-gray-300 hover:text-gray-500 active:cursor-grabbing dark:text-dark-600 dark:hover:text-dark-400"
                >
                  <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M7 2a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 2a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM7 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 8a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM7 14a2 2 0 1 0 0 4 2 2 0 0 0 0-4zM13 14a2 2 0 1 0 0 4 2 2 0 0 0 0-4z"/>
                  </svg>
                </span>
                <div class="min-w-0 flex-1 space-y-3">
                  <div class="flex items-center gap-3">
                    <Select v-model="benefit.benefit_type" :options="benefitTypeOptions" class="flex-1" />
                    <label class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                      <Toggle v-model="benefit.enabled" />
                      {{ t('admin.tierConfig.enabled') }}
                    </label>
                    <button
                      type="button"
                      class="text-xs text-red-500 hover:text-red-600"
                      @click="removeBenefit(index)"
                    >
                      {{ t('admin.tierConfig.removeBenefit') }}
                    </button>
                  </div>

                  <div v-if="benefit.benefit_type === 'balance_credit'" class="grid gap-3 sm:grid-cols-2">
                    <div>
                      <label class="input-label">{{ t('admin.tierConfig.amount') }}</label>
                      <input v-model.number="benefit.amount" type="number" step="0.01" min="0" class="input" />
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.tierConfig.validityDays') }}</label>
                      <input v-model.number="benefit.validity_days" type="number" min="0" class="input" />
                    </div>
                  </div>

                  <div v-else class="grid gap-3 sm:grid-cols-2">
                    <div>
                      <label class="input-label">{{ t('admin.tierConfig.group') }}</label>
                      <Select
                        v-model="benefit.group_id"
                        :options="groupOptions"
                        :placeholder="t('admin.tierConfig.selectGroup')"
                      />
                    </div>
                    <div>
                      <label class="input-label">{{ t('admin.tierConfig.rateMultiplier') }}</label>
                      <input v-model.number="benefit.rate_multiplier" type="number" step="0.01" min="0" class="input" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </VueDraggable>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="showTierDialog = false">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="tier-form" :disabled="saving" class="btn btn-primary">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.tierConfig.deleteTier')"
      :message="deleteMessage"
      :confirm-text="t('common.delete')"
      danger
      @confirm="handleDeleteTier"
      @cancel="showDeleteDialog = false"
    />

    <!-- 关闭总开关对用户立即生效，因此先确认；取消则不改动 -->
    <ConfirmDialog
      :show="showDisableSwitchDialog"
      :title="t('admin.tierConfig.switchDisableTitle')"
      :message="t('admin.tierConfig.switchDisableConfirm')"
      :confirm-text="t('admin.tierConfig.switchDisableOk')"
      danger
      @confirm="confirmDisableFeatureSwitch"
      @cancel="showDisableSwitchDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import { useAppStore } from '@/stores/app'
import adminAPI from '@/api/admin'
import userTiersAPI from '@/api/admin/userTiers'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatCurrency, formatDateTime } from '@/utils/format'
import type {
  AdminGroup,
  AdminTier,
  AdminTierBenefitInput,
  AdminTierSaveRequest,
  AdminUserTierResponse,
  UserTierBenefitType,
  UserTierTriggerType,
} from '@/types'
import type { SelectOption } from '@/components/common/Select.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'

const { t } = useI18n()
const appStore = useAppStore()

// ==================== Feature master switch ====================

/** 总开关状态；未加载完成时按开启展示（与 opt-out 语义一致） */
const featureEnabled = ref(true)
const switchLoading = ref(false)
const switchSaving = ref(false)
const showDisableSwitchDialog = ref(false)

async function loadFeatureSwitch() {
  switchLoading.value = true
  try {
    const result = await userTiersAPI.getFeatureSwitch()
    featureEnabled.value = result.enabled
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.switchLoadFailed')))
  } finally {
    switchLoading.value = false
  }
}

/** 关闭对用户立即生效，先弹确认框；开启无破坏性，直接生效。 */
function requestToggleFeatureSwitch() {
  if (switchLoading.value || switchSaving.value) return
  if (featureEnabled.value) {
    showDisableSwitchDialog.value = true
    return
  }
  void applyFeatureSwitch(true)
}

function confirmDisableFeatureSwitch() {
  showDisableSwitchDialog.value = false
  void applyFeatureSwitch(false)
}

async function applyFeatureSwitch(enabled: boolean) {
  switchSaving.value = true
  try {
    const result = await userTiersAPI.updateFeatureSwitch(enabled)
    featureEnabled.value = result.enabled
    appStore.showSuccess(
      t(enabled ? 'admin.tierConfig.switchEnableSuccess' : 'admin.tierConfig.switchDisableSuccess'),
    )
  } catch (err: unknown) {
    // 失败时保持原状态，只有后端确认成功才翻转开关
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.switchSaveFailed')))
  } finally {
    switchSaving.value = false
  }
}

// ==================== Tiers ====================

const loading = ref(false)
const saving = ref(false)
const tiers = ref<AdminTier[]>([])
/** 拖拽期间的本地顺序副本，@end 之后回写并提交 */
const localTiers = ref<AdminTier[]>([])

const groups = ref<AdminGroup[]>([])

const triggerTypeOptions = computed((): SelectOption[] => [
  { value: 'consumption', label: t('admin.tierConfig.triggerConsumption') },
  { value: 'first_recharge', label: t('admin.tierConfig.triggerFirstRecharge') },
])

const benefitTypeOptions = computed((): SelectOption[] => [
  { value: 'balance_credit', label: t('admin.tierConfig.benefitTypeBalanceCredit') },
  { value: 'group_rate', label: t('admin.tierConfig.benefitTypeGroupRate') },
])

const groupOptions = computed((): SelectOption[] =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    platform: group.platform,
  })),
)

function triggerTypeLabel(value: UserTierTriggerType): string {
  return value === 'first_recharge'
    ? t('admin.tierConfig.triggerFirstRecharge')
    : t('admin.tierConfig.triggerConsumption')
}

function benefitTypeLabel(value: UserTierBenefitType): string {
  return value === 'group_rate'
    ? t('admin.tierConfig.benefitTypeGroupRate')
    : t('admin.tierConfig.benefitTypeBalanceCredit')
}

function groupName(groupId: number): string {
  return groups.value.find((group) => group.id === groupId)?.name || `#${groupId}`
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch {
    /* 分组只用于展示与选择，加载失败不阻塞档位配置 */
  }
}

async function loadTiers() {
  loading.value = true
  try {
    tiers.value = await userTiersAPI.listTiers()
    localTiers.value = [...tiers.value]
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.loadFailed')))
  } finally {
    loading.value = false
  }
}

/** 拖拽结束后按当前本地顺序提交档位排序 */
async function applyOrder() {
  const ids = localTiers.value.map((item) => item.id)
  try {
    await userTiersAPI.reorderTiers(ids)
    tiers.value = [...localTiers.value]
    appStore.showSuccess(t('admin.tierConfig.orderSaved'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.orderFailed')))
    await loadTiers()
  }
}

/** Quick toggle enabled from the list */
async function toggleEnabled(tier: AdminTier) {
  try {
    await userTiersAPI.updateTier(tier.id, { enabled: !tier.enabled })
    tier.enabled = !tier.enabled
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.saveFailed')))
  }
}

// ==================== Edit Dialog ====================

type TierBenefitForm = {
  id?: number
  benefit_type: UserTierBenefitType
  amount: number
  validity_days: number
  group_id: number
  rate_multiplier: number
  enabled: boolean
}

const showTierDialog = ref(false)
const editingTierId = ref<number | null>(null)

const form = reactive({
  code: '',
  name: '',
  description: '',
  trigger_type: 'consumption' as UserTierTriggerType,
  threshold_usd: 0 as number | null,
  enabled: true,
  benefits: [] as TierBenefitForm[],
})

/** 已产生授予记录的档位不允许改 code（后端以 code 关联历史授予） */
const tierCodeLocked = computed(() => {
  if (editingTierId.value === null) return false
  const tier = tiers.value.find((item) => item.id === editingTierId.value)
  return (tier?.award_count ?? 0) > 0
})

function openTierEdit(tier: AdminTier | null) {
  editingTierId.value = tier?.id ?? null
  form.code = tier?.code ?? ''
  form.name = tier?.name ?? ''
  form.description = tier?.description ?? ''
  form.trigger_type = tier?.trigger_type ?? 'consumption'
  form.threshold_usd = tier?.threshold_usd ?? 0
  form.enabled = tier?.enabled ?? true
  form.benefits = (tier?.benefits ?? []).map((benefit) => ({
    id: benefit.id,
    benefit_type: benefit.benefit_type,
    amount: benefit.amount,
    validity_days: benefit.validity_days,
    group_id: benefit.group_id,
    rate_multiplier: benefit.rate_multiplier,
    enabled: benefit.enabled,
  }))
  showTierDialog.value = true
}

function addBenefit() {
  form.benefits.push({
    benefit_type: 'balance_credit',
    amount: 0,
    validity_days: 0,
    group_id: 0,
    rate_multiplier: 0,
    enabled: true,
  })
}

function removeBenefit(index: number) {
  form.benefits.splice(index, 1)
}

/** 提交前逐项校验，返回第一条错误文案；全部通过返回 null */
function validateForm(): string | null {
  if (!form.name.trim()) return t('admin.tierConfig.nameRequired')
  if (!form.code.trim()) return t('admin.tierConfig.codeRequired')
  if (form.trigger_type === 'consumption') {
    const threshold = Number(form.threshold_usd)
    if (!Number.isFinite(threshold) || threshold <= 0) return t('admin.tierConfig.thresholdRequired')
  }
  for (const benefit of form.benefits) {
    if (benefit.benefit_type === 'balance_credit') {
      const amount = Number(benefit.amount)
      if (!Number.isFinite(amount) || amount <= 0) return t('admin.tierConfig.amountInvalid')
      continue
    }
    const groupId = Number(benefit.group_id)
    if (!Number.isFinite(groupId) || groupId <= 0) return t('admin.tierConfig.groupRequired')
    const rate = Number(benefit.rate_multiplier)
    if (!Number.isFinite(rate) || rate <= 0) return t('admin.tierConfig.rateMultiplierInvalid')
  }
  return null
}

function buildBenefitsPayload(): AdminTierBenefitInput[] {
  return form.benefits.map((benefit) => {
    const isBalanceCredit = benefit.benefit_type === 'balance_credit'
    return {
      id: benefit.id,
      benefit_type: benefit.benefit_type,
      amount: isBalanceCredit ? Math.max(0, Number(benefit.amount) || 0) : 0,
      validity_days: isBalanceCredit ? Math.max(0, Math.floor(Number(benefit.validity_days) || 0)) : 0,
      group_id: isBalanceCredit ? 0 : Math.max(0, Math.floor(Number(benefit.group_id) || 0)),
      rate_multiplier: isBalanceCredit ? 0 : Math.max(0, Number(benefit.rate_multiplier) || 0),
      enabled: benefit.enabled,
    }
  })
}

async function saveTier() {
  const error = validateForm()
  if (error) {
    appStore.showError(error)
    return
  }

  const payload: AdminTierSaveRequest = {
    name: form.name.trim(),
    description: form.description,
    trigger_type: form.trigger_type,
    threshold_usd: form.trigger_type === 'consumption' ? Number(form.threshold_usd) : null,
    enabled: form.enabled,
    benefits: buildBenefitsPayload(),
  }
  // code 只在创建时提交；已产生授予记录的档位不允许修改
  if (editingTierId.value === null || !tierCodeLocked.value) {
    payload.code = form.code.trim()
  }

  saving.value = true
  try {
    if (editingTierId.value === null) {
      await userTiersAPI.createTier(payload)
      appStore.showSuccess(t('admin.tierConfig.createSuccess'))
    } else {
      await userTiersAPI.updateTier(editingTierId.value, payload)
      appStore.showSuccess(t('admin.tierConfig.updateSuccess'))
    }
    showTierDialog.value = false
    await loadTiers()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.saveFailed')))
  } finally {
    saving.value = false
  }
}

// ==================== Delete ====================

const showDeleteDialog = ref(false)
const deletingTier = ref<AdminTier | null>(null)

const deleteMessage = computed(() =>
  deletingTier.value
    ? t('admin.tierConfig.deleteTierConfirm', { name: deletingTier.value.name })
    : '',
)

function confirmDeleteTier(tier: AdminTier) {
  // 有授予记录的档位后端会返回 409，这里直接在入口禁用
  if (tier.award_count > 0) return
  deletingTier.value = tier
  showDeleteDialog.value = true
}

async function handleDeleteTier() {
  const tier = deletingTier.value
  if (!tier) return
  try {
    await userTiersAPI.deleteTier(tier.id)
    appStore.showSuccess(t('admin.tierConfig.deleteSuccess'))
    showDeleteDialog.value = false
    await loadTiers()
  } catch (err: unknown) {
    // 后端 409 USER_TIER_HAS_AWARDS 的文案按 error code 映射后展示
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.deleteFailed')))
  }
}

// ==================== Read-only user lookup ====================

const lookupUserId = ref<number | null>(null)
const lookupLoading = ref(false)
const lookupResult = ref<AdminUserTierResponse | null>(null)

async function lookupUserTier() {
  const userId = Number(lookupUserId.value)
  if (!Number.isFinite(userId) || userId <= 0) {
    appStore.showError(t('admin.tierConfig.lookupEmpty'))
    return
  }
  lookupLoading.value = true
  try {
    lookupResult.value = await userTiersAPI.getUserTier(userId)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'admin.tierConfig', t('admin.tierConfig.lookupFailed')))
  } finally {
    lookupLoading.value = false
  }
}

// ==================== Lifecycle ====================

onMounted(() => {
  loadGroups()
  loadTiers()
  loadFeatureSwitch()
})
</script>

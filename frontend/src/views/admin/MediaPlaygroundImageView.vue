<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.mediaPlaygroundImage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.mediaPlaygroundImage.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadModels">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" class="mr-2" />
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-secondary" type="button" @click="openCallRecordsDialog">
            {{ t('admin.mediaPlaygroundImage.callRecords.button') }}
          </button>
          <button class="btn btn-secondary" type="button" @click="openProbeRunsDialog">
            {{ t('admin.mediaPlaygroundImage.probeRuns.button') }}
          </button>
          <button class="btn btn-primary" type="button" @click="openCreateDialog">
            {{ t('admin.mediaPlaygroundImage.createModel') }}
          </button>
        </div>
      </div>

      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <DataTable :columns="columns" :data="models" :loading="loading">
          <template #cell-display_name="{ row }">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ row.display_name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.model }}</div>
            </div>
          </template>
          <template #cell-provider_name="{ row }">
            <span class="badge badge-gray">{{ row.provider_name }}</span>
          </template>
          <template #cell-api_mode="{ row }">
            <span class="badge badge-gray">{{ apiModeLabel(row.api_mode) }}</span>
          </template>
          <template #cell-prices="{ row }">
            <div class="space-y-1 font-mono text-xs">
              <div>1k {{ row.price_1k }}</div>
              <div>2k {{ row.price_2k }}</div>
              <div>4k {{ row.price_4k }}</div>
            </div>
          </template>
          <template #cell-supported_sizes="{ row }">
            <div class="flex flex-wrap gap-1">
              <span v-for="size in row.supported_sizes" :key="size" class="badge badge-gray">{{ size }}</span>
            </div>
          </template>
          <template #cell-sort_order="{ row }">
            <span class="font-mono text-xs text-gray-600 dark:text-gray-300">{{ row.sort_order }}</span>
          </template>
          <template #cell-health="{ row }">
            <div class="min-w-[220px] space-y-1 text-xs text-gray-600 dark:text-gray-300">
              <div class="flex flex-wrap items-center gap-2">
                <span :class="healthBadgeClass(row.health_status)">
                  {{ healthStatusLabel(row.health_status) }}
                </span>
                <span v-if="row.cooldown_until" class="whitespace-nowrap">
                  {{ t('admin.mediaPlaygroundImage.health.cooldownUntil') }} {{ formatDateTime(row.cooldown_until) }}
                </span>
              </div>
              <div class="flex flex-wrap gap-x-3 gap-y-1 font-mono">
                <span>{{ t('admin.mediaPlaygroundImage.health.failures') }} {{ row.consecutive_failures ?? 0 }}</span>
                <span>{{ t('admin.mediaPlaygroundImage.health.cooldowns') }} {{ row.cooldown_count ?? 0 }}</span>
                <span>{{ t('admin.mediaPlaygroundImage.health.halfOpenAttempts') }} {{ row.half_open_attempts ?? 0 }}</span>
              </div>
              <div
                v-if="row.last_health_error"
                class="max-w-[260px] truncate text-red-600 dark:text-red-300"
                :title="row.last_health_error"
              >
                {{ t('admin.mediaPlaygroundImage.health.lastError') }} {{ row.last_health_error }}
              </div>
            </div>
          </template>
          <template #cell-enabled="{ row }">
            <button
              type="button"
              :class="row.enabled ? 'badge badge-success' : 'badge badge-gray'"
              :disabled="togglingId === row.id"
              @click="toggleModelEnabled(row)"
            >
              {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
            </button>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center gap-2">
              <button class="btn btn-sm btn-secondary" :disabled="probingModelIds.has(row.id)" @click="runSingleModelProbe(row)">
                <Icon name="refresh" size="sm" :class="probingModelIds.has(row.id) ? 'animate-spin' : ''" class="mr-1" />
                {{ t('admin.mediaPlaygroundImage.probeRuns.singleRunButton') }}
              </button>
              <button class="btn btn-sm btn-secondary" @click="openReuseDialog(row)">
                {{ t('admin.mediaPlaygroundImage.reuse') }}
              </button>
              <button class="btn btn-sm btn-secondary" @click="openEditDialog(row)">
                {{ t('common.edit') }}
              </button>
              <button class="btn btn-sm btn-danger" @click="requestDeleteModel(row)">
                {{ t('common.delete') }}
              </button>
            </div>
          </template>
        </DataTable>
      </section>

      <BaseDialog
        :show="showDialog"
        :title="dialogTitle"
        width="wide"
        :close-on-escape="!saving"
        :show-close-button="!saving"
        @close="closeDialog"
      >
        <form id="media-playground-image-model-form" class="space-y-4" @submit.prevent="saveModel">
          <label v-if="!editingId && models.length" class="block">
            <span class="input-label">{{ t('admin.mediaPlaygroundImage.reuseFrom') }}</span>
            <select class="input" @change="handleReuseSelect">
              <option value="">{{ t('admin.mediaPlaygroundImage.reuseFromPlaceholder') }}</option>
              <option v-for="model in models" :key="model.id" :value="model.id">
                {{ model.display_name }} · {{ model.model }}
              </option>
            </select>
          </label>

          <div class="grid gap-4 lg:grid-cols-2">
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.displayName') }}</span>
              <input v-model.trim="form.display_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.model') }}</span>
              <input v-model.trim="form.model" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.apiMode') }}</span>
              <select v-model="form.api_mode" class="input" required>
                <option value="images">{{ t('admin.mediaPlaygroundImage.apiModes.images') }}</option>
                <option value="responses">{{ t('admin.mediaPlaygroundImage.apiModes.responses') }}</option>
                <option value="gemini_generate_content">{{ t('admin.mediaPlaygroundImage.apiModes.geminiGenerateContent') }}</option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.providerName') }}</span>
              <input v-model.trim="form.provider_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.upstreamBaseURL') }}</span>
              <input v-model.trim="form.upstream_base_url" class="input" placeholder="https://api.example.com" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.upstreamAPIKey') }}</span>
              <div class="relative">
                <input
                  v-model.trim="form.upstream_api_key"
                  class="input pr-10"
                  :type="upstreamKeyVisible ? 'text' : 'password'"
                  autocomplete="off"
                  :placeholder="upstreamKeyPlaceholder"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  @click="upstreamKeyVisible = !upstreamKeyVisible"
                >
                  <Icon :name="upstreamKeyVisible ? 'eyeOff' : 'eye'" size="md" />
                </button>
              </div>
            </label>
          </div>

          <div class="grid gap-4 sm:grid-cols-3">
              <label class="block">
                <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.price1k') }}</span>
                <input v-model.number="form.price_1k" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.price2k') }}</span>
                <input v-model.number="form.price_2k" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.price4k') }}</span>
                <input v-model.number="form.price_4k" class="input" type="number" min="0" step="0.000001" required />
              </label>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
              <label class="block">
                <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.timeoutSeconds') }}</span>
                <input v-model.number="form.timeout_seconds" class="input" type="number" min="1" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.sortOrder') }}</span>
                <input v-model.number="form.sort_order" class="input" type="number" />
              </label>
          </div>

          <div>
            <span class="input-label">{{ t('admin.mediaPlaygroundImage.fields.supportedSizes') }}</span>
            <div class="mt-2 flex flex-wrap gap-4">
              <label v-for="size in sizeOptions" :key="size" class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                <input v-model="form.supported_sizes" :value="size" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ size }}
              </label>
            </div>
          </div>

          <div class="flex flex-wrap gap-4">
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.mediaPlaygroundImage.fields.enabled') }}
            </label>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.fallback_to_responses_enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.mediaPlaygroundImage.fields.fallbackToResponses') }}
            </label>
          </div>
        </form>

        <template #footer>
          <div class="flex justify-end gap-3">
            <button class="btn btn-secondary" type="button" :disabled="saving" @click="closeDialog">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-primary" type="submit" form="media-playground-image-model-form" :disabled="saving">
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </template>
      </BaseDialog>

      <BaseDialog
        :show="showCallRecordsDialog"
        :title="t('admin.mediaPlaygroundImage.callRecords.title')"
        width="wide"
        @close="closeCallRecordsDialog"
      >
        <div class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.mediaPlaygroundImage.callRecords.description') }}
            </p>
            <button class="btn btn-secondary" type="button" :disabled="callRecordsLoading" @click="loadCallRecords(callRecordsPage)">
              <Icon name="refresh" size="md" :class="callRecordsLoading ? 'animate-spin' : ''" class="mr-2" />
              {{ t('common.refresh') }}
            </button>
          </div>

          <DataTable :columns="callRecordColumns" :data="callRecords" :loading="callRecordsLoading">
            <template #cell-created_at="{ value }">
              <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(value) }}</span>
            </template>
            <template #cell-id="{ row }">
              <div>
                <div class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ row.model_config_id }}</div>
                <div class="font-mono text-xs text-gray-900 dark:text-white">{{ row.id }}</div>
              </div>
            </template>
            <template #cell-user="{ row }">
              <div class="font-mono text-xs">
                <div>用户ID {{ row.user_id || '-' }}</div>
                <div>API Key {{ row.api_key_suffix || '-' }}</div>
              </div>
            </template>
            <template #cell-model="{ row }">
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ row.model || '-' }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ apiModeLabel(row.api_mode) }} · {{ row.size_tier || '-' }}</div>
              </div>
            </template>
            <template #cell-upstream_base_url="{ value }">
              <span class="inline-block max-w-[220px] truncate" :title="value || '-'">{{ value || '-' }}</span>
            </template>
            <template #cell-status="{ row }">
              <span :class="row.status === 'succeeded' || row.status === 'completed' ? 'badge badge-success' : row.status === 'failed' ? 'badge badge-danger' : 'badge badge-gray'">
                {{ row.status || '-' }}
              </span>
            </template>
            <template #cell-upstream_status_code="{ value }">
              <span class="font-mono text-xs">{{ value || '-' }}</span>
            </template>
            <template #cell-response_bytes="{ value }">
              <span class="font-mono text-xs">{{ formatBytes(value || 0) }}</span>
            </template>
            <template #cell-error_message="{ row }">
              <span class="inline-block max-w-[260px] truncate text-sm text-gray-600 dark:text-gray-300" :title="row.error_message || row.error_code || ''">
                {{ row.error_message || row.error_code || '-' }}
              </span>
            </template>
          </DataTable>

          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-200 pt-4 dark:border-dark-700">
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.mediaPlaygroundImage.callRecords.pageInfo', { page: callRecordsPage, total: callRecordsTotal }) }}
            </div>
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary" type="button" :disabled="callRecordsLoading || callRecordsPage <= 1" @click="loadCallRecords(callRecordsPage - 1)">
                {{ t('admin.mediaPlaygroundImage.callRecords.previous') }}
              </button>
              <button class="btn btn-secondary" type="button" :disabled="callRecordsLoading || !hasNextCallRecordsPage" @click="loadCallRecords(callRecordsPage + 1)">
                {{ t('admin.mediaPlaygroundImage.callRecords.next') }}
              </button>
            </div>
          </div>
        </div>
      </BaseDialog>

      <BaseDialog
        :show="showProbeRunsDialog"
        :title="t('admin.mediaPlaygroundImage.probeRuns.title')"
        width="wide"
        @close="closeProbeRunsDialog"
      >
        <div class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.mediaPlaygroundImage.probeRuns.description') }}
            </p>
            <button class="btn btn-secondary" type="button" :disabled="probeRunsLoading" @click="loadProbeRuns(probeRunsPage)">
              <Icon name="refresh" size="md" :class="probeRunsLoading ? 'animate-spin' : ''" class="mr-2" />
              {{ t('common.refresh') }}
            </button>
          </div>

          <DataTable :columns="probeRunColumns" :data="probeRuns" :loading="probeRunsLoading">
            <template #cell-created_at="{ value }">
              <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(value) }}</span>
            </template>
            <template #cell-model="{ row }">
              <div>
                <div class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ row.model_config_id }}</div>
                <div class="font-medium text-gray-900 dark:text-white">{{ row.model || '-' }}</div>
              </div>
            </template>
            <template #cell-api_mode="{ row }">
              <span class="badge badge-gray">{{ apiModeLabel(row.api_mode) }}</span>
            </template>
            <template #cell-upstream_base_url="{ value }">
              <span class="inline-block max-w-[220px] truncate" :title="value || '-'">{{ value || '-' }}</span>
            </template>
            <template #cell-status="{ row }">
              <span :class="row.status === 'success' ? 'badge badge-success' : 'badge badge-danger'">
                {{ probeStatusLabel(row.status) }}
              </span>
            </template>
            <template #cell-http_status_code="{ value }">
              <span class="font-mono text-xs">{{ value || '-' }}</span>
            </template>
            <template #cell-elapsed_ms="{ value }">
              <span class="font-mono text-xs">{{ formatDurationMs(value) }}</span>
            </template>
            <template #cell-response_bytes="{ value }">
              <span class="font-mono text-xs">{{ formatBytes(value || 0) }}</span>
            </template>
            <template #cell-error_message="{ value }">
              <span class="inline-block max-w-[260px] truncate text-sm text-gray-600 dark:text-gray-300" :title="value || ''">
                {{ value || '-' }}
              </span>
            </template>
          </DataTable>

          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-200 pt-4 dark:border-dark-700">
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.mediaPlaygroundImage.probeRuns.pageInfo', { page: probeRunsPage, total: probeRunsTotal }) }}
            </div>
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary" type="button" :disabled="probeRunsLoading || probeRunsPage <= 1" @click="loadProbeRuns(probeRunsPage - 1)">
                {{ t('admin.mediaPlaygroundImage.probeRuns.previous') }}
              </button>
              <button class="btn btn-secondary" type="button" :disabled="probeRunsLoading || !hasNextProbeRunsPage" @click="loadProbeRuns(probeRunsPage + 1)">
                {{ t('admin.mediaPlaygroundImage.probeRuns.next') }}
              </button>
            </div>
          </div>
        </div>
      </BaseDialog>

      <ConfirmDialog
        :show="showDeleteDialog"
        :title="t('admin.mediaPlaygroundImage.deleteTitle')"
        :message="t('admin.mediaPlaygroundImage.deleteConfirm', { name: deletingModel?.display_name || '' })"
        :confirm-text="t('common.delete')"
        :cancel-text="t('common.cancel')"
        :danger="true"
        @confirm="confirmDeleteModel"
        @cancel="closeDeleteDialog"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { MediaPlaygroundImageModel, MediaPlaygroundImageModelPayload, MediaPlaygroundImageProbeRun, MediaPlaygroundImageUpstreamRequest, ImageSizeTier } from '@/api/admin'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const sizeOptions: ImageSizeTier[] = ['1k', '2k', '4k']

const models = ref<MediaPlaygroundImageModel[]>([])
const probeRuns = ref<MediaPlaygroundImageProbeRun[]>([])
const callRecords = ref<MediaPlaygroundImageUpstreamRequest[]>([])
const loading = ref(false)
const probeRunsLoading = ref(false)
const callRecordsLoading = ref(false)
const probingModelIds = ref<Set<number>>(new Set())
const saving = ref(false)
const togglingId = ref<number | null>(null)
const editingId = ref<number | null>(null)
const editingKeyMask = ref('')
const reusedKeyMask = ref('')
const upstreamKeyVisible = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const showProbeRunsDialog = ref(false)
const showCallRecordsDialog = ref(false)
const deletingModel = ref<MediaPlaygroundImageModel | null>(null)
const probeRunsPage = ref(1)
const probeRunsPageSize = 20
const probeRunsTotal = ref(0)
const callRecordsPage = ref(1)
const callRecordsPageSize = 20
const callRecordsTotal = ref(0)

const defaultForm = (): MediaPlaygroundImageModelPayload => ({
  display_name: '',
  model: '',
  api_mode: 'images',
  provider_name: '',
  upstream_base_url: '',
  upstream_api_key: '',
  price_1k: 1,
  price_2k: 2,
  price_4k: 4,
  supported_sizes: ['1k', '2k', '4k'],
  timeout_seconds: 600,
  fallback_to_responses_enabled: true,
  enabled: true,
  sort_order: 0,
})

const form = reactive<MediaPlaygroundImageModelPayload>(defaultForm())

const dialogTitle = computed(() => {
  return editingId.value ? t('admin.mediaPlaygroundImage.editModel') : t('admin.mediaPlaygroundImage.createModel')
})

const upstreamKeyPlaceholder = computed(() => {
  if (editingId.value && editingKeyMask.value) {
    return t('admin.mediaPlaygroundImage.keyConfigured', { mask: editingKeyMask.value })
  }
  if (reusedKeyMask.value) {
    return t('admin.mediaPlaygroundImage.keyNotCopied', { mask: reusedKeyMask.value })
  }
  return 'sk-...'
})

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.mediaPlaygroundImage.columns.name') },
  { key: 'api_mode', label: t('admin.mediaPlaygroundImage.columns.apiMode') },
  { key: 'provider_name', label: t('admin.mediaPlaygroundImage.columns.provider') },
  { key: 'upstream_base_url', label: t('admin.mediaPlaygroundImage.columns.upstream') },
  { key: 'prices', label: t('admin.mediaPlaygroundImage.columns.prices') },
  { key: 'supported_sizes', label: t('admin.mediaPlaygroundImage.columns.sizes') },
  { key: 'sort_order', label: t('admin.mediaPlaygroundImage.columns.sortOrder') },
  { key: 'health', label: t('admin.mediaPlaygroundImage.columns.health') },
  { key: 'enabled', label: t('admin.mediaPlaygroundImage.columns.enabled') },
  { key: 'actions', label: t('common.actions') },
])

const probeRunColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.mediaPlaygroundImage.probeRuns.columns.createdAt') },
  { key: 'model', label: t('admin.mediaPlaygroundImage.probeRuns.columns.model') },
  { key: 'api_mode', label: t('admin.mediaPlaygroundImage.probeRuns.columns.apiMode') },
  { key: 'upstream_base_url', label: t('admin.mediaPlaygroundImage.probeRuns.columns.upstream') },
  { key: 'attempt', label: t('admin.mediaPlaygroundImage.probeRuns.columns.attempt') },
  { key: 'status', label: t('admin.mediaPlaygroundImage.probeRuns.columns.status') },
  { key: 'http_status_code', label: t('admin.mediaPlaygroundImage.probeRuns.columns.httpStatus') },
  { key: 'elapsed_ms', label: t('admin.mediaPlaygroundImage.probeRuns.columns.elapsed') },
  { key: 'response_bytes', label: t('admin.mediaPlaygroundImage.probeRuns.columns.responseBytes') },
  { key: 'image_count', label: t('admin.mediaPlaygroundImage.probeRuns.columns.imageCount') },
  { key: 'error_message', label: t('admin.mediaPlaygroundImage.probeRuns.columns.error') },
])

const hasNextProbeRunsPage = computed(() => probeRunsPage.value * probeRunsPageSize < probeRunsTotal.value)
const hasNextCallRecordsPage = computed(() => callRecordsPage.value * callRecordsPageSize < callRecordsTotal.value)

const callRecordColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.mediaPlaygroundImage.callRecords.columns.createdAt') },
  { key: 'id', label: t('admin.mediaPlaygroundImage.callRecords.columns.task') },
  { key: 'user', label: t('admin.mediaPlaygroundImage.callRecords.columns.user') },
  { key: 'model', label: t('admin.mediaPlaygroundImage.callRecords.columns.model') },
  { key: 'upstream_base_url', label: t('admin.mediaPlaygroundImage.callRecords.columns.upstream') },
  { key: 'status', label: t('admin.mediaPlaygroundImage.callRecords.columns.status') },
  { key: 'upstream_status_code', label: t('admin.mediaPlaygroundImage.callRecords.columns.httpStatus') },
  { key: 'response_bytes', label: t('admin.mediaPlaygroundImage.callRecords.columns.responseBytes') },
  { key: 'image_count', label: t('admin.mediaPlaygroundImage.callRecords.columns.imageCount') },
  { key: 'error_message', label: t('admin.mediaPlaygroundImage.callRecords.columns.error') },
])

async function loadModels() {
  loading.value = true
  try {
    models.value = await adminAPI.mediaPlaygroundImage.listModels()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function loadProbeRuns(page = 1) {
  probeRunsLoading.value = true
  try {
    const result = await adminAPI.mediaPlaygroundImage.listProbeRuns({ page, page_size: probeRunsPageSize })
    probeRuns.value = result.items || []
    probeRunsPage.value = result.page || page
    probeRunsTotal.value = result.total || 0
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.probeRuns.loadFailed')))
  } finally {
    probeRunsLoading.value = false
  }
}

async function loadCallRecords(page = 1) {
  callRecordsLoading.value = true
  try {
    const result = await adminAPI.mediaPlaygroundImage.listUpstreamRequests({ page, page_size: callRecordsPageSize })
    callRecords.value = result.items || []
    callRecordsPage.value = result.page || page
    callRecordsTotal.value = result.total || 0
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.callRecords.loadFailed')))
  } finally {
    callRecordsLoading.value = false
  }
}

function assignFormFromModel(model: MediaPlaygroundImageModel, options: { copyKey: boolean }) {
  Object.assign(form, {
    display_name: model.display_name,
    model: model.model,
    api_mode: model.api_mode || 'images',
    provider_name: model.provider_name,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: options.copyKey ? (model.upstream_api_key_mask || model.upstream_api_key || '') : '',
    price_1k: model.price_1k,
    price_2k: model.price_2k,
    price_4k: model.price_4k,
    supported_sizes: model.supported_sizes?.length ? [...model.supported_sizes] : [...sizeOptions],
    timeout_seconds: model.timeout_seconds,
    fallback_to_responses_enabled: model.fallback_to_responses_enabled ?? true,
    enabled: model.enabled,
    sort_order: model.sort_order,
  })
}

function openProbeRunsDialog() {
  showProbeRunsDialog.value = true
  void loadProbeRuns(1)
}

function closeProbeRunsDialog() {
  showProbeRunsDialog.value = false
}

function openCallRecordsDialog() {
  showCallRecordsDialog.value = true
  void loadCallRecords(1)
}

function closeCallRecordsDialog() {
  showCallRecordsDialog.value = false
}

async function runSingleModelProbe(model: MediaPlaygroundImageModel) {
  probingModelIds.value = new Set(probingModelIds.value).add(model.id)
  try {
    await adminAPI.mediaPlaygroundImage.runModelProbe(model.id)
    appStore.showSuccess(t('admin.mediaPlaygroundImage.probeRuns.singleRunSuccess', { name: model.display_name || model.model }))
    await loadModels()
    if (showProbeRunsDialog.value) {
      await loadProbeRuns(1)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.probeRuns.singleRunFailed')))
  } finally {
    const next = new Set(probingModelIds.value)
    next.delete(model.id)
    probingModelIds.value = next
  }
}

function openCreateDialog() {
  resetForm()
  showDialog.value = true
}

function openEditDialog(model: MediaPlaygroundImageModel) {
  editingId.value = model.id
  editingKeyMask.value = model.upstream_api_key_mask || model.upstream_api_key || ''
  reusedKeyMask.value = ''
  upstreamKeyVisible.value = false
  assignFormFromModel(model, { copyKey: true })
  showDialog.value = true
}

function openReuseDialog(model: MediaPlaygroundImageModel) {
  resetForm()
  assignFormFromModel(model, { copyKey: false })
  reusedKeyMask.value = model.upstream_api_key_mask || model.upstream_api_key || ''
  showDialog.value = true
}

function handleReuseSelect(event: Event) {
  const id = Number((event.target as HTMLSelectElement).value)
  const model = models.value.find((item) => item.id === id)
  if (model) {
    openReuseDialog(model)
  }
}

function apiModeLabel(mode: string) {
  if (mode === 'responses') return t('admin.mediaPlaygroundImage.apiModes.responses')
  if (mode === 'gemini_generate_content') return t('admin.mediaPlaygroundImage.apiModes.geminiGenerateContent')
  return t('admin.mediaPlaygroundImage.apiModes.images')
}

function probeStatusLabel(status: string) {
  return status === 'success'
    ? t('admin.mediaPlaygroundImage.probeRuns.status.success')
    : t('admin.mediaPlaygroundImage.probeRuns.status.failed')
}

function healthStatusLabel(status: string) {
  const normalized = status || 'available'
  if (normalized === 'temporary_unavailable') return t('admin.mediaPlaygroundImage.health.status.temporaryUnavailable')
  if (normalized === 'half_open') return t('admin.mediaPlaygroundImage.health.status.halfOpen')
  if (normalized === 'disabled') return t('admin.mediaPlaygroundImage.health.status.disabled')
  return t('admin.mediaPlaygroundImage.health.status.available')
}

function healthBadgeClass(status: string) {
  if (status === 'temporary_unavailable' || status === 'half_open') return 'badge badge-warning'
  if (status === 'disabled') return 'badge badge-danger'
  return 'badge badge-success'
}

function formatDurationMs(value: number) {
  if (!value || value < 0) return '-'
  if (value < 1000) return `${value}ms`
  return `${(value / 1000).toFixed(2)}s`
}

function resetForm() {
  editingId.value = null
  editingKeyMask.value = ''
  reusedKeyMask.value = ''
  upstreamKeyVisible.value = false
  Object.assign(form, defaultForm())
}

function closeDialog() {
  if (saving.value) return
  finishDialog()
}

function finishDialog() {
  showDialog.value = false
  resetForm()
}

function buildSavePayload(): MediaPlaygroundImageModelPayload {
  const payload: MediaPlaygroundImageModelPayload = { ...form, supported_sizes: [...form.supported_sizes] }
  if (editingId.value && editingKeyMask.value && payload.upstream_api_key === editingKeyMask.value) {
    payload.upstream_api_key = ''
  }
  return payload
}

function buildUpdatePayloadFromModel(model: MediaPlaygroundImageModel, overrides: Partial<MediaPlaygroundImageModelPayload> = {}): MediaPlaygroundImageModelPayload {
  return {
    display_name: model.display_name,
    model: model.model,
    api_mode: model.api_mode || 'images',
    provider_name: model.provider_name,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: '',
    price_1k: model.price_1k,
    price_2k: model.price_2k,
    price_4k: model.price_4k,
    supported_sizes: model.supported_sizes?.length ? [...model.supported_sizes] : [...sizeOptions],
    timeout_seconds: model.timeout_seconds,
    fallback_to_responses_enabled: model.fallback_to_responses_enabled ?? true,
    enabled: model.enabled,
    sort_order: model.sort_order,
    ...overrides,
  }
}

async function toggleModelEnabled(model: MediaPlaygroundImageModel) {
  togglingId.value = model.id
  try {
    await adminAPI.mediaPlaygroundImage.updateModel(model.id, buildUpdatePayloadFromModel(model, { enabled: !model.enabled }))
    await loadModels()
    appStore.showSuccess(model.enabled ? t('admin.mediaPlaygroundImage.disabledToast') : t('admin.mediaPlaygroundImage.enabledToast'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.saveFailed')))
  } finally {
    togglingId.value = null
  }
}

async function saveModel() {
  if (!form.supported_sizes.length) {
    appStore.showError(t('admin.mediaPlaygroundImage.sizeRequired'))
    return
  }
  saving.value = true
  try {
    const payload = buildSavePayload()
    if (editingId.value) {
      await adminAPI.mediaPlaygroundImage.updateModel(editingId.value, payload)
    } else {
      await adminAPI.mediaPlaygroundImage.createModel(payload)
    }
    finishDialog()
    await loadModels()
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.saveFailed')))
  } finally {
    saving.value = false
  }
}

function requestDeleteModel(model: MediaPlaygroundImageModel) {
  deletingModel.value = model
  showDeleteDialog.value = true
}

function closeDeleteDialog() {
  showDeleteDialog.value = false
  deletingModel.value = null
}

async function confirmDeleteModel() {
  if (!deletingModel.value) return
  try {
    await adminAPI.mediaPlaygroundImage.deleteModel(deletingModel.value.id)
    closeDeleteDialog()
    await loadModels()
    appStore.showSuccess(t('common.deleted'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.mediaPlaygroundImage.deleteFailed')))
  }
}

onMounted(loadModels)
</script>

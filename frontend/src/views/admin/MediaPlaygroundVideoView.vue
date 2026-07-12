<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.videoPlayground.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.videoPlayground.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadModels">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" class="mr-2" />
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-secondary" type="button" @click="openCallRecordsDialog">
            {{ t('admin.videoPlayground.callRecords.button') }}
          </button>
          <button class="btn btn-primary" type="button" @click="openCreateDialog">
            {{ t('admin.videoPlayground.createModel') }}
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
            <span class="badge badge-gray">{{ t(`admin.videoPlayground.apiModes.${row.api_mode}`) }}</span>
          </template>
          <template #cell-price_quota="{ row }">
            <span class="font-mono text-sm">{{ row.price_quota }}</span>
          </template>
          <template #cell-billing_mode="{ row }">
            {{ t(`admin.videoPlayground.billingModes.${row.billing_mode}`) }}
          </template>
          <template #cell-refund_enabled="{ row }">
            <span :class="row.refund_enabled ? 'badge badge-success' : 'badge badge-gray'">
              {{ row.refund_enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
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
              <button class="btn btn-sm btn-secondary" @click="openReuseDialog(row)">
                {{ t('admin.videoPlayground.reuse') }}
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
        width="extra-wide"
        :close-on-escape="!saving"
        :show-close-button="!saving"
        @close="closeDialog"
      >
        <form id="video-playground-model-form" class="space-y-4" @submit.prevent="saveModel">
          <label v-if="!editingId && models.length" class="block">
            <span class="input-label">{{ t('admin.videoPlayground.reuseFrom') }}</span>
            <select class="input" @change="handleReuseSelect">
              <option value="">{{ t('admin.videoPlayground.reuseFromPlaceholder') }}</option>
              <option v-for="model in models" :key="model.id" :value="model.id">
                {{ model.display_name }} · {{ model.model }}
              </option>
            </select>
          </label>

          <div class="grid gap-4 lg:grid-cols-2">
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.displayName') }}</span>
              <input v-model.trim="form.display_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.model') }}</span>
              <input v-model.trim="form.model" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.apiMode') }}</span>
              <select v-model="form.api_mode" class="input" required>
                <option value="openai_videos">{{ t('admin.videoPlayground.apiModes.openai_videos') }}</option>
                <option value="seedance_content_generation">{{ t('admin.videoPlayground.apiModes.seedance_content_generation') }}</option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.sortOrder') }}</span>
              <input v-model.number="form.sort_order" class="input" type="number" />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.providerName') }}</span>
              <input v-model.trim="form.provider_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.upstreamBaseURL') }}</span>
              <input v-model.trim="form.upstream_base_url" class="input" placeholder="https://api.example.com" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.upstreamAPIKey') }}</span>
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
                <span class="input-label">{{ t('admin.videoPlayground.fields.priceQuota') }}</span>
                <input v-model.number="form.price_quota" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.timeoutSeconds') }}</span>
                <input v-model.number="form.timeout_seconds" class="input" type="number" min="1" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.billingMode') }}</span>
                <select v-model="form.billing_mode" class="input">
                  <option value="balance_prepaid">{{ t('admin.videoPlayground.billingModes.balance_prepaid') }}</option>
                </select>
              </label>
          </div>

          <div class="flex flex-wrap gap-4">
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.refund_enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.videoPlayground.fields.refundEnabled') }}
            </label>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.videoPlayground.fields.enabled') }}
            </label>
          </div>
        </form>

        <template #footer>
          <div class="flex justify-end gap-3">
            <button class="btn btn-secondary" type="button" :disabled="saving" @click="closeDialog">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-primary" type="submit" form="video-playground-model-form" :disabled="saving">
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </template>
      </BaseDialog>

      <BaseDialog
        :show="showCallRecordsDialog"
        :title="t('admin.videoPlayground.callRecords.title')"
        width="wide"
        @close="closeCallRecordsDialog"
      >
        <div class="space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.videoPlayground.callRecords.description') }}
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
            <template #cell-task_id="{ row }">
              <div>
                <div class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ row.model_config_id }}</div>
                <div class="font-mono text-xs text-gray-900 dark:text-white">{{ row.task_id || '-' }}</div>
              </div>
            </template>
            <template #cell-model="{ row }">
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ row.model || '-' }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ row.provider_name || '-' }}</div>
              </div>
            </template>
            <template #cell-endpoint="{ row }">
              <div>
                <div class="font-mono text-xs">{{ row.method || '-' }} {{ row.endpoint || '-' }}</div>
                <div class="max-w-[220px] truncate text-xs text-gray-500 dark:text-gray-400" :title="row.upstream_base_url || '-'">{{ row.upstream_base_url || '-' }}</div>
              </div>
            </template>
            <template #cell-http_status_code="{ value }">
              <span class="font-mono text-xs">{{ value || '-' }}</span>
            </template>
            <template #cell-user_id="{ row }">
              <div class="font-mono text-xs">
                <div>用户ID {{ row.user_id || '-' }}</div>
                <div>API Key {{ row.api_key_suffix || '-' }}</div>
              </div>
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
              {{ t('admin.videoPlayground.callRecords.pageInfo', { page: callRecordsPage, total: callRecordsTotal }) }}
            </div>
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary" type="button" :disabled="callRecordsLoading || callRecordsPage <= 1" @click="loadCallRecords(callRecordsPage - 1)">
                {{ t('admin.videoPlayground.callRecords.previous') }}
              </button>
              <button class="btn btn-secondary" type="button" :disabled="callRecordsLoading || !hasNextCallRecordsPage" @click="loadCallRecords(callRecordsPage + 1)">
                {{ t('admin.videoPlayground.callRecords.next') }}
              </button>
            </div>
          </div>
        </div>
      </BaseDialog>

      <ConfirmDialog
        :show="showDeleteDialog"
        :title="t('admin.videoPlayground.deleteTitle')"
        :message="t('admin.videoPlayground.deleteConfirm', { name: deletingModel?.display_name || '' })"
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
import type { MediaPlaygroundVideoModel, MediaPlaygroundVideoModelPayload, MediaPlaygroundVideoUpstreamRequest } from '@/api/admin'
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

const models = ref<MediaPlaygroundVideoModel[]>([])
const callRecords = ref<MediaPlaygroundVideoUpstreamRequest[]>([])
const loading = ref(false)
const callRecordsLoading = ref(false)
const saving = ref(false)
const togglingId = ref<number | null>(null)
const editingId = ref<number | null>(null)
const editingUpstreamKeyMask = ref('')
const reusedUpstreamKeyMask = ref('')
const upstreamKeyVisible = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const showCallRecordsDialog = ref(false)
const deletingModel = ref<MediaPlaygroundVideoModel | null>(null)
const callRecordsPage = ref(1)
const callRecordsPageSize = 20
const callRecordsTotal = ref(0)

const defaultForm = (): MediaPlaygroundVideoModelPayload => ({
  display_name: '',
  model: '',
  provider_name: '',
  upstream_base_url: '',
  upstream_api_key: '',
  price_quota: 1,
  billing_mode: 'balance_prepaid',
  refund_enabled: true,
  timeout_seconds: 1800,
  enabled: true,
  sort_order: 0,
  api_mode: 'openai_videos',
})

const form = reactive<MediaPlaygroundVideoModelPayload>(defaultForm())

const dialogTitle = computed(() => {
  return editingId.value ? t('admin.videoPlayground.editModel') : t('admin.videoPlayground.createModel')
})

const upstreamKeyPlaceholder = computed(() => {
  if (editingId.value && editingUpstreamKeyMask.value) {
    return t('admin.videoPlayground.keyConfigured', { mask: editingUpstreamKeyMask.value })
  }
  if (reusedUpstreamKeyMask.value) {
    return t('admin.videoPlayground.keyNotCopied', { mask: reusedUpstreamKeyMask.value })
  }
  return 'sk-...'
})

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.videoPlayground.columns.name') },
  { key: 'provider_name', label: t('admin.videoPlayground.columns.provider') },
  { key: 'api_mode', label: t('admin.videoPlayground.columns.apiMode') },
  { key: 'upstream_base_url', label: t('admin.videoPlayground.columns.upstream') },
  { key: 'price_quota', label: t('admin.videoPlayground.columns.price') },
  { key: 'billing_mode', label: t('admin.videoPlayground.columns.billingMode') },
  { key: 'refund_enabled', label: t('admin.videoPlayground.columns.refund') },
  { key: 'enabled', label: t('admin.videoPlayground.columns.enabled') },
  { key: 'actions', label: t('common.actions') },
])

const callRecordColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.videoPlayground.callRecords.columns.createdAt') },
  { key: 'task_id', label: t('admin.videoPlayground.callRecords.columns.task') },
  { key: 'user_id', label: t('admin.videoPlayground.callRecords.columns.user') },
  { key: 'model', label: t('admin.videoPlayground.callRecords.columns.model') },
  { key: 'endpoint', label: t('admin.videoPlayground.callRecords.columns.endpoint') },
  { key: 'http_status_code', label: t('admin.videoPlayground.callRecords.columns.httpStatus') },
  { key: 'elapsed_ms', label: t('admin.videoPlayground.callRecords.columns.elapsed') },
  { key: 'response_bytes', label: t('admin.videoPlayground.callRecords.columns.responseBytes') },
  { key: 'error_message', label: t('admin.videoPlayground.callRecords.columns.error') },
])

const hasNextCallRecordsPage = computed(() => callRecordsPage.value * callRecordsPageSize < callRecordsTotal.value)

async function loadModels() {
  loading.value = true
  try {
    models.value = await adminAPI.mediaPlaygroundVideo.listModels()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.videoPlayground.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function loadCallRecords(page = 1) {
  callRecordsLoading.value = true
  try {
    const result = await adminAPI.mediaPlaygroundVideo.listUpstreamRequests({ page, page_size: callRecordsPageSize })
    callRecords.value = result.items || []
    callRecordsPage.value = result.page || page
    callRecordsTotal.value = result.total || 0
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.videoPlayground.callRecords.loadFailed')))
  } finally {
    callRecordsLoading.value = false
  }
}

function openCallRecordsDialog() {
  showCallRecordsDialog.value = true
  void loadCallRecords(1)
}

function closeCallRecordsDialog() {
  showCallRecordsDialog.value = false
}

function formatDurationMs(value: number) {
  if (!value || value < 0) return '-'
  if (value < 1000) return `${value}ms`
  return `${(value / 1000).toFixed(2)}s`
}

function assignFormFromModel(model: MediaPlaygroundVideoModel) {
  Object.assign(form, {
    display_name: model.display_name,
    model: model.model,
    provider_name: model.provider_name,
    api_mode: model.api_mode,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: '',
    price_quota: model.price_quota,
    billing_mode: model.billing_mode,
    refund_enabled: model.refund_enabled,
    timeout_seconds: model.timeout_seconds,
    enabled: model.enabled,
    sort_order: model.sort_order,
  })
}

function openCreateDialog() {
  resetForm()
  showDialog.value = true
}

function openEditDialog(model: MediaPlaygroundVideoModel) {
  editingId.value = model.id
  editingUpstreamKeyMask.value = model.upstream_api_key_mask || ''
  reusedUpstreamKeyMask.value = ''
  upstreamKeyVisible.value = false
  assignFormFromModel(model)
  showDialog.value = true
}

function openReuseDialog(model: MediaPlaygroundVideoModel) {
  resetForm()
  assignFormFromModel(model)
  reusedUpstreamKeyMask.value = model.upstream_api_key_mask || ''
  showDialog.value = true
}

function handleReuseSelect(event: Event) {
  const id = Number((event.target as HTMLSelectElement).value)
  const model = models.value.find((item) => item.id === id)
  if (model) {
    openReuseDialog(model)
  }
}

function resetForm() {
  editingId.value = null
  editingUpstreamKeyMask.value = ''
  reusedUpstreamKeyMask.value = ''
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

function buildUpdatePayloadFromModel(model: MediaPlaygroundVideoModel, overrides: Partial<MediaPlaygroundVideoModelPayload> = {}): MediaPlaygroundVideoModelPayload {
  return {
    display_name: model.display_name,
    model: model.model,
    provider_name: model.provider_name,
    api_mode: model.api_mode,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: '',
    price_quota: model.price_quota,
    billing_mode: model.billing_mode,
    refund_enabled: model.refund_enabled,
    timeout_seconds: model.timeout_seconds,
    enabled: model.enabled,
    sort_order: model.sort_order,
    ...overrides,
  }
}

async function toggleModelEnabled(model: MediaPlaygroundVideoModel) {
  togglingId.value = model.id
  try {
    await adminAPI.mediaPlaygroundVideo.updateModel(model.id, buildUpdatePayloadFromModel(model, { enabled: !model.enabled }))
    await loadModels()
    appStore.showSuccess(model.enabled ? t('admin.videoPlayground.disabledToast') : t('admin.videoPlayground.enabledToast'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.videoPlayground.saveFailed')))
  } finally {
    togglingId.value = null
  }
}

async function saveModel() {
  saving.value = true
  try {
    const payload = { ...form }
    if (editingId.value && !payload.upstream_api_key.trim()) {
      payload.upstream_api_key = ''
    }
    if (editingId.value) {
      await adminAPI.mediaPlaygroundVideo.updateModel(editingId.value, payload)
    } else {
      await adminAPI.mediaPlaygroundVideo.createModel(payload)
    }
    finishDialog()
    await loadModels()
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.videoPlayground.saveFailed')))
  } finally {
    saving.value = false
  }
}

function requestDeleteModel(model: MediaPlaygroundVideoModel) {
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
    await adminAPI.mediaPlaygroundVideo.deleteModel(deletingModel.value.id)
    closeDeleteDialog()
    await loadModels()
    appStore.showSuccess(t('common.deleted'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.videoPlayground.deleteFailed')))
  }
}

onMounted(loadModels)
</script>

<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.imagePlayground.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.imagePlayground.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadModels">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" class="mr-2" />
            {{ t('common.refresh') }}
          </button>
          <button class="btn btn-primary" type="button" @click="openCreateDialog">
            {{ t('admin.imagePlayground.createModel') }}
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
                {{ t('admin.imagePlayground.reuse') }}
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
        <form id="image-playground-model-form" class="space-y-4" @submit.prevent="saveModel">
          <label v-if="!editingId && models.length" class="block">
            <span class="input-label">{{ t('admin.imagePlayground.reuseFrom') }}</span>
            <select class="input" @change="handleReuseSelect">
              <option value="">{{ t('admin.imagePlayground.reuseFromPlaceholder') }}</option>
              <option v-for="model in models" :key="model.id" :value="model.id">
                {{ model.display_name }} · {{ model.model }}
              </option>
            </select>
          </label>

          <div class="grid gap-4 lg:grid-cols-2">
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.displayName') }}</span>
              <input v-model.trim="form.display_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.model') }}</span>
              <input v-model.trim="form.model" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.apiMode') }}</span>
              <select v-model="form.api_mode" class="input" required>
                <option value="images">{{ t('admin.imagePlayground.apiModes.images') }}</option>
                <option value="responses">{{ t('admin.imagePlayground.apiModes.responses') }}</option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.providerName') }}</span>
              <input v-model.trim="form.provider_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.upstreamBaseURL') }}</span>
              <input v-model.trim="form.upstream_base_url" class="input" placeholder="https://api.example.com" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.upstreamAPIKey') }}</span>
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
                <span class="input-label">{{ t('admin.imagePlayground.fields.price1k') }}</span>
                <input v-model.number="form.price_1k" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.imagePlayground.fields.price2k') }}</span>
                <input v-model.number="form.price_2k" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.imagePlayground.fields.price4k') }}</span>
                <input v-model.number="form.price_4k" class="input" type="number" min="0" step="0.000001" required />
              </label>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
              <label class="block">
                <span class="input-label">{{ t('admin.imagePlayground.fields.timeoutSeconds') }}</span>
                <input v-model.number="form.timeout_seconds" class="input" type="number" min="1" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.imagePlayground.fields.sortOrder') }}</span>
                <input v-model.number="form.sort_order" class="input" type="number" />
              </label>
          </div>

          <div>
            <span class="input-label">{{ t('admin.imagePlayground.fields.supportedSizes') }}</span>
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
              {{ t('admin.imagePlayground.fields.enabled') }}
            </label>
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.fallback_to_responses_enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.imagePlayground.fields.fallbackToResponses') }}
            </label>
          </div>
        </form>

        <template #footer>
          <div class="flex justify-end gap-3">
            <button class="btn btn-secondary" type="button" :disabled="saving" @click="closeDialog">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-primary" type="submit" form="image-playground-model-form" :disabled="saving">
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </template>
      </BaseDialog>

      <ConfirmDialog
        :show="showDeleteDialog"
        :title="t('admin.imagePlayground.deleteTitle')"
        :message="t('admin.imagePlayground.deleteConfirm', { name: deletingModel?.display_name || '' })"
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
import type { ImagePlaygroundModel, ImagePlaygroundModelPayload, ImageSizeTier } from '@/api/admin'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const sizeOptions: ImageSizeTier[] = ['1k', '2k', '4k']

const models = ref<ImagePlaygroundModel[]>([])
const loading = ref(false)
const saving = ref(false)
const togglingId = ref<number | null>(null)
const editingId = ref<number | null>(null)
const editingKeyMask = ref('')
const reusedKeyMask = ref('')
const upstreamKeyVisible = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const deletingModel = ref<ImagePlaygroundModel | null>(null)

const defaultForm = (): ImagePlaygroundModelPayload => ({
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

const form = reactive<ImagePlaygroundModelPayload>(defaultForm())

const dialogTitle = computed(() => {
  return editingId.value ? t('admin.imagePlayground.editModel') : t('admin.imagePlayground.createModel')
})

const upstreamKeyPlaceholder = computed(() => {
  if (editingId.value && editingKeyMask.value) {
    return t('admin.imagePlayground.keyConfigured', { mask: editingKeyMask.value })
  }
  if (reusedKeyMask.value) {
    return t('admin.imagePlayground.keyNotCopied', { mask: reusedKeyMask.value })
  }
  return 'sk-...'
})

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.imagePlayground.columns.name') },
  { key: 'api_mode', label: t('admin.imagePlayground.columns.apiMode') },
  { key: 'provider_name', label: t('admin.imagePlayground.columns.provider') },
  { key: 'upstream_base_url', label: t('admin.imagePlayground.columns.upstream') },
  { key: 'prices', label: t('admin.imagePlayground.columns.prices') },
  { key: 'supported_sizes', label: t('admin.imagePlayground.columns.sizes') },
  { key: 'sort_order', label: t('admin.imagePlayground.columns.sortOrder') },
  { key: 'enabled', label: t('admin.imagePlayground.columns.enabled') },
  { key: 'actions', label: t('common.actions') },
])

async function loadModels() {
  loading.value = true
  try {
    models.value = await adminAPI.imagePlayground.listModels()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.imagePlayground.loadFailed')))
  } finally {
    loading.value = false
  }
}

function assignFormFromModel(model: ImagePlaygroundModel, options: { copyKey: boolean }) {
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

function openCreateDialog() {
  resetForm()
  showDialog.value = true
}

function openEditDialog(model: ImagePlaygroundModel) {
  editingId.value = model.id
  editingKeyMask.value = model.upstream_api_key_mask || model.upstream_api_key || ''
  reusedKeyMask.value = ''
  upstreamKeyVisible.value = false
  assignFormFromModel(model, { copyKey: true })
  showDialog.value = true
}

function openReuseDialog(model: ImagePlaygroundModel) {
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
  return mode === 'responses'
    ? t('admin.imagePlayground.apiModes.responses')
    : t('admin.imagePlayground.apiModes.images')
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

function buildSavePayload(): ImagePlaygroundModelPayload {
  const payload: ImagePlaygroundModelPayload = { ...form, supported_sizes: [...form.supported_sizes] }
  if (editingId.value && editingKeyMask.value && payload.upstream_api_key === editingKeyMask.value) {
    payload.upstream_api_key = ''
  }
  return payload
}

function buildUpdatePayloadFromModel(model: ImagePlaygroundModel, overrides: Partial<ImagePlaygroundModelPayload> = {}): ImagePlaygroundModelPayload {
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

async function toggleModelEnabled(model: ImagePlaygroundModel) {
  togglingId.value = model.id
  try {
    await adminAPI.imagePlayground.updateModel(model.id, buildUpdatePayloadFromModel(model, { enabled: !model.enabled }))
    await loadModels()
    appStore.showSuccess(model.enabled ? t('admin.imagePlayground.disabledToast') : t('admin.imagePlayground.enabledToast'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.imagePlayground.saveFailed')))
  } finally {
    togglingId.value = null
  }
}

async function saveModel() {
  if (!form.supported_sizes.length) {
    appStore.showError(t('admin.imagePlayground.sizeRequired'))
    return
  }
  saving.value = true
  try {
    const payload = buildSavePayload()
    if (editingId.value) {
      await adminAPI.imagePlayground.updateModel(editingId.value, payload)
    } else {
      await adminAPI.imagePlayground.createModel(payload)
    }
    finishDialog()
    await loadModels()
    appStore.showSuccess(t('common.saved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.imagePlayground.saveFailed')))
  } finally {
    saving.value = false
  }
}

function requestDeleteModel(model: ImagePlaygroundModel) {
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
    await adminAPI.imagePlayground.deleteModel(deletingModel.value.id)
    closeDeleteDialog()
    await loadModels()
    appStore.showSuccess(t('common.deleted'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.imagePlayground.deleteFailed')))
  }
}

onMounted(loadModels)
</script>

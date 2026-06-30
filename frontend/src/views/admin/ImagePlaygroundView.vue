<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.imagePlayground.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.imagePlayground.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" @click="loadModels">
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" class="mr-2" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
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
            <template #cell-enabled="{ row }">
              <span :class="row.enabled ? 'badge badge-success' : 'badge badge-gray'">
                {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center gap-2">
                <button class="btn btn-sm btn-secondary" @click="editModel(row)">
                  {{ t('common.edit') }}
                </button>
                <button class="btn btn-sm btn-danger" @click="deleteModel(row)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </template>
          </DataTable>
        </div>

        <form class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900" @submit.prevent="saveModel">
          <div class="mb-4 flex items-center justify-between gap-3">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ editingId ? t('admin.imagePlayground.editModel') : t('admin.imagePlayground.createModel') }}
            </h2>
            <button v-if="editingId" type="button" class="btn btn-sm btn-secondary" @click="resetForm">
              {{ t('common.cancel') }}
            </button>
          </div>

          <div class="space-y-4">
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.displayName') }}</span>
              <input v-model.trim="form.display_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.imagePlayground.fields.model') }}</span>
              <input v-model.trim="form.model" class="input" required />
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
              <input v-model.trim="form.upstream_api_key" class="input" type="password" autocomplete="off" placeholder="sk-..." />
            </label>
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
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.imagePlayground.fields.enabled') }}
            </label>
            <button class="btn btn-primary w-full" type="submit" :disabled="saving">
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </form>
      </section>
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
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const sizeOptions: ImageSizeTier[] = ['1k', '2k', '4k']

const models = ref<ImagePlaygroundModel[]>([])
const loading = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const editingKeyMask = ref('')

const defaultForm = (): ImagePlaygroundModelPayload => ({
  display_name: '',
  model: '',
  provider_name: '',
  upstream_base_url: '',
  upstream_api_key: '',
  price_1k: 1,
  price_2k: 2,
  price_4k: 4,
  supported_sizes: ['1k', '2k', '4k'],
  timeout_seconds: 600,
  enabled: true,
  sort_order: 0,
})

const form = reactive<ImagePlaygroundModelPayload>(defaultForm())

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.imagePlayground.columns.name') },
  { key: 'provider_name', label: t('admin.imagePlayground.columns.provider') },
  { key: 'upstream_base_url', label: t('admin.imagePlayground.columns.upstream') },
  { key: 'prices', label: t('admin.imagePlayground.columns.prices') },
  { key: 'supported_sizes', label: t('admin.imagePlayground.columns.sizes') },
  { key: 'enabled', label: t('admin.imagePlayground.columns.enabled') },
  { key: 'actions', label: t('common.actions') },
])

async function loadModels() {
  loading.value = true
  try {
    models.value = await adminAPI.imagePlayground.listModels()
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.imagePlayground.loadFailed')))
  } finally {
    loading.value = false
  }
}

function editModel(model: ImagePlaygroundModel) {
  editingId.value = model.id
  editingKeyMask.value = model.upstream_api_key_mask || model.upstream_api_key || ''
  Object.assign(form, {
    display_name: model.display_name,
    model: model.model,
    provider_name: model.provider_name,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: editingKeyMask.value,
    price_1k: model.price_1k,
    price_2k: model.price_2k,
    price_4k: model.price_4k,
    supported_sizes: model.supported_sizes?.length ? [...model.supported_sizes] : [...sizeOptions],
    timeout_seconds: model.timeout_seconds,
    enabled: model.enabled,
    sort_order: model.sort_order,
  })
}

function resetForm() {
  editingId.value = null
  editingKeyMask.value = ''
  Object.assign(form, defaultForm())
}

function buildSavePayload(): ImagePlaygroundModelPayload {
  const payload: ImagePlaygroundModelPayload = { ...form, supported_sizes: [...form.supported_sizes] }
  if (editingId.value && editingKeyMask.value && payload.upstream_api_key === editingKeyMask.value) {
    payload.upstream_api_key = ''
  }
  return payload
}

async function saveModel() {
  if (!form.supported_sizes.length) {
    appStore.showToast('error', t('admin.imagePlayground.sizeRequired'))
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
    resetForm()
    await loadModels()
    appStore.showToast('success', t('common.saved'))
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.imagePlayground.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function deleteModel(model: ImagePlaygroundModel) {
  if (!window.confirm(t('admin.imagePlayground.deleteConfirm', { name: model.display_name }))) {
    return
  }
  try {
    await adminAPI.imagePlayground.deleteModel(model.id)
    await loadModels()
    appStore.showToast('success', t('common.deleted'))
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.imagePlayground.deleteFailed')))
  }
}

onMounted(loadModels)
</script>

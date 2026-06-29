<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.videoPlayground.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.videoPlayground.description') }}</p>
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
              {{ editingId ? t('admin.videoPlayground.editModel') : t('admin.videoPlayground.createModel') }}
            </h2>
            <button v-if="editingId" type="button" class="btn btn-sm btn-secondary" @click="resetForm">
              {{ t('common.cancel') }}
            </button>
          </div>

          <div class="space-y-4">
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.displayName') }}</span>
              <input v-model.trim="form.display_name" class="input" required />
            </label>
            <label class="block">
              <span class="input-label">{{ t('admin.videoPlayground.fields.model') }}</span>
              <input v-model.trim="form.model" class="input" required />
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
              <input v-model.trim="form.upstream_api_key" class="input" type="password" autocomplete="off" placeholder="sk-..." />
            </label>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.priceQuota') }}</span>
                <input v-model.number="form.price_quota" class="input" type="number" min="0" step="0.000001" required />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.timeoutSeconds') }}</span>
                <input v-model.number="form.timeout_seconds" class="input" type="number" min="1" required />
              </label>
            </div>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.billingMode') }}</span>
                <select v-model="form.billing_mode" class="input">
                  <option value="balance_prepaid">{{ t('admin.videoPlayground.billingModes.balance_prepaid') }}</option>
                </select>
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.sortOrder') }}</span>
                <input v-model.number="form.sort_order" class="input" type="number" />
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
import type { VideoPlaygroundModel, VideoPlaygroundModelPayload } from '@/api/admin'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<VideoPlaygroundModel[]>([])
const loading = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const defaultForm = (): VideoPlaygroundModelPayload => ({
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
})

const form = reactive<VideoPlaygroundModelPayload>(defaultForm())

const columns = computed<Column[]>(() => [
  { key: 'display_name', label: t('admin.videoPlayground.columns.name') },
  { key: 'provider_name', label: t('admin.videoPlayground.columns.provider') },
  { key: 'upstream_base_url', label: t('admin.videoPlayground.columns.upstream') },
  { key: 'price_quota', label: t('admin.videoPlayground.columns.price') },
  { key: 'billing_mode', label: t('admin.videoPlayground.columns.billingMode') },
  { key: 'refund_enabled', label: t('admin.videoPlayground.columns.refund') },
  { key: 'enabled', label: t('admin.videoPlayground.columns.enabled') },
  { key: 'actions', label: t('common.actions') },
])

async function loadModels() {
  loading.value = true
  try {
    models.value = await adminAPI.videoPlayground.listModels()
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.videoPlayground.loadFailed')))
  } finally {
    loading.value = false
  }
}

function editModel(model: VideoPlaygroundModel) {
  editingId.value = model.id
  Object.assign(form, {
    display_name: model.display_name,
    model: model.model,
    provider_name: model.provider_name,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: model.upstream_api_key || '',
    price_quota: model.price_quota,
    billing_mode: model.billing_mode,
    refund_enabled: model.refund_enabled,
    timeout_seconds: model.timeout_seconds,
    enabled: model.enabled,
    sort_order: model.sort_order,
  })
}

function resetForm() {
  editingId.value = null
  Object.assign(form, defaultForm())
}

async function saveModel() {
  saving.value = true
  try {
    if (editingId.value) {
      await adminAPI.videoPlayground.updateModel(editingId.value, { ...form })
    } else {
      await adminAPI.videoPlayground.createModel({ ...form })
    }
    resetForm()
    await loadModels()
    appStore.showToast('success', t('common.saved'))
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.videoPlayground.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function deleteModel(model: VideoPlaygroundModel) {
  if (!window.confirm(t('admin.videoPlayground.deleteConfirm', { name: model.display_name }))) {
    return
  }
  try {
    await adminAPI.videoPlayground.deleteModel(model.id)
    await loadModels()
    appStore.showToast('success', t('common.deleted'))
  } catch (err) {
    appStore.showToast('error', extractApiErrorMessage(err, t('admin.videoPlayground.deleteFailed')))
  }
}

onMounted(loadModels)
</script>

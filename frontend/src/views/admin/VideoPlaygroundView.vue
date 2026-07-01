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
              <span class="input-label">{{ t('admin.videoPlayground.fields.studioTemplate') }}</span>
              <select v-model="form.studio_model_id" class="input" @change="applyStudioTemplate">
                <option value="">{{ t('admin.videoPlayground.templates.baseModel') }}</option>
                <option v-for="template in seedanceTemplates" :key="template.id" :value="template.id">
                  {{ template.name }}
                </option>
              </select>
            </label>
            <div class="grid gap-4 sm:grid-cols-2">
              <div class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.modelKind') }}</span>
                <div class="grid grid-cols-2 gap-2">
                  <label
                    v-for="kind in modelKindOptions"
                    :key="kind"
                    class="flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-sm transition"
                    :class="form.model_kind === kind ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-200' : 'border-gray-200 text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:text-gray-200'"
                  >
                    <input
                      type="checkbox"
                      class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="form.model_kind === kind"
                      @change="selectModelKind(kind)"
                    />
                    <span>{{ t(`admin.videoPlayground.modelKinds.${kind}`) }}</span>
                  </label>
                </div>
              </div>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.sortOrder') }}</span>
                <input v-model.number="form.sort_order" class="input" type="number" />
              </label>
            </div>
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
            </div>
            <div class="space-y-3">
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.inputSchemaJSON') }}</span>
                <textarea v-model.trim="form.input_schema_json" class="input font-mono text-xs" rows="4" />
              </label>
              <label class="block">
                <span class="input-label">{{ t('admin.videoPlayground.fields.payloadMappingJSON') }}</span>
                <textarea v-model.trim="form.payload_mapping_json" class="input font-mono text-xs" rows="3" />
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
const editingUpstreamKeyMask = ref('')
const upstreamKeyVisible = ref(false)
const modelKindOptions = ['t2v', 'i2v', 'reference_video', 'extend'] as const

const seedanceTemplates = [
  { id: 'seedance-lite-t2v', name: 'Seedance Lite T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9', '9:21'], default: '16:9' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' } } }, mapping: {} },
  { id: 'seedance-pro-t2v', name: 'Seedance Pro T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9', '9:21'], default: '16:9' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' } } }, mapping: {} },
  { id: 'seedance-pro-t2v-fast', name: 'Seedance Pro Fast T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '4:3', '3:4', '21:9'], default: '16:9' }, duration: { default: 5, minValue: 2, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' } } }, mapping: {} },
  { id: 'seedance-v1.5-pro-t2v', name: 'Seedance v1.5 Pro T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '3:4', '4:3', '21:9'], default: '16:9' }, duration: { default: 5, minValue: 4, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '720p' } } }, mapping: {} },
  { id: 'seedance-v1.5-pro-t2v-fast', name: 'Seedance v1.5 Pro Fast T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '3:4', '4:3', '21:9'], default: '16:9' }, duration: { default: 5, minValue: 4, maxValue: 12, step: 1 }, resolution: { enum: ['720p', '1080p'], default: '720p' } } }, mapping: {} },
  { id: 'seedance-v2.0-t2v', name: 'Seedance 2.0 T2V', kind: 't2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '4:3', '3:4'], default: '16:9' }, duration: { enum: [5, 10, 15], default: 5 }, quality: { enum: ['high', 'basic'], default: 'basic' } } }, mapping: {} },
  { id: 'seedance-lite-i2v', name: 'Seedance Lite I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' }, camera_fixed: { type: 'boolean', default: false } } }, mapping: { image_field: 'image_url', last_image_field: 'last_image', max_images: 2 } },
  { id: 'seedance-pro-i2v', name: 'Seedance Pro I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' }, camera_fixed: { type: 'boolean', default: false } } }, mapping: { image_field: 'image_url', max_images: 1 } },
  { id: 'seedance-pro-i2v-fast', name: 'Seedance Pro Fast I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '480p' }, camera_fixed: { type: 'boolean', default: false } } }, mapping: { image_field: 'image_url', max_images: 1 } },
  { id: 'seedance-v1.5-pro-i2v', name: 'Seedance v1.5 Pro I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '3:4', '4:3', '21:9'], default: '16:9' }, duration: { default: 5, minValue: 4, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p', '1080p'], default: '720p' }, generate_audio: { type: 'boolean', default: true }, camera_fixed: { type: 'boolean', default: false } } }, mapping: { image_field: 'image_url', last_image_field: 'last_image', max_images: 2 } },
  { id: 'seedance-v1.5-pro-i2v-fast', name: 'Seedance v1.5 Pro Fast I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '1:1', '3:4', '4:3', '21:9'], default: '16:9' }, duration: { default: 5, minValue: 4, maxValue: 12, step: 1 }, resolution: { enum: ['720p', '1080p'], default: '720p' }, generate_audio: { type: 'boolean', default: true }, camera_fixed: { type: 'boolean', default: false } } }, mapping: { image_field: 'image_url', last_image_field: 'last_image', max_images: 2 } },
  { id: 'seedance-v2.0-i2v', name: 'Seedance 2.0 I2V', kind: 'i2v', schema: { inputs: { prompt: { type: 'string' }, aspect_ratio: { enum: ['16:9', '9:16', '4:3', '3:4'], default: '16:9' }, duration: { enum: [5, 10, 15], default: 5 }, quality: { enum: ['high', 'basic'], default: 'basic' } } }, mapping: { image_field: 'images_list', max_images: 5 } },
  { id: 'seedance-lite-reference-video', name: 'Seedance Lite Reference Video', kind: 'reference_video', schema: { inputs: { prompt: { type: 'string' }, duration: { default: 5, minValue: 3, maxValue: 12, step: 1 }, resolution: { enum: ['480p', '720p'], default: '480p' } } }, mapping: { image_field: 'images_list', max_images: 4 } },
  { id: 'seedance-v2.0-extend', name: 'Seedance 2.0 Extend', kind: 'extend', schema: { inputs: { request_id: { type: 'string' }, prompt: { type: 'string' }, duration: { enum: [5, 10, 15], default: 5 }, quality: { enum: ['high', 'basic'], default: 'basic' } } }, mapping: { requires_source_task: true } },
] as const

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
  studio_model_id: '',
  model_kind: 't2v',
  input_schema_json: '{}',
  payload_mapping_json: '{}',
})

const form = reactive<VideoPlaygroundModelPayload>(defaultForm())

const upstreamKeyPlaceholder = computed(() => {
  if (editingId.value && editingUpstreamKeyMask.value) {
    return t('admin.videoPlayground.keyConfigured', { mask: editingUpstreamKeyMask.value })
  }
  return 'sk-...'
})

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
  upstreamKeyVisible.value = false
  Object.assign(form, {
    display_name: model.display_name,
    model: model.model,
    provider_name: model.provider_name,
    upstream_base_url: model.upstream_base_url,
    upstream_api_key: '',
    price_quota: model.price_quota,
    billing_mode: model.billing_mode,
    refund_enabled: model.refund_enabled,
    timeout_seconds: model.timeout_seconds,
    enabled: model.enabled,
    sort_order: model.sort_order,
    studio_model_id: model.studio_model_id || '',
    model_kind: model.model_kind || 't2v',
    input_schema_json: model.input_schema_json || '{}',
    payload_mapping_json: model.payload_mapping_json || '{}',
  })
  editingUpstreamKeyMask.value = model.upstream_api_key_mask || ''
}

function applyStudioTemplate() {
  const template = seedanceTemplates.find((item) => item.id === form.studio_model_id)
  if (!template) {
    form.model_kind = 't2v'
    form.input_schema_json = '{}'
    form.payload_mapping_json = '{}'
    return
  }
  form.model_kind = template.kind
  form.input_schema_json = JSON.stringify(template.schema)
  form.payload_mapping_json = JSON.stringify(template.mapping)
  if (!form.display_name) form.display_name = template.name
  if (!form.model) form.model = template.id
}

function selectModelKind(kind: typeof modelKindOptions[number]) {
  form.model_kind = kind
}

function resetForm() {
  editingId.value = null
  editingUpstreamKeyMask.value = ''
  upstreamKeyVisible.value = false
  Object.assign(form, defaultForm())
}

async function saveModel() {
  saving.value = true
  try {
    parseJSONObject(form.input_schema_json, t('admin.videoPlayground.fields.inputSchemaJSON'))
    parseJSONObject(form.payload_mapping_json, t('admin.videoPlayground.fields.payloadMappingJSON'))
    const payload = { ...form }
    if (editingId.value && !payload.upstream_api_key.trim()) {
      payload.upstream_api_key = ''
    }
    if (editingId.value) {
      await adminAPI.videoPlayground.updateModel(editingId.value, payload)
    } else {
      await adminAPI.videoPlayground.createModel(payload)
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

function parseJSONObject(raw: string, label: string) {
  try {
    const parsed = JSON.parse(raw || '{}')
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error(t('admin.videoPlayground.jsonMustBeObject', { field: label }))
    }
  } catch (err) {
    if (err instanceof Error && err.message.includes(label)) {
      throw err
    }
    throw new Error(t('admin.videoPlayground.invalidJSON', { field: label }))
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

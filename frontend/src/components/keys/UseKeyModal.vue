<template>
  <BaseDialog
    :show="show"
    :title="t('keys.useKeyModal.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <!-- No Group Assigned Warning -->
      <div v-if="!platform" class="flex items-start gap-3 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
        <svg class="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
        </svg>
        <div>
          <p class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
            {{ t('keys.useKeyModal.noGroupTitle') }}
          </p>
          <p class="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
            {{ t('keys.useKeyModal.noGroupDescription') }}
          </p>
        </div>
      </div>

      <!-- Platform-specific content -->
      <template v-else>
        <!-- Description -->
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ platformDescription }}
        </p>

        <!-- 管理员配置无效提示：配置已被忽略，但回退必须可见，不能静默 -->
        <div
          v-if="showUseKeyConfigInvalidNotice"
          data-testid="use-key-config-invalid-notice"
          class="flex items-start gap-3 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800"
        >
          <Icon name="exclamationCircle" size="md" class="text-amber-500 flex-shrink-0 mt-0.5" />
          <p class="text-sm text-amber-800 dark:text-amber-200">
            {{ t('keys.useKeyModal.useKeyConfigInvalid') }}
          </p>
        </div>

        <!-- 白名单交集为空提示：fail-closed，不展示任何客户端，但必须显式说明 -->
        <div
          v-if="showNoClientAllowedNotice"
          data-testid="use-key-config-no-client-notice"
          class="flex items-start gap-3 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800"
        >
          <Icon name="exclamationCircle" size="md" class="text-amber-500 flex-shrink-0 mt-0.5" />
          <p class="text-sm text-amber-800 dark:text-amber-200">
            {{ t('keys.useKeyModal.useKeyConfigNoClient') }}
          </p>
        </div>

        <!-- 无可用客户端时不渲染任何 tab、认证模式、目录区块与配置文件内容 -->
        <template v-if="hasAvailableClients">
        <!-- Client Tabs -->
        <div v-if="clientTabs.length" class="overflow-x-auto border-b border-gray-200 dark:border-dark-700">
          <nav class="-mb-px flex min-w-max gap-4 sm:gap-6" aria-label="Client">
            <button
              v-for="tab in clientTabs"
              :key="tab.id"
              type="button"
              @click="activeClientTab = tab.id"
              :class="[
                'whitespace-nowrap py-2.5 px-1 border-b-2 font-medium text-sm transition-colors',
                activeClientTab === tab.id
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              ]"
            >
              <span class="flex items-center gap-2">
                <component :is="tab.icon" class="w-4 h-4" />
                {{ tab.label }}
              </span>
            </button>
          </nav>
        </div>

        <!-- Codex Authentication Mode -->
        <div
          v-if="showCodexAuthMode"
          class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
        >
          <div class="mb-2">
            <p class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('keys.useKeyModal.openai.authModeTitle') }}
            </p>
            <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('keys.useKeyModal.openai.authModeDescription') }}
            </p>
          </div>
          <div
            class="grid grid-cols-2 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
            role="radiogroup"
            :aria-label="t('keys.useKeyModal.openai.authModeTitle')"
          >
            <button
              type="button"
              role="radio"
              data-testid="codex-auth-mode-legacy"
              :aria-checked="codexAuthMode === 'legacy'"
              :class="[
                'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                codexAuthMode === 'legacy'
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                  : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
              ]"
              @click="codexAuthMode = 'legacy'"
            >
              {{ t('keys.useKeyModal.openai.authModeLegacy') }}
            </button>
            <button
              type="button"
              role="radio"
              data-testid="codex-auth-mode-api-key"
              :aria-checked="codexAuthMode === 'api-key'"
              :class="[
                'rounded-md px-3 py-2 text-sm font-medium transition-colors',
                codexAuthMode === 'api-key'
                  ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                  : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
              ]"
              @click="codexAuthMode = 'api-key'"
            >
              {{ t('keys.useKeyModal.openai.authModeApiKey') }}
            </button>
          </div>
          <div
            v-if="codexAuthMode === 'api-key'"
            data-testid="codex-api-key-restart-notice"
            class="mt-3 flex items-start gap-2 border-l-2 border-amber-400 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800 dark:border-amber-500 dark:bg-amber-950/30 dark:text-amber-200"
          >
            <Icon name="exclamationCircle" size="sm" class="mt-0.5 flex-shrink-0" />
            <p>{{ t('keys.useKeyModal.openai.authModeApiKeyRestartNotice') }}</p>
          </div>
        </div>

        <!-- OS/Shell Tabs -->
        <div v-if="showShellTabs" class="overflow-x-auto border-b border-gray-200 dark:border-dark-700">
          <nav class="-mb-px flex min-w-max gap-4" aria-label="Tabs">
            <button
              v-for="tab in currentTabs"
              :key="tab.id"
              type="button"
              @click="activeTab = tab.id"
              :class="[
                'whitespace-nowrap py-2.5 px-1 border-b-2 font-medium text-sm transition-colors',
                activeTab === tab.id
                  ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
              ]"
            >
              <span class="flex items-center gap-2">
                <component :is="tab.icon" class="w-4 h-4" />
                {{ tab.label }}
              </span>
            </button>
          </nav>
        </div>

        <!-- Code Blocks (Stacked for multi-file platforms) -->
        <div class="space-y-4">
          <div
            v-for="(file, index) in currentFiles"
            :key="index"
            class="relative"
          >
            <!-- File Hint (if exists) -->
            <p v-if="file.hint" class="text-xs text-amber-600 dark:text-amber-400 mb-1.5 flex items-center gap-1">
              <Icon name="exclamationCircle" size="sm" class="flex-shrink-0" />
              {{ file.hint }}
            </p>
            <div class="bg-gray-900 dark:bg-dark-900 rounded-xl overflow-hidden">
              <!-- Code Header -->
              <div class="flex items-center justify-between px-4 py-2 bg-gray-800 dark:bg-dark-800 border-b border-gray-700 dark:border-dark-700">
                <span class="min-w-0 truncate text-xs text-gray-400 font-mono">{{ file.path }}</span>
                <button
                  type="button"
                  @click="copyContent(file.content, index)"
                  class="flex flex-shrink-0 items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition-colors"
                  :class="copiedIndex === index
                    ? 'bg-green-500/20 text-green-400'
                    : 'bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white'"
                >
                  <svg v-if="copiedIndex === index" class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                  </svg>
                  {{ copiedIndex === index ? t('keys.useKeyModal.copied') : t('keys.useKeyModal.copy') }}
                </button>
              </div>
              <!-- Code Content -->
              <pre class="p-4 text-sm font-mono text-gray-100 overflow-x-auto"><code v-if="file.highlighted" v-html="file.highlighted"></code><code v-else v-text="file.content"></code></pre>
            </div>
          </div>
        </div>

        <section
          v-if="showCodexModelCatalog"
          data-testid="codex-model-catalog"
          class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800/50"
        >
          <div class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="min-w-0">
              <h3 class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t('keys.useKeyModal.codexModelCatalog.title') }}
              </h3>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('keys.useKeyModal.codexModelCatalog.description') }}
              </p>
              <p class="mt-1 truncate font-mono text-xs text-gray-700 dark:text-gray-300">
                {{ codexModelCatalogMode === 'remote' ? codexModelCatalogUrl : codexModelCatalogPath }}
              </p>
              <select
                v-model="codexModelCatalogMode"
                data-testid="codex-model-catalog-mode"
                :aria-label="t('keys.useKeyModal.codexModelCatalog.mode')"
                class="input mt-2 text-sm"
              >
                <option value="remote" :disabled="codexModelCatalogOversized">{{ t('keys.useKeyModal.codexModelCatalog.remote') }}</option>
                <option value="file">{{ t('keys.useKeyModal.codexModelCatalog.local') }}</option>
              </select>
              <p v-if="codexModelCatalogOversized" class="mt-2 text-xs text-amber-700 dark:text-amber-300">
                {{ t('keys.useKeyModal.codexModelCatalog.oversized') }}
              </p>
            </div>
            <button
              v-if="codexModelManifestState === 'ready'"
              type="button"
              class="btn btn-primary min-h-9 flex-shrink-0 px-3 text-xs"
              @click="downloadCodexModelManifest"
            >
              <Icon name="download" size="sm" class="mr-1.5" />
              {{ t('keys.useKeyModal.codexModelCatalog.download') }}
            </button>
            <button
              v-else
              type="button"
              data-testid="codex-model-catalog-fetch"
              class="btn btn-primary min-h-9 flex-shrink-0 px-3 text-xs"
              :disabled="codexModelManifestState === 'loading' || !apiKey"
              @click="loadCodexModelManifest"
            >
              <Icon
                name="refresh"
                size="sm"
                class="mr-1.5"
                :class="codexModelManifestState === 'loading' ? 'animate-spin' : ''"
              />
              {{ codexModelManifestState === 'error'
                ? t('keys.useKeyModal.codexModelCatalog.retry')
                : t('keys.useKeyModal.codexModelCatalog.fetch') }}
            </button>
          </div>
          <p
            v-if="codexModelManifestState === 'ready'"
            class="border-t border-gray-200 px-4 py-2 text-xs text-emerald-700 dark:border-dark-700 dark:text-emerald-300"
          >
            {{ t('keys.useKeyModal.codexModelCatalog.modelsCount', { count: codexModelManifestModelCount }) }}
          </p>
          <p
            v-else-if="codexModelManifestState === 'error'"
            class="border-t border-red-200 px-4 py-2 text-xs text-red-700 dark:border-red-900 dark:text-red-300"
          >
            {{ t('keys.useKeyModal.codexModelCatalog.errorDescription') }}
          </p>
        </section>

        <!-- Usage Note -->
        <div v-if="showPlatformNote" class="flex items-start gap-3 p-3 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-100 dark:border-blue-800">
          <Icon name="infoCircle" size="md" class="text-blue-500 flex-shrink-0 mt-0.5" />
          <p class="text-sm text-blue-700 dark:text-blue-300">
            {{ platformNote }}
          </p>
        </div>
        </template>
      </template>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          @click="emit('close')"
          class="btn btn-secondary"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, h, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { saveAs } from 'file-saver'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { buildCodexModelCatalogUrl, fetchCodexModelsManifest } from '@/api/codex'
import type { GroupPlatform, UseKeyConfig } from '@/types'
import {
  findCodexCatalogModel,
  formatCodexReasoningEffortTomlLine,
  parseCodexCatalogModels,
  selectCodexConfigReasoningEffort
} from '@/utils/codexCatalogConfig'

interface Props {
  show: boolean
  apiKey: string
  baseUrl: string
  platform: GroupPlatform | null
  claudeCodeOnly?: boolean
  allowMessagesDispatch?: boolean
  /** 管理端「使用密钥」弹窗配置（公开设置 use_key_config）；未配置或解析失败时为 null。 */
  useKeyConfig?: UseKeyConfig | null
}

interface Emits {
  (e: 'close'): void
}

interface TabConfig {
  id: string
  label: string
  icon: Component
}

interface FileConfig {
  path: string
  content: string
  hint?: string  // Optional hint message for this file
  highlighted?: string
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const { copyToClipboard: clipboardCopy } = useClipboard()

const copiedIndex = ref<number | null>(null)
const activeTab = ref<string>('unix')
const activeClientTab = ref<string>('claude')
type CodexAuthMode = 'legacy' | 'api-key'
const codexAuthMode = ref<CodexAuthMode>('legacy')
type CodexModelManifestState = 'idle' | 'loading' | 'ready' | 'error'
const codexModelManifestState = ref<CodexModelManifestState>('idle')
const codexModelManifestContent = ref('')
const codexModelManifestModelCount = ref(0)
const codexModelCatalogMode = ref<'remote' | 'file'>('remote')
const codexModelManifestResponseBytes = ref(0)
const codexModelCatalogOversized = computed(() => codexModelManifestResponseBytes.value > 1024 * 1024)
const codexModelCatalogUrl = computed(() => buildCodexModelCatalogUrl(props.baseUrl))
const codexLocalCatalogToml = computed(() => codexModelCatalogMode.value === 'file'
  ? `model_catalog_json = "${CODEX_MODEL_CATALOG_CONFIG_PATH}"\n`
  : '')
let codexModelManifestController: AbortController | null = null
let codexModelManifestRequestID = 0

const showCodexModelCatalog = computed(() =>
  props.show &&
  (activeClientTab.value === 'codex' ||
    (props.platform === 'openai' && activeClientTab.value === 'codex-ws'))
)

const codexModelCatalogPath = computed(() => {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'
  return joinConfigPath(configDir, 'codex-models.json', isWindows)
})

// Codex expands a leading ~/ on every platform but not %userprofile%, which it
// resolves relative to the config directory, so config.toml always uses ~/.
const CODEX_MODEL_CATALOG_CONFIG_PATH = '~/.codex/codex-models.json'

const codexManifestContext = computed(() => {
  if (!showCodexModelCatalog.value) return ''
  return `${props.platform}|${props.baseUrl}|${props.apiKey}`
})

// Reset tabs when platform changes
const defaultClientTab = computed(() => {
  if (props.claudeCodeOnly) return 'claude'
  switch (props.platform) {
    case 'openai':
      return 'codex'
    case 'grok':
      return 'grok'
    case 'gemini':
      return 'gemini'
    case 'antigravity':
      return 'claude'
    case 'typesafe':
      return 'systemone'
    default:
      return 'claude'
  }
})

watch(() => [props.platform, props.claudeCodeOnly], () => {
  activeTab.value = 'unix'
  activeClientTab.value = defaultClientTab.value
  codexAuthMode.value = 'legacy'
}, { immediate: true })

watch(() => props.show, (show) => {
  if (show) {
    codexAuthMode.value = 'legacy'
  } else {
    resetCodexModelManifest()
  }
})

watch(codexManifestContext, (context, previousContext) => {
  if (context !== previousContext) {
    resetCodexModelManifest()
  }
})

// Reset shell tab when client changes
watch(activeClientTab, () => {
  activeTab.value = 'unix'
})

// Icon components
const AppleIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'w-4 h-4'
    }, [
      h('path', { d: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' })
    ])
  }
}

const WindowsIcon = {
  render() {
    return h('svg', {
      fill: 'currentColor',
      viewBox: '0 0 24 24',
      class: 'w-4 h-4'
    }, [
      h('path', { d: 'M3 12V6.75l6-1.32v6.48L3 12zm17-9v8.75l-10 .15V5.21L20 3zM3 13l6 .09v6.81l-6-1.15V13zm7 .25l10 .15V21l-10-1.91v-5.84z' })
    ])
  }
}

// Terminal icon for Claude Code
const TerminalIcon = {
  render() {
    return h('svg', {
      fill: 'none',
      stroke: 'currentColor',
      viewBox: '0 0 24 24',
      'stroke-width': '1.5',
      class: 'w-4 h-4'
    }, [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'm6.75 7.5 3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0 0 21 17.25V6.75A2.25 2.25 0 0 0 18.75 4.5H5.25A2.25 2.25 0 0 0 3 6.75v10.5A2.25 2.25 0 0 0 5.25 20.25Z'
      })
    ])
  }
}

// Sparkle icon for Gemini
const SparkleIcon = {
  render() {
    return h('svg', {
      fill: 'none',
      stroke: 'currentColor',
      viewBox: '0 0 24 24',
      'stroke-width': '1.5',
      class: 'w-4 h-4'
    }, [
      h('path', {
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z'
      })
    ])
  }
}

// 管理端「使用密钥」配置解析结果。v1 只覆盖 openai 分支，其他平台分支行为不变。
interface OpenAiUseKeyOverrides {
  providerId: string | null
  providerName: string | null
  model: string | null
  reviewModel: string | null
  reasoningEffort: string | null
  tabs: string[] | null
}

const emptyOpenAiUseKeyOverrides: OpenAiUseKeyOverrides = {
  providerId: null,
  providerName: null,
  model: null,
  reviewModel: null,
  reasoningEffort: null,
  tabs: null
}

// [model_providers.<id>] 是 TOML 裸表键：带空格或其他字符会让整个 config.toml 语法非法，
// 所以 provider id 只能用裸键字符集，长度上限与后端校验一致。
const CODEX_PROVIDER_ID_PATTERN = /^[A-Za-z0-9_-]+$/
const CODEX_PROVIDER_ID_MAX_LENGTH = 64

function isValidCodexProviderId(value: string): boolean {
  return value.length <= CODEX_PROVIDER_ID_MAX_LENGTH && CODEX_PROVIDER_ID_PATTERN.test(value)
}

// 空字符串与缺省一律视为「未配置」，回退到前端硬编码值；非字符串属于结构不合法。
// 换行同样判为不合法，而不是静默转义：这些值最终拼进 config.toml 的双引号内，真实换行会让
// 整份 config.toml 语法非法；而静默转义会生成管理员没写过的值且用户看不到任何提示（项目禁止
// 「回退但看不到」）。`"` 与 `\` 的转义是语义保真的，不能与换行类比。
// 公开设置路径原样透传库值，后端写入校验挡不住手工改库，这里是唯一防线。
function readUseKeyConfigString(
  source: Record<string, unknown>,
  key: string
): { value: string | null; invalid: boolean } {
  const raw = source[key]
  if (raw === undefined || raw === null) return { value: null, invalid: false }
  if (typeof raw !== 'string') return { value: null, invalid: true }
  if (/[\n\r]/.test(raw)) return { value: null, invalid: true }
  return { value: raw.trim() || null, invalid: false }
}

const useKeyConfigState = computed((): { overrides: OpenAiUseKeyOverrides; invalid: boolean } => {
  const ignored = { overrides: emptyOpenAiUseKeyOverrides, invalid: false }
  const raw: unknown = props.useKeyConfig
  if (raw === undefined || raw === null) return ignored
  if (typeof raw !== 'object' || Array.isArray(raw)) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }

  const config = raw as Record<string, unknown>
  // 判据必须与后端一致（后端写入要求 Version == 1，缺省即 0 → 400；公开路径解析也把 0 判为无效）。
  // 容忍缺省会让管理员看到「配置无效」而用户弹窗实际在应用它，两侧结论相反。
  // 同时这是 TOML 生成前的最后一道防线，不能假设上游一定清洗过。
  if (config.version !== 1) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }
  if (config.enabled !== true) return ignored

  const platforms = config.platforms === undefined || config.platforms === null ? {} : config.platforms
  if (typeof platforms !== 'object' || Array.isArray(platforms)) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }
  const openai = (platforms as Record<string, unknown>).openai
  if (openai === undefined || openai === null) return ignored
  if (typeof openai !== 'object' || Array.isArray(openai)) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }
  const platformConfig = openai as Record<string, unknown>

  let tabs: string[] | null = null
  if (platformConfig.tabs !== undefined && platformConfig.tabs !== null) {
    const rawTabs = platformConfig.tabs
    if (!Array.isArray(rawTabs) || rawTabs.some((tab) => typeof tab !== 'string')) {
      return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
    }
    const allowed = (rawTabs as string[]).map((tab) => tab.trim()).filter((tab) => tab !== '')
    // 空数组 = 不限制，沿用现有 tab 全集。
    tabs = allowed.length ? allowed : null
  }

  const codexRaw = platformConfig.codex === undefined || platformConfig.codex === null
    ? {}
    : platformConfig.codex
  if (typeof codexRaw !== 'object' || Array.isArray(codexRaw)) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }
  const codex = codexRaw as Record<string, unknown>
  const providerId = readUseKeyConfigString(codex, 'provider_id')
  const providerName = readUseKeyConfigString(codex, 'provider_name')
  const model = readUseKeyConfigString(codex, 'model')
  const reviewModel = readUseKeyConfigString(codex, 'review_model')
  const reasoningEffort = readUseKeyConfigString(codex, 'reasoning_effort')
  if (providerId.invalid || providerName.invalid || model.invalid || reviewModel.invalid || reasoningEffort.invalid) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }
  // 非法 provider id 绝不透传进 TOML，按「配置无效」整体回退。
  if (providerId.value !== null && !isValidCodexProviderId(providerId.value)) {
    return { overrides: emptyOpenAiUseKeyOverrides, invalid: true }
  }

  return {
    overrides: {
      providerId: providerId.value,
      providerName: providerName.value,
      model: model.value,
      reviewModel: reviewModel.value,
      reasoningEffort: reasoningEffort.value,
      tabs
    },
    invalid: false
  }
})

const useKeyOpenAiOverrides = computed(() => useKeyConfigState.value.overrides)

// 结构不合法的配置按未配置处理（回退硬编码），但必须在弹窗内显式提示，不静默忽略。
const showUseKeyConfigInvalidNotice = computed(() =>
  props.platform === 'openai' && useKeyConfigState.value.invalid
)

// 管理员白名单与现有 tab 集、分组字段门取交集。交集为空时 fail-closed：不展示任何客户端，
// 由模板给出显式提示。回退为「不限制」等于把管理员的白名单静默放大成全集（fail-open），
// 会出现「配了只留 Claude Code、用户反而看到三个客户端」这种与管理员意图相反的静默降级。
function applyUseKeyTabWhitelist(tabs: TabConfig[]): TabConfig[] {
  const allowed = useKeyOpenAiOverrides.value.tabs
  if (!allowed) return tabs
  return tabs.filter((tab) => allowed.includes(tab.id))
}

const clientTabs = computed((): TabConfig[] => {
  if (!props.platform) return []
  // 显式例外（计划 §3.2 v1.2 记录）：claude_code_only 分组只提供 Claude Code，不参与白名单交集。
  // 方向上是安全的 —— 不会因配置把该分组唯一的客户端入口藏掉，故不按交集公式处理。
  if (props.claudeCodeOnly) {
    return [{ id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon }]
  }
  switch (props.platform) {
    case 'openai': {
      const tabs: TabConfig[] = [
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'codex-ws', label: t('keys.useKeyModal.cliTabs.codexCliWs'), icon: TerminalIcon },
      ]
      if (props.allowMessagesDispatch) {
        tabs.push({ id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon })
      }
      tabs.push({ id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon })
      return applyUseKeyTabWhitelist(tabs)
    }
    case 'gemini':
      return [
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: SparkleIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    case 'antigravity':
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'gemini', label: t('keys.useKeyModal.cliTabs.geminiCli'), icon: SparkleIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    case 'grok':
      return [
        { id: 'grok', label: t('keys.useKeyModal.cliTabs.grokCli'), icon: TerminalIcon },
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    case 'typesafe':
      return [
        { id: 'systemone', label: t('keys.useKeyModal.cliTabs.systemOne'), icon: TerminalIcon }
      ]
    case 'deepseek':
    case 'minimax':
    case 'composite':
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
    default:
      return [
        { id: 'claude', label: t('keys.useKeyModal.cliTabs.claudeCode'), icon: TerminalIcon },
        { id: 'codex', label: t('keys.useKeyModal.cliTabs.codexCli'), icon: TerminalIcon },
        { id: 'opencode', label: t('keys.useKeyModal.cliTabs.opencode'), icon: TerminalIcon }
      ]
  }
})

// 白名单交集为空 = fail-closed：模板不渲染任何客户端与配置文件内容，只显示显式提示。
const hasAvailableClients = computed(() => clientTabs.value.length > 0)
const showNoClientAllowedNotice = computed(() =>
  props.platform === 'openai' && !hasAvailableClients.value
)

// 白名单可能排除默认 tab，此时落到白名单内的第一个 tab，避免无 tab 选中。
watch(clientTabs, (tabs) => {
  if (!tabs.length) return
  if (tabs.some((tab) => tab.id === activeClientTab.value)) return
  activeClientTab.value = tabs[0]!.id
}, { immediate: true })

// Shell tabs (3 types for environment variable based configs)
const shellTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'cmd', label: 'Windows CMD', icon: WindowsIcon },
  { id: 'powershell', label: 'PowerShell', icon: WindowsIcon }
]

// OpenAI tabs (2 OS types)
const openaiTabs: TabConfig[] = [
  { id: 'unix', label: 'macOS / Linux', icon: AppleIcon },
  { id: 'windows', label: 'Windows', icon: WindowsIcon }
]

const showShellTabs = computed(() => activeClientTab.value !== 'opencode')

const showCodexAuthMode = computed(() =>
  props.platform === 'openai' &&
  (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws')
)

const currentTabs = computed(() => {
  if (!showShellTabs.value) return []
  if (activeClientTab.value === 'codex' || activeClientTab.value === 'codex-ws' || activeClientTab.value === 'grok') {
    return openaiTabs
  }
  return shellTabs
})

const platformDescription = computed(() => {
  if (activeClientTab.value === 'codex' &&
    props.platform !== 'openai' &&
    props.platform !== 'grok' &&
    props.platform !== 'deepseek' &&
    props.platform !== 'minimax' &&
    props.platform !== 'composite') {
    return t('keys.useKeyModal.routedCodex.description')
  }
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.description')
      }
      return t('keys.useKeyModal.openai.description')
    case 'gemini':
      return t('keys.useKeyModal.gemini.description')
    case 'antigravity':
      return t('keys.useKeyModal.antigravity.description')
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.grok.claudeDescription')
      }
      if (activeClientTab.value === 'codex') {
        return t('keys.useKeyModal.grok.codexDescription')
      }
      return t('keys.useKeyModal.grok.description')
    case 'deepseek':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.deepseek.codexDescription')
        : t('keys.useKeyModal.deepseek.description')
    case 'minimax':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.minimax.codexDescription')
        : t('keys.useKeyModal.minimax.description')
    case 'composite':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.composite.codexDescription')
        : t('keys.useKeyModal.composite.description')
    case 'typesafe':
      return t('keys.useKeyModal.typesafe.description')
    default:
      return t('keys.useKeyModal.description')
  }
})

const platformNote = computed(() => {
  if (activeClientTab.value === 'codex' &&
    props.platform !== 'openai' &&
    props.platform !== 'grok' &&
    props.platform !== 'deepseek' &&
    props.platform !== 'minimax' &&
    props.platform !== 'composite') {
    return t('keys.useKeyModal.routedCodex.note')
  }
  switch (props.platform) {
    case 'openai':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.note')
      }
      return activeTab.value === 'windows'
        ? t('keys.useKeyModal.openai.noteWindows')
        : t('keys.useKeyModal.openai.note')
    case 'gemini':
      return t('keys.useKeyModal.gemini.note')
    case 'antigravity':
      return activeClientTab.value === 'claude'
        ? t('keys.useKeyModal.antigravity.claudeNote')
        : t('keys.useKeyModal.antigravity.geminiNote')
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return t('keys.useKeyModal.grok.claudeNote')
      }
      if (activeClientTab.value === 'codex') {
        return activeTab.value === 'windows'
          ? t('keys.useKeyModal.grok.codexNoteWindows')
          : t('keys.useKeyModal.grok.codexNote')
      }
      // Grok CLI: shell-specific path guidance (env + ~/.grok/config.toml).
      if (activeClientTab.value === 'grok' && (activeTab.value === 'cmd' || activeTab.value === 'powershell')) {
        return t('keys.useKeyModal.grok.noteWindows')
      }
      if (activeClientTab.value === 'grok' && activeTab.value === 'windows') {
        return t('keys.useKeyModal.grok.noteWindows')
      }
      return t('keys.useKeyModal.grok.note')
    case 'deepseek':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.deepseek.codexNote')
        : t('keys.useKeyModal.note')
    case 'minimax':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.minimax.codexNote')
        : t('keys.useKeyModal.note')
    case 'composite':
      return activeClientTab.value === 'codex'
        ? t('keys.useKeyModal.composite.codexNote')
        : t('keys.useKeyModal.note')
    case 'typesafe':
      return t('keys.useKeyModal.typesafe.note')
    default:
      return t('keys.useKeyModal.note')
  }
})

const showPlatformNote = computed(() => activeClientTab.value !== 'opencode')

function resetCodexModelManifest() {
  codexModelManifestController?.abort()
  codexModelManifestController = null
  codexModelManifestRequestID += 1
  codexModelManifestState.value = 'idle'
  codexModelManifestContent.value = ''
  codexModelManifestModelCount.value = 0
  codexModelManifestResponseBytes.value = 0
}

async function loadCodexModelManifest() {
  if (!showCodexModelCatalog.value || !props.apiKey) return

  codexModelManifestController?.abort()
  const controller = new AbortController()
  const requestID = ++codexModelManifestRequestID
  codexModelManifestController = controller
  codexModelManifestState.value = 'loading'

  try {
    const result = await fetchCodexModelsManifest(props.baseUrl, props.apiKey, controller.signal)
    if (requestID !== codexModelManifestRequestID) return
    codexModelManifestContent.value = result.content
    codexModelManifestModelCount.value = result.modelCount
    codexModelManifestResponseBytes.value = result.responseBytes
    if (codexModelCatalogOversized.value) codexModelCatalogMode.value = 'file'
    codexModelManifestState.value = 'ready'
  } catch (error) {
    const errorName = error && typeof error === 'object' && 'name' in error
      ? String((error as { name?: unknown }).name || '')
      : ''
    if (requestID !== codexModelManifestRequestID || errorName === 'AbortError') return
    codexModelManifestState.value = 'error'
  } finally {
    if (requestID === codexModelManifestRequestID) {
      codexModelManifestController = null
    }
  }
}

function downloadCodexModelManifest() {
  if (!codexModelManifestContent.value) return
  saveAs(
    new Blob([codexModelManifestContent.value], { type: 'application/json;charset=utf-8' }),
    'codex-models.json'
  )
}

const codexCatalogModelSlugs = computed(() =>
  parseCodexCatalogModels(codexModelManifestContent.value).map((model) => model.slug)
)

function selectCodexCatalogModel(preferredModel: string): string {
  if (codexCatalogModelSlugs.value.includes(preferredModel)) return preferredModel
  return codexCatalogModelSlugs.value[0] || preferredModel
}

function codexReasoningEffortTomlLine(modelSlug: string): string {
  return formatCodexReasoningEffortTomlLine(
    selectCodexConfigReasoningEffort(findCodexCatalogModel(codexModelManifestContent.value, modelSlug))
  )
}

// 管理员配置的模型优先：非空时直接使用，不再做清单替换（清单只用于推导 reasoning effort）。
// 各字段为空时逐个回退到现有硬编码行为。
function resolveOpenAiCodexConfig(): {
  providerId: string
  model: string
  reviewModel: string
  providerName: string
  reasoningEffortLine: string
} {
  const overrides = useKeyOpenAiOverrides.value
  const model = overrides.model || selectCodexCatalogModel('gpt-5.5')
  return {
    // provider_id 同时驱动 model_provider 与 [model_providers.<id>]，两行永远一致。
    providerId: overrides.providerId || 'OpenAI',
    model,
    reviewModel: overrides.reviewModel || model,
    providerName: overrides.providerName || 'OpenAI',
    reasoningEffortLine: overrides.reasoningEffort
      ? formatCodexReasoningEffortTomlLine(escapeTomlBasicString(overrides.reasoningEffort))
      : codexReasoningEffortTomlLine(model)
  }
}

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const wrapToken = (className: string, value: string) =>
  `<span class="${className}">${escapeHtml(value)}</span>`

const keyword = (value: string) => wrapToken('text-emerald-300', value)
const variable = (value: string) => wrapToken('text-sky-200', value)
const operator = (value: string) => wrapToken('text-slate-400', value)
const string = (value: string) => wrapToken('text-amber-200', value)
const comment = (value: string) => wrapToken('text-slate-500', value)

// Syntax highlighting helpers
// Generate file configs based on platform and active tab
const currentFiles = computed((): FileConfig[] => {
  const baseUrl = props.baseUrl || window.location.origin
  const apiKey = props.apiKey
  const baseRoot = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  const ensureV1 = (value: string) => {
    const trimmed = value.replace(/\/+$/, '')
    return trimmed.endsWith('/v1') ? trimmed : `${trimmed}/v1`
  }
  const apiBase = ensureV1(baseRoot)
  const antigravityBase = ensureV1(`${baseRoot}/antigravity`)
  const antigravityGeminiBase = (() => {
    const trimmed = `${baseRoot}/antigravity`.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()
  const geminiBase = (() => {
    const trimmed = baseRoot.replace(/\/+$/, '')
    return trimmed.endsWith('/v1beta') ? trimmed : `${trimmed}/v1beta`
  })()

  if (activeClientTab.value === 'opencode') {
    switch (props.platform) {
      case 'anthropic':
        return [generateOpenCodeConfig('anthropic', apiBase, apiKey)]
      case 'openai':
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
      case 'gemini':
        return [generateOpenCodeConfig('gemini', geminiBase, apiKey)]
      case 'antigravity':
        return [
          generateOpenCodeConfig('antigravity-claude', antigravityBase, apiKey, 'opencode.json (Claude)'),
          generateOpenCodeConfig('antigravity-gemini', antigravityGeminiBase, apiKey, 'opencode.json (Gemini)')
        ]
      case 'grok':
        return [generateOpenCodeConfig('grok', apiBase, apiKey)]
      default:
        return [generateOpenCodeConfig('openai', apiBase, apiKey)]
    }
  }

  switch (props.platform) {
    case 'typesafe':
      return [generateSystemOneCurl(baseRoot, apiKey)]
    case 'openai':
      if (activeClientTab.value === 'claude') {
        // Anthropic clients append /v1/messages themselves.
        return generateAnthropicFiles(baseRoot, apiKey)
      }
      if (activeClientTab.value === 'codex-ws') {
        return generateOpenAIWsFiles(apiBase, apiKey)
      }
      // Codex appends /responses directly and does not add /v1.
      return generateOpenAIFiles(apiBase, apiKey)
    case 'gemini':
      if (activeClientTab.value === 'codex') {
        return generateRoutedCodexFiles(apiBase, apiKey, 'gemini')
      }
      return [generateGeminiCliContent(baseUrl, apiKey)]
    case 'antigravity':
      if (activeClientTab.value === 'codex') {
        return generateRoutedCodexFiles(apiBase, apiKey, 'antigravity')
      }
      if (activeClientTab.value === 'gemini') {
        return [generateGeminiCliContent(`${baseUrl}/antigravity`, apiKey)]
      }
      return generateAnthropicFiles(`${baseUrl}/antigravity`, apiKey)
    case 'grok':
      if (activeClientTab.value === 'claude') {
        return generateGrokClaudeFiles(baseRoot, apiKey)
      }
      if (activeClientTab.value === 'codex') {
        return generateGrokCodexFiles(apiBase, apiKey)
      }
      return generateGrokFiles(apiBase, apiKey)
    case 'deepseek':
      if (activeClientTab.value === 'codex') {
        return generateRoutedCodexFiles(apiBase, apiKey, 'deepseek')
      }
      return generateAnthropicFiles(baseRoot, apiKey)
    case 'minimax':
      if (activeClientTab.value === 'codex') {
        return generateRoutedCodexFiles(apiBase, apiKey, 'minimax')
      }
      return generateAnthropicFiles(baseRoot, apiKey)
    case 'composite':
      if (activeClientTab.value === 'codex') {
        return generateRoutedCodexFiles(apiBase, apiKey, 'composite')
      }
      return generateAnthropicFiles(baseRoot, apiKey)
    default:
      if (activeClientTab.value === 'codex' && props.platform) {
        return generateRoutedCodexFiles(apiBase, apiKey, props.platform)
      }
      return generateAnthropicFiles(baseUrl, apiKey)
  }
})

function generateSystemOneCurl(baseUrl: string, apiKey: string): FileConfig {
  const endpoint = `${baseUrl}/v1/systemone`
  const payload = `{
  "model": "jev-latest",
  "state": "Text to evaluate",
  "questions": {
    "safety": {
      "type": "noul",
      "instructions": "Evaluate whether the text is unsafe"
    }
  }
}`
  if (activeTab.value === 'powershell') {
    return {
      path: 'PowerShell',
      content: `$headers = @{ Authorization = "Bearer ${apiKey}" }
$body = @'
${payload}
'@
Invoke-RestMethod -Method Post -Uri "${endpoint}" -Headers $headers -ContentType "application/json" -Body $body`
    }
  }
  if (activeTab.value === 'cmd') {
    return {
      path: 'Command Prompt',
      content: `curl -X POST "${endpoint}" ^
  -H "Authorization: Bearer ${apiKey}" ^
  -H "Content-Type: application/json" ^
  --data "{\"model\":\"jev-latest\",\"state\":\"Text to evaluate\",\"questions\":{\"safety\":{\"type\":\"noul\",\"instructions\":\"Evaluate whether the text is unsafe\"}}}"`
    }
  }
  return {
    path: 'Terminal',
    content: `curl -X POST "${endpoint}" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -H "Content-Type: application/json" \\
  --data '${payload}'`
  }
}

function generateAnthropicFiles(baseUrl: string, apiKey: string): FileConfig[] {
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export ANTHROPIC_BASE_URL="${baseUrl}"
export ANTHROPIC_AUTH_TOKEN="${apiKey}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set ANTHROPIC_BASE_URL=${baseUrl}
set ANTHROPIC_AUTH_TOKEN=${apiKey}
set CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:ANTHROPIC_BASE_URL="${baseUrl}"
$env:ANTHROPIC_AUTH_TOKEN="${apiKey}"
$env:CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const vscodeSettingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%USERPROFILE%\\.claude\\settings.json'

  const vscodeContent = `{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "env": {
    "ANTHROPIC_BASE_URL": "${baseUrl}",
    "ANTHROPIC_AUTH_TOKEN": "${apiKey}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
  }
}`

  return [
    { path, content },
    {
      path: vscodeSettingsPath,
      content: vscodeContent,
      hint: t('keys.useKeyModal.claudeSettingsHint')
    }
  ]
}

function generateGrokClaudeFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const environment = {
    ANTHROPIC_BASE_URL: baseUrl,
    ANTHROPIC_AUTH_TOKEN: apiKey,
    ANTHROPIC_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_OPUS_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_SONNET_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_HAIKU_MODEL: 'grok-4.5',
    ANTHROPIC_DEFAULT_FABLE_MODEL: 'grok-4.5',
    CLAUDE_CODE_SUBAGENT_MODEL: 'grok-4.5',
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC: '1'
  }
  let path: string
  let content: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = Object.entries(environment)
        .map(([name, value]) => `export ${name}="${value}"`)
        .join('\n')
      break
    case 'cmd':
      path = 'Command Prompt'
      content = Object.entries(environment)
        .map(([name, value]) => `set ${name}=${value}`)
        .join('\n')
      break
    case 'powershell':
      path = 'PowerShell'
      content = Object.entries(environment)
        .map(([name, value]) => `$env:${name}="${value}"`)
        .join('\n')
      break
    default:
      path = 'Terminal'
      content = ''
  }

  const settingsPath = activeTab.value === 'unix'
    ? '~/.claude/settings.json'
    : '%USERPROFILE%\\.claude\\settings.json'

  return [
    { path, content },
    {
      path: settingsPath,
      content: JSON.stringify({
        $schema: 'https://json.schemastore.org/claude-code-settings.json',
        env: environment
      }, null, 2),
      hint: t('keys.useKeyModal.claudeSettingsHint')
    }
  ]
}

function generateGeminiCliContent(baseUrl: string, apiKey: string): FileConfig {
  const model = 'gemini-2.0-flash'
  const modelComment = t('keys.useKeyModal.gemini.modelComment')
  let path: string
  let content: string
  let highlighted: string

  switch (activeTab.value) {
    case 'unix':
      path = 'Terminal'
      content = `export GOOGLE_GEMINI_BASE_URL="${baseUrl}"
export GEMINI_API_KEY="${apiKey}"
export GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('export')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('export')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('export')} ${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    case 'cmd':
      path = 'Command Prompt'
      content = `set GOOGLE_GEMINI_BASE_URL=${baseUrl}
set GEMINI_API_KEY=${apiKey}
set GEMINI_MODEL=${model}`
      highlighted = `${keyword('set')} ${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(baseUrl)}
${keyword('set')} ${variable('GEMINI_API_KEY')}${operator('=')}${string(apiKey)}
${keyword('set')} ${variable('GEMINI_MODEL')}${operator('=')}${string(model)}
${comment(`REM ${modelComment}`)}`
      break
    case 'powershell':
      path = 'PowerShell'
      content = `$env:GOOGLE_GEMINI_BASE_URL="${baseUrl}"
$env:GEMINI_API_KEY="${apiKey}"
$env:GEMINI_MODEL="${model}"  # ${modelComment}`
      highlighted = `${keyword('$env:')}${variable('GOOGLE_GEMINI_BASE_URL')}${operator('=')}${string(`"${baseUrl}"`)}
${keyword('$env:')}${variable('GEMINI_API_KEY')}${operator('=')}${string(`"${apiKey}"`)}
${keyword('$env:')}${variable('GEMINI_MODEL')}${operator('=')}${string(`"${model}"`)}  ${comment(`# ${modelComment}`)}`
      break
    default:
      path = 'Terminal'
      content = ''
      highlighted = ''
  }

  return { path, content, highlighted }
}

function generateOpenAIFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'

  const { providerId, model, reviewModel, providerName, reasoningEffortLine } = resolveOpenAiCodexConfig()

  // config.toml content
  const configContent = `model_provider = "${providerId}"
model = "${escapeTomlBasicString(model)}"
review_model = "${escapeTomlBasicString(reviewModel)}"
${reasoningEffortLine}disable_response_storage = true
${codexLocalCatalogToml.value}network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.${providerId}]
name = "${escapeTomlBasicString(providerName)}"
base_url = "${baseUrl}"
${codexModelCatalogMode.value === 'remote' ? `model_catalog_url = "${escapeTomlBasicString(buildCodexModelCatalogUrl(baseUrl))}"\n` : ''}wire_api = "responses"
${generateCodexProviderAuthConfig(apiKey)}

[features]
${codexModelCatalogMode.value === 'remote' ? 'api_key_model_discovery = true\n' : ''}goals = true`

  return buildOpenAICodexFileConfigs(configDir, configContent, apiKey)
}

function generateCodexProviderAuthConfig(apiKey: string): string {
  if (codexAuthMode.value === 'api-key') {
    return `requires_openai_auth = false
experimental_bearer_token = "${escapeTomlBasicString(apiKey)}"
http_headers = { "x-openai-actor-authorization" = "local-image-extension" }`
  }

  return 'requires_openai_auth = true'
}

function buildOpenAICodexFileConfigs(
  configDir: string,
  configContent: string,
  apiKey: string
): FileConfig[] {
  const files: FileConfig[] = [
    {
      path: `${configDir}/config.toml`,
      content: configContent,
      hint: t('keys.useKeyModal.openai.configTomlHint')
    }
  ]

  if (codexAuthMode.value === 'legacy') {
    files.push({
      path: `${configDir}/auth.json`,
      content: JSON.stringify({ OPENAI_API_KEY: apiKey }, null, 2)
    })
  }

  return files
}

function joinConfigPath(dir: string, file: string, windows: boolean): string {
  if (!windows) return `${dir}/${file}`
  return `${dir}\\${file}`
}

function escapeTomlBasicString(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

function generateGrokFiles(baseUrl: string, apiKey: string): FileConfig[] {
  // Prefer unix/cmd/powershell when shell tabs are shown; fall back to windows tab.
  const shell = activeTab.value
  const isWindowsPath = shell === 'windows' || shell === 'cmd' || shell === 'powershell'
  const configDir = isWindowsPath ? '%userprofile%\\.grok' : '~/.grok'

  let envPath: string
  let envContent: string
  switch (shell) {
    case 'cmd':
      envPath = 'Command Prompt'
      envContent = `set GROK_MODELS_BASE_URL=${baseUrl}
set XAI_API_KEY=${apiKey}`
      break
    case 'powershell':
    case 'windows':
      envPath = 'PowerShell'
      envContent = `$env:GROK_MODELS_BASE_URL="${baseUrl}"
$env:XAI_API_KEY="${apiKey}"`
      break
    default:
      envPath = 'Terminal'
      envContent = `export GROK_MODELS_BASE_URL="${baseUrl}"
export XAI_API_KEY="${apiKey}"`
  }

  // Shape follows Grok Build user guide (~/.grok/docs + custom-models) and production-ready Sub2API setups.
  // Text models only (Responses). Image/video: Imagine model IDs on media endpoints / feature overrides.
  // Credential order: api_key field → env_key → signed-in session → XAI_API_KEY global fallback.
  const modelsListUrl = `${baseUrl.replace(/\/+$/, '')}/models`
  const configContent = `# Grok Build CLI → Sub2API Grok group (API key auth).
# Docs: ~/.grok/docs/user-guide/05-configuration.md + 11-custom-models.md
# Verify after save: grok inspect
#
# IMPORTANT: api_backend must be "responses" for Sub2API Grok (POST /v1/responses).
# If omitted, Grok Build defaults to chat_completions (/v1/chat/completions).
# Keep api_backend = "responses" on every model entry.
#
# Prefer env_key over hardcoding api_key (never commit secrets).
# Also export GROK_MODELS_BASE_URL + XAI_API_KEY in the shell block above.

# Global inference / catalog endpoints (same role as env GROK_MODELS_BASE_URL).
# When models_base_url is set, Grok uses API-key Bearer auth (no grok login required).
[endpoints]
models_base_url = "${baseUrl}"              # inference base; model list defaults to {base}/models
models_list_url = "${modelsListUrl}"        # optional override (env: GROK_MODELS_LIST_URL)
xai_api_base_url = "${baseUrl}"             # public xAI API base override for gateway routing
cli_chat_proxy_base_url = "${baseUrl}"      # CLI chat-proxy base (env: GROK_CLI_CHAT_PROXY_BASE_URL)

# Prefer API key when using a custom gateway (matches Sub2API).
# Requires XAI_API_KEY env or per-model env_key / api_key.
[auth]
preferred_method = "api_key"

[model."grok-4.5"]
model = "grok-4.5"                          # id sent to the API
name = "Grok 4.5"                           # shown in /model picker
description = "Grok 4.5 via Sub2API (Responses)"
# base_url inherits from [endpoints].models_base_url; override only if needed:
# base_url = "${baseUrl}"
env_key = "XAI_API_KEY"                     # or: api_key = "${apiKey}"  (not recommended)
api_backend = "responses"                   # chat_completions | responses | messages
context_window = 500000                     # drives auto-compaction timing
# Optional sampling (global defaults can live under [models] instead):
# temperature = 0.7
# top_p = 0.95
# max_completion_tokens = 8192
# Server-side (backend) web_search tools — only if your gateway exposes them:
supports_backend_search = true

[model."grok-build-0.1"]
model = "grok-build-0.1"
name = "Grok Build"
description = "Coding / agent sessions (xAI recommends grok-build* for coding)"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 256000
supports_backend_search = true

# Text multi-agent / client web_search sub-agent (NOT Imagine image/video).
[model."grok-4.20-multi-agent-0309"]
model = "grok-4.20-multi-agent-0309"
name = "Grok 4.20 Multi Agent (text / web_search)"
description = "Text multi-agent; use for web_search sub-agent, not image/video"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true

[model."grok-4.3"]
model = "grok-4.3"
name = "Grok 4.3"
env_key = "XAI_API_KEY"
api_backend = "responses"
context_window = 1000000
supports_backend_search = true

# Optional short alias for /model grok:
# [model."grok"]
# model = "grok-4.5"
# name = "Grok"
# env_key = "XAI_API_KEY"
# api_backend = "responses"
# context_window = 1000000
# supports_backend_search = true

[models]
# xAI recommends grok-build* for coding/agent sessions; use grok-4.5 for general chat.
default = "grok-4.5"
web_search = "grok-4.5"                     # client-side web_search tool model (must exist as [model.*])
image_description = "grok-4.5"              # vision/describe-image helper model
# Optional environment-wide sampling defaults (per-model values win):
# temperature = 0.7
# top_p = 0.95
# max_completion_tokens = 8192
# max_retries = 8

[session]
auto_compact_threshold_percent = 80         # auto-compact at this % of context_window (default 85)

# Imagine tools: model IDs go to Sub2API media endpoints (not the text [model.*] catalog).
# Enable only if the Grok group allows image/video generation.
[features]
image_gen = true
video_gen = true
image_gen_model_override = "grok-imagine-image-quality"   # or grok-imagine-image
image_edit_model_override = "grok-imagine-edit"
# Optional feature flags (defaults shown in docs):
# telemetry = false
# remote_fetch = true                         # set false for air-gapped / pure-gateway catalogs
# lsp_tools = false`

  return [
    { path: envPath, content: envContent },
    {
      path: joinConfigPath(configDir, 'config.toml', isWindowsPath),
      content: configContent,
      hint: t('keys.useKeyModal.grok.configTomlHint')
    }
  ]
}

function generateGrokCodexFiles(baseUrl: string, apiKey: string): FileConfig[] {
  // Codex config reference: wire_api = "responses" only; prefer env_key over experimental_bearer_token.
  // Non-OpenAI gateways should set supports_websockets = false (HTTP/SSE).
  const shell = activeTab.value
  const isWindowsPath = shell === 'windows' || shell === 'cmd' || shell === 'powershell'
  const configDir = isWindowsPath ? '%userprofile%\\.codex' : '~/.codex'
  const model = selectCodexCatalogModel('grok-4.5')

  let envPath: string
  let envContent: string
  switch (shell) {
    case 'cmd':
      envPath = 'Command Prompt'
      envContent = `set SUB2API_API_KEY=${apiKey}`
      break
    case 'powershell':
    case 'windows':
      envPath = 'PowerShell'
      envContent = `$env:SUB2API_API_KEY="${apiKey}"`
      break
    default:
      envPath = 'Terminal'
      envContent = `export SUB2API_API_KEY="${apiKey}"`
  }

  const configContent = `# Codex CLI → Sub2API Grok group
# Docs: Codex config reference (model_providers.*, wire_api = "responses")
#
# Text models only. Image/video: grok-imagine-image / grok-imagine-video on media endpoints.
# Switch model: grok-4.5 | grok-4.3 | grok-build-0.1 | grok-4.20-multi-agent-0309 (text / web_search)

model_provider = "sub2api"
model = "${model}"
${codexLocalCatalogToml.value}# Optional:
# review_model = "${model}"
# model_reasoning_effort = "medium"
# model_context_window = 500000
# disable_response_storage = true
# network_access = "enabled"
# windows_wsl_setup_acknowledged = true

[model_providers.sub2api]
name = "Sub2API Grok"
base_url = "${baseUrl}"
${codexModelCatalogMode.value === 'remote' ? `model_catalog_url = "${escapeTomlBasicString(buildCodexModelCatalogUrl(baseUrl))}"\n` : ''}# Prefer env_key (variable NAME). Do not combine with experimental_bearer_token.
env_key = "SUB2API_API_KEY"
# Fallback only if you cannot set env (discouraged — keeps secret on disk):
# experimental_bearer_token = "${apiKey}"
wire_api = "responses"
# API-key providers: do not require ChatGPT OAuth login
requires_openai_auth = false
# Grok/Sub2API path is HTTP/SSE; disable WS (Codex may otherwise try WebSocket first)
supports_websockets = false

${codexModelCatalogMode.value === 'remote' ? '[features]\napi_key_model_discovery = true\n\n# Optional:' : '# Optional:\n# [features]'}
# goals = true`

  return [
    { path: envPath, content: envContent },
    {
      path: joinConfigPath(configDir, 'config.toml', isWindowsPath),
      content: configContent,
      hint: t('keys.useKeyModal.grok.codexConfigTomlHint')
    }
  ]
}

function generateRoutedCodexFiles(
  baseUrl: string,
  apiKey: string,
  platform: GroupPlatform
): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'
  const preferredModels: Partial<Record<GroupPlatform, string>> = {
    openai: 'gpt-5.5',
    anthropic: 'claude-sonnet-4-6',
    gemini: 'gemini-2.5-pro',
    antigravity: 'claude-sonnet-4-6',
    grok: 'grok-4.5',
    kimi: 'kimi-k2.5',
    zhipu: 'glm-4.7',
    deepseek: 'deepseek-v4-pro',
    minimax: 'MiniMax-M3',
    opencode_go: 'glm-5.3',
    composite: 'gpt-5.5'
  }
  const preferredModel = preferredModels[platform] || ''
  const model = selectCodexCatalogModel(preferredModel)
  const labels: Record<GroupPlatform, string> = {
    anthropic: 'Anthropic',
    openai: 'OpenAI',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    grok: 'Grok',
    kimi: 'Kimi',
    zhipu: 'Zhipu',
    deepseek: 'DeepSeek',
    minimax: 'MiniMax',
    opencode_go: 'OpenCode',
    typesafe: 'TypeSafe / Jev',
    command_code: 'Command Code',
    cline: 'Cline',
    composite: 'Composite'
  }
  const label = labels[platform]
  const envContent = isWindows
    ? `$env:SUB2API_API_KEY="${apiKey}"`
    : `export SUB2API_API_KEY="${apiKey}"`

  const configContent = `# Codex CLI -> Sub2API ${label} group
model_provider = "sub2api"
model = "${model}"
review_model = "${model}"
disable_response_storage = true
${codexLocalCatalogToml.value}
[model_providers.sub2api]
name = "Sub2API ${label}"
base_url = "${baseUrl}"
${codexModelCatalogMode.value === 'remote' ? `model_catalog_url = "${escapeTomlBasicString(buildCodexModelCatalogUrl(baseUrl))}"\n` : ''}env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false${codexModelCatalogMode.value === 'remote' ? '\n\n[features]\napi_key_model_discovery = true' : ''}`

  return [
    { path: isWindows ? 'PowerShell' : 'Terminal', content: envContent },
    {
      path: joinConfigPath(configDir, 'config.toml', isWindows),
      content: configContent,
      hint: t(
        platform === 'deepseek' || platform === 'minimax' || platform === 'composite'
          ? `keys.useKeyModal.${platform}.codexConfigTomlHint`
          : 'keys.useKeyModal.routedCodex.configTomlHint'
      )
    }
  ]
}

function generateOpenAIWsFiles(baseUrl: string, apiKey: string): FileConfig[] {
  const isWindows = activeTab.value === 'windows'
  const configDir = isWindows ? '%userprofile%\\.codex' : '~/.codex'
  const { providerId, model, reviewModel, providerName, reasoningEffortLine } = resolveOpenAiCodexConfig()

  // config.toml content with WebSocket v2
  const configContent = `model_provider = "${providerId}"
model = "${escapeTomlBasicString(model)}"
review_model = "${escapeTomlBasicString(reviewModel)}"
${reasoningEffortLine}disable_response_storage = true
${codexLocalCatalogToml.value}network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.${providerId}]
name = "${escapeTomlBasicString(providerName)}"
base_url = "${baseUrl}"
${codexModelCatalogMode.value === 'remote' ? `model_catalog_url = "${escapeTomlBasicString(buildCodexModelCatalogUrl(baseUrl))}"\n` : ''}wire_api = "responses"
supports_websockets = true
${generateCodexProviderAuthConfig(apiKey)}

[features]
${codexModelCatalogMode.value === 'remote' ? 'api_key_model_discovery = true\n' : ''}responses_websockets_v2 = true
goals = true`

  return buildOpenAICodexFileConfigs(configDir, configContent, apiKey)
}

function generateOpenCodeConfig(platform: string, baseUrl: string, apiKey: string, pathLabel?: string): FileConfig {
  const provider: Record<string, any> = {
    [platform]: {
      options: {
        baseURL: baseUrl,
        apiKey
      }
    }
  }
  const openaiModels = {
    'gpt-6': {
      name: 'GPT-6 (Astra)',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-6-astra': {
      name: 'GPT-6 Astra',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.2': {
      name: 'GPT-5.2',
      limit: {
        context: 400000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.6': {
      name: 'GPT-5.6 (Sol)',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-6.1-terra': {
      name: 'GPT-6.1 Terra',
      limit: { context: 1050000, output: 128000 },
      options: { store: false },
      variants: { none: {}, low: {}, medium: {}, high: {}, xhigh: {}, max: {} }
    },
    'gpt-6.1-sol': {
      name: 'GPT-6.1 Sol',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-6-sol': {
      name: 'GPT-6 Sol',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        none: {},
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-sol': {
      name: 'GPT-5.6 Sol',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-terra': {
      name: 'GPT-5.6 Terra',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-6-luna': {
      name: 'GPT-6 Luna',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        none: {},
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.6-luna': {
      name: 'GPT-5.6 Luna',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {},
        max: {}
      }
    },
    'gpt-5.5': {
      name: 'GPT-5.5',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4': {
      name: 'GPT-5.4',
      limit: {
        context: 1050000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.4-mini': {
      name: 'GPT-5.4 Mini',
      limit: {
        context: 400000,
        output: 128000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'gpt-5.3-codex-spark': {
      name: 'GPT-5.3 Codex Spark',
      limit: {
        context: 128000,
        output: 32000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {},
        xhigh: {}
      }
    },
    'codex-mini-latest': {
      name: 'Codex Mini',
      limit: {
        context: 200000,
        output: 100000
      },
      options: {
        store: false
      },
      variants: {
        low: {},
        medium: {},
        high: {}
      }
    }
  }
  const geminiModels = {
    'gemini-2.0-flash': {
      name: 'Gemini 2.0 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-2.5-pro': {
      name: 'Gemini 2.5 Pro',
      limit: {
        context: 2097152,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.5-flash': {
      name: 'Gemini 3.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-flash-preview': {
      name: 'Gemini 3 Flash Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      }
    },
    'gemini-3-pro-preview': {
      name: 'Gemini 3 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-preview': {
      name: 'Gemini 3.1 Pro Preview',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }

  const antigravityGeminiModels = {
    'gemini-2.5-flash': {
      name: 'Gemini 2.5 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'disable'
        }
      }
    },
    'gemini-2.5-flash-lite': {
      name: 'Gemini 2.5 Flash Lite',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-thinking': {
      name: 'Gemini 2.5 Flash (Thinking)',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3-flash': {
      name: 'Gemini 3 Flash',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-low': {
      name: 'Gemini 3.1 Pro Low',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-pro-high': {
      name: 'Gemini 3.1 Pro High',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-2.5-flash-image': {
      name: 'Gemini 2.5 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'gemini-3.1-flash-image': {
      name: 'Gemini 3.1 Flash Image',
      limit: {
        context: 1048576,
        output: 65536
      },
      modalities: {
        input: ['text', 'image'],
        output: ['image']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  const claudeModels = {
    'claude-fable-5-1': {
      name: 'Claude Fable 5.1',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    'claude-fable-5': {
      name: 'Claude Fable 5',
      limit: {
        context: 1048576,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          type: 'adaptive'
        }
      }
    },
    'claude-opus-4-6-thinking': {
      name: 'Claude 4.6 Opus (Thinking)',
      limit: {
        context: 200000,
        output: 128000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    },
    'claude-sonnet-4-6': {
      name: 'Claude 4.6 Sonnet',
      limit: {
        context: 200000,
        output: 64000
      },
      modalities: {
        input: ['text', 'image', 'pdf'],
        output: ['text']
      },
      options: {
        thinking: {
          budgetTokens: 24576,
          type: 'enabled'
        }
      }
    }
  }
  // Align context_window with Grok Build official sample (docs.x.ai/build/settings) where known.
  // Image/video: grok-imagine-image / grok-imagine-video on media endpoints — not this list.
  const grokModels = {
    'grok-4.5': {
      name: 'Grok 4.5',
      limit: { context: 500000, output: 64000 }
    },
    'grok-build-0.1': {
      name: 'Grok Build 0.1',
      limit: { context: 256000, output: 64000 }
    },
    'grok-4.20-multi-agent-0309': {
      name: 'Grok 4.20 Multi Agent (text / web_search)',
      limit: { context: 1000000, output: 64000 }
    },
    'grok-4.3': {
      name: 'Grok 4.3',
      limit: { context: 1000000, output: 64000 }
    },
    'grok-composer-2.5-fast': {
      name: 'Grok Composer 2.5 Fast',
      limit: { context: 500000, output: 64000 }
    }
  }

  if (platform === 'gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].models = geminiModels
  } else if (platform === 'anthropic') {
    provider[platform].npm = '@ai-sdk/anthropic'
    provider[platform].models = {
      'claude-opus-5-5': {
        name: 'Claude Opus 5.5',
        limit: { context: 1000000, output: 128000 },
        modalities: { input: ['text', 'image', 'pdf'], output: ['text'] },
        options: { thinking: { type: 'adaptive' }, effort: 'medium' },
        variants: {
          low: { effort: 'low' },
          medium: { effort: 'medium' },
          high: { effort: 'high' },
          xhigh: { effort: 'xhigh' },
          max: { effort: 'max' }
        }
      },
      'claude-sonnet-5-5': {
        name: 'Claude Sonnet 5.5',
        limit: { context: 1000000, output: 128000 },
        modalities: { input: ['text', 'image', 'pdf'], output: ['text'] },
        options: { thinking: { type: 'adaptive' }, effort: 'high' },
        variants: {
          low: { effort: 'low' },
          medium: { effort: 'medium' },
          high: { effort: 'high' },
          xhigh: { effort: 'xhigh' },
          max: { effort: 'max' }
        }
      }
    }
  } else if (platform === 'antigravity-claude') {
    provider[platform].npm = '@ai-sdk/anthropic'
    provider[platform].name = 'Antigravity (Claude)'
    provider[platform].models = claudeModels
  } else if (platform === 'antigravity-gemini') {
    provider[platform].npm = '@ai-sdk/google'
    provider[platform].name = 'Antigravity (Gemini)'
    provider[platform].models = antigravityGeminiModels
  } else if (platform === 'openai') {
    provider[platform].models = openaiModels
  } else if (platform === 'grok') {
    // Custom provider pointing at Sub2API OpenAI-compatible Responses/Chat endpoints.
    provider[platform].npm = '@ai-sdk/openai-compatible'
    provider[platform].name = 'Grok via Sub2API'
    provider[platform].models = grokModels
  }

  const agent =
    platform === 'openai'
      ? {
          build: {
            options: {
              store: false
            }
          },
          plan: {
            options: {
              store: false
            }
          }
        }
      : undefined

  const content = JSON.stringify(
    {
      provider,
      ...(agent ? { agent } : {}),
      $schema: 'https://opencode.ai/config.json'
    },
    null,
    2
  )

  return {
    path: pathLabel ?? 'opencode.json',
    content,
    hint: t('keys.useKeyModal.opencode.hint')
  }
}

const copyContent = async (content: string, index: number) => {
  const success = await clipboardCopy(content, t('keys.copied'))
  if (success) {
    copiedIndex.value = index
    setTimeout(() => {
      copiedIndex.value = null
    }, 2000)
  }
}
</script>

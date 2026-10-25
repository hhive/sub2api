import { getModelIcon } from '@/constants/modelIconPaths'

/** 主视觉模型卡的一项：vendor 取自图标表的 key，label 是展示名 */
export interface OrbitModelSpec {
  vendor: string
  label: string
}

export const MAX_ORBIT_MODELS = 24
export const MAX_ORBIT_MODEL_LABEL_LENGTH = 28

/** 主视觉不展示的厂商：图标表里有，但产品上不放到首页 */
export const EXCLUDED_ORBIT_VENDORS = ['hunyuan', 'wenxin', 'spark', 'yi']

/** 后台未配置时展示的默认名单 */
export const DEFAULT_ORBIT_MODELS: OrbitModelSpec[] = [
  { vendor: 'openai', label: 'GPT-6 Astra' },
  { vendor: 'openai', label: 'GPT-6 Sol' },
  { vendor: 'claude', label: 'Claude Opus 5.5' },
  { vendor: 'claude', label: 'CL Fable 5.1' },
  { vendor: 'gemini', label: 'Gemini 4 Argon' },
  { vendor: 'gemini', label: 'GM-3.6 Flash' },
  { vendor: 'xai', label: 'Grok 4.7' },
  { vendor: 'xai', label: 'Grok-4 Heavy' },
  { vendor: 'deepseek', label: 'DeepSeek V4.1 Flash' },
  { vendor: 'deepseek', label: 'DeepSeek V4 Pro' },
  { vendor: 'qwen', label: 'Qwen3.5-Max' },
  { vendor: 'qwen', label: 'Qwen3.8-Flash' },
  { vendor: 'zhipu', label: 'GLM-5' },
  { vendor: 'zhipu', label: 'GLM-5.3 Flash' },
  { vendor: 'moonshot', label: 'Kimi K3.1 Pro' },
  { vendor: 'doubao', label: 'Doubao 2.0' },
  { vendor: 'doubao', label: 'Seedance 2.0' },
  { vendor: 'minimax', label: 'MiniMax M3.1 Flash' },
  { vendor: 'mistral', label: 'Mistral L' },
  { vendor: 'meta', label: 'Llama 5' }
]

/**
 * 把后台配置的名单整理成可渲染列表：丢弃没有图标的 vendor，
 * 标签去空白并限长，超过上限截断；空结果回落到默认名单。
 */
export function normalizeOrbitModels(raw: unknown): OrbitModelSpec[] {
  const entries = Array.isArray(raw) ? raw : []
  const models: OrbitModelSpec[] = []

  for (const entry of entries) {
    if (!entry || typeof entry !== 'object') continue

    const record = entry as Record<string, unknown>
    const vendor = typeof record.vendor === 'string' ? record.vendor.trim() : ''
    if (!vendor || !getModelIcon(vendor)) continue
    if (EXCLUDED_ORBIT_VENDORS.includes(vendor)) continue

    const rawLabel = typeof record.label === 'string' ? record.label.trim() : ''
    const label = (rawLabel || vendor).slice(0, MAX_ORBIT_MODEL_LABEL_LENGTH)

    models.push({ vendor, label })
    if (models.length >= MAX_ORBIT_MODELS) break
  }

  return models.length > 0 ? models : DEFAULT_ORBIT_MODELS.map((model) => ({ ...model }))
}

import { apiClient } from '../client'

export type ImageSizeTier = '1k' | '2k' | '4k'
export type ImageAPIMode = 'images' | 'responses' | 'gemini_generate_content'
export type ImagePlaygroundHealthStatus = 'available' | 'temporary_unavailable' | 'half_open' | 'disabled'

export interface ImagePlaygroundModel {
  id: number
  display_name: string
  model: string
  api_mode: ImageAPIMode
  provider_name: string
  upstream_base_url: string
  upstream_api_key: string
  upstream_api_key_mask?: string
  price_1k: number
  price_2k: number
  price_4k: number
  supported_sizes: ImageSizeTier[]
  timeout_seconds: number
  fallback_to_responses_enabled: boolean
  health_status: ImagePlaygroundHealthStatus | string
  consecutive_failures: number
  half_open_attempts: number
  cooldown_count: number
  cooldown_until?: string | null
  last_health_error: string
  enabled: boolean
  sort_order: number
}

export interface ImagePlaygroundProbeRun {
  id: number
  run_id: string
  model_config_id: number
  model: string
  api_mode: ImageAPIMode | string
  upstream_base_url: string
  attempt: number
  status: string
  http_status_code: number
  error_message: string
  elapsed_ms: number
  response_bytes: number
  image_count: number
  created_at: string
}

export interface ImagePlaygroundProbeRunPage {
  items: ImagePlaygroundProbeRun[]
  total: number
  page: number
  page_size: number
}

export interface ImagePlaygroundUpstreamRequest {
  id: string
  user_id: number
  api_key_id: number
  api_key_suffix: string
  model_config_id: number
  model: string
  api_mode: ImageAPIMode | string
  provider_name: string
  upstream_base_url: string
  size_tier: string
  status: string
  upstream_status_code: number
  response_bytes: number
  image_count: number
  error_code: string
  error_message: string
  created_at: string
  updated_at: string
  completed_at?: string | null
}

export interface ImagePlaygroundUpstreamRequestPage {
  items: ImagePlaygroundUpstreamRequest[]
  total: number
  page: number
  page_size: number
}

export type ImagePlaygroundModelPayload = Omit<
  ImagePlaygroundModel,
  | 'id'
  | 'upstream_api_key_mask'
  | 'health_status'
  | 'consecutive_failures'
  | 'half_open_attempts'
  | 'cooldown_count'
  | 'cooldown_until'
  | 'last_health_error'
>

export async function listModels(): Promise<ImagePlaygroundModel[]> {
  const { data } = await apiClient.get<ImagePlaygroundModel[]>('/admin/image-playground/models')
  return data
}

export async function listProbeRuns(params: { page?: number; page_size?: number } = {}): Promise<ImagePlaygroundProbeRunPage> {
  const { data } = await apiClient.get<ImagePlaygroundProbeRunPage>('/admin/image-playground/model-probe-runs', { params })
  return data
}

export async function listUpstreamRequests(params: { page?: number; page_size?: number } = {}): Promise<ImagePlaygroundUpstreamRequestPage> {
  const { data } = await apiClient.get<ImagePlaygroundUpstreamRequestPage>('/admin/image-playground/upstream-requests', { params })
  return data
}

export async function runProbe(): Promise<{ ok: boolean; running?: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean; running?: boolean }>('/admin/image-playground/model-probe-runs/run')
  return data
}

export async function runModelProbe(id: number): Promise<{ ok: boolean; running?: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean; running?: boolean }>(`/admin/image-playground/models/${id}/probe`)
  return data
}

export async function createModel(payload: ImagePlaygroundModelPayload): Promise<ImagePlaygroundModel> {
  const { data } = await apiClient.post<ImagePlaygroundModel>('/admin/image-playground/models', payload)
  return data
}

export async function updateModel(id: number, payload: ImagePlaygroundModelPayload): Promise<ImagePlaygroundModel> {
  const { data } = await apiClient.patch<ImagePlaygroundModel>(`/admin/image-playground/models/${id}`, payload)
  return data
}

export async function deleteModel(id: number): Promise<{ ok: boolean }> {
  const { data } = await apiClient.delete<{ ok: boolean }>(`/admin/image-playground/models/${id}`)
  return data
}

const imagePlaygroundAPI = {
  listModels,
  listProbeRuns,
  listUpstreamRequests,
  runProbe,
  runModelProbe,
  createModel,
  updateModel,
  deleteModel,
}

export default imagePlaygroundAPI

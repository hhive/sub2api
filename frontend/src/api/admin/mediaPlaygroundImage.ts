import { apiClient } from '../client'

export type ImageSizeTier = '1k' | '2k' | '4k'
export type ImageAPIMode = 'images' | 'responses' | 'gemini_generate_content'
export type MediaPlaygroundImageHealthStatus = 'available' | 'temporary_unavailable' | 'half_open' | 'disabled'

export interface MediaPlaygroundImageModel {
	 id: number
	media_type: 'image'
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
  health_status: MediaPlaygroundImageHealthStatus | string
  consecutive_failures: number
  half_open_attempts: number
  cooldown_count: number
  cooldown_until?: string | null
  last_health_error: string
  enabled: boolean
  sort_order: number
}

export interface MediaPlaygroundImageProbeRun {
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

export interface MediaPlaygroundImageProbeRunPage {
  items: MediaPlaygroundImageProbeRun[]
  total: number
  page: number
  page_size: number
}

export interface MediaPlaygroundImageUpstreamRequest {
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

export interface MediaPlaygroundImageUpstreamRequestPage {
  items: MediaPlaygroundImageUpstreamRequest[]
  total: number
  page: number
  page_size: number
}

export type MediaPlaygroundImageModelPayload = Omit<
  MediaPlaygroundImageModel,
  | 'id'
  | 'media_type'
  | 'upstream_api_key_mask'
  | 'health_status'
  | 'consecutive_failures'
  | 'half_open_attempts'
  | 'cooldown_count'
  | 'cooldown_until'
  | 'last_health_error'
>

export async function listModels(): Promise<MediaPlaygroundImageModel[]> {
  const { data } = await apiClient.get<MediaPlaygroundImageModel[]>('/admin/media-playground/image/models')
  return data
}

export async function listProbeRuns(params: { page?: number; page_size?: number } = {}): Promise<MediaPlaygroundImageProbeRunPage> {
  const { data } = await apiClient.get<MediaPlaygroundImageProbeRunPage>('/admin/media-playground/image/model-probe-runs', { params })
  return data
}

export async function listUpstreamRequests(params: { page?: number; page_size?: number } = {}): Promise<MediaPlaygroundImageUpstreamRequestPage> {
  const { data } = await apiClient.get<MediaPlaygroundImageUpstreamRequestPage>('/admin/media-playground/image/upstream-requests', { params })
  return data
}

export async function runProbe(): Promise<{ ok: boolean; running?: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean; running?: boolean }>('/admin/media-playground/image/model-probe-runs/run')
  return data
}

export async function runModelProbe(id: number): Promise<{ ok: boolean; running?: boolean }> {
  const { data } = await apiClient.post<{ ok: boolean; running?: boolean }>(`/admin/media-playground/image/models/${id}/probe`)
  return data
}

export async function createModel(payload: MediaPlaygroundImageModelPayload): Promise<MediaPlaygroundImageModel> {
  const { data } = await apiClient.post<MediaPlaygroundImageModel>('/admin/media-playground/image/models', payload)
  return data
}

export async function updateModel(id: number, payload: MediaPlaygroundImageModelPayload): Promise<MediaPlaygroundImageModel> {
  const { data } = await apiClient.patch<MediaPlaygroundImageModel>(`/admin/media-playground/image/models/${id}`, payload)
  return data
}

export async function deleteModel(id: number): Promise<{ ok: boolean }> {
  const { data } = await apiClient.delete<{ ok: boolean }>(`/admin/media-playground/image/models/${id}`)
  return data
}

const mediaPlaygroundImageAPI = {
  listModels,
  listProbeRuns,
  listUpstreamRequests,
  runProbe,
  runModelProbe,
  createModel,
  updateModel,
  deleteModel,
}

export default mediaPlaygroundImageAPI

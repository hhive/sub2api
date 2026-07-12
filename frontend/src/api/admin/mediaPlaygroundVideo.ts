import { apiClient } from '../client'

export type VideoAPIMode = 'openai_videos' | 'seedance_content_generation'

export interface MediaPlaygroundVideoModel {
  id: number
  media_type: 'video'
  display_name: string
  model: string
  provider_name: string
  api_mode: VideoAPIMode
  upstream_base_url: string
  upstream_api_key: string
  upstream_api_key_mask?: string
  price_quota: number
  billing_mode: 'balance_prepaid'
  refund_enabled: boolean
  timeout_seconds: number
  enabled: boolean
  sort_order: number
}

export type MediaPlaygroundVideoModelPayload = Omit<MediaPlaygroundVideoModel, 'id' | 'media_type' | 'upstream_api_key_mask'>

export interface MediaPlaygroundVideoUpstreamRequest {
  id: number
  task_id: string
  user_id: number
  api_key_suffix: string
  model_config_id: number
  model: string
  provider_name: string
  upstream_base_url: string
  method: string
  endpoint: string
  content_type: string
  http_status_code: number
  error_message: string
  elapsed_ms: number
  response_bytes: number
  created_at: string
}

export interface MediaPlaygroundVideoUpstreamRequestPage {
  items: MediaPlaygroundVideoUpstreamRequest[]
  total: number
  page: number
  page_size: number
}

export async function listModels(): Promise<MediaPlaygroundVideoModel[]> {
  const { data } = await apiClient.get<MediaPlaygroundVideoModel[]>('/admin/media-playground/video/models')
  return data
}

export async function listUpstreamRequests(params: { page?: number; page_size?: number } = {}): Promise<MediaPlaygroundVideoUpstreamRequestPage> {
  const { data } = await apiClient.get<MediaPlaygroundVideoUpstreamRequestPage>('/admin/media-playground/video/upstream-requests', { params })
  return data
}

export async function createModel(payload: MediaPlaygroundVideoModelPayload): Promise<MediaPlaygroundVideoModel> {
  const { data } = await apiClient.post<MediaPlaygroundVideoModel>('/admin/media-playground/video/models', payload)
  return data
}

export async function updateModel(id: number, payload: MediaPlaygroundVideoModelPayload): Promise<MediaPlaygroundVideoModel> {
  const { data } = await apiClient.patch<MediaPlaygroundVideoModel>(`/admin/media-playground/video/models/${id}`, payload)
  return data
}

export async function deleteModel(id: number): Promise<{ ok: boolean }> {
  const { data } = await apiClient.delete<{ ok: boolean }>(`/admin/media-playground/video/models/${id}`)
  return data
}

const mediaPlaygroundVideoAPI = {
  listModels,
  listUpstreamRequests,
  createModel,
  updateModel,
  deleteModel,
}

export default mediaPlaygroundVideoAPI

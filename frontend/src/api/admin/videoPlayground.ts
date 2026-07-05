import { apiClient } from '../client'

export interface VideoPlaygroundModel {
  id: number
  display_name: string
  model: string
  provider_name: string
  upstream_base_url: string
  upstream_api_key: string
  upstream_api_key_mask?: string
  price_quota: number
  billing_mode: 'balance_prepaid'
  refund_enabled: boolean
  timeout_seconds: number
  enabled: boolean
  sort_order: number
  studio_model_id: string
  model_kind: 't2v' | 'i2v' | 'reference_video' | 'extend'
  input_schema_json: string
  payload_mapping_json: string
}

export type VideoPlaygroundModelPayload = Omit<VideoPlaygroundModel, 'id' | 'upstream_api_key_mask'>

export interface VideoPlaygroundUpstreamRequest {
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

export interface VideoPlaygroundUpstreamRequestPage {
  items: VideoPlaygroundUpstreamRequest[]
  total: number
  page: number
  page_size: number
}

export async function listModels(): Promise<VideoPlaygroundModel[]> {
  const { data } = await apiClient.get<VideoPlaygroundModel[]>('/admin/video-playground/models')
  return data
}

export async function listUpstreamRequests(params: { page?: number; page_size?: number } = {}): Promise<VideoPlaygroundUpstreamRequestPage> {
  const { data } = await apiClient.get<VideoPlaygroundUpstreamRequestPage>('/admin/video-playground/upstream-requests', { params })
  return data
}

export async function createModel(payload: VideoPlaygroundModelPayload): Promise<VideoPlaygroundModel> {
  const { data } = await apiClient.post<VideoPlaygroundModel>('/admin/video-playground/models', payload)
  return data
}

export async function updateModel(id: number, payload: VideoPlaygroundModelPayload): Promise<VideoPlaygroundModel> {
  const { data } = await apiClient.patch<VideoPlaygroundModel>(`/admin/video-playground/models/${id}`, payload)
  return data
}

export async function deleteModel(id: number): Promise<{ ok: boolean }> {
  const { data } = await apiClient.delete<{ ok: boolean }>(`/admin/video-playground/models/${id}`)
  return data
}

const videoPlaygroundAPI = {
  listModels,
  listUpstreamRequests,
  createModel,
  updateModel,
  deleteModel,
}

export default videoPlaygroundAPI

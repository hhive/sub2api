import { apiClient } from '../client'

export type VideoAPIMode = 'openai_videos' | 'openai_videos_v2' | 'seedance_content_generation'

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

export interface MediaPlaygroundVideoTask {
  task_id: string
  user_id: number
  model_config_id: number
  model: string
  request: Record<string, unknown>
  status: string
  progress: number
  upstream_task_id?: string
  upstream_response?: Record<string, unknown>
  error_message: string
  charged_quota: number
  refund_status: string
  refund_reason: string
  created_at: string
  updated_at: string
  completed_at?: string | null
  duration_ms?: number | null
}

export interface MediaPlaygroundVideoTaskPage {
  items: MediaPlaygroundVideoTask[]
  total: number
  page: number
  page_size: number
}

export interface MediaPlaygroundVideoTaskDetail {
  task: MediaPlaygroundVideoTask
}

export async function listModels(): Promise<MediaPlaygroundVideoModel[]> {
  const { data } = await apiClient.get<MediaPlaygroundVideoModel[]>('/admin/media-playground/video/models')
  return data
}

export async function listTasks(params: { page?: number; page_size?: number; status?: string; model?: string } = {}): Promise<MediaPlaygroundVideoTaskPage> {
  const { data } = await apiClient.get<MediaPlaygroundVideoTaskPage>('/admin/media-playground/video/tasks', { params })
  return data
}

export async function getTask(id: string): Promise<MediaPlaygroundVideoTaskDetail> {
  const { data } = await apiClient.get<MediaPlaygroundVideoTaskDetail>(`/admin/media-playground/video/tasks/${encodeURIComponent(id)}`)
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
  listTasks,
  getTask,
  createModel,
  updateModel,
  deleteModel,
}

export default mediaPlaygroundVideoAPI

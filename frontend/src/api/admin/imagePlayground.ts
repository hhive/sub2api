import { apiClient } from '../client'

export type ImageSizeTier = '1k' | '2k' | '4k'

export interface ImagePlaygroundModel {
  id: number
  display_name: string
  model: string
  provider_name: string
  upstream_base_url: string
  upstream_api_key: string
  upstream_api_key_mask?: string
  price_1k: number
  price_2k: number
  price_4k: number
  supported_sizes: ImageSizeTier[]
  timeout_seconds: number
  enabled: boolean
  sort_order: number
}

export type ImagePlaygroundModelPayload = Omit<ImagePlaygroundModel, 'id' | 'upstream_api_key_mask'>

export async function listModels(): Promise<ImagePlaygroundModel[]> {
  const { data } = await apiClient.get<ImagePlaygroundModel[]>('/admin/image-playground/models')
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
  createModel,
  updateModel,
  deleteModel,
}

export default imagePlaygroundAPI

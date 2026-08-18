import { apiClient } from '../client'

export type VendorHallWindow = '3h' | '24h' | '3d'
export type VendorHallSort = 'availability' | 'cache_hit_rate' | 'user_ttft' | 'requests' | 'updated_at'
export type VendorHallSchedulingStatus = 'schedulable' | 'paused' | 'disabled' | 'unknown'

export interface VendorHallTrendPoint {
  timestamp: string
  ttft_p95_ms: number | null
}

export interface VendorHallAccount {
  account_id: number
  account_name: string
  platform: string
  group_name: string | null
  rate_multiplier: number | null
  scheduling_status: VendorHallSchedulingStatus
  availability: number | null
  cache_hit_rate: number | null
  average_latency_ms: number | null
  p95_latency_ms: number | null
  user_ttft_p95_ms: number | null
  request_count: number
  collected_at: string | null
  temp_unschedulable_until?: string | null
  trend: VendorHallTrendPoint[]
}

export interface VendorHallSummary {
  total_accounts: number
  healthy_accounts: number
  paused_accounts: number
  average_availability: number | null
  updated_at: string | null
}

export interface VendorHallResponse {
  summary: VendorHallSummary
  items: VendorHallAccount[]
  total: number
  page: number
  page_size: number
}

export interface VendorHallQuery {
  window?: VendorHallWindow
  search?: string
  group?: string
  status?: string
  sort_by?: VendorHallSort
  sort_order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export interface PauseSchedulingResponse {
  account_id: number
  temp_unschedulable_until: string
}

export async function list(params: VendorHallQuery = {}): Promise<VendorHallResponse> {
  const { data } = await apiClient.get<VendorHallResponse>('/admin/ops/vendor-hall', { params })
  return data
}

export async function pauseScheduling(accountId: number): Promise<PauseSchedulingResponse> {
  const { data } = await apiClient.post<PauseSchedulingResponse>(
    `/admin/accounts/${accountId}/pause-scheduling`,
  )
  return data
}

export const vendorHallAPI = { list, pauseScheduling }

export default vendorHallAPI

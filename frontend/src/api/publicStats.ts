import { apiClient } from './client'

/**
 * 公开可靠性指标（无认证、仅聚合数据）。
 * availability_30d 为 0~1 小数，无可用数据时后端返回 null。
 */
export interface PublicStats {
  running_since: string
  running_days: number
  total_requests: number
  total_tokens: number
  availability_30d: number | null
  availability_window_days: number
  updated_at: string
}

export async function getPublicStats(): Promise<PublicStats> {
  const { data } = await apiClient.get<PublicStats>('/settings/public-stats', {
    timeout: 3000
  })
  return data
}

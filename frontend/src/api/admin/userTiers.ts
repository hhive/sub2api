/**
 * Admin User Tiers API endpoints
 * Handles user tier configuration and per-user read-only tier lookups
 */

import { apiClient } from '../client'
import type { AdminTier, AdminTierSaveRequest, AdminUserTierResponse } from '@/types'

/**
 * Get all tier configurations.
 * 默认包含已停用档位；传 false 只取启用中的档位。
 */
export async function listTiers(includeDisabled = true): Promise<AdminTier[]> {
  const { data } = await apiClient.get<AdminTier[]>('/admin/tiers', {
    params: includeDisabled ? undefined : { include_disabled: false }
  })
  return data
}

/**
 * Create a new tier (with its benefits submitted as a whole array)
 */
export async function createTier(request: AdminTierSaveRequest): Promise<AdminTier> {
  const { data } = await apiClient.post<AdminTier>('/admin/tiers', request)
  return data
}

/**
 * Update an existing tier
 */
export async function updateTier(id: number, request: AdminTierSaveRequest): Promise<AdminTier> {
  const { data } = await apiClient.put<AdminTier>(`/admin/tiers/${id}`, request)
  return data
}

/**
 * Reorder tiers by the given id sequence
 */
export async function reorderTiers(ids: number[]): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>('/admin/tiers/reorder', { ids })
  return data
}

/**
 * Delete a tier. 已产生授予记录（award_count > 0）的档位后端返回 409 USER_TIER_HAS_AWARDS，只能停用。
 */
export async function deleteTier(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/tiers/${id}`)
  return data
}

/**
 * Read-only lookup of a single user's tier state, awards and effects
 */
export async function getUserTier(userId: number): Promise<AdminUserTierResponse> {
  const { data } = await apiClient.get<AdminUserTierResponse>(`/admin/users/${userId}/tier`)
  return data
}

export const userTiersAPI = {
  listTiers,
  createTier,
  updateTier,
  reorderTiers,
  deleteTier,
  getUserTier,
}

export default userTiersAPI

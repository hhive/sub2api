/**
 * Admin User Tiers API endpoints
 * Handles user tier configuration and per-user read-only tier lookups
 */

import { apiClient } from '../client'
import type {
  AdminTier,
  AdminTierAssignmentSaveRequest,
  AdminTierSaveRequest,
  AdminUserTierAssignmentRemoveResponse,
  AdminUserTierAssignmentResponse,
  AdminUserTierResponse,
  UserTierFeatureSwitch,
} from '@/types'

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

/**
 * Read the user-tier feature master switch.
 * 关闭后用户端隐藏等级入口与页面，且不再发放新的等级权益（已发放的权益不回收）。
 */
export async function getFeatureSwitch(): Promise<UserTierFeatureSwitch> {
  const { data } = await apiClient.get<UserTierFeatureSwitch>('/admin/tiers/switch')
  return data
}

/**
 * Resolve a user by email and read their current tier assignment.
 * 邮箱不存在返回 404 USER_TIER_ASSIGN_USER_NOT_FOUND；无指派时 assignment 为 null。
 */
export async function getTierAssignment(email: string): Promise<AdminUserTierAssignmentResponse> {
  const { data } = await apiClient.get<AdminUserTierAssignmentResponse>('/admin/tiers/assignments', {
    params: { email }
  })
  return data
}

/**
 * Create or overwrite the tier assignment (one assignment per user; re-assigning replaces it).
 * 只接受消费档且必须是启用中的档位，否则后端返回 400。
 */
export async function saveTierAssignment(
  request: AdminTierAssignmentSaveRequest,
): Promise<AdminUserTierAssignmentResponse> {
  const { data } = await apiClient.put<AdminUserTierAssignmentResponse>('/admin/tiers/assignments', request)
  return data
}

/**
 * Remove the tier assignment. 幂等：本来就没有指派时 removed 为 false 且仍返回 200。
 */
export async function removeTierAssignment(email: string): Promise<AdminUserTierAssignmentRemoveResponse> {
  const { data } = await apiClient.delete<AdminUserTierAssignmentRemoveResponse>('/admin/tiers/assignments', {
    params: { email }
  })
  return data
}

/**
 * Update the user-tier feature master switch. The response echoes the stored value.
 */
export async function updateFeatureSwitch(enabled: boolean): Promise<UserTierFeatureSwitch> {
  const { data } = await apiClient.put<UserTierFeatureSwitch>('/admin/tiers/switch', { enabled })
  return data
}

export const userTiersAPI = {
  listTiers,
  createTier,
  updateTier,
  reorderTiers,
  deleteTier,
  getUserTier,
  getFeatureSwitch,
  updateFeatureSwitch,
  getTierAssignment,
  saveTierAssignment,
  removeTierAssignment,
}

export default userTiersAPI

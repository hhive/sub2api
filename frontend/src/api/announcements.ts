/**
 * User Announcements API endpoints
 */

import { apiClient } from './client'
import type { AnnouncementSite, UserAnnouncement } from '@/types'

export async function list(unreadOnly: boolean = false, site?: AnnouncementSite): Promise<UserAnnouncement[]> {
  const { data } = await apiClient.get<UserAnnouncement[]>('/announcements', {
    params: {
      ...(unreadOnly ? { unread_only: 1 } : {}),
      ...(site ? { site } : {})
    }
  })
  return data
}

export async function markRead(id: number, site?: AnnouncementSite): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/announcements/${id}/read`, undefined, {
    params: site ? { site } : {}
  })
  return data
}

const announcementsAPI = {
  list,
  markRead
}

export default announcementsAPI

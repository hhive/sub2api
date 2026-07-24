import { apiClient } from './client'

export interface LaunchResponse {
  redirect_url: string
}

export interface AdminExternalApp {
  app_id: string
  label_en: string
  label_zh: string
  enabled: boolean
  sort_order: number
}

export interface AdminExternalAppLaunchResponse extends LaunchResponse {
  expires_in: number
}

export async function listAdminExternalApps(): Promise<AdminExternalApp[]> {
  const { data } = await apiClient.get<AdminExternalApp[]>('/admin/external-apps')
  return data
}

export async function launchAdminExternalApp(appID: string): Promise<AdminExternalAppLaunchResponse> {
  const { data } = await apiClient.post<AdminExternalAppLaunchResponse>(
    `/admin/external-apps/${encodeURIComponent(appID)}/launch`,
  )
  return data
}

export async function launchMediaPlayground(): Promise<LaunchResponse> {
  const { data } = await apiClient.post<LaunchResponse>('/media-playground/launch')
  return data
}

export async function launchLobeHub(): Promise<LaunchResponse> {
  const { data } = await apiClient.post<LaunchResponse>('/lobehub/launch')
  return data
}

export async function launchVictoryMenu(menuId: string): Promise<LaunchResponse> {
  const { data } = await apiClient.post<LaunchResponse>('/menu-launch/victory', { menu_id: menuId })
  return data
}

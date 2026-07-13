import { apiClient } from './client'

export interface LaunchResponse {
  redirect_url: string
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

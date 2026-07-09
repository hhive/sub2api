import { apiClient } from './client'

export interface OnyxLaunchResponse {
  redirect_url: string
}

export async function launchOnyx(): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/onyx/launch')
  return data
}

export async function launchImagePlayground(): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/image-playground/launch')
  return data
}

export async function launchVideoPlayground(): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/video-playground/launch')
  return data
}

export async function launchLobeHub(): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/lobehub/launch')
  return data
}

export async function launchVictoryMenu(menuId: string): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/menu-launch/victory', { menu_id: menuId })
  return data
}

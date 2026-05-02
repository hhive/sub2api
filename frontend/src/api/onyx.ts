import { apiClient } from './client'

export interface OnyxLaunchResponse {
  redirect_url: string
}

export async function launchOnyx(): Promise<OnyxLaunchResponse> {
  const { data } = await apiClient.post<OnyxLaunchResponse>('/onyx/launch')
  return data
}

import { sanitizeUrl } from '@/utils/url'

export class InvalidAdminExternalAppUrlError extends Error {
  readonly code = 'INVALID_ADMIN_EXTERNAL_APP_URL'

  constructor() {
    super('Invalid administrator external app URL')
    this.name = 'InvalidAdminExternalAppUrlError'
  }
}

type LaunchResult = { redirect_url: string }
type OpenWindow = (url?: string | URL, target?: string, features?: string) => Window | null

export async function openAdminExternalAppWindow(
  requestLaunch: () => Promise<LaunchResult>,
  openWindow: OpenWindow = window.open.bind(window),
): Promise<void> {
  const launchWindow = openWindow('', '_blank')
  if (launchWindow) {
    launchWindow.opener = null
  }

  try {
    const result = await requestLaunch()
    const redirectURL = sanitizeUrl(result.redirect_url)
    const protocol = redirectURL ? new URL(redirectURL).protocol.toLowerCase() : ''
    if (!redirectURL || (protocol !== 'http:' && protocol !== 'https:')) {
      throw new InvalidAdminExternalAppUrlError()
    }

    if (launchWindow) {
      launchWindow.location.href = redirectURL
    } else {
      window.location.href = redirectURL
    }
  } catch (error) {
    launchWindow?.close()
    throw error
  }
}

import { describe, expect, it, vi } from 'vitest'

import { openAdminExternalAppWindow } from '@/utils/adminExternalAppLaunch'

function createPopup() {
  return {
    close: vi.fn(),
    location: { href: '' },
    opener: window,
  }
}

describe('openAdminExternalAppWindow', () => {
  it('opens synchronously, removes opener access, and navigates only after launch resolves', async () => {
    const popup = createPopup()
    const open = vi.fn(() => popup as unknown as Window)
    let resolveLaunch!: (value: { redirect_url: string }) => void
    const requestLaunch = vi.fn(
      () => new Promise<{ redirect_url: string }>((resolve) => { resolveLaunch = resolve }),
    )

    const pending = openAdminExternalAppWindow(requestLaunch, open)

    expect(open).toHaveBeenCalledWith('', '_blank')
    expect(popup.opener).toBeNull()
    expect(popup.location.href).toBe('')

    resolveLaunch({ redirect_url: 'https://media.example.com/admin?token=one-time' })
    await pending

    expect(popup.location.href).toBe('https://media.example.com/admin?token=one-time')
  })

  it('closes the pre-opened window when launch fails', async () => {
    const popup = createPopup()

    await expect(
      openAdminExternalAppWindow(
        () => Promise.reject(new Error('launch failed')),
        () => popup as unknown as Window,
      ),
    ).rejects.toThrow('launch failed')
    expect(popup.close).toHaveBeenCalledOnce()
  })

  it('rejects non-http launch URLs and closes the pre-opened window', async () => {
    const popup = createPopup()

    await expect(
      openAdminExternalAppWindow(
        () => Promise.resolve({ redirect_url: 'javascript:alert(1)' }),
        () => popup as unknown as Window,
      ),
    ).rejects.toMatchObject({ code: 'INVALID_ADMIN_EXTERNAL_APP_URL' })
    expect(popup.location.href).toBe('')
    expect(popup.close).toHaveBeenCalledOnce()
  })
})

import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import {
  launchAdminExternalApp,
  launchMediaPlayground,
  launchLobeHub,
  launchVictoryMenu,
  listAdminExternalApps,
} from '@/api/launch'

describe('launch api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({
      data: [
        {
          app_id: 'media-management',
          label_en: 'Media Management',
          label_zh: '媒体管理',
          enabled: true,
          sort_order: 10,
        },
      ],
    })
    post.mockResolvedValue({ data: { redirect_url: 'https://example.com/launch?token=abc' } })
  })

  it('launches media playground through the backend launch endpoint', async () => {
    const result = await launchMediaPlayground()

    expect(post).toHaveBeenCalledWith('/media-playground/launch')
    expect(result.redirect_url).toBe('https://example.com/launch?token=abc')
  })

  it('launches LobeHub through the backend launch endpoint', async () => {
    const result = await launchLobeHub()

    expect(post).toHaveBeenCalledWith('/lobehub/launch')
    expect(result.redirect_url).toBe('https://example.com/launch?token=abc')
  })

  it('launches a victory menu item through the backend launch endpoint', async () => {
    const result = await launchVictoryMenu('xiaoni-offer')

    expect(post).toHaveBeenCalledWith('/menu-launch/victory', { menu_id: 'xiaoni-offer' })
    expect(result.redirect_url).toBe('https://example.com/launch?token=abc')
  })

  it('lists enabled administrator external apps', async () => {
    const result = await listAdminExternalApps()

    expect(get).toHaveBeenCalledWith('/admin/external-apps')
    expect(result).toEqual([
      {
        app_id: 'media-management',
        label_en: 'Media Management',
        label_zh: '媒体管理',
        enabled: true,
        sort_order: 10,
      },
    ])
  })

  it('launches an administrator external app without sending identity or redirect data', async () => {
    const result = await launchAdminExternalApp('media-management')

    expect(post).toHaveBeenCalledWith('/admin/external-apps/media-management/launch')
    expect(result.redirect_url).toBe('https://example.com/launch?token=abc')
  })
})

import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import { launchMediaPlayground, launchLobeHub, launchVictoryMenu } from '@/api/launch'

describe('launch api', () => {
  beforeEach(() => {
    post.mockReset()
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
})

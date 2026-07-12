import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import { launchImagePlayground, launchLobeHub, launchOnyx, launchVictoryMenu } from '@/api/onyx'

describe('onyx api', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: { redirect_url: 'https://onyx.example.com/api/sub2api/exchange?token=abc' } })
  })

  it('launches Onyx through the backend launch endpoint', async () => {
    const result = await launchOnyx()

    expect(post).toHaveBeenCalledWith('/onyx/launch')
    expect(result.redirect_url).toBe('https://onyx.example.com/api/sub2api/exchange?token=abc')
  })

  it('launches image playground through the backend launch endpoint', async () => {
    post.mockResolvedValueOnce({ data: { redirect_url: 'https://xiaoni-ai.zle.ee/image_playground?apiMode=images' } })

    const result = await launchImagePlayground()

    expect(post).toHaveBeenCalledWith('/image-playground/launch')
    expect(result.redirect_url).toBe('https://xiaoni-ai.zle.ee/image_playground?apiMode=images')
  })

  it('launches LobeHub through the backend launch endpoint', async () => {
    post.mockResolvedValueOnce({ data: { redirect_url: 'https://lobe.example.com/api/sub2api/launch?token=abc' } })

    const result = await launchLobeHub()

    expect(post).toHaveBeenCalledWith('/lobehub/launch')
    expect(result.redirect_url).toBe('https://lobe.example.com/api/sub2api/launch?token=abc')
  })

  it('launches a victory menu item through the backend launch endpoint', async () => {
    post.mockResolvedValueOnce({ data: { redirect_url: 'https://offer.xiaoni-ai.top?apikey=sk-test' } })

    const result = await launchVictoryMenu('xiaoni-offer')

    expect(post).toHaveBeenCalledWith('/menu-launch/victory', { menu_id: 'xiaoni-offer' })
    expect(result.redirect_url).toBe('https://offer.xiaoni-ai.top?apikey=sk-test')
  })
})

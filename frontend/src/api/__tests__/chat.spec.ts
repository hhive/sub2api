import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const adapter = vi.fn()

vi.mock('@/i18n', () => ({
  getLocale: () => 'zh-CN',
}))

describe('chat api', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('loads chat models from the user chat endpoint', async () => {
    const { apiClient } = await import('@/api/client')
    apiClient.defaults.adapter = adapter.mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          models: [{ id: 'gpt-5.5', name: 'GPT-5.5', provider: 'openai' }],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    const { getChatModels } = await import('@/api/chat')

    const models = await getChatModels()

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/models')
    expect(models).toEqual([{ id: 'gpt-5.5', name: 'GPT-5.5', provider: 'openai' }])
  })

  it('loads text chat models from the explicit chat endpoint', async () => {
    const { apiClient } = await import('@/api/client')
    apiClient.defaults.adapter = adapter.mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          models: [{ id: 'gpt-5.5', name: 'GPT-5.5', provider: 'openai' }],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    const { getTextChatModels } = await import('@/api/chat')

    const models = await getTextChatModels()

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/models/chat')
    expect(models).toEqual([{ id: 'gpt-5.5', name: 'GPT-5.5', provider: 'openai' }])
  })

  it('loads image generation models from the user chat endpoint', async () => {
    const { apiClient } = await import('@/api/client')
    apiClient.defaults.adapter = adapter.mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          models: [{ id: 'gpt-image-2', name: 'GPT Image 2', provider: 'openai' }],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    const { getImageGenerationModels } = await import('@/api/chat')

    const models = await getImageGenerationModels()

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/models/images')
    expect(models).toEqual([{ id: 'gpt-image-2', name: 'GPT Image 2', provider: 'openai' }])
  })

  it('posts chat completions to the user chat endpoint', async () => {
    const { apiClient } = await import('@/api/client')
    apiClient.defaults.adapter = adapter.mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          id: 'chatcmpl-1',
          choices: [{ message: { role: 'assistant', content: '你好' } }],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    const { sendChatCompletion } = await import('@/api/chat')

    const completion = await sendChatCompletion({
      model: 'gpt-5.5',
      messages: [{ role: 'user', content: '你好' }],
      stream: false,
    })

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/completions')
    expect(JSON.parse(adapter.mock.calls[0][0].data)).toEqual({
      model: 'gpt-5.5',
      messages: [{ role: 'user', content: '你好' }],
      stream: false,
    })
    expect(completion.choices[0].message.content).toBe('你好')
  })

  it('posts image generation requests to the user chat endpoint', async () => {
    const { apiClient } = await import('@/api/client')
    apiClient.defaults.adapter = adapter.mockResolvedValue({
      status: 200,
      data: {
        code: 0,
        data: {
          created: 1776873600,
          data: [{ url: 'https://example.com/image.png' }],
        },
      },
      headers: {},
      config: {},
      statusText: 'OK',
    })
    const { sendImageGeneration } = await import('@/api/chat')

    const response = await sendImageGeneration({
      model: 'gpt-image-2',
      prompt: '腾云的孙悟空',
      response_format: 'url',
    })

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/images/generations')
    expect(JSON.parse(adapter.mock.calls[0][0].data)).toEqual({
      model: 'gpt-image-2',
      prompt: '腾云的孙悟空',
      response_format: 'url',
    })
    expect(response.data[0].url).toBe('https://example.com/image.png')
  })
})

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
          models: [{ id: 'gpt-4o-mini', name: 'gpt-4o-mini', provider: 'openai' }],
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
    expect(models).toEqual([{ id: 'gpt-4o-mini', name: 'gpt-4o-mini', provider: 'openai' }])
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
      model: 'gpt-4o-mini',
      messages: [{ role: 'user', content: '你好' }],
      stream: false,
    })

    expect(adapter).toHaveBeenCalledTimes(1)
    expect(adapter.mock.calls[0][0].url).toBe('/chat/completions')
    expect(JSON.parse(adapter.mock.calls[0][0].data)).toEqual({
      model: 'gpt-4o-mini',
      messages: [{ role: 'user', content: '你好' }],
      stream: false,
    })
    expect(completion.choices[0].message.content).toBe('你好')
  })
})

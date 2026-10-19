import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import OpenAIGPT61SolAccountSelector from '../OpenAIGPT61SolAccountSelector.vue'

const messages: Record<string, string> = {
  'admin.settings.openaiGPT61SolLongContextBilling.accountMissing': 'Account missing or inaccessible',
  'admin.settings.openaiGPT61SolLongContextBilling.accountIdFallback': 'Account #{id}',
  'admin.settings.openaiGPT61SolLongContextBilling.removeAccount': 'Remove account',
  'admin.settings.openaiGPT61SolLongContextBilling.accountSearchPlaceholder': 'Search accounts',
  'admin.settings.openaiGPT61SolLongContextBilling.accountSearchEmpty': 'No accounts found',
  'common.loading': 'Loading',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const message = messages[key] ?? key
      return params
        ? Object.entries(params).reduce(
            (value, [name, replacement]) => value.replace(`{${name}}`, String(replacement)),
            message,
          )
        : message
    },
  }),
}))

const mockList = vi.fn()
const mockGetById = vi.fn()

vi.mock('@/api', () => ({
  adminAPI: {
    accounts: {
      list: (...args: unknown[]) => mockList(...args),
      getById: (...args: unknown[]) => mockGetById(...args),
    },
  },
}))

function accountItem(
  id: number,
  name: string,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return {
    id,
    name,
    platform: 'openai',
    type: 'apikey',
    parent_account_id: null,
    ...overrides,
  }
}

function listResponse(items: Record<string, unknown>[]) {
  return { items, total: items.length, page: 1, page_size: 20, pages: 1 }
}

async function search(wrapper: ReturnType<typeof mount>, query: string) {
  const input = wrapper.get('input')
  await input.trigger('focus')
  await input.setValue(query)
  await input.trigger('input')
  vi.advanceTimersByTime(300)
  await flushPromises()
}

describe('OpenAIGPT61SolAccountSelector', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockList.mockReset()
    mockGetById.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('hydrates existing IDs to account names without changing the saved IDs', async () => {
    mockGetById.mockResolvedValue(accountItem(7, 'existing-account'))

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [7] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(mockGetById).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('existing-account')
    expect(wrapper.text()).toContain('#7')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('searches OpenAI accounts after the debounce and adds the selected account ID', async () => {
    mockList.mockResolvedValue(listResponse([accountItem(9, 'team-a')]))

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [] },
      global: { stubs: { Icon: true } },
    })
    await search(wrapper, 'team')

    expect(mockList).toHaveBeenCalledWith(1, 20, { platform: 'openai', search: 'team' })
    const result = wrapper
      .findAll('button')
      .find((button) => button.text().includes('team-a'))
    expect(result).toBeDefined()
    await result!.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[[9]]])
  })

  it('filters subscription and shadow accounts out of the search results', async () => {
    mockList.mockResolvedValue(
      listResponse([
        accountItem(1, 'api-key-account'),
        accountItem(2, 'oauth-account', { type: 'oauth' }),
        accountItem(3, 'setup-token-account', { type: 'setup-token' }),
        accountItem(4, 'shadow-account', { parent_account_id: 1 }),
      ]),
    )

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [] },
      global: { stubs: { Icon: true } },
    })
    await search(wrapper, 'account')

    expect(wrapper.text()).toContain('api-key-account')
    expect(wrapper.text()).not.toContain('oauth-account')
    expect(wrapper.text()).not.toContain('setup-token-account')
    expect(wrapper.text()).not.toContain('shadow-account')
  })

  it('removes a selected account ID without touching the others', async () => {
    mockGetById.mockImplementation(async (id: number) => accountItem(id, `acct-${id}`))

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [5, 6] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    const removeButtons = wrapper.findAll('button[aria-label="Remove account"]')
    expect(removeButtons).toHaveLength(2)
    await removeButtons[0].trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([[[6]]])
  })

  it('keeps an unresolved saved ID visible, flagged as missing, and removable', async () => {
    mockGetById.mockRejectedValue(new Error('not found'))

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [42] },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Account #42')
    expect(wrapper.text()).toContain('Account missing or inaccessible')
    await wrapper.get('button[aria-label="Remove account"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[[]]])
  })

  it('ignores search and removal while disabled', async () => {
    mockGetById.mockResolvedValue(accountItem(3, 'acct-three'))

    const wrapper = mount(OpenAIGPT61SolAccountSelector, {
      props: { modelValue: [3], disabled: true },
      global: { stubs: { Icon: true } },
    })
    await flushPromises()

    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    expect(wrapper.get('button[aria-label="Remove account"]').attributes('disabled')).toBeDefined()

    await search(wrapper, 'team')

    expect(mockList).not.toHaveBeenCalled()
    expect(wrapper.find('.z-50').exists()).toBe(false)
  })
})

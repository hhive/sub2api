import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import RedeemView from '../RedeemView.vue'

const { getHistory, getPublicSettings } = vi.hoisted(() => ({
  getHistory: vi.fn(),
  getPublicSettings: vi.fn(),
}))

vi.mock('@/api', () => ({
  redeemAPI: {
    getHistory,
    redeem: vi.fn(),
  },
  authAPI: {
    getPublicSettings,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { balance: 0, concurrency: 0 },
    refreshUser: vi.fn(),
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    fetchActiveSubscriptions: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('RedeemView', () => {
  it('renders configured redeem purchase url as a purchase entry above the redeem button', async () => {
    getHistory.mockResolvedValue([])
    getPublicSettings.mockResolvedValue({
      contact_info: '',
      redeem_purchase_url: 'https://pay.ldxp.cn/shop/xiaoni-ai',
    })

    const wrapper = mount(RedeemView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const link = wrapper.get('a[href="https://pay.ldxp.cn/shop/xiaoni-ai"]')
    expect(link.text()).toContain('redeem.purchaseLink')
    expect(link.text()).not.toContain('https://pay.ldxp.cn/shop/xiaoni-ai')
    expect(wrapper.text()).not.toContain('https://pay.ldxp.cn/shop/xiaoni-ai')

    const redeemButton = wrapper.get('button[type="submit"]')
    expect(link.element.compareDocumentPosition(redeemButton.element)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    )
    expect(link.classes()).toContain('redeem-purchase-entry')
  })
})

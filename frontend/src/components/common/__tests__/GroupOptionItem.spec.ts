import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'

import GroupOptionItem from '../GroupOptionItem.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null })
}))

const global = {
  plugins: [createPinia()],
  stubs: {
    GroupBadge: {
      props: ['name'],
      template: '<span class="group-badge">{{ name }}</span>'
    }
  }
}

describe('GroupOptionItem', () => {
  it('shows a compact supported model summary', () => {
    const wrapper = mount(GroupOptionItem, {
      global,
      props: {
        name: 'API Group',
        platform: 'openai',
        subscriptionType: 'standard',
        rateMultiplier: 1,
        supportedModels: [
          { name: 'gpt-5.4', platform: 'openai', pricing: null },
          { name: 'gpt-5.4-mini', platform: 'openai', pricing: null },
          { name: 'gpt-5.3-codex', platform: 'openai', pricing: null },
          { name: 'gpt-5.2', platform: 'openai', pricing: null }
        ],
        noModelsLabel: 'No available models'
      }
    })

    expect(wrapper.text()).toContain('gpt-5.4')
    expect(wrapper.text()).toContain('gpt-5.4-mini')
    expect(wrapper.text()).toContain('gpt-5.3-codex')
    expect(wrapper.text()).toContain('+1')
    expect(wrapper.text()).not.toContain('No available models')
  })

  it('shows an empty model label when no models are available', () => {
    const wrapper = mount(GroupOptionItem, {
      global,
      props: {
        name: 'API Group',
        platform: 'openai',
        subscriptionType: 'standard',
        rateMultiplier: 1,
        supportedModels: [],
        noModelsLabel: 'No available models'
      }
    })

    expect(wrapper.text()).toContain('No available models')
  })

  it('applies multiline and overflow-safe description styles', () => {
    const description = 'First section\nvery-long-unbroken-description-value-that-must-not-overflow'
    const wrapper = mount(GroupOptionItem, {
      global,
      props: {
        name: 'Example group',
        platform: 'openai',
        description
      }
    })

    const descriptionElement = wrapper
      .findAll('span')
      .find((element) => element.text() === description)

    expect(descriptionElement).toBeDefined()
    expect(descriptionElement?.classes()).toContain('whitespace-pre-line')
    expect(descriptionElement?.classes()).toContain('[overflow-wrap:anywhere]')
    expect(descriptionElement?.classes()).toContain('line-clamp-3')
    expect(wrapper.find('[title]').attributes('title')).toBe(description)
  })
})

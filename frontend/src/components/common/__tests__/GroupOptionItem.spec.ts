import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import GroupOptionItem from '../GroupOptionItem.vue'

const global = {
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
})

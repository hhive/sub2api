import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import VendorMetricHelp from '../VendorMetricHelp.vue'

describe('VendorMetricHelp', () => {
  it('shows the metric definition on click and closes it with Escape', async () => {
    const wrapper = mount(VendorMetricHelp, {
      props: { label: 'User TTFT P95', description: 'Time to the first response token.' },
      global: { stubs: { teleport: true } },
    })

    const button = wrapper.get('button')
    expect(button.attributes('aria-expanded')).toBe('false')

    await button.trigger('click')
    expect(button.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[role="dialog"]').text()).toContain('Time to the first response token.')

    await button.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { UserSupportedModel } from '@/types'
import KeyModelsCell from '../KeyModelsCell.vue'

const createModels = (): UserSupportedModel[] => [
  { name: 'claude-opus-4-1', platform: 'anthropic' },
  { name: 'gpt-5.5', platform: 'openai' },
  { name: 'claude-sonnet-5-5', platform: 'anthropic' },
  { name: 'gemini-3-pro', platform: 'gemini' },
  { name: 'gpt-5.4', platform: 'openai' },
]

const mounted: VueWrapper[] = []

const mountCell = (models: UserSupportedModel[] = createModels()): VueWrapper => {
  const wrapper = mount(KeyModelsCell, {
    props: { models, emptyLabel: 'No models' },
    attachTo: document.body,
    global: { stubs: { Teleport: true } },
  })
  mounted.push(wrapper)
  return wrapper
}

const popover = (wrapper: VueWrapper) => wrapper.find('[data-test="key-models-popover"]')

describe('KeyModelsCell', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    mounted.splice(0).forEach((wrapper) => wrapper.unmount())
    vi.useRealTimers()
  })

  it('previews the first three models with a hidden count', () => {
    const wrapper = mountCell()

    const trigger = wrapper.get('[data-test="key-models-trigger"]')
    expect(trigger.findAll('[data-test="model-preview-pill"]').map((pill) => pill.text())).toEqual([
      'claude-opus-4-1',
      'gpt-5.5',
      'claude-sonnet-5-5',
    ])
    expect(trigger.get('[data-test="model-hidden-count"]').text()).toBe('+2')
  })

  it('opens on hover with the full list grouped by platform and sorted by name', async () => {
    const wrapper = mountCell()

    await wrapper.get('[data-test="key-models-trigger"]').trigger('mouseenter')
    await nextTick()

    expect(
      popover(wrapper)
        .findAll('[data-test="model-popover-pill"]')
        .map((pill) => pill.text())
    ).toEqual([
      'claude-opus-4-1',
      'claude-sonnet-5-5',
      'gemini-3-pro',
      'gpt-5.4',
      'gpt-5.5',
    ])
    expect(
      popover(wrapper)
        .findAll('[data-test="model-platform"]')
        .map((heading) => heading.text())
    ).toEqual(['Anthropic', 'Gemini', 'OpenAI'])
  })

  it('opens on keyboard focus', async () => {
    const wrapper = mountCell()

    await wrapper.get('[data-test="key-models-trigger"]').trigger('focus')
    await nextTick()

    expect(popover(wrapper).exists()).toBe(true)
  })

  it('closes shortly after the pointer leaves and stays open while it is over the popover', async () => {
    const wrapper = mountCell()
    const trigger = wrapper.get('[data-test="key-models-trigger"]')

    await trigger.trigger('mouseenter')
    await nextTick()
    await trigger.trigger('mouseleave')

    vi.advanceTimersByTime(100)
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await popover(wrapper).trigger('mouseenter')
    vi.advanceTimersByTime(200)
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await popover(wrapper).trigger('mouseleave')
    vi.advanceTimersByTime(149)
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    vi.advanceTimersByTime(1)
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)
  })

  it('pins open on click and closes on a second click or an outside click', async () => {
    const wrapper = mountCell()
    const trigger = wrapper.get('[data-test="key-models-trigger"]')

    await trigger.trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await trigger.trigger('mouseleave')
    vi.advanceTimersByTime(200)
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await trigger.trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)

    await trigger.trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    document.body.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)
  })

  it('stays open when pressing inside the popover', async () => {
    const wrapper = mountCell()
    const trigger = wrapper.get('[data-test="key-models-trigger"]')

    await trigger.trigger('click')
    await nextTick()

    popover(wrapper).element.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await nextTick()

    expect(popover(wrapper).exists()).toBe(true)
  })

  it('closes a pinned popover when another instance is pressed', async () => {
    const first = mountCell()
    const second = mountCell()

    await first.get('[data-test="key-models-trigger"]').trigger('click')
    await nextTick()
    expect(popover(first).exists()).toBe(true)

    second
      .get('[data-test="key-models-trigger"]')
      .element.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true }))
    await nextTick()

    expect(popover(first).exists()).toBe(false)
    expect(popover(second).exists()).toBe(false)
  })

  it('closes when the model list becomes empty', async () => {
    const wrapper = mountCell()

    await wrapper.get('[data-test="key-models-trigger"]').trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await wrapper.setProps({ models: [] })
    await nextTick()

    expect(popover(wrapper).exists()).toBe(false)
    expect(wrapper.get('[data-test="key-models-empty"]').text()).toBe('No models')
  })

  it('sorts version-like model names numerically within a platform', async () => {
    const wrapper = mountCell([
      { name: 'gpt-5.10', platform: 'openai' },
      { name: 'gpt-5.4', platform: 'openai' },
    ])

    await wrapper.get('[data-test="key-models-trigger"]').trigger('mouseenter')
    await nextTick()

    expect(
      popover(wrapper)
        .findAll('[data-test="model-popover-pill"]')
        .map((pill) => pill.text())
    ).toEqual(['gpt-5.4', 'gpt-5.10'])
  })

  it('stays closed after the model list is cleared and refilled', async () => {
    const wrapper = mountCell()

    await wrapper.get('[data-test="key-models-trigger"]').trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    await wrapper.setProps({ models: [] })
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)

    await wrapper.setProps({ models: createModels() })
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)
    expect(wrapper.get('[data-test="key-models-trigger"]').exists()).toBe(true)
  })

  it('removes document and window listeners on unmount', async () => {
    const removeDocumentListener = vi.spyOn(document, 'removeEventListener')
    const removeWindowListener = vi.spyOn(window, 'removeEventListener')
    const wrapper = mountCell()

    await wrapper.get('[data-test="key-models-trigger"]').trigger('click')
    await nextTick()

    wrapper.unmount()
    mounted.splice(mounted.indexOf(wrapper), 1)

    expect(removeDocumentListener).toHaveBeenCalledWith('pointerdown', expect.any(Function), true)
    expect(removeWindowListener).toHaveBeenCalledWith('keydown', expect.any(Function))
    removeDocumentListener.mockRestore()
    removeWindowListener.mockRestore()
  })

  it('closes on Escape', async () => {
    const wrapper = mountCell()
    const trigger = wrapper.get('[data-test="key-models-trigger"]')

    await trigger.trigger('click')
    await nextTick()
    expect(popover(wrapper).exists()).toBe(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(popover(wrapper).exists()).toBe(false)
  })

  it('renders the empty label without a trigger when there are no models', () => {
    const wrapper = mountCell([])

    expect(wrapper.get('[data-test="key-models-empty"]').text()).toBe('No models')
    expect(wrapper.find('[data-test="key-models-trigger"]').exists()).toBe(false)
    expect(popover(wrapper).exists()).toBe(false)
  })
})

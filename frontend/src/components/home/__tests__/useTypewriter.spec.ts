import { effectScope, nextTick, ref, type EffectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useTypewriter } from '../useTypewriter'

let scope: EffectScope | null = null

function run<T>(factory: () => T): T {
  scope = effectScope()
  return scope.run(factory) as T
}

describe('useTypewriter', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    scope?.stop()
    scope = null
    vi.useRealTimers()
  })

  it('reveals the whole text at once when disabled', () => {
    const state = run(() => useTypewriter('统一入口', { enabled: false }))

    expect(state.visible.value).toBe('统一入口')
    expect(state.done.value).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('advances one character per delay until the text is complete', async () => {
    const state = run(() => useTypewriter('abc', { charDelayMs: 100 }))

    expect(state.visible.value).toBe('')
    expect(state.done.value).toBe(false)

    vi.advanceTimersByTime(100)
    expect(state.visible.value).toBe('a')
    expect(state.done.value).toBe(false)

    vi.advanceTimersByTime(100)
    expect(state.visible.value).toBe('ab')

    vi.advanceTimersByTime(200)
    expect(state.visible.value).toBe('abc')
    expect(state.done.value).toBe(true)
    expect(vi.getTimerCount()).toBe(0)

    await nextTick()
  })

  it('stops advancing after stop() keeps the revealed prefix', () => {
    const state = run(() => useTypewriter('abcd', { charDelayMs: 50 }))

    vi.advanceTimersByTime(50)
    state.stop()

    vi.advanceTimersByTime(500)

    expect(state.visible.value).toBe('a')
    expect(state.done.value).toBe(false)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('replays from an empty prefix on restart()', () => {
    const state = run(() => useTypewriter('ab', { charDelayMs: 50 }))

    vi.advanceTimersByTime(100)
    expect(state.done.value).toBe(true)

    state.restart()

    expect(state.visible.value).toBe('')
    expect(state.done.value).toBe(false)

    vi.advanceTimersByTime(50)
    expect(state.visible.value).toBe('a')
  })

  it('restarts when the source text changes', async () => {
    const source = ref('ab')
    const state = run(() => useTypewriter(source, { charDelayMs: 50 }))

    vi.advanceTimersByTime(100)
    expect(state.visible.value).toBe('ab')

    source.value = 'xyz'
    await nextTick()

    expect(state.visible.value).toBe('')
    vi.advanceTimersByTime(50)
    expect(state.visible.value).toBe('x')
  })

  it('leaves no pending timer after the owning scope is disposed', () => {
    const state = run(() => useTypewriter('abcdef', { charDelayMs: 30 }))

    vi.advanceTimersByTime(30)
    expect(vi.getTimerCount()).toBe(1)

    scope?.stop()
    scope = null

    expect(vi.getTimerCount()).toBe(0)
    expect(state.visible.value).toBe('a')
  })
})

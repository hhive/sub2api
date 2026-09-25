import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

function getMessage(messages: Record<string, unknown>, key: string): unknown {
  return key.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, messages)
}

// 用户端等级功能在侧边栏入口（nav.myTier）与等级页标题（membership.title）两处渲染，
// 两处必须同名，统一叫「等级权益」（en: Tier Benefits）；
// 同时与管理端的「等级与权益」（nav.tierConfig）保持区分，避免两端合成同一个名字。
describe('user tier entry labels', () => {
  it('names the user-facing tier feature 等级权益 in zh', () => {
    expect(getMessage(zh, 'nav.myTier')).toBe('等级权益')
    expect(getMessage(zh, 'membership.title')).toBe('等级权益')
    expect(getMessage(zh, 'nav.tierConfig')).toBe('等级与权益')
  })

  it('names the user-facing tier feature Tier Benefits in en', () => {
    expect(getMessage(en, 'nav.myTier')).toBe('Tier Benefits')
    expect(getMessage(en, 'membership.title')).toBe('Tier Benefits')
  })
})

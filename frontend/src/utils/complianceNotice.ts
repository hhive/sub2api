import type { PublicSettings } from '@/types'

export const COMPLIANCE_NOTICE_STORAGE_KEY = 'sub2api_compliance_notice_consent'
export const COMPLIANCE_NOTICE_EVENT = 'sub2api:compliance-notice-decision'

export type ComplianceNoticeDecision = 'accepted' | 'declined'

export interface ComplianceNoticeState {
  revision: string
  decision: ComplianceNoticeDecision
  decided_at: string
}

export function getComplianceNoticeRevision(settings: PublicSettings | null | undefined): string {
  return (settings?.compliance_notice_revision || '').trim()
}

export function isComplianceNoticeEnabled(settings: PublicSettings | null | undefined): boolean {
  return settings?.compliance_notice_enabled === true && getComplianceNoticeRevision(settings) !== ''
}

export function readComplianceNoticeState(): ComplianceNoticeState | null {
  try {
    const raw = localStorage.getItem(COMPLIANCE_NOTICE_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<ComplianceNoticeState>
    if (!parsed.revision || (parsed.decision !== 'accepted' && parsed.decision !== 'declined')) {
      return null
    }
    return {
      revision: parsed.revision,
      decision: parsed.decision,
      decided_at: parsed.decided_at || ''
    }
  } catch {
    return null
  }
}

export function hasAcceptedComplianceNotice(settings: PublicSettings | null | undefined): boolean {
  if (!isComplianceNoticeEnabled(settings)) return true
  const state = readComplianceNoticeState()
  return state?.revision === getComplianceNoticeRevision(settings) && state.decision === 'accepted'
}

export function hasDeclinedComplianceNotice(settings: PublicSettings | null | undefined): boolean {
  if (!isComplianceNoticeEnabled(settings)) return false
  const state = readComplianceNoticeState()
  return state?.revision === getComplianceNoticeRevision(settings) && state.decision === 'declined'
}

export function saveComplianceNoticeDecision(
  revision: string,
  decision: ComplianceNoticeDecision
): void {
  localStorage.setItem(
    COMPLIANCE_NOTICE_STORAGE_KEY,
    JSON.stringify({
      revision,
      decision,
      decided_at: new Date().toISOString()
    })
  )
  window.dispatchEvent(new CustomEvent(COMPLIANCE_NOTICE_EVENT, { detail: { revision, decision } }))
}

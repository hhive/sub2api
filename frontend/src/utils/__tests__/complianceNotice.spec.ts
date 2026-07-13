import { beforeEach, describe, expect, it } from 'vitest'
import type { PublicSettings } from '@/types'
import {
  COMPLIANCE_NOTICE_STORAGE_KEY,
  hasAcceptedComplianceNotice,
  hasDeclinedComplianceNotice,
  isComplianceNoticeEnabled,
  saveComplianceNoticeDecision
} from '../complianceNotice'

function settings(overrides: Partial<PublicSettings> = {}): PublicSettings {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    redeem_purchase_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    payment_enabled: false,
    risk_control_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [10, 20, 50, 100],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    wechat_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: '',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    channel_monitor_enabled: true,
    channel_monitor_default_interval_seconds: 60,
    available_channels_enabled: false,
    affiliate_enabled: false,
    lobehub_enabled: false,
    lobehub_menu_label: 'LobeHub',
    lobehub_launch_path: '/api/v1/lobehub/launch',
    compliance_notice_enabled: true,
    compliance_notice_revision: '2026-05-26',
    ...overrides
  }
}

describe('complianceNotice', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('requires a non-empty revision before enabling the notice', () => {
    expect(isComplianceNoticeEnabled(settings({ compliance_notice_revision: '' }))).toBe(false)
  })

  it('records accepted and declined decisions per revision', () => {
    const publicSettings = settings()

    saveComplianceNoticeDecision('2026-05-26', 'declined')

    expect(hasAcceptedComplianceNotice(publicSettings)).toBe(false)
    expect(hasDeclinedComplianceNotice(publicSettings)).toBe(true)

    saveComplianceNoticeDecision('2026-05-26', 'accepted')

    expect(hasAcceptedComplianceNotice(publicSettings)).toBe(true)
    expect(hasDeclinedComplianceNotice(publicSettings)).toBe(false)
  })

  it('ignores stale decisions when revision changes', () => {
    saveComplianceNoticeDecision('2026-05-26', 'accepted')

    expect(hasAcceptedComplianceNotice(settings({ compliance_notice_revision: '2026-05-27' }))).toBe(
      false
    )
  })

  it('ignores malformed local storage state', () => {
    localStorage.setItem(COMPLIANCE_NOTICE_STORAGE_KEY, '{bad-json')

    expect(hasAcceptedComplianceNotice(settings())).toBe(false)
    expect(hasDeclinedComplianceNotice(settings())).toBe(false)
  })
})

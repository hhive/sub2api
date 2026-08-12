import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'

const {
  getPublicSettingsMock,
  validateInvitationCodeMock,
  registerMock,
  showErrorMock,
  routeState,
} = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  validateInvitationCodeMock: vi.fn(),
  registerMock: vi.fn(),
  showErrorMock: vi.fn(),
  routeState: {
    query: {} as Record<string, string>,
  },
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
  useRoute: () => routeState,
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, string | number>) => {
      if (key === 'auth.signUpToStart') {
        return `Sign up to start with ${params?.siteName ?? 'Sub2API'}`
      }
      if (key === 'auth.emailDomainRegistrationLimit') {
        return '该邮箱域名无法注册新账户。请使用主流邮箱注册；如需使用企业邮箱，请联系客服添加域名白名单。'
      }
      return key
    },
    locale: { value: 'zh-CN' },
  }),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: (...args: any[]) => registerMock(...args),
  }),
  useAppStore: () => ({
    showError: (...args: any[]) => showErrorMock(...args),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
  isWeChatWebOAuthEnabled: () => false,
  validatePromoCode: vi.fn(),
  validateInvitationCode: (...args: any[]) => validateInvitationCodeMock(...args),
}))

const defaultPublicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: false,
  password_reset_enabled: true,
  invitation_code_enabled: true,
  affiliate_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: '小逆AI',
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
  table_page_size_options: [10, 20, 50],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  wechat_oauth_enabled: false,
  wechat_oauth_open_enabled: false,
  wechat_oauth_mp_enabled: false,
  wechat_oauth_mobile_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  backend_mode_enabled: false,
  version: '',
  balance_low_notify_enabled: false,
}

const publicSettings = defaultPublicSettings

function mountRegisterView() {
  return mount(RegisterView, {
    global: {
      stubs: {
        AuthLayout: {
          template: '<div><main><slot /></main><footer><slot name="footer" /></footer></div>',
        },
        Icon: true,
        TurnstileWidget: true,
        LoginAgreementPrompt: true,
        EmailOAuthButtons: true,
        LinuxDoOAuthSection: true,
        WechatOAuthSection: true,
        OidcOAuthSection: true,
        RouterLink: {
          props: ['to'],
          template: '<a><slot /></a>',
        },
      },
    },
  })
}

const mountRegister = mountRegisterView

describe('RegisterView', () => {
  beforeEach(() => {
    routeState.query = {}
    getPublicSettingsMock.mockReset()
    validateInvitationCodeMock.mockReset()
    registerMock.mockReset()
    showErrorMock.mockReset()
    getPublicSettingsMock.mockResolvedValue({ ...defaultPublicSettings })
    validateInvitationCodeMock.mockResolvedValue({ valid: true })
    registerMock.mockResolvedValue(undefined)
    sessionStorage.clear()
    localStorage.clear()
  })

  it('fills and validates invitation code from registration link query', async () => {
    routeState.query = { invitation_code: 'INVITE-123' }

    const wrapper = mountRegisterView()

    await flushPromises()

    const invitationInput = wrapper.get<HTMLInputElement>('#invitation_code')
    expect(invitationInput.element.value).toBe('INVITE-123')
    expect(validateInvitationCodeMock).toHaveBeenCalledWith('INVITE-123')
    expect(wrapper.text()).toContain('auth.invitationCodeValid')
  })

  it('shows affiliate link code in invitation input without submitting it as an invitation code', async () => {
    routeState.query = { aff: '4W52TTV5G5A3' }

    const wrapper = mountRegisterView()

    await flushPromises()

    const invitationInput = wrapper.get<HTMLInputElement>('#invitation_code')
    expect(invitationInput.element.value).toBe('4W52TTV5G5A3')
    expect(validateInvitationCodeMock).not.toHaveBeenCalled()

    await wrapper.get<HTMLInputElement>('#email').setValue('new-user@example.com')
    await wrapper.get<HTMLInputElement>('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({
        aff_code: '4W52TTV5G5A3',
        email: 'new-user@example.com',
        password: 'secret-123',
      })
    )
    expect(registerMock.mock.calls[0][0]).not.toHaveProperty('invitation_code')
  })

  it('keeps the optional affiliate invitation field before Turnstile', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...defaultPublicSettings,
      invitation_code_enabled: false,
      affiliate_enabled: true,
      turnstile_enabled: true,
      turnstile_site_key: 'site-key'
    })

    const wrapper = mountRegisterView()
    await flushPromises()

    const invitationField = wrapper.get('[data-testid="affiliate-invitation-field"]')
    const turnstile = wrapper.get('[data-testid="registration-turnstile"]')

    expect(invitationField.get('input').attributes('id')).toBe('affiliate_code')
    expect(invitationField.text()).toContain('common.optional')
    expect(
      invitationField.element.compareDocumentPosition(turnstile.element) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
  })

  it('uses the mandatory invitation field without duplicating the affiliate field', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...defaultPublicSettings,
      invitation_code_enabled: true,
      affiliate_enabled: true
    })

    const wrapper = mountRegisterView()
    await flushPromises()

    expect(wrapper.find('[data-testid="affiliate-invitation-field"]').exists()).toBe(false)
    expect(wrapper.get('#invitation_code').exists()).toBe(true)
  })

  it('submits a non-whitelist email domain so the backend can enforce its registration quota', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com'],
      registration_email_domain_quota_enabled: true
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('first@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({ email: 'first@custom.example' })
    )
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('shows the localized registration domain quota message returned by the backend', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com'],
      registration_email_domain_quota_enabled: true
    })
    registerMock.mockRejectedValueOnce({
      reason: 'EMAIL_DOMAIN_REGISTRATION_LIMIT',
      message: 'raw backend message'
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('second@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(showErrorMock).toHaveBeenCalledWith(
      '该邮箱域名无法注册新账户。请使用主流邮箱注册；如需使用企业邮箱，请联系客服添加域名白名单。'
    )
  })

  // 域名限量注册开关默认关闭：恢复 PR5423 之前的客户端白名单预检，非白名单域名不发起注册请求。
  it('rejects a non-whitelist email domain locally when the domain quota switch is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com']
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('first@custom.example')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).not.toHaveBeenCalled()
    // 校验失败通过 validationToastMessage watcher 弹 toast
    expect(showErrorMock).toHaveBeenCalledWith('auth.emailSuffixNotAllowedWithAllowed')
    expect(wrapper.get('#email').classes()).toContain('input-error')
  })

  it('still submits whitelisted email domains when the domain quota switch is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: false,
      registration_email_suffix_whitelist: ['allowed.com']
    })

    const wrapper = mountRegister()
    await flushPromises()
    await wrapper.get('#email').setValue('user@allowed.com')
    await wrapper.get('#password').setValue('secret-123')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({ email: 'user@allowed.com' })
    )
    expect(showErrorMock).not.toHaveBeenCalled()
  })
})

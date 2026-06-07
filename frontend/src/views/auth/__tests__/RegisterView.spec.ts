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
})

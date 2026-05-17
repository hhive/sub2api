import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'
import RegisterView from '@/views/auth/RegisterView.vue'

const { getPublicSettingsMock, showErrorMock, showWarningMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  showErrorMock: vi.fn(),
  showWarningMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
  }),
  useRoute: () => ({
    query: {},
  }),
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
    login: vi.fn(),
    login2FA: vi.fn(),
    register: vi.fn(),
  }),
  useAppStore: () => ({
    showError: (...args: any[]) => showErrorMock(...args),
    showWarning: (...args: any[]) => showWarningMock(...args),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/api/auth', () => {
  return {
    getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
    isTotp2FARequired: () => false,
    isWeChatWebOAuthEnabled: () => false,
    validatePromoCode: vi.fn(),
    validateInvitationCode: vi.fn(),
  }
})

const defaultPublicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: false,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  turnstile_enabled: false,
  turnstile_site_key: '',
  site_name: '小逆AI',
  site_logo: '',
  site_subtitle: '',
  api_base_url: '',
  contact_info: '2910703711',
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

function mountAuthView(component: typeof LoginView | typeof RegisterView) {
  return mount(component, {
    global: {
      stubs: {
        AuthLayout: {
          template: '<div><main><slot /></main><footer><slot name="footer" /></footer></div>',
        },
        Icon: true,
        TurnstileWidget: true,
        TotpLoginModal: true,
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

describe('auth contact footer', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    showErrorMock.mockReset()
    showWarningMock.mockReset()
    sessionStorage.clear()
    localStorage.clear()
    getPublicSettingsMock.mockResolvedValue({ ...defaultPublicSettings })
  })

  it('shows configured service QQ on the login footer', async () => {
    const wrapper = mountAuthView(LoginView)

    await flushPromises()

    expect(wrapper.text()).toContain('客服QQ：2910703711')
  })

  it('shows configured service QQ on the register footer', async () => {
    const wrapper = mountAuthView(RegisterView)

    await flushPromises()

    expect(wrapper.text()).toContain('客服QQ：2910703711')
  })
})

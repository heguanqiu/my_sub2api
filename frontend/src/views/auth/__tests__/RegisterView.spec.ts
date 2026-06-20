import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'

const {
  routeState,
  pushMock,
  registerMock,
  showErrorMock,
  showSuccessMock,
  showWarningMock,
  getPublicSettingsMock,
  validatePromoCodeMock,
  validateInvitationCodeMock,
} = vi.hoisted(() => ({
  routeState: {
    query: {} as Record<string, unknown>,
  },
  pushMock: vi.fn(),
  registerMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showWarningMock: vi.fn(),
  getPublicSettingsMock: vi.fn(),
  validatePromoCodeMock: vi.fn(),
  validateInvitationCodeMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    push: (...args: any[]) => pushMock(...args),
  }),
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'en' },
  }),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    register: (...args: any[]) => registerMock(...args),
  }),
  useAppStore: () => ({
    showError: (...args: any[]) => showErrorMock(...args),
    showSuccess: (...args: any[]) => showSuccessMock(...args),
    showWarning: (...args: any[]) => showWarningMock(...args),
  }),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: (...args: any[]) => getPublicSettingsMock(...args),
  isWeChatWebOAuthEnabled: () => false,
  validatePromoCode: (...args: any[]) => validatePromoCodeMock(...args),
  validateInvitationCode: (...args: any[]) => validateInvitationCodeMock(...args),
}))

describe('RegisterView', () => {
  beforeEach(() => {
    routeState.query = {}
    pushMock.mockReset()
    registerMock.mockReset()
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
    showWarningMock.mockReset()
    getPublicSettingsMock.mockReset()
    validatePromoCodeMock.mockReset()
    validateInvitationCodeMock.mockReset()
    sessionStorage.clear()
    localStorage.clear()

    getPublicSettingsMock.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: false,
      promo_code_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: 'Sub2API',
      linuxdo_oauth_enabled: false,
      oidc_oauth_enabled: false,
      github_oauth_enabled: false,
      google_oauth_enabled: false,
      registration_email_suffix_whitelist: [],
    })
    registerMock.mockResolvedValue({})
    validatePromoCodeMock.mockResolvedValue({ valid: true })
    validateInvitationCodeMock.mockResolvedValue({ valid: true })
  })

  it('submits the invitation code from a sales registration link', async () => {
    routeState.query = { invitation_code: 'SALES123' }

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
          EmailOAuthButtons: true,
          Icon: true,
          LinuxDoOAuthSection: true,
          LoginAgreementPrompt: true,
          OidcOAuthSection: true,
          RouterLink: true,
          TurnstileWidget: true,
          WechatOAuthSection: true,
          transition: false,
        },
      },
    })
    await flushPromises()

    await wrapper.find('input[type="email"]').setValue('customer@example.com')
    await wrapper.find('input[type="password"]').setValue('secret-123')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(registerMock).toHaveBeenCalledWith(
      expect.objectContaining({
        email: 'customer@example.com',
        password: 'secret-123',
        invitation_code: 'SALES123',
      })
    )
  })
})

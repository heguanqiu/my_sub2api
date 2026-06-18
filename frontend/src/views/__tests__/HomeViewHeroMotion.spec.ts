import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomeView from '@/views/HomeView.vue'

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false,
    isSales: false,
    checkAuth: vi.fn()
  }),
  useAppStore: () => ({
    cachedPublicSettings: {
      home_content: '',
      site_logo: '',
      doc_url: '',
      contact_info: '',
      contact_image_url: ''
    },
    siteLogo: '',
    docUrl: '',
    contactInfo: '',
    contactImageUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn()
  })
}))

describe('HomeView hero motion', () => {
  beforeEach(() => {
    window.scrollTo = vi.fn()
  })

  it('marks every hero geometry element for ambient motion', () => {
    const wrapper = mount(HomeView, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>'
          }
        }
      }
    })

    const stage = wrapper.find('.hero-geo-stage')
    const movingElements = stage.findAll('.hero-geo-motion')

    expect(stage.exists()).toBe(true)
    expect(stage.element.children).toHaveLength(7)
    expect(movingElements).toHaveLength(7)
  })
})

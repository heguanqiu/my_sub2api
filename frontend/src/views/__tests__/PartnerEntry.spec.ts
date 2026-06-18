import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import router from '@/router'
import HomeView from '@/views/HomeView.vue'
import TokenMerchantPartnerView from '@/views/public/TokenMerchantPartnerView.vue'

const push = vi.fn()

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

describe('Partner entry', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    push.mockReset()
    window.scrollTo = vi.fn()
  })

  it('adds a public partner route at /token-merchant', () => {
    const partnerRoute = router.getRoutes().find((route) => route.path === '/token-merchant')

    expect(partnerRoute?.name).toBe('TokenMerchantPartner')
    expect(partnerRoute?.meta.requiresAuth).toBe(false)
    expect(partnerRoute?.meta.title).toBe('Become Partner')
  })

  it('renders a home entry that navigates to the partner page', async () => {
    const testRouter = createRouter({
      history: createWebHistory(),
      routes: [
        { path: '/', redirect: '/home' },
        { path: '/home', component: HomeView },
        { path: '/token-merchant', component: defineComponent({ template: '<div>partner</div>' }) }
      ]
    })
    vi.spyOn(testRouter, 'push').mockImplementation(push)

    const wrapper = mount(HomeView, {
      global: {
        plugins: [testRouter],
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to.path" @click.prevent="$router.push(to)"><slot /></a>'
          }
        }
      }
    })

    const partnerLink = wrapper.find('[data-test="home-partner-entry"]')

    expect(partnerLink.exists()).toBe(true)
    expect(partnerLink.text()).toContain('成为合伙人')

    await partnerLink.trigger('click')

    expect(push).toHaveBeenCalledWith('/token-merchant')
  })

  it('aligns partner surfaces with the light home landing theme', () => {
    const homeWrapper = mount(HomeView, {
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>'
          }
        }
      }
    })
    const partnerWrapper = mount(TokenMerchantPartnerView, {
      global: {
        plugins: [
          createRouter({
            history: createWebHistory(),
            routes: [
              { path: '/', component: TokenMerchantPartnerView },
              { path: '/register', component: defineComponent({ template: '<div>register</div>' }) },
              { path: '/affiliate', component: defineComponent({ template: '<div>affiliate</div>' }) },
              { path: '/home', component: defineComponent({ template: '<div>home</div>' }) }
            ]
          })
        ]
      }
    })
    const themedHtml = `${homeWrapper.html()} ${partnerWrapper.html()}`

    expect(partnerWrapper.find('.partner-page').classes()).toEqual(
      expect.arrayContaining(['bg-white', 'text-[#0f1729]'])
    )
    expect(themedHtml).toContain('#1d6ff2')
    expect(themedHtml).toContain('#f7f9fc')
    expect(themedHtml).toContain('#e6e9ef')
    expect(themedHtml).not.toMatch(/#(?:070707|101010)/i)
  })

  it('uses home-style reveal motion, multi-color icon tones, and an animated svg hero illustration', () => {
    const partnerWrapper = mount(TokenMerchantPartnerView, {
      global: {
        plugins: [
          createRouter({
            history: createWebHistory(),
            routes: [
              { path: '/', component: TokenMerchantPartnerView },
              { path: '/register', component: defineComponent({ template: '<div>register</div>' }) },
              { path: '/affiliate', component: defineComponent({ template: '<div>affiliate</div>' }) },
              { path: '/home', component: defineComponent({ template: '<div>home</div>' }) }
            ]
          })
        ]
      }
    })

    const heroVisual = partnerWrapper.find('.partner-hero-visual')
    const toneStyles = partnerWrapper
      .findAll('.partner-tone-icon')
      .map((icon) => icon.attributes('style'))
      .filter(Boolean)

    expect(partnerWrapper.findAll('.reveal-init').length).toBeGreaterThan(8)
    expect(new Set(toneStyles).size).toBeGreaterThan(3)
    expect(heroVisual.find('img').exists()).toBe(false)
    expect(heroVisual.find('svg').exists()).toBe(true)
    expect(heroVisual.findAll('.partner-visual-motion').length).toBeGreaterThan(5)
  })
})

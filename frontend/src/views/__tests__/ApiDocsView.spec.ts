import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ApiDocsView from '../ApiDocsView.vue'

const { fetchPublicSettings, publicSettings } = vi.hoisted(() => ({
  fetchPublicSettings: vi.fn(),
  publicSettings: {
    site_name: 'Example Gateway',
    api_base_url: 'https://api.example.test/'
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        key === 'apiDocs.title' ? `${params?.siteName} API Documentation` : key
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: publicSettings,
    siteName: 'Sub2API',
    apiBaseUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false
  })
}))

describe('ApiDocsView', () => {
  beforeEach(() => {
    fetchPublicSettings.mockReset()
    publicSettings.api_base_url = 'https://api.example.test/'
  })

  function mountView() {
    return mount(ApiDocsView, {
      global: {
        stubs: {
          PublicSiteNav: true,
          RouterLink: { template: '<a><slot /></a>' },
          Icon: true
        }
      }
    })
  }

  it('uses the configured site name and normalized API base URL in examples', () => {
    const wrapper = mountView()

    expect(wrapper.text()).toContain('Example Gateway API Documentation')
    expect(wrapper.text()).toContain('https://api.example.test')
    expect(wrapper.find('pre').text()).toContain('https://api.example.test/v1/responses')
    expect(wrapper.find('pre').text()).not.toContain('https://api.example.test//v1')
  })

  it('does not duplicate the version path when the configured base URL already contains v1', () => {
    publicSettings.api_base_url = 'https://api.example.test/v1/'
    const wrapper = mountView()

    expect(wrapper.text()).toContain('https://api.example.test/v1')
    expect(wrapper.find('pre').text()).toContain('https://api.example.test/v1/responses')
    expect(wrapper.find('pre').text()).not.toContain('/v1/v1/')
  })

  it('switches the first-request example between supported protocols', async () => {
    const wrapper = mountView()
    const protocolTabs = wrapper.findAll('[role="tab"]').slice(0, 4)

    await protocolTabs[1].trigger('click')
    expect(wrapper.find('pre').text()).toContain('/v1/chat/completions')
    expect(wrapper.find('pre').text()).toContain('Authorization: Bearer')

    await protocolTabs[3].trigger('click')
    expect(wrapper.find('pre').text()).toContain('/v1beta/models/your-model-id:generateContent')
    expect(wrapper.find('pre').text()).toContain('x-goog-api-key')
  })
})

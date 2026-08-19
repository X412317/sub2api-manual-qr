import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelPlazaContent from '../ModelPlazaContent.vue'
import PlazaFilterBar from '../PlazaFilterBar.vue'
import type { ModelPlazaResponse } from '@/api/modelPlaza'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isAuthenticated: false })
}))

const response: ModelPlazaResponse = {
  description: '',
  groups: [
    {
      id: 1,
      name: 'OpenAI Standard',
      description: '',
      platform: 'openai',
      subscription_type: 'standard',
      rate_multiplier: 0.8,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      is_exclusive: false,
      models: [
        { name: 'gpt-alpha', platform: 'openai', pricing: null, official_pricing: null },
        { name: 'shared-model', platform: 'openai', pricing: null, official_pricing: null }
      ]
    },
    {
      id: 2,
      name: 'Claude Standard',
      description: '',
      platform: 'anthropic',
      subscription_type: 'standard',
      rate_multiplier: 1,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      is_exclusive: false,
      models: [
        { name: 'claude-beta', platform: 'anthropic', pricing: null, official_pricing: null },
        { name: 'shared-model', platform: 'anthropic', pricing: null, official_pricing: null }
      ]
    }
  ]
}

function mountContent(props: { response: ModelPlazaResponse | null; loading: boolean; error?: boolean }) {
  return mount(ModelPlazaContent, {
    props,
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        Icon: true,
        PlazaGroupSection: { template: '<div class="group-stub" />' }
      }
    }
  })
}

describe('ModelPlazaContent', () => {
  it('summarizes unique models, groups, platforms, and the lowest rate', () => {
    const wrapper = mountContent({ response, loading: false })
    const stats = wrapper.get('[data-testid="plaza-stats"]').text()

    expect(stats).toContain('modelPlaza.stats.models')
    expect(stats).toContain('3')
    expect(stats).toContain('modelPlaza.stats.groups')
    expect(stats).toContain('2')
    expect(stats).toContain('modelPlaza.stats.platforms')
    expect(stats).toContain('0.8x')
  })

  it('updates result counts when searching and resets all filters', async () => {
    const wrapper = mountContent({ response, loading: false })
    const filter = wrapper.findComponent(PlazaFilterBar)

    filter.vm.$emit('update:search', 'claude')
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="plaza-result-summary"]').text()).toContain('"groups":1')
    expect(wrapper.get('[data-testid="plaza-result-summary"]').text()).toContain('"models":1')

    await wrapper.get('[data-testid="plaza-reset-filters"]').trigger('click')
    expect(wrapper.get('[data-testid="plaza-result-summary"]').text()).toContain('"groups":2')
    expect(wrapper.get('[data-testid="plaza-result-summary"]').text()).toContain('"models":4')
  })

  it('emits retry from the error state', async () => {
    const wrapper = mountContent({ response: null, loading: false, error: true })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })
})

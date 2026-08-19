import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AmountInput from '@/components/payment/AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AmountInput', () => {
  it('shows only currency-labelled tier buttons in fixed mode', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 50,
        amounts: [10, 20, 50, 100],
        fixedOnly: true,
        currencySymbol: '¥',
      },
    })

    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.findAll('button').map(button => button.text())).toEqual([
      '¥10',
      '¥20',
      '¥50',
      '¥100',
    ])

    await wrapper.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[20]])
  })

  it('keeps quick amounts and the custom input in standard mode', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10, 20],
        fixedOnly: false,
      },
    })

    expect(wrapper.findAll('button').map(button => button.text())).toEqual(['10', '20'])
    expect(wrapper.find('input').exists()).toBe(true)
  })
})

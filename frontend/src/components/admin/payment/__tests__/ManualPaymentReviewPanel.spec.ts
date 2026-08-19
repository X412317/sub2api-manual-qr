import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import ManualPaymentReviewPanel from '@/components/admin/payment/ManualPaymentReviewPanel.vue'
import type { PaymentOrder } from '@/types/payment'

const getManualProof = vi.hoisted(() => vi.fn())
const getManualProofImage = vi.hoisted(() => vi.fn())
const reviewManualProof = vi.hoisted(() => vi.fn())
const stepUpRun = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin/payment', () => {
  const api = { getManualProof, getManualProofImage, reviewManualProof }
  return { adminPaymentAPI: api, default: api }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: stepUpRun }),
  isStepUpBlocked: () => false,
  isStepUpCancelled: () => false,
  stepUpBlockReason: () => '',
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh-CN' } }),
  }
})

const order: PaymentOrder = {
  id: 101,
  user_id: 7,
  amount: 10,
  pay_amount: 10,
  currency: 'CNY',
  fee_rate: 0,
  payment_type: 'alipay',
  out_trade_no: 'sub2_manual_101',
  status: 'PENDING',
  order_type: 'balance',
  created_at: '2026-08-03T10:00:00Z',
  expires_at: '2026-08-03T10:30:00Z',
  refund_amount: 0,
  provider_key: 'manual_qr',
}

describe('ManualPaymentReviewPanel', () => {
  beforeEach(() => {
    getManualProof.mockReset().mockResolvedValue({
      data: {
        order_id: 101,
        submission_no: 1,
        channel: 'alipay',
        transaction_no: '20260803123456',
        status: 'submitted',
        has_image: true,
        submitted_at: '2026-08-03T10:05:00Z',
        attempts_used: 1,
        attempts_remaining: 2,
        can_submit: false,
      },
    })
    getManualProofImage.mockReset().mockResolvedValue({ data: new Blob(['proof'], { type: 'image/png' }) })
    reviewManualProof.mockReset().mockResolvedValue({ data: { status: 'approved' } })
    stepUpRun.mockReset().mockImplementation(async (request: () => Promise<unknown>) => request())
    showError.mockReset()
    showSuccess.mockReset()
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:manual-proof') })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })
  })

  it('routes approval through TOTP step-up and sends the exact expected amount', async () => {
    const wrapper = shallowMount(ManualPaymentReviewPanel, { props: { order } })
    await flushPromises()
    await flushPromises()

    expect(getManualProofImage).toHaveBeenCalledWith(101)
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(reviewManualProof).toHaveBeenCalledWith(101, {
      action: 'approve',
      received_amount: '10.00',
      reason: undefined,
    })
    expect(wrapper.emitted('reviewed')).toHaveLength(1)
    wrapper.unmount()
  })
})

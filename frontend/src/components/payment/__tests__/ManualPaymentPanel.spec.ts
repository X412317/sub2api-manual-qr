import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import ManualPaymentPanel from '@/components/payment/ManualPaymentPanel.vue'

const getManualQR = vi.hoisted(() => vi.fn())
const getManualProof = vi.hoisted(() => vi.fn())
const getOrder = vi.hoisted(() => vi.fn())
const submitManualProof = vi.hoisted(() => vi.fn())
const cancelOrder = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('@/api/payment', () => ({
  paymentAPI: { getManualQR, getManualProof, getOrder, submitManualProof, cancelOrder },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' },
    }),
  }
})

function proofFixture(overrides: Record<string, unknown> = {}) {
  return {
    order_id: 101,
    submission_no: 1,
    channel: 'alipay',
    transaction_no: '20260803123456',
    status: 'submitted',
    mime_type: 'image/png',
    file_size: 100,
    sha256: 'hash',
    has_image: true,
    attempts_used: 1,
    attempts_remaining: 2,
    can_submit: false,
    alipay_launch_payload: 'https://qr.alipay.com/manual-test',
    ...overrides,
  }
}

function orderFixture(overrides: Record<string, unknown> = {}) {
  return {
    id: 101,
    status: 'PENDING',
    expires_at: '2099-01-01T00:10:00.000Z',
    ...overrides,
  }
}

function mountPanel() {
  return shallowMount(ManualPaymentPanel, {
    props: {
      orderId: 101,
      payAmount: 50,
      currency: 'CNY',
      paymentType: 'alipay',
      outTradeNo: 'sub2_manual_101',
      expiresAt: '2024-01-01T00:10:00.000Z',
    },
  })
}

describe('ManualPaymentPanel', () => {
  beforeEach(() => {
    getManualQR.mockReset().mockResolvedValue({ data: new Blob(['qr'], { type: 'image/png' }) })
    getManualProof.mockReset()
    getOrder.mockReset()
    submitManualProof.mockReset()
    cancelOrder.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:manual-qr') })
    Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('suppresses expiry and cancellation while a proof is awaiting review', async () => {
    getManualProof.mockResolvedValue({ data: proofFixture() })
    getOrder.mockResolvedValue({ data: orderFixture({ expires_at: '2024-01-01T00:10:00.000Z' }) })

    const wrapper = mountPanel()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('payment.manual.awaitingReview')
    expect(wrapper.text()).toContain('payment.manual.reviewNoExpiry')
    expect(wrapper.text()).not.toContain('payment.qr.cancelOrder')
    wrapper.unmount()
  })

  it('shows the rejection reason and permits a new proof within the reset timeout', async () => {
    getManualProof.mockResolvedValue({
      data: proofFixture({
        status: 'rejected',
        rejection_reason: 'amount is unreadable',
        can_submit: true,
      }),
    })
    getOrder.mockResolvedValue({ data: orderFixture() })

    const wrapper = mountPanel()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('amount is unreadable')
    expect(wrapper.find('#manual-transaction-no').exists()).toBe(true)
    expect(wrapper.find('#manual-proof-file').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('payment.qr.cancelOrder')
    wrapper.unmount()
  })

  it('emits success only after the order reaches COMPLETED', async () => {
    getManualProof.mockResolvedValue({ data: proofFixture({ status: 'approved' }) })
    getOrder.mockResolvedValueOnce({ data: orderFixture({ status: 'PENDING' }) })

    const pendingWrapper = mountPanel()
    await flushPromises()
    await flushPromises()
    expect(pendingWrapper.emitted('success')).toBeUndefined()
    pendingWrapper.unmount()

    getOrder.mockResolvedValueOnce({ data: orderFixture({ status: 'COMPLETED' }) })
    const completedWrapper = mountPanel()
    await flushPromises()
    await flushPromises()
    expect(completedWrapper.emitted('success')).toHaveLength(1)
    expect(completedWrapper.emitted('settled')).toHaveLength(1)
    completedWrapper.unmount()
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, unknown>,
}))
const routerPush = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const pollOrderStatus = vi.hoisted(() => vi.fn())
const cancelOrder = vi.hoisted(() => vi.fn())
const verifyOrder = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const toCanvas = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({ push: routerPush }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({ pollOrderStatus }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    cancelOrder,
    verifyOrder,
  },
}))

vi.mock('qrcode', () => ({
  default: { toCanvas },
}))

import PaymentQRCodeView from '../PaymentQRCodeView.vue'

const baseOrder = {
  id: 42,
  user_id: 9,
  amount: 88,
  pay_amount: 88,
  fee_rate: 0,
  payment_type: 'alipay',
  out_trade_no: 'sub2_qr_42',
  status: 'PENDING',
  order_type: 'balance',
  created_at: '2026-06-25T10:00:00Z',
  expires_at: '2026-06-25T10:30:00Z',
  refund_amount: 0,
}

function mountView() {
  return mount(PaymentQRCodeView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
      },
    },
  })
}

describe('PaymentQRCodeView active recovery', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    routeState.query = {
      order_id: '42',
      qr: '',
      payment_type: 'alipay',
      expires_at: '2099-01-01T00:30:00Z',
    }
    routerPush.mockReset().mockResolvedValue(undefined)
    pollOrderStatus.mockReset()
    cancelOrder.mockReset()
    verifyOrder.mockReset()
    showError.mockReset()
    toCanvas.mockReset().mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('actively verifies an EasyPay order after its expiry timestamp', async () => {
    vi.setSystemTime(new Date('2026-06-25T11:00:00Z'))
    pollOrderStatus.mockResolvedValue({
      ...baseOrder,
      provider_key: 'easypay',
    })
    verifyOrder.mockResolvedValue({
      data: {
        ...baseOrder,
        status: 'COMPLETED',
        provider_key: 'easypay',
      },
    })
    routeState.query = {
      ...routeState.query,
      expires_at: '2026-06-25T10:30:00Z',
    }

    const wrapper = mountView()
    await flushPromises()

    expect(pollOrderStatus).toHaveBeenCalledWith(42)
    expect(verifyOrder).toHaveBeenCalledWith('sub2_qr_42')
    expect(routerPush).toHaveBeenCalledWith({
      path: '/payment/result',
      query: { order_id: '42', status: 'success' },
    })
    expect(wrapper.text()).not.toContain('payment.qr.expired')
    wrapper.unmount()
  })

  it('actively verifies a pending WeChat order during normal polling', async () => {
    pollOrderStatus.mockResolvedValue({
      ...baseOrder,
      payment_type: 'wxpay',
      provider_key: 'wxpay',
    })
    verifyOrder.mockResolvedValue({
      data: {
        ...baseOrder,
        payment_type: 'wxpay',
        provider_key: 'wxpay',
        status: 'PAID',
      },
    })
    routeState.query = {
      ...routeState.query,
      payment_type: 'wxpay',
    }

    const wrapper = mountView()
    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(verifyOrder).toHaveBeenCalledWith('sub2_qr_42')
    expect(routerPush).toHaveBeenCalledWith({
      path: '/payment/result',
      query: { order_id: '42', status: 'success' },
    })
    wrapper.unmount()
  })

  it('does not actively verify a manual QR order', async () => {
    pollOrderStatus.mockResolvedValue({
      ...baseOrder,
      payment_type: 'wxpay',
      provider_key: 'manual_qr',
    })
    routeState.query = {
      ...routeState.query,
      payment_type: 'wxpay',
      provider_key: 'manual_qr',
    }

    const wrapper = mountView()
    await vi.advanceTimersByTimeAsync(3000)
    await flushPromises()

    expect(pollOrderStatus).toHaveBeenCalledWith(42)
    expect(verifyOrder).not.toHaveBeenCalled()
    expect(routerPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

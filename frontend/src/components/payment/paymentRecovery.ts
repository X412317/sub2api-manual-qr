import { isBuiltInAlipayMethod, isBuiltInWxpayMethod } from './providerConfig'

interface ActiveRecoveryContext {
  orderProviderKey?: string
  checkoutProviderKey?: string
  paymentType: string
  mobileAlipayDeepLink?: boolean
}

function normalizeProviderKey(value: string | undefined): string {
  return String(value || '').trim().toLowerCase()
}

export function isRecoverablePaymentStatus(status: string | null | undefined): boolean {
  const normalized = String(status || '').trim().toUpperCase()
  return normalized === 'PENDING' || normalized === 'EXPIRED'
}

export function shouldActivelyRecoverPayment(context: ActiveRecoveryContext): boolean {
  const paymentType = normalizeProviderKey(context.paymentType)
  const providerKeys = [
    normalizeProviderKey(context.orderProviderKey),
    normalizeProviderKey(context.checkoutProviderKey),
  ].filter(Boolean)

  // Manual QR orders are settled only after administrator proof review.
  if (providerKeys.includes('manual_qr') || paymentType === 'manual_qr') {
    return false
  }
  if (providerKeys.includes('easypay') || paymentType === 'easypay') {
    return true
  }
  if (isBuiltInWxpayMethod(paymentType)) {
    return true
  }
  return context.mobileAlipayDeepLink === true && isBuiltInAlipayMethod(paymentType)
}

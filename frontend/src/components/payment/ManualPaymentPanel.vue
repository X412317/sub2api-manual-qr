<template>
  <div class="space-y-4">
    <section class="card p-5 sm:p-6">
      <div class="flex flex-wrap items-start justify-between gap-4 border-b border-gray-100 pb-4 dark:border-dark-700">
        <div>
          <div class="flex items-center gap-2">
            <span
              class="inline-flex h-9 w-9 items-center justify-center rounded-lg"
              :class="channel === 'alipay' ? 'bg-blue-50 dark:bg-blue-950' : 'bg-green-50 dark:bg-green-950'"
            >
              <img :src="channelIcon" alt="" class="h-6 w-6" />
            </span>
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('payment.manual.title') }}
              </h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t(`payment.methods.${channel}`) }}</p>
            </div>
          </div>
        </div>
        <span :class="statusClass" class="rounded-full px-2.5 py-1 text-xs font-medium">
          {{ statusLabel }}
        </span>
      </div>

      <dl class="mt-4 grid grid-cols-1 gap-3 text-sm sm:grid-cols-3">
        <div>
          <dt class="text-xs text-gray-400">{{ t('payment.paymentAmount') }}</dt>
          <dd class="mt-0.5 text-xl font-bold text-gray-900 dark:text-white">{{ formattedPayAmount }}</dd>
        </div>
        <div class="min-w-0">
          <dt class="text-xs text-gray-400">{{ t('payment.orders.orderNo') }}</dt>
          <dd class="mt-0.5 truncate font-mono text-sm text-gray-800 dark:text-gray-200">{{ outTradeNo }}</dd>
        </div>
        <div>
          <dt class="text-xs text-gray-400">{{ t('payment.qr.expiresIn') }}</dt>
          <dd class="mt-0.5 font-mono text-sm font-semibold" :class="countdownClass">{{ countdownLabel }}</dd>
        </div>
      </dl>
    </section>

    <section v-if="!isCompleted" class="card p-5 sm:p-6">
      <div v-if="loading" class="flex min-h-72 items-center justify-center">
        <span class="h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
      </div>

      <div v-else-if="expired" class="py-8 text-center">
        <Icon name="clock" size="xl" class="mx-auto text-amber-500" />
        <h3 class="mt-3 font-semibold text-gray-900 dark:text-white">{{ t('payment.qr.expired') }}</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('payment.qr.expiredDesc') }}</p>
        <button type="button" class="btn btn-secondary mt-5" @click="emit('done')">
          {{ t('payment.result.backToRecharge') }}
        </button>
      </div>

      <template v-else>
        <div
          v-if="proof.status === 'submitted'"
          class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800/60 dark:bg-amber-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="clock" size="md" class="mt-0.5 shrink-0 text-amber-600 dark:text-amber-400" />
            <div>
              <p class="font-medium text-amber-900 dark:text-amber-100">{{ t('payment.manual.awaitingReview') }}</p>
              <p class="mt-1 text-sm text-amber-700 dark:text-amber-300">{{ t('payment.manual.awaitingReviewHint') }}</p>
              <p class="mt-2 font-mono text-xs text-amber-700 dark:text-amber-300">
                {{ t('payment.manual.transactionNo') }}: {{ proof.transaction_no }}
              </p>
            </div>
          </div>
        </div>

        <div
          v-else-if="proof.status === 'approved'"
          class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800/60 dark:bg-green-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="checkCircle" size="md" class="mt-0.5 shrink-0 text-green-600 dark:text-green-400" />
            <div>
              <p class="font-medium text-green-900 dark:text-green-100">{{ t('payment.manual.approved') }}</p>
              <p class="mt-1 text-sm text-green-700 dark:text-green-300">{{ t('payment.manual.approvedProcessing') }}</p>
            </div>
          </div>
        </div>

        <div
          v-else-if="proof.status === 'rejected'"
          class="mb-5 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800/60 dark:bg-red-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="mt-0.5 shrink-0 text-red-600 dark:text-red-400" />
            <div class="min-w-0">
              <p class="font-medium text-red-900 dark:text-red-100">{{ t('payment.manual.rejected') }}</p>
              <p class="mt-1 whitespace-pre-wrap break-words text-sm text-red-700 dark:text-red-300">{{ proof.rejection_reason }}</p>
              <p class="mt-2 text-xs text-red-600 dark:text-red-400">
                {{ t('payment.manual.attemptsRemaining', { count: proof.attempts_remaining }) }}
              </p>
            </div>
          </div>
        </div>

        <div v-if="proof.status !== 'submitted' && proof.status !== 'approved'" class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.85fr)]">
          <div>
            <div class="mx-auto max-w-sm">
              <div v-if="qrURL && (showQRCode || channel === 'wxpay')" class="aspect-square overflow-hidden rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600">
                <img :src="qrURL" :alt="t('payment.manual.qrAlt')" class="h-full w-full object-contain" />
              </div>
              <div v-else class="flex aspect-square items-center justify-center rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-center">
                  <span class="mx-auto block h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
                  <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">{{ t('payment.manual.openingAlipay') }}</p>
                </div>
              </div>

              <div class="mt-3 flex gap-2">
                <button
                  v-if="channel === 'alipay' && alipayLaunchURL"
                  type="button"
                  class="btn btn-alipay flex-1"
                  @click="launchAlipay"
                >
                  <Icon name="externalLink" size="sm" />
                  {{ t('payment.manual.openAlipay') }}
                </button>
                <button type="button" class="btn btn-secondary flex-1" :disabled="!qrURL" @click="saveQRCode">
                  <Icon name="download" size="sm" />
                  {{ t('payment.qr.saveQRCode') }}
                </button>
              </div>
              <p class="mt-2 text-center text-xs text-gray-500 dark:text-gray-400">
                {{ channel === 'wxpay' ? t('payment.manual.wechatSaveHint') : t('payment.manual.alipayFallbackHint') }}
              </p>
            </div>
          </div>

          <form class="space-y-4" @submit.prevent="submitProof">
            <div>
              <label class="input-label" for="manual-transaction-no">{{ t('payment.manual.transactionNo') }}</label>
              <input
                id="manual-transaction-no"
                v-model.trim="transactionNo"
                type="text"
                class="input font-mono"
                maxlength="128"
                autocomplete="off"
                :placeholder="t('payment.manual.transactionNoPlaceholder')"
              />
            </div>

            <div>
              <label class="input-label" for="manual-proof-file">{{ t('payment.manual.proofImage') }}</label>
              <label
                for="manual-proof-file"
                class="flex min-h-28 cursor-pointer flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 text-center transition-colors hover:border-primary-400 hover:bg-primary-50/40 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-600"
              >
                <Icon name="upload" size="lg" class="text-gray-400" />
                <span class="mt-2 max-w-full truncate text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ proofFile?.name || t('payment.manual.chooseProof') }}
                </span>
                <span class="mt-1 text-xs text-gray-400">{{ t('payment.manual.proofRules') }}</span>
              </label>
              <input
                id="manual-proof-file"
                type="file"
                class="sr-only"
                accept="image/png,image/jpeg,image/webp"
                @change="selectProofFile"
              />
            </div>

            <p v-if="formError" class="text-sm text-red-600 dark:text-red-400">{{ formError }}</p>

            <button type="submit" class="btn btn-primary w-full" :disabled="!canSubmit || submitting">
              <span v-if="submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              <Icon v-else name="upload" size="sm" />
              {{ submitting ? t('common.processing') : t('payment.manual.submitProof') }}
            </button>
            <p class="text-center text-xs text-gray-400">
              {{ t('payment.manual.attemptsRemaining', { count: proof.attempts_remaining }) }}
            </p>
          </form>
        </div>
      </template>
    </section>

    <button
      v-if="proof.status === 'not_submitted' && !expired"
      type="button"
      class="btn btn-secondary w-full"
      :disabled="cancelling"
      @click="cancelOrder"
    >
      {{ cancelling ? t('common.processing') : t('payment.qr.cancelOrder') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatPaymentAmount } from './currency'
import Icon from '@/components/icons/Icon.vue'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import type { ManualPaymentProof, PaymentOrder } from '@/types/payment'

const props = defineProps<{
  orderId: number
  payAmount: number
  currency?: string
  paymentType: string
  outTradeNo: string
  expiresAt: string
}>()

const emit = defineEmits<{
  done: []
  success: []
  settled: []
}>()

const { t, locale } = useI18n()
const appStore = useAppStore()
const channel = computed<'alipay' | 'wxpay'>(() => props.paymentType === 'wxpay' ? 'wxpay' : 'alipay')
const channelIcon = computed(() => channel.value === 'alipay' ? alipayIcon : wxpayIcon)
const formattedPayAmount = computed(() => formatPaymentAmount(props.payAmount, props.currency || 'CNY', locale.value))
const loading = ref(true)
const submitting = ref(false)
const cancelling = ref(false)
const transactionNo = ref('')
const proofFile = ref<File | null>(null)
const formError = ref('')
const qrURL = ref('')
const qrBlob = ref<Blob | null>(null)
const showQRCode = ref(true)
const currentExpiresAt = ref(props.expiresAt)
const orderStatus = ref('PENDING')
const now = ref(Date.now())
const successEmitted = ref(false)
const proof = ref<ManualPaymentProof>({
  order_id: props.orderId,
  channel: channel.value,
  status: 'not_submitted',
  attempts_used: 0,
  attempts_remaining: 3,
  can_submit: true,
})
let refreshTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
let alipayFallbackTimer: ReturnType<typeof setTimeout> | null = null

const alipayLaunchURL = computed(() => buildAlipayLaunchURL(proof.value.alipay_launch_payload || ''))
const isCompleted = computed(() => orderStatus.value === 'COMPLETED')
const isReviewing = computed(() => proof.value.status === 'submitted' || proof.value.status === 'approved')
const expired = computed(() => {
  if (isReviewing.value || isCompleted.value) return false
  if (['EXPIRED', 'CANCELLED'].includes(orderStatus.value)) return true
  const timestamp = Date.parse(currentExpiresAt.value)
  return Number.isFinite(timestamp) && timestamp <= now.value
})

const remainingSeconds = computed(() => {
  const timestamp = Date.parse(currentExpiresAt.value)
  if (!Number.isFinite(timestamp)) return 0
  return Math.max(0, Math.ceil((timestamp - now.value) / 1000))
})

const countdownLabel = computed(() => {
  if (proof.value.status === 'submitted') return t('payment.manual.reviewNoExpiry')
  if (proof.value.status === 'approved') return t('payment.manual.crediting')
  if (expired.value) return t('payment.qr.expired')
  const minutes = Math.floor(remainingSeconds.value / 60)
  const seconds = remainingSeconds.value % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
})

const countdownClass = computed(() => expired.value
  ? 'text-red-600 dark:text-red-400'
  : isReviewing.value
    ? 'text-amber-600 dark:text-amber-400'
    : 'text-gray-800 dark:text-gray-200')

const statusLabel = computed(() => {
  if (isCompleted.value) return t('payment.status.completed')
  if (proof.value.status === 'submitted') return t('payment.manual.statusSubmitted')
  if (proof.value.status === 'approved') return t('payment.manual.statusApproved')
  if (proof.value.status === 'rejected') return t('payment.manual.statusRejected')
  if (expired.value) return t('payment.status.expired')
  return t('payment.status.pending')
})

const statusClass = computed(() => {
  if (isCompleted.value || proof.value.status === 'approved') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (proof.value.status === 'submitted') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (proof.value.status === 'rejected' || expired.value) return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
})

const canSubmit = computed(() => proof.value.can_submit
  && proof.value.attempts_remaining > 0
  && /^[A-Za-z0-9_-]{6,128}$/.test(transactionNo.value)
  && !!proofFile.value
  && !expired.value)

function buildAlipayLaunchURL(payload: string): string {
  const value = payload.trim()
  if (!value) return ''
  if (/^alipays?:\/\/platformapi/i.test(value)) return value
  if (/^https:\/\/qr\.alipay\.com\//i.test(value)) {
    return `alipays://platformapi/startapp?saId=10000007&qrcode=${encodeURIComponent(value)}`
  }
  return ''
}

function isMobileBrowser(): boolean {
  return /Android|iPhone|iPad|iPod|Mobile/i.test(window.navigator.userAgent)
}

function launchAlipay() {
  if (!alipayLaunchURL.value) {
    showQRCode.value = true
    return
  }
  showQRCode.value = false
  window.location.href = alipayLaunchURL.value
  if (alipayFallbackTimer) clearTimeout(alipayFallbackTimer)
  alipayFallbackTimer = setTimeout(() => { showQRCode.value = true }, 1400)
}

async function loadQRCode() {
  const response = await paymentAPI.getManualQR(props.orderId)
  qrBlob.value = response.data
  if (qrURL.value) URL.revokeObjectURL(qrURL.value)
  qrURL.value = URL.createObjectURL(response.data)
}

async function refreshStatus(showError = false) {
  try {
    const [proofResponse, orderResponse] = await Promise.all([
      paymentAPI.getManualProof(props.orderId),
      paymentAPI.getOrder(props.orderId),
    ])
    const previousStatus = proof.value.status
    proof.value = proofResponse.data
    const order = orderResponse.data as PaymentOrder
    orderStatus.value = order.status
    currentExpiresAt.value = order.expires_at || currentExpiresAt.value
    if (previousStatus !== 'rejected' && proof.value.status === 'rejected') {
      transactionNo.value = ''
      proofFile.value = null
    }
    if (order.status === 'COMPLETED' && !successEmitted.value) {
      successEmitted.value = true
      emit('settled')
      emit('success')
    }
  } catch (error: unknown) {
    if (showError) {
      appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
    }
  }
}

async function initialize() {
  loading.value = true
  try {
    await Promise.all([loadQRCode(), refreshStatus(true)])
    if (channel.value === 'alipay' && alipayLaunchURL.value && isMobileBrowser() && proof.value.status === 'not_submitted') {
      launchAlipay()
    }
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function selectProofFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] || null
  input.value = ''
  formError.value = ''
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    formError.value = t('payment.manual.invalidProofType')
    proofFile.value = null
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    formError.value = t('payment.manual.proofTooLarge')
    proofFile.value = null
    return
  }
  proofFile.value = file
}

async function submitProof() {
  if (!canSubmit.value || !proofFile.value || submitting.value) return
  submitting.value = true
  formError.value = ''
  try {
    const response = await paymentAPI.submitManualProof(props.orderId, transactionNo.value, proofFile.value)
    proof.value = response.data
    proofFile.value = null
    appStore.showSuccess(t('payment.manual.submitSuccess'))
    await refreshStatus()
  } catch (error: unknown) {
    formError.value = extractI18nErrorMessage(error, t, 'payment.errors', t('payment.manual.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function saveQRCode() {
  if (!qrURL.value || !qrBlob.value) return
  const ext = qrBlob.value.type === 'image/jpeg' ? 'jpg' : qrBlob.value.type === 'image/webp' ? 'webp' : 'png'
  const anchor = document.createElement('a')
  anchor.href = qrURL.value
  anchor.download = `${channel.value}-${props.outTradeNo || props.orderId}.${ext}`
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

async function cancelOrder() {
  if (cancelling.value || proof.value.status !== 'not_submitted') return
  cancelling.value = true
  try {
    await paymentAPI.cancelOrder(props.orderId)
    emit('settled')
    emit('done')
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    cancelling.value = false
  }
}

onMounted(() => {
  initialize()
  countdownTimer = setInterval(() => { now.value = Date.now() }, 1000)
  refreshTimer = setInterval(() => { void refreshStatus() }, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  if (countdownTimer) clearInterval(countdownTimer)
  if (alipayFallbackTimer) clearTimeout(alipayFallbackTimer)
  if (qrURL.value) URL.revokeObjectURL(qrURL.value)
})
</script>

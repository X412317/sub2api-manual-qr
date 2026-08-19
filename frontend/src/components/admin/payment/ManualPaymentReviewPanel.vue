<template>
  <section class="space-y-4 border-t border-gray-200 pt-4 dark:border-dark-700">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('payment.admin.manualReview.title') }}</h4>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.manualReview.privateHint') }}</p>
      </div>
      <span v-if="proof" :class="proofStatusClass" class="rounded-full px-2.5 py-1 text-xs font-medium">
        {{ proofStatusLabel }}
      </span>
    </div>

    <div v-if="loading" class="flex min-h-48 items-center justify-center">
      <span class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
    </div>

    <template v-else-if="proof">
      <div class="grid gap-4 lg:grid-cols-[minmax(260px,0.85fr)_minmax(0,1.15fr)]">
        <div class="min-w-0">
          <div class="aspect-[4/3] overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800">
            <img
              v-if="proofImageURL"
              :src="proofImageURL"
              :alt="t('payment.admin.manualReview.proofImage')"
              class="h-full w-full cursor-zoom-in object-contain"
              @click="previewOpen = true"
            />
            <div v-else class="flex h-full items-center justify-center px-4 text-center text-sm text-gray-400">
              {{ t('payment.admin.manualReview.imageUnavailable') }}
            </div>
          </div>
        </div>

        <div class="space-y-4">
          <dl class="grid grid-cols-2 gap-3 text-sm">
            <div>
              <dt class="text-xs text-gray-400">{{ t('payment.paymentMethod') }}</dt>
              <dd class="mt-0.5 font-medium text-gray-800 dark:text-gray-200">{{ t(`payment.methods.${proof.channel}`) }}</dd>
            </div>
            <div>
              <dt class="text-xs text-gray-400">{{ t('payment.admin.manualReview.submission') }}</dt>
              <dd class="mt-0.5 font-medium text-gray-800 dark:text-gray-200">{{ proof.submission_no }}/3</dd>
            </div>
            <div class="col-span-2 min-w-0">
              <dt class="text-xs text-gray-400">{{ t('payment.manual.transactionNo') }}</dt>
              <dd class="mt-0.5 break-all font-mono text-gray-800 dark:text-gray-200">{{ proof.transaction_no }}</dd>
            </div>
            <div>
              <dt class="text-xs text-gray-400">{{ t('payment.admin.manualReview.expectedAmount') }}</dt>
              <dd class="mt-0.5 font-semibold text-gray-900 dark:text-white">{{ expectedAmount }}</dd>
            </div>
            <div>
              <dt class="text-xs text-gray-400">{{ t('payment.admin.manualReview.submittedAt') }}</dt>
              <dd class="mt-0.5 text-gray-700 dark:text-gray-300">{{ formatDateTime(proof.submitted_at) }}</dd>
            </div>
            <div v-if="proof.received_amount !== undefined">
              <dt class="text-xs text-gray-400">{{ t('payment.admin.manualReview.receivedAmount') }}</dt>
              <dd class="mt-0.5 font-semibold text-gray-900 dark:text-white">{{ formatAmount(proof.received_amount) }}</dd>
            </div>
            <div v-if="proof.reviewed_at">
              <dt class="text-xs text-gray-400">{{ t('payment.admin.manualReview.reviewedAt') }}</dt>
              <dd class="mt-0.5 text-gray-700 dark:text-gray-300">{{ formatDateTime(proof.reviewed_at) }}</dd>
            </div>
            <div v-if="proof.rejection_reason" class="col-span-2 rounded-lg bg-red-50 p-3 dark:bg-red-900/20">
              <dt class="text-xs font-medium text-red-600 dark:text-red-400">{{ t('payment.admin.manualReview.rejectionReason') }}</dt>
              <dd class="mt-1 whitespace-pre-wrap break-words text-sm text-red-800 dark:text-red-200">{{ proof.rejection_reason }}</dd>
            </div>
          </dl>

          <form v-if="proof.status === 'submitted'" class="space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700" @submit.prevent="submitReview">
            <div class="inline-flex w-full rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
              <button
                type="button"
                class="flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
                :class="action === 'approve' ? 'bg-green-600 text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'"
                @click="action = 'approve'"
              >
                {{ t('payment.admin.manualReview.approve') }}
              </button>
              <button
                type="button"
                class="flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
                :class="action === 'reject' ? 'bg-red-600 text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'"
                @click="action = 'reject'"
              >
                {{ t('payment.admin.manualReview.reject') }}
              </button>
            </div>

            <div>
              <label class="input-label" for="manual-received-amount">{{ t('payment.admin.manualReview.receivedAmount') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">{{ currencyMark }}</span>
                <input
                  id="manual-received-amount"
                  v-model.trim="receivedAmount"
                  type="text"
                  inputmode="decimal"
                  class="input pl-8 font-mono"
                  :placeholder="order.pay_amount.toFixed(2)"
                />
              </div>
              <p v-if="action === 'approve' && receivedAmount && !amountMatches" class="mt-1 text-xs text-red-600 dark:text-red-400">
                {{ t('payment.admin.manualReview.amountMismatch', { amount: expectedAmount }) }}
              </p>
            </div>

            <div v-if="action === 'reject'">
              <label class="input-label" for="manual-rejection-reason">{{ t('payment.admin.manualReview.rejectionReason') }}</label>
              <textarea
                id="manual-rejection-reason"
                v-model.trim="reason"
                rows="3"
                maxlength="1000"
                class="input"
                :placeholder="t('payment.admin.manualReview.reasonPlaceholder')"
              />
            </div>

            <div
              v-if="action === 'approve'"
              class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs leading-relaxed text-amber-800 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-200"
            >
              {{ t('payment.admin.manualReview.approveWarning') }}
            </div>

            <button
              type="submit"
              class="btn w-full"
              :class="action === 'approve' ? 'bg-green-600 text-white hover:bg-green-700' : 'bg-red-600 text-white hover:bg-red-700'"
              :disabled="!canReview || submitting"
            >
              <span v-if="submitting" class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              <Icon v-else :name="action === 'approve' ? 'check' : 'x'" size="sm" />
              {{ submitting ? t('common.processing') : actionLabel }}
            </button>
          </form>
        </div>
      </div>
    </template>

    <Teleport to="body">
      <div
        v-if="previewOpen && proofImageURL"
        class="fixed inset-0 z-[70] flex items-center justify-center bg-black/80 p-4"
        @click="previewOpen = false"
      >
        <img :src="proofImageURL" :alt="t('payment.admin.manualReview.proofImage')" class="max-h-[92vh] max-w-[94vw] object-contain" />
      </div>
    </Teleport>
    <TotpStepUpDialog :controller="reviewStepUp" />
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminPaymentAPI } from '@/api/admin/payment'
import { useAppStore } from '@/stores'
import { isStepUpBlocked, isStepUpCancelled, stepUpBlockReason, useStepUp } from '@/composables/useStepUp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { currencySymbol, formatPaymentAmount } from '@/components/payment/currency'
import { formatOrderDateTime } from '@/components/payment/orderUtils'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { ManualPaymentProof, PaymentOrder } from '@/types/payment'

const props = defineProps<{
  order: PaymentOrder
}>()

const emit = defineEmits<{
  reviewed: []
}>()

const { t, locale } = useI18n()
const appStore = useAppStore()
const reviewStepUp = useStepUp()
const proof = ref<ManualPaymentProof | null>(null)
const proofImageURL = ref('')
const loading = ref(true)
const submitting = ref(false)
const previewOpen = ref(false)
const action = ref<'approve' | 'reject'>('approve')
const receivedAmount = ref('')
const reason = ref('')

const currencyMark = computed(() => currencySymbol(props.order.currency))
const expectedAmount = computed(() => formatAmount(props.order.pay_amount))
const amountMatches = computed(() => {
  if (!/^\d+(\.\d{1,2})?$/.test(receivedAmount.value)) return false
  const received = Number(receivedAmount.value)
  return Number.isFinite(received) && Math.round(received * 100) === Math.round(props.order.pay_amount * 100)
})
const canReview = computed(() => action.value === 'approve'
  ? amountMatches.value
  : reason.value.length >= 2 && reason.value.length <= 1000 && (!receivedAmount.value || /^\d+(\.\d{1,2})?$/.test(receivedAmount.value)))
const actionLabel = computed(() => action.value === 'approve'
  ? t('payment.admin.manualReview.confirmApprove')
  : t('payment.admin.manualReview.confirmReject'))
const proofStatusLabel = computed(() => proof.value
  ? t(`payment.admin.manualReview.status.${proof.value.status}`)
  : '')
const proofStatusClass = computed(() => {
  if (proof.value?.status === 'approved') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (proof.value?.status === 'rejected') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
})

function formatAmount(value: number): string {
  return formatPaymentAmount(value, props.order.currency || 'CNY', locale.value)
}

function formatDateTime(value?: string): string {
  return value ? formatOrderDateTime(value) : '-'
}

function clearProofImage() {
  if (proofImageURL.value) URL.revokeObjectURL(proofImageURL.value)
  proofImageURL.value = ''
}

async function loadProof() {
  loading.value = true
  clearProofImage()
  try {
    const response = await adminPaymentAPI.getManualProof(props.order.id)
    proof.value = response.data
    if (proof.value.has_image) {
      try {
        const image = await adminPaymentAPI.getManualProofImage(props.order.id)
        proofImageURL.value = URL.createObjectURL(image.data)
      } catch {
        proofImageURL.value = ''
      }
    }
    receivedAmount.value = proof.value.received_amount !== undefined
      ? proof.value.received_amount.toFixed(2)
      : props.order.pay_amount.toFixed(2)
    reason.value = ''
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function reportStepUpBlocked(error: unknown): boolean {
  if (!isStepUpBlocked(error)) return false
  appStore.showError(stepUpBlockReason(error) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
    ? t('stepUp.adminApiKeyForbidden')
    : t('stepUp.notEnabled'))
  return true
}

async function submitReview() {
  if (!proof.value || proof.value.status !== 'submitted' || !canReview.value || submitting.value) return
  submitting.value = true
  try {
    const request = () => adminPaymentAPI.reviewManualProof(props.order.id, {
      action: action.value,
      received_amount: receivedAmount.value || undefined,
      reason: action.value === 'reject' ? reason.value : undefined,
    })
    if (action.value === 'approve') {
      await reviewStepUp.run(request)
    } else {
      await request()
    }
    appStore.showSuccess(action.value === 'approve'
      ? t('payment.admin.manualReview.approveSuccess')
      : t('payment.admin.manualReview.rejectSuccess'))
    await loadProof()
    emit('reviewed')
  } catch (error: unknown) {
    if (isStepUpCancelled(error) || reportStepUpBlocked(error)) return
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    submitting.value = false
  }
}

watch(() => props.order.id, loadProof, { immediate: true })
onBeforeUnmount(clearProofImage)
</script>

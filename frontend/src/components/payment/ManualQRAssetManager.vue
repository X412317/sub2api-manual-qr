<template>
  <section class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-700">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.settings.payment.manualQRAssetsTitle') }}
        </h4>
        <p class="mt-1 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.payment.manualQRAssetsHint') }}
        </p>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="loading"
        :title="t('common.refresh')"
        @click="loadAssets"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
      <button
        v-for="channel in channels"
        :key="channel"
        type="button"
        class="min-w-28 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
        :class="activeChannel === channel
          ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
          : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
        @click="activeChannel = channel"
      >
        {{ t(`payment.methods.${channel}`) }}
      </button>
    </div>

    <div v-if="loading && !assets.length" class="flex justify-center py-10">
      <span class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
    </div>

    <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
      <div
        v-for="slot in slots"
        :key="slot.key"
        class="relative min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800/60"
      >
        <div class="aspect-square w-full bg-white dark:bg-dark-900">
          <img
            v-if="slot.asset && previewURLs[slot.asset.id]"
            :src="previewURLs[slot.asset.id]"
            :alt="slot.label"
            class="h-full w-full object-contain p-2"
          />
          <div v-else class="flex h-full flex-col items-center justify-center gap-2 px-3 text-center text-gray-400">
            <Icon name="upload" size="lg" />
            <span class="text-xs">{{ t('admin.settings.payment.manualQRNotUploaded') }}</span>
          </div>
        </div>

        <div class="border-t border-gray-200 p-2.5 dark:border-dark-700">
          <p class="truncate text-sm font-medium text-gray-800 dark:text-gray-200">{{ slot.label }}</p>
          <p v-if="slot.asset" class="mt-0.5 truncate text-[11px] text-gray-400">
            {{ formatBytes(slot.asset.file_size) }}
          </p>
          <div class="mt-2 flex items-center gap-1.5">
            <label
              class="btn btn-secondary btn-sm flex-1 cursor-pointer justify-center"
              :class="uploadingKey === slot.key ? 'pointer-events-none opacity-60' : ''"
            >
              <span v-if="uploadingKey === slot.key" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
              <Icon v-else name="upload" size="sm" />
              <input
                type="file"
                class="sr-only"
                accept="image/png,image/jpeg,image/webp"
                :disabled="uploadingKey !== ''"
                @change="uploadAsset(slot.amount, $event)"
              />
            </label>
            <button
              v-if="slot.asset"
              type="button"
              class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
              :disabled="deletingId === slot.asset.id"
              :title="t('common.delete')"
              @click="deleteAsset(slot.asset)"
            >
              <span v-if="deletingId === slot.asset.id" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
              <Icon v-else name="trash" size="sm" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <p class="text-xs text-gray-400 dark:text-gray-500">
      {{ t('admin.settings.payment.manualQRUploadRules') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminPaymentAPI } from '@/api/admin/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import Icon from '@/components/icons/Icon.vue'
import type { ManualQRAsset } from '@/types/payment'

const props = defineProps<{
  providerId: number
}>()

const { t } = useI18n()
const appStore = useAppStore()
const channels = ['alipay', 'wxpay'] as const
const fixedAmounts = [10, 20, 50, 100, 200, 500, 1000] as const
const activeChannel = ref<(typeof channels)[number]>('alipay')
const assets = ref<ManualQRAsset[]>([])
const loading = ref(false)
const uploadingKey = ref('')
const deletingId = ref<number | null>(null)
const previewURLs = ref<Record<number, string>>({})

const slots = computed(() => [null, ...fixedAmounts].map((amount) => {
  const asset = assets.value.find(item => item.channel === activeChannel.value && (item.amount ?? null) === amount)
  return {
    key: `${activeChannel.value}:${amount ?? 'generic'}`,
    amount,
    asset,
    label: amount === null
      ? t('admin.settings.payment.manualQRGeneric')
      : t('admin.settings.payment.manualQRFixed', { amount }),
  }
}))

function clearPreviewURLs() {
  Object.values(previewURLs.value).forEach(url => URL.revokeObjectURL(url))
  previewURLs.value = {}
}

async function loadAssets() {
  if (!props.providerId) return
  loading.value = true
  try {
    const response = await adminPaymentAPI.getManualQRAssets(props.providerId)
    assets.value = response.data || []
    clearPreviewURLs()
    const entries = await Promise.all(assets.value.map(async (asset) => {
      try {
        const image = await adminPaymentAPI.getManualQRAssetImage(props.providerId, asset.id)
        return [asset.id, URL.createObjectURL(image.data)] as const
      } catch {
        return null
      }
    }))
    previewURLs.value = Object.fromEntries(entries.filter((entry): entry is readonly [number, string] => entry !== null))
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

async function uploadAsset(amount: number | null, event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  const key = `${activeChannel.value}:${amount ?? 'generic'}`
  uploadingKey.value = key
  try {
    await adminPaymentAPI.saveManualQRAsset(props.providerId, activeChannel.value, amount, file)
    appStore.showSuccess(t('admin.settings.payment.manualQRUploadSuccess'))
    await loadAssets()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    uploadingKey.value = ''
  }
}

async function deleteAsset(asset: ManualQRAsset) {
  deletingId.value = asset.id
  try {
    await adminPaymentAPI.deleteManualQRAsset(props.providerId, asset.id)
    appStore.showSuccess(t('admin.settings.payment.manualQRDeleteSuccess'))
    await loadAssets()
  } catch (error: unknown) {
    appStore.showError(extractI18nErrorMessage(error, t, 'payment.errors', t('common.error')))
  } finally {
    deletingId.value = null
  }
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 KB'
  return `${Math.max(1, Math.round(value / 1024))} KB`
}

watch(() => props.providerId, loadAssets, { immediate: true })
onBeforeUnmount(clearPreviewURLs)
</script>

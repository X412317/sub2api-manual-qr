<template>
  <div class="plaza-pricing-table" :style="accentStyle">
    <div v-if="view === 'grid'" class="grid gap-3 p-3 sm:grid-cols-2 xl:grid-cols-3">
      <article v-for="m in sortedModels" :key="`card-${m.name}`" class="model-card group flex min-h-[208px] flex-col rounded-xl border border-gray-200/90 bg-white p-4 transition hover:-translate-y-0.5 hover:border-[var(--plaza-accent)] hover:shadow-md dark:border-dark-700/90 dark:bg-dark-900/70 dark:hover:border-[var(--plaza-accent)]">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex min-w-0 items-center gap-2">
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-gray-50 shadow-sm dark:border-dark-700 dark:bg-dark-800"
                :class="platformIconClass(m.platform || platform || '')"
              >
                <PlatformIcon :platform="(m.platform || platform) as GroupPlatform" size="md" />
              </span>
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <h3 class="truncate font-mono text-sm font-semibold text-gray-950 dark:text-white" :title="m.name">{{ m.name }}</h3>
                  <span v-if="modelKind(m)" class="hidden shrink-0 rounded-full bg-gray-100 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-gray-500 sm:inline-flex dark:bg-dark-800 dark:text-dark-400">
                    {{ modelKind(m) }}
                  </span>
                </div>
              </div>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
              <span>{{ platform || 'model' }}</span>
              <span class="text-gray-300 dark:text-dark-600">·</span>
              <span>{{ effectiveRate }}x</span>
              <span v-if="billingMode(m) !== BILLING_MODE_TOKEN" class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] dark:bg-dark-800">{{ billingModeLabel(m) }}</span>
            </div>
          </div>
          <CopyButton :text="m.name" :copy-label="t('modelPlaza.copyModel')" :copied-label="t('modelPlaza.modelCopied')" compact class="text-gray-400 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-500 dark:hover:bg-dark-800 dark:hover:text-white" />
        </div>
        <p class="mt-4 min-h-10 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ mDescription(m) }}</p>
        <div v-if="billingMode(m) === BILLING_MODE_TOKEN" class="mt-auto grid grid-cols-3 gap-2 pt-4">
          <div v-for="item in cardPrices(m)" :key="item.label" class="rounded-lg bg-gray-50 px-2.5 py-2 dark:bg-dark-800/75">
            <span class="block text-[10px] text-gray-400 dark:text-dark-500">{{ item.label }}</span>
            <strong class="mt-1 block truncate font-mono text-sm text-gray-950 dark:text-white">{{ item.value }}</strong>
          </div>
        </div>
        <div v-else class="mt-auto pt-4">
          <span class="text-[10px] text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.paidPrice') }}</span>
          <div v-if="requestIntervals(m).length" class="mt-1.5 flex flex-wrap gap-1.5">
            <span
              v-for="(iv, idx) in requestIntervals(m)"
              :key="idx"
              class="inline-flex items-baseline gap-1 rounded-md border border-gray-200 bg-gray-50 px-2 py-1 dark:border-dark-700 dark:bg-dark-800/80"
            >
              <span class="text-[10px] text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
              <strong class="font-mono text-sm text-gray-950 dark:text-white">{{ paidRequestPrice(iv.per_request_price) }}</strong>
              <span class="text-[10px] text-gray-400">{{ perUnitSuffix(m) }}</span>
            </span>
          </div>
          <strong v-else class="mt-1 block font-mono text-lg text-gray-950 dark:text-white">{{ paidRequestPrice(m.pricing?.per_request_price) }}<span class="ml-1 text-xs font-normal text-gray-400">{{ perUnitSuffix(m) }}</span></strong>
        </div>
        <details class="mt-3 border-t border-gray-100 pt-3 dark:border-dark-800">
          <summary class="flex cursor-pointer list-none items-center justify-between text-xs font-semibold text-primary-600 dark:text-primary-400">{{ t('modelPlaza.controls.details') }}<Icon name="chevronDown" size="xs" /></summary>
          <div class="mt-3 space-y-1 text-xs text-gray-500 dark:text-dark-400">
            <p>{{ t('modelPlaza.table.rate') }}: {{ effectiveRate }}x</p>
            <p>{{ t('modelPlaza.table.officialPrice') }}: {{ official(m.official_pricing?.output_price) }}</p>
            <p v-if="m.pricing?.intervals?.length">{{ t('modelPlaza.controls.tieredPricing') }}: {{ m.pricing.intervals.length }}</p>
          </div>
        </details>
      </article>
    </div>
    <div v-else class="divide-y divide-gray-100 md:hidden dark:divide-dark-800">
      <article v-for="m in sortedModels" :key="`mobile-${m.name}`" class="px-4 py-4">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span
                class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800"
                :class="platformIconClass(m.platform || platform || '')"
              >
                <PlatformIcon :platform="(m.platform || platform) as GroupPlatform" size="sm" />
              </span>
              <h3 class="break-all text-sm font-semibold text-gray-950 dark:text-white">{{ m.name }}</h3>
              <span v-if="billingMode(m) !== BILLING_MODE_TOKEN" class="rounded-md bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700/70 dark:text-dark-300">
                {{ billingModeLabel(m) }}
              </span>
            </div>
            <div class="mt-1.5 flex items-center gap-1.5 text-xs">
              <span class="text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.rate') }}</span>
              <template v-if="hasCustomRate">
                <span class="text-gray-400 line-through dark:text-dark-500">{{ rateMultiplier }}x</span>
                <span class="font-semibold text-primary-600 dark:text-primary-400">{{ effectiveRate }}x</span>
              </template>
              <span v-else class="font-semibold text-gray-700 dark:text-gray-300">{{ effectiveRate }}x</span>
            </div>
          </div>
          <CopyButton
            :text="m.name"
            :copy-label="t('modelPlaza.copyModel')"
            :copied-label="t('modelPlaza.modelCopied')"
            compact
            class="text-gray-400 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-500 dark:hover:bg-dark-800 dark:hover:text-white"
          />
        </div>

        <div v-if="billingMode(m) === BILLING_MODE_TOKEN" class="mt-4">
          <div v-if="tokenIntervals(m).length" class="space-y-2">
            <div v-for="(iv, idx) in tokenIntervals(m)" :key="idx" class="grid grid-cols-[1fr_auto_auto] items-center gap-3 rounded-md bg-gray-50 px-3 py-2 text-xs dark:bg-dark-900">
              <span class="font-medium text-gray-500 dark:text-dark-400">{{ tierLabel(iv) }}</span>
              <span class="text-right"><span class="block text-[10px] text-gray-400">{{ t('modelPlaza.table.input') }}</span><strong class="font-mono text-gray-900 dark:text-white">{{ paidPerMillion(iv.input_price) }}</strong></span>
              <span class="text-right"><span class="block text-[10px] text-gray-400">{{ t('modelPlaza.table.output') }}</span><strong class="font-mono text-gray-900 dark:text-white">{{ paidPerMillion(iv.output_price) }}</strong></span>
            </div>
          </div>
          <div v-else class="grid grid-cols-3 gap-2 text-xs">
            <div class="rounded-md bg-gray-50 px-2.5 py-2 dark:bg-dark-900">
              <span class="block text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.input') }}</span>
              <strong class="mt-1 block font-mono text-gray-900 dark:text-white">{{ paidPerMillion(m.pricing?.input_price) }}</strong>
            </div>
            <div class="rounded-md bg-gray-50 px-2.5 py-2 dark:bg-dark-900">
              <span class="block text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.output') }}</span>
              <strong class="mt-1 block font-mono text-gray-900 dark:text-white">{{ paidPerMillion(m.pricing?.output_price) }}</strong>
            </div>
            <div class="rounded-md bg-gray-50 px-2.5 py-2 dark:bg-dark-900">
              <span class="block text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cache') }}</span>
              <strong class="mt-1 block font-mono text-gray-900 dark:text-white">{{ paidPerMillion(m.pricing?.cache_read_price) }}</strong>
              <span v-if="m.pricing?.cache_write_price != null" class="mt-0.5 block font-mono text-[10px] text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cacheWrite') }} {{ paidPerMillion(m.pricing.cache_write_price) }}</span>
            </div>
          </div>
        </div>

        <div v-else class="mt-4">
          <div v-if="requestIntervals(m).length" class="flex flex-wrap gap-1.5">
            <span v-for="(iv, idx) in requestIntervals(m)" :key="idx" class="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-gray-800 dark:bg-dark-700/60 dark:text-gray-200">
              <span class="font-sans text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
              {{ paidRequestPrice(iv.per_request_price) }}<span class="font-sans text-gray-400">{{ perUnitSuffix(m) }}</span>
            </span>
          </div>
          <p v-else class="font-mono text-base font-semibold text-gray-950 dark:text-white">
            {{ paidRequestPrice(m.pricing?.per_request_price) }}<span class="ml-1 font-sans text-xs font-normal text-gray-400">{{ perUnitSuffix(m) }}</span>
          </p>
        </div>

        <details class="group mt-3 border-t border-gray-100 pt-3 dark:border-dark-800">
          <summary class="flex cursor-pointer list-none items-center justify-between text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t('modelPlaza.table.officialReference') }}
            <span class="flex items-center gap-1 text-[11px] text-gray-400">
              {{ unitLabel }}
              <Icon name="chevronDown" size="xs" class="transition-transform group-open:rotate-180" />
            </span>
          </summary>
          <div class="mt-3 grid grid-cols-3 gap-2 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('modelPlaza.table.input') }} <strong class="block font-mono font-medium text-gray-800 dark:text-gray-200">{{ official(m.official_pricing?.input_price) }}</strong></span>
            <span>{{ t('modelPlaza.table.output') }} <strong class="block font-mono font-medium text-gray-800 dark:text-gray-200">{{ official(m.official_pricing?.output_price) }}</strong></span>
            <span>{{ t('modelPlaza.table.cacheRead') }} <strong class="block font-mono font-medium text-gray-800 dark:text-gray-200">{{ official(m.official_pricing?.cache_read_price) }}</strong></span>
          </div>
        </details>
      </article>
    </div>

    <div v-if="view !== 'grid'" class="hidden overflow-x-auto md:block">
    <table class="w-full min-w-[860px] table-fixed border-collapse text-sm tabular-nums">
      <colgroup>
        <col class="w-[22%]" />
        <col class="w-[10%]" />
        <col class="w-[10%]" />
        <col class="w-[14%]" />
        <col class="w-[10%]" />
        <col class="w-[10%]" />
        <col class="w-[14%]" />
        <col class="w-[10%]" />
      </colgroup>
      <thead>
        <tr
          class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-dark-400"
        >
          <th
            rowspan="2"
            class="border-r border-gray-100 py-2.5 pl-5 pr-4 text-left align-middle dark:border-dark-700/60"
          >
            {{ t('modelPlaza.table.model') }}
          </th>
          <th colspan="3" class="pz-bg pt-2 text-center">
            <div class="pz-title border-b pb-2 font-semibold">
              {{ t('modelPlaza.table.paidPrice') }}
              <span class="pz-unit ml-1 normal-case font-normal">{{ unitLabel }}</span>
            </div>
          </th>
          <th
            colspan="3"
            class="border-l border-gray-100 pt-2 text-center dark:border-dark-700/60"
          >
            <div class="border-b border-gray-200 pb-2 text-gray-400 dark:border-dark-600 dark:text-dark-500">
              {{ t('modelPlaza.table.officialPrice') }}
              <span class="ml-1 normal-case font-normal text-gray-400 dark:text-dark-500">{{ unitLabel }}</span>
            </div>
          </th>
          <th
            rowspan="2"
            class="border-l border-gray-100 py-2.5 pl-3 pr-5 text-right align-middle dark:border-dark-700/60"
          >
            {{ t('modelPlaza.table.rate') }}
          </th>
        </tr>
        <tr
          class="border-b border-gray-200 text-left text-[11px] font-medium uppercase leading-4 tracking-wide text-gray-400 dark:border-dark-700 dark:text-dark-500"
        >
          <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.input') }}</th>
          <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.output') }}</th>
          <th class="pz-bg px-3 py-2 font-medium">{{ t('modelPlaza.table.cache') }}</th>
          <th class="border-l border-gray-100 px-3 py-2 font-medium dark:border-dark-700/60">
            {{ t('modelPlaza.table.input') }}
          </th>
          <th class="px-3 py-2 font-medium">{{ t('modelPlaza.table.output') }}</th>
          <th class="px-3 py-2 font-medium">{{ t('modelPlaza.table.cache') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="m in sortedModels"
          :key="m.name"
          class="group border-b border-gray-100 transition-colors last:border-b-0 hover:bg-gray-50/70 dark:border-dark-800 dark:hover:bg-dark-800/50"
        >
          <!-- 模型名 + 非 token 计费模式徽章 -->
          <td class="border-r border-gray-100 py-2.5 pl-5 pr-4 align-middle dark:border-dark-700/60">
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ m.name }}</span>
              <CopyButton
                :text="m.name"
                :copy-label="t('modelPlaza.copyModel')"
                :copied-label="t('modelPlaza.modelCopied')"
                compact
                class="opacity-0 text-gray-400 hover:bg-gray-100 hover:text-gray-900 focus:opacity-100 group-hover:opacity-100 dark:text-dark-500 dark:hover:bg-dark-700 dark:hover:text-white"
              />
              <span
                v-if="billingMode(m) !== BILLING_MODE_TOKEN"
                class="rounded-md bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700/70 dark:text-dark-300"
              >
                {{ billingModeLabel(m) }}
              </span>
            </div>
          </td>

          <!-- token 计费:输入 / 输出(阶梯内联)/ 缓存(写/读) -->
          <template v-if="billingMode(m) === BILLING_MODE_TOKEN">
            <td class="pz-cell px-3 py-2.5 align-middle font-mono font-semibold text-gray-900 dark:text-gray-50">
              <template v-if="tokenIntervals(m).length">
                <div
                  v-for="(iv, idx) in tokenIntervals(m)"
                  :key="idx"
                  class="whitespace-nowrap text-xs leading-5"
                >
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
                  {{ paidPerMillion(iv.input_price) }}
                </div>
              </template>
              <template v-else>{{ paidPerMillion(m.pricing?.input_price) }}</template>
            </td>
            <td class="pz-cell px-3 py-2.5 align-middle font-mono font-semibold text-gray-900 dark:text-gray-50">
              <template v-if="tokenIntervals(m).length">
                <div
                  v-for="(iv, idx) in tokenIntervals(m)"
                  :key="idx"
                  class="whitespace-nowrap text-xs leading-5"
                >
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
                  {{ paidPerMillion(iv.output_price) }}
                </div>
              </template>
              <template v-else>{{ paidPerMillion(m.pricing?.output_price) }}</template>
            </td>
            <td class="pz-cell px-3 py-2.5 align-middle">
              <div
                v-if="hasCachePricing(m)"
                class="space-y-0.5 font-mono text-xs text-gray-800 dark:text-gray-200"
              >
                <div>
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWrite') }}</span>
                  {{ paidPerMillion(m.pricing?.cache_write_price) }}
                </div>
                <div>
                  <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheRead') }}</span>
                  {{ paidPerMillion(m.pricing?.cache_read_price) }}
                </div>
              </div>
              <span v-else class="text-gray-400 dark:text-dark-500">-</span>
            </td>
          </template>

          <!-- 按次 / 按图片计费:实付区整体合并,阶梯芯片或单一按次价 -->
          <template v-else>
            <td colspan="3" class="pz-cell px-3 py-2.5 align-middle">
              <div
                v-if="requestIntervals(m).length"
                class="flex flex-wrap items-center gap-1.5"
              >
                <span
                  v-for="(iv, idx) in requestIntervals(m)"
                  :key="idx"
                  class="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-800 dark:bg-dark-700/60 dark:text-gray-200"
                >
                  <span class="font-sans text-gray-400 dark:text-dark-500">{{ tierLabel(iv) }}</span>
                  {{ paidRequestPrice(iv.per_request_price)
                  }}<span class="font-sans text-gray-400 dark:text-dark-500">{{ perUnitSuffix(m) }}</span>
                </span>
              </div>
              <template v-else-if="m.pricing?.per_request_price != null">
                <span class="font-mono font-semibold text-gray-900 dark:text-gray-50">
                  {{ paidRequestPrice(m.pricing.per_request_price) }}
                </span>
                <span class="ml-1 text-xs text-gray-400 dark:text-dark-500">{{ perUnitSuffix(m) }}</span>
              </template>
              <span v-else class="text-gray-400 dark:text-dark-500">-</span>
            </td>
          </template>

          <!-- 官方价格(LiteLLM 参考价,不乘倍率) -->
          <td
            class="border-l border-gray-100 px-3 py-2.5 align-middle font-mono text-xs text-gray-500 dark:border-dark-700/60 dark:text-dark-400"
          >
            {{ official(m.official_pricing?.input_price) }}
          </td>
          <td class="px-3 py-2.5 align-middle font-mono text-xs text-gray-500 dark:text-dark-400">
            {{ official(m.official_pricing?.output_price) }}
          </td>
          <td class="px-3 py-2.5 align-middle">
            <div
              v-if="m.official_pricing && hasOfficialCache(m.official_pricing)"
              class="space-y-0.5 font-mono text-xs text-gray-500 dark:text-dark-400"
            >
              <div>
                <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheWrite') }}</span>
                {{ official(m.official_pricing.cache_write_price)
                }}<template v-if="m.official_pricing.cache_write_1h_price != null"
                  ><span class="font-sans text-gray-400 dark:text-dark-500"> (1h </span>{{ official(m.official_pricing.cache_write_1h_price)
                  }}<span class="font-sans text-gray-400 dark:text-dark-500">)</span></template
                >
              </div>
              <div>
                <span class="mr-1 font-sans font-normal text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.cacheRead') }}</span>
                {{ official(m.official_pricing.cache_read_price) }}
              </div>
            </div>
            <span v-else class="text-gray-400 dark:text-dark-500">-</span>
          </td>

          <!-- 折扣倍率(专属倍率划线展示原倍率) -->
          <td
            class="border-l border-gray-100 py-2.5 pl-3 pr-5 text-right align-middle font-mono text-xs dark:border-dark-700/60"
          >
            <template v-if="hasCustomRate">
              <span class="mr-1 text-gray-400 line-through dark:text-dark-500">{{ rateMultiplier }}x</span>
              <span class="font-bold text-primary-600 dark:text-primary-400">{{ effectiveRate }}x</span>
            </template>
            <span v-else class="font-bold text-gray-700 dark:text-gray-300">{{ effectiveRate }}x</span>
          </td>
        </tr>
      </tbody>
    </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/common/CopyButton.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { formatScaled } from '@/utils/pricing'
import { platformAccentColor, platformIconClass } from '@/utils/platformColors'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_IMAGE,
  type BillingMode
} from '@/constants/channel'
import type { PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  models: PlazaModel[]
  /** 分组平台;实付分区底色随平台着色,未知平台回退品牌青。 */
  platform?: string
  /** 分组默认倍率。 */
  rateMultiplier: number
  /** 用户专属倍率;与默认不同,实付价按此计算并划线展示原倍率。 */
  userRateMultiplier?: number | null
  priceMode?: 'paid' | 'official'
  unit?: 'million' | 'thousand'
  view?: 'grid' | 'list'
  sort?: 'latest' | 'name' | 'price'
}>()

const { t } = useI18n()

/** 实付分区只从平台拿一个主色,浅底/标题/下划线全部由 scoped CSS 用 color-mix 派生。 */
const accentStyle = computed(() => ({ '--plaza-accent': platformAccentColor(props.platform ?? '') }))

const unitScale = computed(() => props.unit === 'thousand' ? 1_000 : 1_000_000)
const unitLabel = computed(() => props.unit === 'thousand'
  ? t('modelPlaza.table.unitPerThousand')
  : t('modelPlaza.table.unitPerMillion'))

/**
 * 展示顺序:
 * 最新排序按模型名中的数字版本倒序，兼容 gpt-5.6、claude-3-7 等常见命名。
 * 价格排序仍将 token 与按图/按次计费分开，避免不同量纲混排。
 */
const sortedModels = computed(() => {
  return [...props.models].sort((a, b) => {
    if (props.sort === 'name') return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
    if (props.sort === 'latest') return b.name.localeCompare(a.name, undefined, { numeric: true, sensitivity: 'base' })
    const ta = billingMode(a) === BILLING_MODE_TOKEN
    const tb = billingMode(b) === BILLING_MODE_TOKEN
    if (ta !== tb) return ta ? -1 : 1
    const pa = a.official_pricing?.output_price ?? null
    const pb = b.official_pricing?.output_price ?? null
    if (pa != null && pb != null && pa !== pb) return pb - pa
    if (pa != null && pb == null) return -1
    if (pa == null && pb != null) return 1
    return b.name.localeCompare(a.name)
  })
})

const effectiveRate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const hasCustomRate = computed(
  () => props.userRateMultiplier != null && props.userRateMultiplier !== props.rateMultiplier
)

function billingMode(m: PlazaModel): BillingMode {
  return (m.pricing?.billing_mode || BILLING_MODE_TOKEN) as BillingMode
}

function billingModeLabel(m: PlazaModel): string {
  return billingMode(m) === BILLING_MODE_IMAGE
    ? t('modelPlaza.table.perImage')
    : t('modelPlaza.table.perRequest')
}

function modelKind(m: PlazaModel): string {
  if (billingMode(m) === BILLING_MODE_IMAGE) return 'Image'
  if (/^gpt(?:-|$)/i.test(m.name)) return 'GPT'
  if (/^claude(?:-|$)/i.test(m.name)) return 'Claude'
  if (/^gemini(?:-|$)/i.test(m.name)) return 'Gemini'
  return ''
}

/** 价格统一保底 2 位小数,更长的有效小数原样保留。 */
const MIN_DECIMALS = 2

/** 实付价 = 渠道单价 × 生效倍率,按 $/1M token 展示。 */
function paidPerMillion(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value * effectiveRate.value, unitScale.value, MIN_DECIMALS)
}

/** 按次 / 按图片单价(乘生效倍率,不换算 1M)。 */
function paidRequestPrice(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value * effectiveRate.value, 1, MIN_DECIMALS)
}

/** 官方参考价不乘倍率。 */
function official(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value, unitScale.value, MIN_DECIMALS)
}

function cardPrices(m: PlazaModel): Array<{ label: string; value: string }> {
  const source = props.priceMode === 'official' ? m.official_pricing : m.pricing
  const render = (value: number | null | undefined) => props.priceMode === 'official' ? official(value) : paidPerMillion(value)
  return [
    { label: t('modelPlaza.table.input'), value: render(source?.input_price) },
    { label: t('modelPlaza.table.output'), value: render(source?.output_price) },
    { label: t('modelPlaza.table.cacheRead'), value: render(source?.cache_read_price) }
  ]
}

function mDescription(m: PlazaModel): string {
  if (billingMode(m) === BILLING_MODE_IMAGE) return t('modelPlaza.controls.imagePricing')
  if (m.pricing?.intervals?.length) return t('modelPlaza.controls.tieredPricing')
  return t('modelPlaza.controls.modelPricing')
}

/** 非 token 计费的单位后缀:按图片 → “/ 张”,按次 → “/ 次”。 */
function perUnitSuffix(m: PlazaModel): string {
  return billingMode(m) === BILLING_MODE_IMAGE
    ? t('modelPlaza.table.perUnitImage')
    : t('modelPlaza.table.perUnitRequest')
}

function hasCachePricing(m: PlazaModel): boolean {
  return m.pricing?.cache_write_price != null || m.pricing?.cache_read_price != null
}

function hasOfficialCache(o: NonNullable<PlazaModel['official_pricing']>): boolean {
  return o.cache_write_price != null || o.cache_read_price != null || o.cache_write_1h_price != null
}

/** token 模式的阶梯定价(内联进输入/输出列)。 */
function tokenIntervals(m: PlazaModel): UserPricingInterval[] {
  return m.pricing?.intervals ?? []
}

/** 按次/按图模式的阶梯定价(仅保留配了按次价的档位)。 */
function requestIntervals(m: PlazaModel): UserPricingInterval[] {
  return (m.pricing?.intervals ?? []).filter((iv) => iv.per_request_price != null)
}

/** 档位标签:优先管理员配置的 tier_label,否则按 token 区间生成(≤200K / >200K / 200K–1M)。 */
function tierLabel(iv: UserPricingInterval): string {
  if (iv.tier_label) return iv.tier_label
  const { min_tokens: min, max_tokens: max } = iv
  if (max == null) return `>${formatTokenCount(min)}`
  if (min === 0) return `≤${formatTokenCount(max)}`
  return `${formatTokenCount(min)}–${formatTokenCount(max)}`
}

function formatTokenCount(n: number): string {
  if (n >= 1_000_000) return `${trimZero(n / 1_000_000)}M`
  if (n >= 1_000) return `${trimZero(n / 1_000)}K`
  return String(n)
}

function trimZero(n: number): string {
  return String(Math.round(n * 100) / 100)
}
</script>

<style scoped>
/* 实付分区配色统一从 --plaza-accent(平台主色)派生,新增平台无需扩展样式 */
.plaza-pricing-table {
  --pz-title: color-mix(in srgb, var(--plaza-accent) 88%, black);
  --pz-bg: color-mix(in srgb, var(--plaza-accent) 7%, transparent);
  --pz-bg-hover: color-mix(in srgb, var(--plaza-accent) 13%, transparent);
}

.dark .plaza-pricing-table {
  --pz-title: color-mix(in srgb, var(--plaza-accent) 70%, white);
  --pz-bg: color-mix(in srgb, var(--plaza-accent) 6%, transparent);
  --pz-bg-hover: color-mix(in srgb, var(--plaza-accent) 10%, transparent);
}

.pz-bg,
.pz-cell {
  background-color: var(--pz-bg);
}

.pz-cell {
  transition: background-color 150ms cubic-bezier(0.4, 0, 0.2, 1);
}

tbody tr:hover .pz-cell {
  background-color: var(--pz-bg-hover);
}

.pz-title {
  /* color-mix 不可用的老浏览器回退为平台原色 */
  color: var(--plaza-accent);
  color: var(--pz-title);
  border-color: color-mix(in srgb, var(--pz-title) 30%, transparent);
}

.pz-unit {
  color: color-mix(in srgb, var(--pz-title) 62%, transparent);
}
</style>

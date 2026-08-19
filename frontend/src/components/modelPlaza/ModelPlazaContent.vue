<template>
  <div class="space-y-7">
    <header v-if="!embedded" class="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="mb-2 text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">
          {{ t('modelPlaza.eyebrow') }}
        </p>
        <h1 class="text-3xl font-bold text-gray-950 dark:text-white sm:text-4xl">{{ t('modelPlaza.title') }}</h1>
        <p class="mt-2 text-base text-gray-600 dark:text-dark-300">{{ t('modelPlaza.description') }}</p>
      </div>
      <RouterLink to="/docs" class="inline-flex w-fit items-center gap-1.5 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
        {{ t('modelPlaza.viewDocs') }}
        <Icon name="arrowRight" size="xs" />
      </RouterLink>
    </header>

    <div v-if="loading" class="space-y-5" aria-live="polite">
      <div class="grid grid-cols-2 border-y border-gray-200 md:grid-cols-4 dark:border-dark-800">
        <div
          v-for="n in 4"
          :key="n"
          class="px-4 py-5 md:px-6"
          :class="[
            n > 2 ? 'border-t border-gray-200 md:border-t-0 dark:border-dark-800' : '',
            n % 2 === 0 ? 'border-l border-gray-200 dark:border-dark-800' : '',
            n === 3 ? 'md:border-l md:border-gray-200 md:dark:border-dark-800' : ''
          ]"
        >
          <div class="h-3 w-16 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          <div class="mt-3 h-7 w-12 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
        </div>
      </div>
      <div class="h-40 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-900"></div>
      <div class="h-56 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-900"></div>
    </div>

    <div v-else-if="error" class="flex min-h-[320px] flex-col items-center justify-center rounded-lg border border-red-200 bg-red-50 px-5 text-center dark:border-red-900/50 dark:bg-red-950/20">
      <span class="flex h-11 w-11 items-center justify-center rounded-full bg-red-100 text-red-600 dark:bg-red-950/60 dark:text-red-300">
        <Icon name="exclamationCircle" size="lg" />
      </span>
      <h2 class="mt-4 text-base font-semibold text-red-900 dark:text-red-100">{{ t('modelPlaza.loadFailed') }}</h2>
      <p class="mt-1 text-sm text-red-700/80 dark:text-red-300/80">{{ t('modelPlaza.loadFailedHint') }}</p>
      <button type="button" class="mt-5 inline-flex h-9 items-center gap-2 rounded-md border border-red-300 bg-white px-3 text-sm font-medium text-red-700 hover:bg-red-100 dark:border-red-800 dark:bg-dark-900 dark:text-red-200 dark:hover:bg-red-950/50" @click="$emit('retry')">
        <Icon name="refresh" size="sm" />
        {{ t('modelPlaza.retry') }}
      </button>
    </div>

    <template v-else>
      <div data-testid="plaza-stats" class="grid grid-cols-2 border-y border-gray-200 md:grid-cols-4 dark:border-dark-800">
        <div
          v-for="(stat, index) in stats"
          :key="stat.label"
          class="px-4 py-5 md:px-6"
          :class="[
            index >= 2 ? 'border-t border-gray-200 md:border-t-0 dark:border-dark-800' : '',
            index % 2 === 1 ? 'border-l border-gray-200 dark:border-dark-800' : '',
            index === 2 ? 'md:border-l md:border-gray-200 md:dark:border-dark-800' : ''
          ]"
        >
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t(stat.label) }}</p>
          <p class="mt-1.5 text-2xl font-semibold text-gray-950 dark:text-white">{{ stat.value }}</p>
        </div>
      </div>

      <div v-if="descriptionHtml" class="plaza-description rounded-lg border border-gray-200 bg-gray-50 px-5 py-4 text-sm dark:border-dark-800 dark:bg-dark-900/50" v-html="descriptionHtml"></div>

      <div v-if="!isAuthenticated" class="flex flex-col gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900 sm:flex-row sm:items-center sm:justify-between dark:border-blue-900/50 dark:bg-blue-950/25 dark:text-blue-200">
        <p class="flex items-start gap-2 leading-6">
          <Icon name="infoCircle" size="sm" class="mt-1 shrink-0" />
          {{ t('modelPlaza.anonymousHint') }}
        </p>
        <RouterLink :to="{ path: '/login', query: { redirect: '/model-plaza' } }" class="inline-flex shrink-0 items-center gap-1 font-medium text-blue-700 hover:text-blue-900 dark:text-blue-300 dark:hover:text-blue-100">
          {{ t('modelPlaza.signInForRates') }}
          <Icon name="arrowRight" size="xs" />
        </RouterLink>
      </div>

      <PlazaFilterBar
        :platforms="platforms"
        :groups="groupOptions"
        :rates="rates"
        :platform="selectedPlatform"
        :group-id="selectedGroupId"
        :rate="selectedRate"
        :search="searchQuery"
        @update:platform="selectedPlatform = $event"
        @update:group-id="selectedGroupId = $event"
        @update:rate="selectedRate = $event"
        @update:search="searchQuery = $event"
      />

      <div class="flex min-h-8 flex-wrap items-center justify-between gap-3">
        <p data-testid="plaza-result-summary" class="text-sm text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.resultSummary', { groups: filteredGroups.length, models: filteredModelCount }) }}
        </p>
        <button v-if="filtersActive" data-testid="plaza-reset-filters" type="button" class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 hover:text-gray-950 dark:text-dark-300 dark:hover:text-white" @click="resetFilters">
          <Icon name="x" size="xs" />
          {{ t('modelPlaza.resetFilters') }}
        </button>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 dark:border-dark-800 dark:bg-dark-900/35">
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800/70" role="group" :aria-label="t('modelPlaza.controls.priceMode')">
            <button v-for="mode in priceModes" :key="mode.value" type="button" class="min-h-8 rounded px-2.5 text-xs font-semibold transition" :class="priceMode === mode.value ? 'bg-primary-600 text-white shadow-sm dark:bg-primary-500' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'" @click="priceMode = mode.value">
              {{ t(mode.label) }}
            </button>
          </div>
          <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800/70" role="group" :aria-label="t('modelPlaza.controls.unit')">
            <button v-for="unitOption in unitOptions" :key="unitOption.value" type="button" class="min-h-8 rounded px-2.5 text-xs font-semibold transition" :class="unit === unitOption.value ? 'bg-primary-600 text-white shadow-sm dark:bg-primary-500' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'" @click="unit = unitOption.value">
              {{ t(unitOption.label) }}
            </button>
          </div>
          <button type="button" class="inline-flex min-h-8 items-center gap-1.5 rounded-md border border-gray-200 px-2.5 text-xs font-semibold text-gray-600 hover:bg-gray-50 dark:border-dark-700 dark:text-dark-300 dark:hover:bg-dark-800" :title="t('modelPlaza.controls.sort')" @click="cycleSort">
            <Icon name="sort" size="xs" />
            {{ t(`modelPlaza.controls.${sort}`) }}
          </button>
        </div>
        <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800/70" role="group" :aria-label="t('modelPlaza.controls.view')">
          <button type="button" class="inline-flex h-8 w-8 items-center justify-center rounded transition" :class="view === 'grid' ? 'bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-400 hover:text-gray-700 dark:text-dark-400 dark:hover:text-white'" :title="t('modelPlaza.controls.grid')" :aria-label="t('modelPlaza.controls.grid')" @click="view = 'grid'"><Icon name="grid" size="sm" /></button>
          <button type="button" class="inline-flex h-8 w-8 items-center justify-center rounded transition" :class="view === 'list' ? 'bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-400 hover:text-gray-700 dark:text-dark-400 dark:hover:text-white'" :title="t('modelPlaza.controls.list')" :aria-label="t('modelPlaza.controls.list')" @click="view = 'list'"><Icon name="menu" size="sm" /></button>
        </div>
      </div>

      <div v-if="filteredGroups.length > 0" class="space-y-5">
        <PlazaGroupSection v-for="g in filteredGroups" :key="g.id" :group="g" :price-mode="priceMode" :unit="unit" :view="view" :sort="sort" />
      </div>
      <div v-else class="rounded-lg border border-dashed border-gray-300 px-5 py-14 text-center dark:border-dark-600">
        <Icon name="search" size="lg" class="mx-auto text-gray-300 dark:text-dark-600" />
        <p class="mt-3 text-sm font-medium text-gray-700 dark:text-dark-300">{{ filtersActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}</p>
        <button v-if="filtersActive" type="button" class="mt-3 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="resetFilters">{{ t('modelPlaza.resetFilters') }}</button>
      </div>

      <p class="text-xs leading-5 text-gray-400 dark:text-dark-500">{{ t('modelPlaza.pricingNote') }}</p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import type { ModelPlazaGroup, ModelPlazaResponse } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内):隐藏页头。 */
  embedded?: boolean
}>()

defineEmits<{
  retry: []
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')
const priceMode = ref<'paid' | 'official'>('paid')
const unit = ref<'million' | 'thousand'>('million')
const view = ref<'grid' | 'list'>('grid')
const sort = ref<'latest' | 'name' | 'price'>('latest')

const priceModes = [
  { value: 'paid' as const, label: 'modelPlaza.controls.paid' },
  { value: 'official' as const, label: 'modelPlaza.controls.official' }
]
const unitOptions = [
  { value: 'million' as const, label: 'modelPlaza.controls.perMillion' },
  { value: 'thousand' as const, label: 'modelPlaza.controls.perThousand' }
]
const sortOptions: Array<'latest' | 'price' | 'name'> = ['latest', 'price', 'name']

function cycleSort() {
  const index = sortOptions.indexOf(sort.value)
  sort.value = sortOptions[(index + 1) % sortOptions.length]
}

const filtersActive = computed(() =>
  searchQuery.value.trim() !== '' ||
  selectedPlatform.value !== 'all' ||
  selectedGroupId.value !== 'all' ||
  selectedRate.value !== 'all'
)

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
function effectiveRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

const platforms = computed(() =>
  [...new Set((props.response?.groups ?? []).map((g) => g.platform).filter(Boolean))].sort()
)

const groupOptions = computed(() =>
  (props.response?.groups ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    platform: g.platform,
    rate: effectiveRate(g)
  }))
)

/** 全量生效倍率;当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set((props.response?.groups ?? []).map(effectiveRate))].sort((a, b) => a - b)
)

const modelCount = computed(() => new Set(
  (props.response?.groups ?? []).flatMap((group) => group.models.map((model) => model.name))
).size)

const stats = computed(() => [
  { label: 'modelPlaza.stats.models', value: modelCount.value },
  { label: 'modelPlaza.stats.groups', value: props.response?.groups.length ?? 0 },
  { label: 'modelPlaza.stats.platforms', value: platforms.value.length },
  { label: 'modelPlaza.stats.bestRate', value: rates.value.length ? `${rates.value[0]}x` : '-' }
])

/** 数据刷新后选中的倍率可能不复存在,重置为全部。 */
watch(rates, (list) => {
  if (selectedRate.value !== 'all' && !list.includes(selectedRate.value)) {
    selectedRate.value = 'all'
  }
})

const filteredGroups = computed(() => {
  let groups = props.response?.groups ?? []
  if (selectedPlatform.value !== 'all') {
    groups = groups.filter((g) => g.platform === selectedPlatform.value)
  }
  if (selectedGroupId.value !== 'all') {
    groups = groups.filter((g) => g.id === selectedGroupId.value)
  }
  if (selectedRate.value !== 'all') {
    groups = groups.filter((g) => effectiveRate(g) === selectedRate.value)
  }
  // 模型名搜索:分组内只留命中的模型,整组无命中则隐藏该分组。
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    groups = groups
      .map((g) => ({ ...g, models: g.models.filter((m) => m.name.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  // 专属倍率会改变生效值,不能只依赖后端按默认倍率的排序。
  return [...groups].sort(
    (a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name)
  )
})

const filteredModelCount = computed(() => filteredGroups.value.reduce((total, group) => total + group.models.length, 0))

function resetFilters() {
  selectedPlatform.value = 'all'
  selectedGroupId.value = 'all'
  selectedRate.value = 'all'
  searchQuery.value = ''
}
</script>

<style scoped>
.plaza-description {
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.plaza-description :deep(h1),
.plaza-description :deep(h2),
.plaza-description :deep(h3) {
  @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white;
}

.plaza-description :deep(p) {
  @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300;
}

.plaza-description :deep(ul) {
  @apply mb-2 list-disc pl-5;
}

.plaza-description :deep(ol) {
  @apply mb-2 list-decimal pl-5;
}

.plaza-description :deep(li) {
  @apply mb-0.5 text-gray-700 dark:text-dark-200;
}

.plaza-description :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs dark:bg-dark-800;
}

.plaza-description :deep(blockquote) {
  @apply my-2 border-l-4 border-gray-300 pl-3 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
</style>

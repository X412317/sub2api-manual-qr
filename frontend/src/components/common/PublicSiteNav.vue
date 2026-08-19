<template>
  <header class="sticky top-0 z-40 border-b border-gray-200/80 bg-white/95 backdrop-blur dark:border-dark-800 dark:bg-dark-950/95">
    <div class="mx-auto flex h-16 max-w-7xl items-center gap-2 px-4 sm:gap-4 sm:px-6 lg:px-8">
      <RouterLink to="/home" class="flex min-w-0 shrink-0 items-center gap-2.5" :aria-label="siteName">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <img :src="siteLogo || '/logo.svg'" alt="" class="h-full w-full object-contain" />
        </span>
        <span class="hidden max-w-40 truncate text-sm font-semibold text-gray-950 dark:text-white md:block">
          {{ siteName }}
        </span>
      </RouterLink>

      <nav class="ml-1 flex h-full min-w-0 items-center sm:ml-3" :aria-label="t('publicNav.primary')">
        <RouterLink
          to="/docs"
          class="relative flex h-full items-center px-2.5 text-sm font-medium transition-colors sm:px-3"
          :class="active === 'docs' ? activeClass : idleClass"
          :aria-current="active === 'docs' ? 'page' : undefined"
        >
          {{ t('publicNav.docs') }}
        </RouterLink>
        <RouterLink
          v-if="showModelPlaza"
          to="/model-plaza"
          class="relative flex h-full items-center whitespace-nowrap px-2.5 text-sm font-medium transition-colors sm:px-3"
          :class="active === 'models' ? activeClass : idleClass"
          :aria-current="active === 'models' ? 'page' : undefined"
        >
          {{ t('publicNav.models') }}
        </RouterLink>
      </nav>

      <div class="ml-auto flex shrink-0 items-center gap-1 sm:gap-2">
        <LocaleSwitcher />
        <button
          type="button"
          class="flex h-9 w-9 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
          :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          @click="toggleTheme"
        >
          <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
        </button>
        <RouterLink
          :to="authTarget"
          class="inline-flex h-9 items-center justify-center gap-1.5 rounded-md bg-gray-950 px-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200 sm:px-3"
        >
          <Icon :name="isAuthenticated ? 'grid' : 'login'" size="sm" />
          <span class="hidden lg:inline">
            {{ isAuthenticated ? t('publicNav.console') : t('publicNav.login') }}
          </span>
        </RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

defineProps<{
  active: 'docs' | 'models'
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const isDark = ref(document.documentElement.classList.contains('dark'))

const settings = computed(() => appStore.cachedPublicSettings)
const siteName = computed(() => settings.value?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() =>
  sanitizeUrl(settings.value?.site_logo || appStore.siteLogo || '', {
    allowRelative: true,
    allowDataUrl: true
  })
)
const showModelPlaza = computed(() => settings.value?.model_plaza_enabled !== false)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const authTarget = computed(() => {
  if (!isAuthenticated.value) return { path: '/login', query: { redirect: window.location.pathname } }
  return authStore.isAdmin ? '/admin/dashboard' : '/dashboard'
})

const activeClass = 'text-gray-950 after:absolute after:inset-x-2.5 after:bottom-0 after:h-0.5 after:bg-primary-500 dark:text-white sm:after:inset-x-3'
const idleClass = 'text-gray-500 hover:text-gray-950 dark:text-dark-400 dark:hover:text-white'

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
})
</script>

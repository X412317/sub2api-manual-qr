<template>
  <div class="min-h-screen bg-white text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <PublicSiteNav active="docs" />

    <header class="border-b border-gray-200 bg-gray-50 dark:border-dark-800 dark:bg-dark-900/35">
      <div class="mx-auto max-w-7xl px-4 py-9 sm:px-6 sm:py-11 lg:px-8">
        <p class="mb-2 text-xs font-semibold uppercase text-primary-600 dark:text-primary-400">
          {{ t('apiDocs.eyebrow') }}
        </p>
        <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <h1 class="text-3xl font-bold text-gray-950 dark:text-white sm:text-4xl">
              {{ t('apiDocs.title', { siteName }) }}
            </h1>
            <p class="mt-3 max-w-2xl text-base leading-7 text-gray-600 dark:text-dark-300">
              {{ t('apiDocs.subtitle') }}
            </p>
          </div>
          <RouterLink
            :to="keyTarget"
            class="inline-flex h-10 w-fit items-center gap-2 rounded-md bg-gray-950 px-4 text-sm font-medium text-white transition-colors hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-200"
          >
            <Icon name="key" size="sm" />
            {{ isAuthenticated ? t('apiDocs.actions.manageKeys') : t('apiDocs.actions.getKey') }}
          </RouterLink>
        </div>

        <div class="mt-7 flex max-w-3xl items-center gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 dark:border-dark-700 dark:bg-dark-900">
          <span class="shrink-0 text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t('apiDocs.baseUrl') }}
          </span>
          <code class="min-w-0 flex-1 truncate text-sm text-gray-900 dark:text-gray-100">{{ apiBaseUrl }}</code>
          <CopyButton
            :text="apiBaseUrl"
            :copy-label="t('apiDocs.actions.copy')"
            :copied-label="t('apiDocs.actions.copied')"
            compact
            class="text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
          />
        </div>
      </div>
    </header>

    <nav class="sticky top-16 z-30 overflow-x-auto border-b border-gray-200 bg-white lg:hidden dark:border-dark-800 dark:bg-dark-950" :aria-label="t('apiDocs.onThisPage')">
      <div class="flex min-w-max px-4 sm:px-6">
        <a
          v-for="item in sectionLinks"
          :key="item.id"
          :href="`#${item.id}`"
          class="border-b-2 px-3 py-3 text-sm font-medium transition-colors"
          :class="activeSection === item.id ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 dark:text-dark-400'"
        >
          {{ t(item.label) }}
        </a>
      </div>
    </nav>

    <div class="mx-auto grid max-w-7xl gap-10 px-4 py-10 sm:px-6 lg:grid-cols-[200px_minmax(0,1fr)] lg:px-8 lg:py-12 xl:gap-16">
      <aside class="hidden lg:block">
        <nav class="sticky top-24 space-y-1" :aria-label="t('apiDocs.onThisPage')">
          <p class="mb-3 px-3 text-xs font-semibold uppercase text-gray-400 dark:text-dark-500">
            {{ t('apiDocs.onThisPage') }}
          </p>
          <a
            v-for="item in sectionLinks"
            :key="item.id"
            :href="`#${item.id}`"
            class="block rounded-md border-l-2 px-3 py-2 text-sm transition-colors"
            :class="activeSection === item.id ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-950/30 dark:text-primary-300' : 'border-transparent text-gray-500 hover:bg-gray-50 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-900 dark:hover:text-white'"
          >
            {{ t(item.label) }}
          </a>
        </nav>
      </aside>

      <main class="min-w-0 max-w-4xl">
        <section id="overview" class="docs-section scroll-mt-32 border-b border-gray-200 pb-12 dark:border-dark-800">
          <div class="mb-6 flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-md bg-blue-50 text-blue-600 dark:bg-blue-950/40 dark:text-blue-300">
              <Icon name="bolt" size="md" />
            </span>
            <div>
              <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.quickStart.title') }}</h2>
              <p class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">{{ t('apiDocs.quickStart.description') }}</p>
            </div>
          </div>

          <ol class="grid border-y border-gray-200 sm:grid-cols-3 dark:border-dark-800">
            <li v-for="(step, index) in quickSteps" :key="step.title" class="relative py-5 sm:px-5 sm:first:pl-0 sm:last:pr-0" :class="index > 0 ? 'border-t border-gray-200 sm:border-l sm:border-t-0 dark:border-dark-800' : ''">
              <span class="text-xs font-semibold text-primary-600 dark:text-primary-400">0{{ index + 1 }}</span>
              <h3 class="mt-2 text-sm font-semibold text-gray-950 dark:text-white">{{ t(step.title) }}</h3>
              <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t(step.description) }}</p>
            </li>
          </ol>

          <div class="mt-6 flex gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/25 dark:text-amber-200">
            <Icon name="shield" size="sm" class="mt-0.5 shrink-0" />
            <p class="leading-6">{{ t('apiDocs.quickStart.securityNote') }}</p>
          </div>
        </section>

        <section id="authentication" class="docs-section scroll-mt-32 border-b border-gray-200 py-12 dark:border-dark-800">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.authentication.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('apiDocs.authentication.description') }}</p>

          <div class="mt-6 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
            <div v-for="(auth, index) in authMethods" :key="auth.name" class="grid gap-2 px-4 py-4 sm:grid-cols-[150px_1fr] sm:items-center" :class="index > 0 ? 'border-t border-gray-200 dark:border-dark-700' : ''">
              <span class="text-sm font-medium text-gray-900 dark:text-white">{{ t(auth.name) }}</span>
              <code class="break-all rounded bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ auth.value }}</code>
            </div>
          </div>
        </section>

        <section id="first-request" class="docs-section scroll-mt-32 border-b border-gray-200 py-12 dark:border-dark-800">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.firstRequest.title') }}</h2>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('apiDocs.firstRequest.description') }}</p>
            </div>
            <RouterLink to="/model-plaza" class="inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300">
              {{ t('apiDocs.actions.viewModels') }}
              <Icon name="arrowRight" size="xs" />
            </RouterLink>
          </div>

          <div class="mt-6 inline-flex max-w-full overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-900" role="tablist" :aria-label="t('apiDocs.firstRequest.protocol')">
            <button
              v-for="protocol in protocols"
              :key="protocol.id"
              type="button"
              role="tab"
              class="min-h-9 whitespace-nowrap rounded-md px-3 text-sm font-medium transition-colors"
              :class="selectedProtocol === protocol.id ? 'bg-white text-gray-950 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'"
              :aria-selected="selectedProtocol === protocol.id"
              @click="selectedProtocol = protocol.id"
            >
              {{ t(protocol.label) }}
            </button>
          </div>

          <DocsCodeBlock
            class="mt-4"
            :code="requestExample"
            label="cURL"
            :copy-label="t('apiDocs.actions.copyCode')"
            :copied-label="t('apiDocs.actions.copied')"
          />
        </section>

        <section id="endpoints" class="docs-section scroll-mt-32 border-b border-gray-200 py-12 dark:border-dark-800">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.endpoints.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('apiDocs.endpoints.description') }}</p>

          <div class="mt-6 hidden overflow-hidden rounded-lg border border-gray-200 sm:block dark:border-dark-700">
            <table class="w-full text-left text-sm">
              <thead class="bg-gray-50 text-xs font-semibold text-gray-500 dark:bg-dark-900 dark:text-dark-400">
                <tr>
                  <th class="px-4 py-3">{{ t('apiDocs.endpoints.method') }}</th>
                  <th class="px-4 py-3">{{ t('apiDocs.endpoints.path') }}</th>
                  <th class="px-4 py-3">{{ t('apiDocs.endpoints.purpose') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
                <tr v-for="endpoint in endpoints" :key="endpoint.path">
                  <td class="px-4 py-3"><span class="rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold" :class="methodClass(endpoint.method)">{{ endpoint.method }}</span></td>
                  <td class="px-4 py-3"><code class="text-xs text-gray-900 dark:text-gray-100">{{ endpoint.path }}</code></td>
                  <td class="px-4 py-3 text-gray-500 dark:text-dark-400">{{ t(endpoint.description) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="mt-6 divide-y divide-gray-200 border-y border-gray-200 sm:hidden dark:divide-dark-800 dark:border-dark-800">
            <div v-for="endpoint in endpoints" :key="`mobile-${endpoint.path}`" class="py-4">
              <div class="flex items-center gap-2">
                <span class="rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold" :class="methodClass(endpoint.method)">{{ endpoint.method }}</span>
                <code class="break-all text-xs text-gray-900 dark:text-gray-100">{{ endpoint.path }}</code>
              </div>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t(endpoint.description) }}</p>
            </div>
          </div>
        </section>

        <section id="sdk" class="docs-section scroll-mt-32 border-b border-gray-200 py-12 dark:border-dark-800">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.sdk.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('apiDocs.sdk.description') }}</p>

          <div class="mt-6 inline-flex max-w-full overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-900" role="tablist" :aria-label="t('apiDocs.sdk.title')">
            <button
              v-for="sdk in sdks"
              :key="sdk.id"
              type="button"
              role="tab"
              class="min-h-9 whitespace-nowrap rounded-md px-3 text-sm font-medium transition-colors"
              :class="selectedSdk === sdk.id ? 'bg-white text-gray-950 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'"
              :aria-selected="selectedSdk === sdk.id"
              @click="selectedSdk = sdk.id"
            >
              {{ sdk.label }}
            </button>
          </div>
          <DocsCodeBlock
            class="mt-4"
            :code="sdkExample"
            :label="selectedSdkLabel"
            :copy-label="t('apiDocs.actions.copyCode')"
            :copied-label="t('apiDocs.actions.copied')"
          />
        </section>

        <section id="errors" class="docs-section scroll-mt-32 border-b border-gray-200 py-12 dark:border-dark-800">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.errors.title') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ t('apiDocs.errors.description') }}</p>
          <div class="mt-6 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
            <div v-for="(error, index) in errors" :key="error.code" class="grid grid-cols-[64px_1fr] gap-3 px-4 py-3.5 text-sm" :class="index > 0 ? 'border-t border-gray-200 dark:border-dark-700' : ''">
              <code class="font-semibold text-gray-900 dark:text-white">{{ error.code }}</code>
              <span class="text-gray-500 dark:text-dark-400">{{ t(error.description) }}</span>
            </div>
          </div>
        </section>

        <section id="faq" class="docs-section scroll-mt-32 pt-12">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('apiDocs.faq.title') }}</h2>
          <div class="mt-5 divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-800 dark:border-dark-800">
            <details v-for="item in faqItems" :key="item.question" class="group py-4">
              <summary class="flex cursor-pointer list-none items-center justify-between gap-4 text-sm font-medium text-gray-900 dark:text-white">
                {{ t(item.question) }}
                <Icon name="chevronDown" size="sm" class="shrink-0 text-gray-400 transition-transform group-open:rotate-180" />
              </summary>
              <p class="mt-3 pr-8 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t(item.answer) }}</p>
            </details>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import PublicSiteNav from '@/components/common/PublicSiteNav.vue'
import CopyButton from '@/components/common/CopyButton.vue'
import DocsCodeBlock from '@/components/docs/DocsCodeBlock.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { sanitizeUrl } from '@/utils/url'

type ProtocolId = 'responses' | 'chat' | 'anthropic' | 'gemini'
type SdkId = 'python-openai' | 'node-openai' | 'python-anthropic'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const selectedProtocol = ref<ProtocolId>('responses')
const selectedSdk = ref<SdkId>('python-openai')
const activeSection = ref('overview')
let sectionObserver: IntersectionObserver | null = null

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const apiBaseUrl = computed(() => {
  const configured = sanitizeUrl(appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl || '')
  const normalized = (configured || window.location.origin).replace(/\/+$/, '')
  return /\/v1$/i.test(normalized) ? normalized : `${normalized}/v1`
})
const gatewayRootUrl = computed(() => apiBaseUrl.value.replace(/\/v1$/i, ''))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const keyTarget = computed(() => isAuthenticated.value ? '/keys' : { path: '/login', query: { redirect: '/keys' } })

const sectionLinks = [
  { id: 'overview', label: 'apiDocs.nav.overview' },
  { id: 'authentication', label: 'apiDocs.nav.authentication' },
  { id: 'first-request', label: 'apiDocs.nav.firstRequest' },
  { id: 'endpoints', label: 'apiDocs.nav.endpoints' },
  { id: 'sdk', label: 'apiDocs.nav.sdk' },
  { id: 'errors', label: 'apiDocs.nav.errors' },
  { id: 'faq', label: 'apiDocs.nav.faq' }
]

const quickSteps = [
  { title: 'apiDocs.quickStart.step1Title', description: 'apiDocs.quickStart.step1Description' },
  { title: 'apiDocs.quickStart.step2Title', description: 'apiDocs.quickStart.step2Description' },
  { title: 'apiDocs.quickStart.step3Title', description: 'apiDocs.quickStart.step3Description' }
]

const authMethods = [
  { name: 'apiDocs.authentication.bearer', value: 'Authorization: Bearer sk-your-api-key' },
  { name: 'apiDocs.authentication.anthropic', value: 'x-api-key: sk-your-api-key' },
  { name: 'apiDocs.authentication.google', value: 'x-goog-api-key: sk-your-api-key' }
]

const protocols: Array<{ id: ProtocolId; label: string }> = [
  { id: 'responses', label: 'apiDocs.firstRequest.responses' },
  { id: 'chat', label: 'apiDocs.firstRequest.chat' },
  { id: 'anthropic', label: 'apiDocs.firstRequest.anthropic' },
  { id: 'gemini', label: 'apiDocs.firstRequest.gemini' }
]

const endpoints = [
  { method: 'GET', path: '/v1/models', description: 'apiDocs.endpoints.models' },
  { method: 'POST', path: '/v1/responses', description: 'apiDocs.endpoints.responses' },
  { method: 'POST', path: '/v1/chat/completions', description: 'apiDocs.endpoints.chat' },
  { method: 'POST', path: '/v1/messages', description: 'apiDocs.endpoints.messages' },
  { method: 'POST', path: '/v1/messages/count_tokens', description: 'apiDocs.endpoints.countTokens' },
  { method: 'POST', path: '/v1beta/models/{model}:generateContent', description: 'apiDocs.endpoints.gemini' },
  { method: 'POST', path: '/v1/images/generations', description: 'apiDocs.endpoints.images' },
  { method: 'POST', path: '/v1/embeddings', description: 'apiDocs.endpoints.embeddings' }
]

const sdks: Array<{ id: SdkId; label: string }> = [
  { id: 'python-openai', label: 'Python · OpenAI' },
  { id: 'node-openai', label: 'Node.js · OpenAI' },
  { id: 'python-anthropic', label: 'Python · Anthropic' }
]

const errors = [
  { code: '400', description: 'apiDocs.errors.badRequest' },
  { code: '401', description: 'apiDocs.errors.unauthorized' },
  { code: '403', description: 'apiDocs.errors.forbidden' },
  { code: '429', description: 'apiDocs.errors.rateLimited' },
  { code: '5xx', description: 'apiDocs.errors.server' }
]

const faqItems = [
  { question: 'apiDocs.faq.modelQuestion', answer: 'apiDocs.faq.modelAnswer' },
  { question: 'apiDocs.faq.balanceQuestion', answer: 'apiDocs.faq.balanceAnswer' },
  { question: 'apiDocs.faq.streamQuestion', answer: 'apiDocs.faq.streamAnswer' },
  { question: 'apiDocs.faq.errorQuestion', answer: 'apiDocs.faq.errorAnswer' }
]

const requestExample = computed(() => {
  const base = apiBaseUrl.value
  if (selectedProtocol.value === 'chat') {
    return `curl ${base}/chat/completions \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'`
  }
  if (selectedProtocol.value === 'anthropic') {
    return `curl ${base}/messages \\
  -H "x-api-key: sk-your-api-key" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'`
  }
  if (selectedProtocol.value === 'gemini') {
    return `curl "${gatewayRootUrl.value}/v1beta/models/your-model-id:generateContent" \\
  -H "x-goog-api-key: sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [{"parts": [{"text": "Hello"}]}]
  }'`
  }
  return `curl ${base}/responses \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "input": "Hello",
    "stream": true
  }'`
})

const sdkExample = computed(() => {
  const base = apiBaseUrl.value
  if (selectedSdk.value === 'node-openai') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.SUB2API_KEY,
  baseURL: "${base}",
});

const response = await client.responses.create({
  model: "your-model-id",
  input: "Hello",
});

console.log(response.output_text);`
  }
  if (selectedSdk.value === 'python-anthropic') {
    return `import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ["SUB2API_KEY"],
    base_url="${gatewayRootUrl.value}",
)

message = client.messages.create(
    model="your-model-id",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
)

print(message.content[0].text)`
  }
  return `import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["SUB2API_KEY"],
    base_url="${base}",
)

response = client.responses.create(
    model="your-model-id",
    input="Hello",
)

print(response.output_text)`
})

const selectedSdkLabel = computed(() => sdks.find((sdk) => sdk.id === selectedSdk.value)?.label || 'SDK')

function methodClass(method: string): string {
  return method === 'GET'
    ? 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
    : 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
}

onMounted(() => {
  if (!appStore.publicSettingsLoaded) void appStore.fetchPublicSettings()
  const sections = sectionLinks
    .map((item) => document.getElementById(item.id))
    .filter((section): section is HTMLElement => section !== null)

  sectionObserver = new IntersectionObserver((entries) => {
    const visible = entries
      .filter((entry) => entry.isIntersecting)
      .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
    if (visible[0]?.target.id) activeSection.value = visible[0].target.id
  }, { rootMargin: '-96px 0px -65% 0px', threshold: 0 })

  sections.forEach((section) => sectionObserver?.observe(section))
})

onBeforeUnmount(() => sectionObserver?.disconnect())
</script>

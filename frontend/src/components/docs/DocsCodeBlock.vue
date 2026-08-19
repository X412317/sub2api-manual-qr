<template>
  <div class="overflow-hidden rounded-lg border border-gray-800 bg-gray-950 shadow-sm dark:border-dark-700">
    <div class="flex h-10 items-center justify-between border-b border-white/10 px-3.5">
      <span class="text-xs font-medium text-gray-400">{{ label }}</span>
      <CopyButton
        :text="code"
        :copy-label="copyLabel"
        :copied-label="copiedLabel"
        compact
        class="text-gray-400 hover:bg-white/10 hover:text-white"
      />
    </div>
    <pre class="overflow-x-auto p-4 text-[13px] leading-6 text-gray-200"><code>{{ code }}</code></pre>
  </div>
</template>

<script setup lang="ts">
import CopyButton from '@/components/common/CopyButton.vue'

withDefaults(defineProps<{
  code: string
  label?: string
  copyLabel?: string
  copiedLabel?: string
}>(), {
  label: 'Example',
  copyLabel: 'Copy',
  copiedLabel: 'Copied'
})
</script>

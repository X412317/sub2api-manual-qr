<template>
  <button
    type="button"
    class="inline-flex shrink-0 items-center justify-center rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500/40"
    :class="compact ? 'h-7 w-7' : 'h-9 w-9'"
    :title="copied ? copiedLabel : copyLabel"
    :aria-label="copied ? copiedLabel : copyLabel"
    @click="copy"
  >
    <Icon
      :name="copied ? 'check' : 'copy'"
      :size="compact ? 'xs' : 'sm'"
      :class="copied ? 'text-emerald-500' : 'text-current'"
    />
  </button>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  text: string
  copyLabel?: string
  copiedLabel?: string
  compact?: boolean
}>(), {
  copyLabel: 'Copy',
  copiedLabel: 'Copied',
  compact: false
})

const copied = ref(false)
let resetTimer: ReturnType<typeof setTimeout> | null = null

function fallbackCopy(text: string): boolean {
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.cssText = 'position:fixed;left:0;top:0;width:1px;height:1px;opacity:0;pointer-events:none'
  document.body.appendChild(textarea)
  textarea.select()
  const success = document.execCommand('copy')
  document.body.removeChild(textarea)
  return success
}

async function copy() {
  if (!props.text) return

  let success = false
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(props.text)
      success = true
    } else {
      success = fallbackCopy(props.text)
    }
  } catch {
    success = fallbackCopy(props.text)
  }

  if (!success) return
  copied.value = true
  if (resetTimer) clearTimeout(resetTimer)
  resetTimer = setTimeout(() => {
    copied.value = false
    resetTimer = null
  }, 1800)
}

onBeforeUnmount(() => {
  if (resetTimer) clearTimeout(resetTimer)
})
</script>

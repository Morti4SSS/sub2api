<template>
  <div class="inline-flex min-w-[4.25rem] flex-col gap-0.5 text-[11px] leading-4" :title="title">
    <div class="flex items-center gap-1">
      <span :class="dotClass(accessOk)" />
      <span class="font-mono text-gray-700 dark:text-gray-200">AT {{ accessText }}</span>
    </div>
    <div class="flex items-center gap-1">
      <span :class="dotClass(refreshOk)" />
      <span class="font-mono text-gray-700 dark:text-gray-200">RT {{ refreshText }}</span>
    </div>
    <div :class="stateClass">{{ stateText }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()

const status = computed(() => props.account.token_status ?? null)
const accessOk = computed(() => status.value?.access_token === 'present')
const refreshOk = computed(() => status.value?.refresh_token === 'present')
const accessText = computed(() => accessOk.value ? 'ok' : '-')
const refreshText = computed(() => refreshOk.value ? 'ok' : '-')

const stateText = computed(() => {
  switch (status.value?.refresh_state) {
    case 'auto':
      return 'auto'
    case 'manual':
      return 'manual'
    case 'failed':
      return 'fail'
    default:
      return '-'
  }
})

const stateClass = computed(() => {
  if (status.value?.refresh_state === 'failed') return 'font-mono text-red-600 dark:text-red-300'
  if (status.value?.refresh_state === 'auto') return 'font-mono text-emerald-600 dark:text-emerald-300'
  return 'font-mono text-gray-500 dark:text-dark-400'
})

const title = computed(() => status.value?.message || stateText.value)

function dotClass(ok: boolean) {
  return [
    'h-1.5 w-1.5 rounded-full',
    ok ? 'bg-emerald-500' : 'bg-amber-500'
  ]
}
</script>

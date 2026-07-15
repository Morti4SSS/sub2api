<template>
  <div class="inline-flex min-w-[8.5rem] max-w-[12rem] flex-col gap-0.5 text-[11px] leading-4" :title="title">
    <div class="flex items-center gap-2 font-mono text-gray-700 dark:text-gray-200">
      <span class="flex items-center gap-1"><span :class="dotClass(accessOk)" />AT {{ accessText }}</span>
      <span class="flex items-center gap-1"><span :class="dotClass(refreshOk)" />RT {{ refreshText }}</span>
    </div>
    <div :class="resultClass" class="flex items-center gap-1 font-mono">
      <span :class="resultDotClass" />
      <span>{{ resultText }}</span>
      <span v-if="triggerText" class="text-gray-400 dark:text-dark-400">/ {{ triggerText }}</span>
    </div>
    <div v-if="status?.last_attempt_at" class="whitespace-nowrap text-gray-500 dark:text-dark-400">
      {{ t('admin.accounts.tokenStatus.lastAttempt') }} {{ compactDateTime(status.last_attempt_at) }}
    </div>
    <div v-if="status?.next_window_start" class="whitespace-nowrap text-gray-500 dark:text-dark-400">
      {{ t('admin.accounts.tokenStatus.nextWindow') }} {{ compactDateTime(status.next_window_start) }}<template v-if="status.next_window_end">-{{ compactTime(status.next_window_end) }}</template>
    </div>
    <div v-if="status?.error" class="truncate text-red-600 dark:text-red-300">
      {{ status.error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'

const props = defineProps<{ account: Account }>()
const { t } = useI18n()

const status = computed(() => props.account.token_status ?? null)
const accessOk = computed(() => status.value?.access_token === 'present')
const refreshOk = computed(() => status.value?.refresh_token === 'present')
const accessText = computed(() => accessOk.value ? 'ok' : '-')
const refreshText = computed(() => refreshOk.value ? 'ok' : '-')

const resultText = computed(() => {
  switch (status.value?.last_result) {
    case 'success':
      return t('admin.accounts.tokenStatus.success')
    case 'failed':
      return t('admin.accounts.tokenStatus.failed')
    default:
      return t('admin.accounts.tokenStatus.noRecord')
  }
})

const triggerText = computed(() => {
  if (status.value?.trigger === 'manual') return t('admin.accounts.tokenStatus.manual')
  if (status.value?.trigger === 'background') return t('admin.accounts.tokenStatus.background')
  return ''
})

const resultClass = computed(() => {
  if (status.value?.last_result === 'failed') return 'text-red-600 dark:text-red-300'
  if (status.value?.last_result === 'success') return 'text-emerald-600 dark:text-emerald-300'
  return 'text-gray-500 dark:text-dark-400'
})

const resultDotClass = computed(() => [
  'h-1.5 w-1.5 rounded-full',
  status.value?.last_result === 'success'
    ? 'bg-emerald-500'
    : status.value?.last_result === 'failed'
      ? 'bg-red-500'
      : 'bg-gray-400'
])

const title = computed(() => {
  const parts = [resultText.value, triggerText.value]
  if (status.value?.last_attempt_at) parts.push(`${t('admin.accounts.tokenStatus.lastAttempt')} ${compactDateTime(status.value.last_attempt_at)}`)
  if (status.value?.next_window_start) parts.push(`${t('admin.accounts.tokenStatus.nextWindow')} ${compactDateTime(status.value.next_window_start)}`)
  if (status.value?.error) parts.push(status.value.error)
  return parts.filter(Boolean).join(' | ')
})

function dotClass(ok: boolean) {
  return [
    'h-1.5 w-1.5 rounded-full',
    ok ? 'bg-emerald-500' : 'bg-amber-500'
  ]
}

function compactDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function compactTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function pad(value: number) {
  return String(value).padStart(2, '0')
}
</script>

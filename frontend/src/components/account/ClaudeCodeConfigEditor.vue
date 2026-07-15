<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <h3 class="input-label mb-0 text-sm font-semibold">
        {{ t('admin.accounts.anthropic.claudeCodeMapping') }}
      </h3>
      <div class="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          class="btn btn-secondary inline-flex items-center gap-1.5 px-3 py-1.5 text-sm"
          :disabled="isSyncingUpstream || !canSyncUpstream"
          data-testid="claude-code-sync-models"
          @click="syncUpstreamModels"
        >
          <Icon name="sync" size="sm" :class="isSyncingUpstream ? 'animate-spin' : ''" />
          {{ isSyncingUpstream
            ? t('admin.accounts.anthropic.claudeCodeSyncModelsLoading')
            : t('admin.accounts.anthropic.claudeCodeSyncModels') }}
        </button>
        <button
          type="button"
          class="btn btn-secondary inline-flex items-center gap-1.5 px-3 py-1.5 text-sm"
          @click="addRoute"
        >
          <Icon name="plus" size="sm" />
          {{ t('admin.accounts.anthropic.claudeCodeAddModel') }}
        </button>
      </div>
    </div>

    <details class="border-b border-gray-100 pb-3 dark:border-dark-700">
      <summary class="cursor-pointer text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.accounts.anthropic.claudeCodeModelSyncAdvanced') }}
      </summary>
      <div class="mt-3">
        <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeModelsURL') }}</label>
        <input
          v-model="upstreamModelsUrl"
          type="url"
          class="input"
          data-testid="claude-code-models-url"
          :placeholder="t('admin.accounts.anthropic.claudeCodeModelsURLPlaceholder')"
        />
      </div>
    </details>

    <div
      v-if="syncResult"
      data-testid="claude-code-sync-result"
      :class="syncResult.kind === 'success'
        ? 'text-xs text-emerald-700 dark:text-emerald-400'
        : 'text-xs text-red-700 dark:text-red-400'"
    >
      <p>{{ syncResult.message }}</p>
      <div v-if="syncResult.kind === 'error'" class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-gray-600 dark:text-gray-300">
        <span v-if="syncResult.requestURL">URL: {{ syncResult.requestURL }}</span>
        <span v-if="syncResult.httpStatus">HTTP: {{ syncResult.httpStatus }}</span>
        <span v-if="syncResult.contentType">Content-Type: {{ syncResult.contentType }}</span>
        <span v-if="syncResult.responseShape">Shape: {{ syncResult.responseShape }}</span>
      </div>
    </div>

    <div
      v-for="(entry, index) in routes"
      :key="index"
      class="space-y-3 border-t border-gray-200 pt-3 first:border-t-0 first:pt-0 dark:border-dark-600"
    >
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-2 xl:grid-cols-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeRequestModel') }}</label>
          <select
            v-model="entry.shellModel"
            class="input"
            :data-testid="`claude-code-shell-${index}`"
            @change="applyShellDefaults(entry)"
          >
            <option value="">{{ t('common.selectOption') }}</option>
            <option
              v-if="entry.shellModel && !shellModelIDs.has(entry.shellModel)"
              :value="entry.shellModel"
            >
              {{ entry.shellModel }}
            </option>
            <option v-for="model in shellModels" :key="model.id" :value="model.id">
              {{ model.display_name }} ({{ model.id }})
            </option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeUpstreamModel') }}</label>
          <input
            v-model="entry.upstreamModel"
            type="text"
            class="input"
            :list="upstreamModelOptions.length > 0 ? 'claude-code-upstream-models' : undefined"
            :data-testid="`claude-code-upstream-model-${index}`"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeDisplayName') }}</label>
          <input v-model="entry.displayName" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeContextWindow') }}</label>
          <div class="flex gap-2">
            <input
              v-model="entry.contextWindow"
              type="number"
              min="1"
              step="1"
              class="input min-w-0 flex-1"
            />
            <select class="input w-24" value="" @change="setContextPreset(entry, $event)">
              <option value="">{{ t('admin.accounts.anthropic.claudeCodeContextPreset') }}</option>
              <option value="128000">128K</option>
              <option value="200000">200K</option>
              <option value="1000000">1M</option>
            </select>
          </div>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
          <input
            v-model="entry.thinkingEnabled"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
            :data-testid="`claude-code-thinking-enabled-${index}`"
          />
          {{ t('admin.accounts.anthropic.claudeCodeThinkingEnabled') }}
        </label>
        <p v-if="modelPreview(entry)" class="text-xs text-gray-500 dark:text-gray-400">
          {{ modelPreview(entry) }}
        </p>
        <button
          type="button"
          class="inline-flex h-8 w-8 items-center justify-center text-red-500 hover:text-red-700"
          :title="t('common.delete')"
          @click="removeRoute(index)"
        >
          <Icon name="trash" size="sm" />
        </button>
      </div>

      <div v-if="entry.thinkingEnabled" class="space-y-3">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div v-if="entry.thinkingMode === 'levels'">
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeEffortTarget') }}</label>
            <select v-model="entry.targetField" class="input">
              <option value="output_config.effort">output_config.effort</option>
              <option value="reasoning_effort">reasoning_effort</option>
              <option value="reasoning.effort">reasoning.effort</option>
            </select>
          </div>
          <div>
            <label class="input-label">
              {{ entry.thinkingMode === 'budget'
                ? t('admin.accounts.anthropic.claudeCodeOrderedBudgets')
                : t('admin.accounts.anthropic.claudeCodeOrderedLevels') }}
            </label>
            <input
              :value="entry.levels.join(', ')"
              type="text"
              class="input"
              @input="updateLevels(entry, $event)"
            />
          </div>
        </div>

        <p v-if="effortSummary(entry)" class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.anthropic.claudeCodeEffortSummary', { summary: effortSummary(entry) }) }}
        </p>

        <details class="border-t border-gray-100 pt-2 dark:border-dark-700">
          <summary class="cursor-pointer text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.accounts.anthropic.claudeCodeAdvancedSettings') }}
          </summary>
          <div class="mt-3 space-y-3">
            <div class="max-w-xs">
              <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeThinkingMode') }}</label>
              <select v-model="entry.thinkingMode" class="input" @change="applyThinkingMode(entry)">
                <option value="levels">{{ t('admin.accounts.anthropic.claudeCodeThinkingModeLevels') }}</option>
                <option value="budget">{{ t('admin.accounts.anthropic.claudeCodeThinkingModeBudget') }}</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeOverrides') }}</label>
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-5">
                <div v-for="level in effortLevels" :key="level">
                  <label class="mb-1 block text-xs text-gray-500">{{ level }}</label>
                  <input v-model="entry.overrides[level]" type="text" class="input" />
                </div>
              </div>
            </div>
          </div>
        </details>
      </div>
    </div>

    <datalist id="claude-code-upstream-models">
      <option v-for="model in upstreamModelOptions" :key="model" :value="model" />
    </datalist>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  accountsAPI,
  getClaudeCodeOptions,
  type ClaudeCodeConfigOptions,
  type SyncUpstreamPreviewParams
} from '@/api/admin/accounts'
import Icon from '@/components/icons/Icon.vue'
import {
  buildClaudeCodeEffortSummary,
  claudeCodeEffortLevels,
  createEmptyClaudeCodeRoute,
  type ClaudeCodeRouteForm
} from './claudeCodeConfig'

const props = withDefaults(defineProps<{
  upstreamModels?: string[]
  accountId?: number
  syncCredentials?: SyncUpstreamPreviewParams
}>(), {
  upstreamModels: () => []
})

const routes = defineModel<ClaudeCodeRouteForm[]>('routes', { required: true })
const upstreamModelsUrl = defineModel<string>('upstreamModelsUrl', { default: '' })
const { t } = useI18n()
const shellModels = ref<ClaudeCodeConfigOptions['models']>([])
const syncedUpstreamModels = ref<string[]>([])
const isSyncingUpstream = ref(false)
const syncResult = ref<{
  kind: 'success' | 'error'
  message: string
  requestURL?: string
  httpStatus?: string
  contentType?: string
  responseShape?: string
} | null>(null)
const effortLevels = claudeCodeEffortLevels
const shellModelIDs = computed(() => new Set(shellModels.value.map(model => model.id)))
const canSyncUpstream = computed(() => Boolean(props.accountId || props.syncCredentials))
const upstreamModelOptions = computed(() => Array.from(new Set([
  ...props.upstreamModels.map(model => model.trim()).filter(Boolean),
  ...syncedUpstreamModels.value
])))

const loadOptions = async () => {
  try {
    const options = await getClaudeCodeOptions()
    shellModels.value = options.models
  } catch {
    shellModels.value = []
  }
}

onMounted(loadOptions)

const syncUpstreamModels = async () => {
  if (isSyncingUpstream.value || !canSyncUpstream.value) return
  isSyncingUpstream.value = true
  syncResult.value = null
  try {
    const result = props.accountId
      ? await accountsAPI.syncUpstreamModels(props.accountId, upstreamModelsUrl.value)
      : await accountsAPI.syncUpstreamModelsPreview({
          ...props.syncCredentials!,
          upstream_models_url: upstreamModelsUrl.value.trim() || undefined
        })
    syncedUpstreamModels.value = Array.from(new Set(result.models.map(model => model.trim()).filter(Boolean)))
    syncResult.value = {
      kind: 'success',
      message: t('admin.accounts.anthropic.claudeCodeSyncModelsSuccess', { count: syncedUpstreamModels.value.length })
    }
  } catch (error: unknown) {
    const syncError = error && typeof error === 'object' ? error as Record<string, unknown> : {}
    const metadata = syncError.metadata && typeof syncError.metadata === 'object'
      ? syncError.metadata as Record<string, unknown>
      : {}
    syncResult.value = {
      kind: 'error',
      message: typeof syncError.message === 'string'
        ? syncError.message
        : t('admin.accounts.anthropic.claudeCodeSyncModelsFailed'),
      requestURL: typeof metadata.request_url === 'string' ? metadata.request_url : undefined,
      httpStatus: typeof metadata.http_status === 'string' ? metadata.http_status : undefined,
      contentType: typeof metadata.content_type === 'string' ? metadata.content_type : undefined,
      responseShape: typeof metadata.response_shape === 'string' ? metadata.response_shape : undefined
    }
  } finally {
    isSyncingUpstream.value = false
  }
}

const addRoute = () => routes.value.push(createEmptyClaudeCodeRoute())
const removeRoute = (index: number) => routes.value.splice(index, 1)

const applyShellDefaults = (entry: ClaudeCodeRouteForm) => {
  if (entry.displayName.trim()) return
  const selected = shellModels.value.find(model => model.id === entry.shellModel)
  if (selected) entry.displayName = selected.display_name
}

const setContextPreset = (entry: ClaudeCodeRouteForm, event: Event) => {
  const value = (event.target as HTMLSelectElement).value
  if (value) entry.contextWindow = value
}

const updateLevels = (entry: ClaudeCodeRouteForm, event: Event) => {
  entry.levels = (event.target as HTMLInputElement).value
    .split(',')
    .map(level => level.trim())
    .filter(Boolean)
}

const applyThinkingMode = (entry: ClaudeCodeRouteForm) => {
  if (entry.thinkingMode === 'budget') entry.targetField = 'thinking.budget_tokens'
  else if (entry.targetField === 'thinking.budget_tokens') entry.targetField = 'output_config.effort'
}

const effortSummary = (entry: ClaudeCodeRouteForm) =>
  buildClaudeCodeEffortSummary(entry.levels, entry.overrides)

const modelPreview = (entry: ClaudeCodeRouteForm) => {
  const shell = entry.shellModel.trim()
  const upstream = entry.upstreamModel.trim()
  if (!shell || !upstream) return ''
  return `${shell} -> ${upstream}`
}
</script>

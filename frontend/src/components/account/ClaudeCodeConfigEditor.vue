<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3">
      <h3 class="input-label mb-0 text-base font-semibold">
        {{ t('admin.accounts.anthropic.claudeCodeMapping') }}
      </h3>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.anthropic.claudeCodeMappingDesc') }}
      </p>
    </div>

    <div class="space-y-3">
      <div
        v-for="(entry, index) in catalog"
        :key="index"
        class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
      >
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeRole') }}</label>
            <input
              v-model="entry.role"
              type="text"
              class="input"
              placeholder="sonnet"
              :data-testid="`claude-code-catalog-role-${index}`"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeDisplayName') }}</label>
            <input
              v-model="entry.displayName"
              type="text"
              class="input"
              placeholder="Relay Sonnet"
              :data-testid="`claude-code-catalog-display-name-${index}`"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeRequestModel') }}</label>
            <input
              v-model="entry.requestModel"
              type="text"
              class="input"
              placeholder="cc-relay-sonnet"
              :data-testid="`claude-code-catalog-request-model-${index}`"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeUpstreamModel') }}</label>
            <input
              v-model="entry.upstreamModel"
              type="text"
              class="input"
              placeholder="provider-sonnet"
              :data-testid="`claude-code-catalog-upstream-model-${index}`"
            />
          </div>
        </div>
        <div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-[auto,1fr]">
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
            <input
              v-model="entry.supports1m"
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
              :data-testid="`claude-code-catalog-supports-1m-${index}`"
            />
            {{ t('admin.accounts.anthropic.claudeCodeSupports1m') }}
          </label>
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeCapabilities') }}</label>
            <input
              v-model="entry.capabilities"
              type="text"
              class="input"
              placeholder="thinking, effort"
              :data-testid="`claude-code-catalog-capabilities-${index}`"
            />
          </div>
        </div>
        <p
          v-if="modelPreview(entry)"
          class="mt-3 rounded-md bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
        >
          {{ modelPreview(entry) }}
        </p>
        <button
          v-if="catalog.length > 1"
          type="button"
          class="mt-3 text-sm text-red-600 hover:text-red-700 dark:text-red-400"
          @click="removeCatalog(index)"
        >
          {{ t('common.delete') }}
        </button>
      </div>
      <button type="button" class="btn btn-secondary text-sm" @click="addCatalog">
        + {{ t('admin.accounts.anthropic.claudeCodeAddModel') }}
      </button>
    </div>

    <div class="mt-5 space-y-3">
      <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeEffortMapping') }}</label>
      <div
        v-for="(entry, index) in effortMappings"
        :key="index"
        class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
      >
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeEffortModel') }}</label>
            <input
              v-model="entry.model"
              type="text"
              class="input"
              placeholder="cc-relay-sonnet"
              :data-testid="`claude-code-effort-model-${index}`"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.anthropic.claudeCodeEffortTarget') }}</label>
            <select
              v-model="entry.targetField"
              class="input"
              :data-testid="`claude-code-effort-target-field-${index}`"
            >
              <option value="output_config.effort">output_config.effort</option>
              <option value="thinking.budget_tokens">thinking.budget_tokens</option>
              <option value="reasoning_effort">reasoning_effort</option>
              <option value="reasoning.effort">reasoning.effort</option>
            </select>
          </div>
        </div>
        <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-5">
          <div v-for="level in effortLevels" :key="level">
            <label class="input-label">{{ level }}</label>
            <input
              v-model="entry[level]"
              type="text"
              class="input"
              :data-testid="`claude-code-effort-${level}-${index}`"
            />
          </div>
        </div>
        <p
          v-if="effortSummary(entry)"
          class="mt-3 rounded-md bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
        >
          {{ t('admin.accounts.anthropic.claudeCodeEffortSummary', { summary: effortSummary(entry) }) }}
        </p>
        <button
          v-if="effortMappings.length > 1"
          type="button"
          class="mt-3 text-sm text-red-600 hover:text-red-700 dark:text-red-400"
          @click="removeEffort(index)"
        >
          {{ t('common.delete') }}
        </button>
      </div>
      <button type="button" class="btn btn-secondary text-sm" @click="addEffort">
        + {{ t('admin.accounts.anthropic.claudeCodeAddEffort') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  buildClaudeCodeEffortSummary,
  createEmptyClaudeCodeCatalogEntry,
  createEmptyClaudeCodeEffortEntry,
  type ClaudeCodeCatalogForm,
  type ClaudeCodeEffortForm
} from './claudeCodeConfig'

const catalog = defineModel<ClaudeCodeCatalogForm[]>('catalog', { required: true })
const effortMappings = defineModel<ClaudeCodeEffortForm[]>('effortMappings', { required: true })

const { t } = useI18n()
const effortLevels = ['low', 'medium', 'high', 'xhigh', 'max'] as const

const modelPreview = (entry: ClaudeCodeCatalogForm) => {
  const visible = entry.displayName.trim() || entry.requestModel.trim()
  const requestModel = entry.requestModel.trim()
  if (!visible || !requestModel) return ''
  const upstream = entry.upstreamModel.trim() || requestModel
  return t('admin.accounts.anthropic.claudeCodeModelPreview', {
    visible,
    requestModel,
    upstream
  })
}

const effortSummary = (entry: ClaudeCodeEffortForm) => buildClaudeCodeEffortSummary(entry)

const addCatalog = () => {
  catalog.value.push(createEmptyClaudeCodeCatalogEntry())
}

const removeCatalog = (index: number) => {
  catalog.value.splice(index, 1)
}

const addEffort = () => {
  effortMappings.value.push(createEmptyClaudeCodeEffortEntry())
}

const removeEffort = (index: number) => {
  effortMappings.value.splice(index, 1)
}
</script>

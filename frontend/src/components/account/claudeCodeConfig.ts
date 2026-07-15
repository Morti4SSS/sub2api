export type ClaudeCodeEffortLevel = 'low' | 'medium' | 'high' | 'xhigh' | 'max'
export type ClaudeCodeThinkingMode = 'levels' | 'budget'

export interface ClaudeCodeRouteForm {
  shellModel: string
  upstreamModel: string
  displayName: string
  contextWindow: string
  thinkingEnabled: boolean
  thinkingMode: ClaudeCodeThinkingMode
  targetField: string
  levels: string[]
  overrides: Record<ClaudeCodeEffortLevel, string>
}

export const claudeCodeEffortLevels: ClaudeCodeEffortLevel[] = ['low', 'medium', 'high', 'xhigh', 'max']

export const buildClaudeCodeEffortSummary = (
  levels: string[],
  overrides: Partial<Record<ClaudeCodeEffortLevel, string>> = {}
) => {
  const mapping = buildClaudeCodeEffortMap(levels, overrides)
  const groups = new Map<string, string[]>()
  for (const level of claudeCodeEffortLevels) {
    const mapped = mapping[level]?.trim() || ''
    if (!mapped) continue
    const existing = groups.get(mapped) ?? []
    groups.set(mapped, existing.concat(level))
  }
  return Array.from(groups.entries())
    .map(([mapped, levels]) => `${levels.join('/')} -> ${mapped}`)
    .join('; ')
}

const stringValue = (value: unknown) => (typeof value === 'string' ? value : '').trim()

const stringListValue = (value: unknown): string[] => {
  if (!Array.isArray(value)) return []
  return value.map((item) => stringValue(item)).filter(Boolean)
}

export const createEmptyClaudeCodeRoute = (): ClaudeCodeRouteForm => ({
  shellModel: '',
  upstreamModel: '',
  displayName: '',
  contextWindow: '',
  thinkingEnabled: false,
  thinkingMode: 'levels',
  targetField: 'output_config.effort',
  levels: [],
  overrides: { low: '', medium: '', high: '', xhigh: '', max: '' }
})

export const buildClaudeCodeEffortMap = (
  rawLevels: string[],
  rawOverrides: Partial<Record<ClaudeCodeEffortLevel, string>> = {}
): Partial<Record<ClaudeCodeEffortLevel, string>> => {
  const levels = rawLevels.map(level => level.trim()).filter(Boolean)
  if (levels.length === 0) return {}

  const mapping: Partial<Record<ClaudeCodeEffortLevel, string>> = {}
  claudeCodeEffortLevels.forEach((level, sourceIndex) => {
    const override = rawOverrides[level]?.trim()
    const targetIndex = Math.round(sourceIndex * (levels.length - 1) / (claudeCodeEffortLevels.length - 1))
    mapping[level] = override || levels[targetIndex]
  })
  return mapping
}

const positiveIntegerString = (value: unknown) => {
  const raw = typeof value === 'number' || typeof value === 'string' ? String(value).trim() : ''
  if (!/^\d+$/.test(raw) || Number(raw) <= 0) return ''
  return raw
}

const readThinkingOverrides = (value: unknown): Record<ClaudeCodeEffortLevel, string> => {
  const raw = value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {}
  return Object.fromEntries(
    claudeCodeEffortLevels.map(level => [level, stringValue(raw[level])])
  ) as Record<ClaudeCodeEffortLevel, string>
}

const readRoute = (value: unknown): ClaudeCodeRouteForm | null => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const raw = value as Record<string, unknown>
  const thinking = raw.thinking && typeof raw.thinking === 'object' && !Array.isArray(raw.thinking)
    ? raw.thinking as Record<string, unknown>
    : {}
  return {
    shellModel: stringValue(raw.shell_model),
    upstreamModel: stringValue(raw.upstream_model),
    displayName: stringValue(raw.display_name),
    contextWindow: positiveIntegerString(raw.context_window),
    thinkingEnabled: Object.keys(thinking).length > 0,
    thinkingMode: stringValue(thinking.mode) === 'budget' ? 'budget' : 'levels',
    targetField: stringValue(thinking.target_field) || 'output_config.effort',
    levels: stringListValue(thinking.levels),
    overrides: readThinkingOverrides(thinking.overrides)
  }
}

export const readClaudeCodeRoutes = (extra?: Record<string, unknown>): ClaudeCodeRouteForm[] => {
  if (extra && Object.prototype.hasOwnProperty.call(extra, 'claude_code_routes')) {
    const raw = extra.claude_code_routes
    if (!Array.isArray(raw)) return [createEmptyClaudeCodeRoute()]
    const routes = raw.map(readRoute).filter((route): route is ClaudeCodeRouteForm => route !== null)
    return routes.length > 0 ? routes : [createEmptyClaudeCodeRoute()]
  }

  const catalog = Array.isArray(extra?.claude_code_model_catalog)
    ? extra.claude_code_model_catalog
    : []
  const effortByModel = extra?.claude_code_effort_mapping && typeof extra.claude_code_effort_mapping === 'object' && !Array.isArray(extra.claude_code_effort_mapping)
    ? extra.claude_code_effort_mapping as Record<string, unknown>
    : {}
  const routes = catalog.map((value) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    const raw = value as Record<string, unknown>
    const shellModel = stringValue(raw.request_model)
    if (!shellModel) return null
    const effort = effortByModel[shellModel] && typeof effortByModel[shellModel] === 'object' && !Array.isArray(effortByModel[shellModel])
      ? effortByModel[shellModel] as Record<string, unknown>
      : {}
    const values = effort.values && typeof effort.values === 'object' && !Array.isArray(effort.values)
      ? effort.values as Record<string, unknown>
      : {}
    const overrides = readThinkingOverrides(values)
    const levels = Array.from(new Set(claudeCodeEffortLevels.map(level => overrides[level]).filter(Boolean)))
    const targetField = stringValue(effort.target_field) || 'output_config.effort'
    return {
      shellModel,
      upstreamModel: stringValue(raw.upstream_model) || shellModel,
      displayName: stringValue(raw.display_name),
      contextWindow: raw.supports_1m === true ? '1000000' : positiveIntegerString(raw.context_window),
      thinkingEnabled: levels.length > 0,
      thinkingMode: targetField === 'thinking.budget_tokens' ? 'budget' as const : 'levels' as const,
      targetField,
      levels,
      overrides
    }
  }).filter((route): route is ClaudeCodeRouteForm => route !== null)
  return routes.length > 0 ? routes : [createEmptyClaudeCodeRoute()]
}

export const readUpstreamModelsURL = (extra?: Record<string, unknown>): string =>
  stringValue(extra?.upstream_models_url).trim()

export const writeUpstreamModelsURLToExtra = (extra: Record<string, unknown>, value: string) => {
  const modelsURL = value.trim()
  if (modelsURL) extra.upstream_models_url = modelsURL
  else delete extra.upstream_models_url
}

export const writeClaudeCodeRoutesToExtra = (
  extra: Record<string, unknown>,
  routes: ClaudeCodeRouteForm[]
) => {
  const routePayload: Record<string, unknown>[] = []
  const catalogPayload: Record<string, unknown>[] = []
  const effortPayload: Record<string, unknown> = {}

  for (const route of routes) {
    const shellModel = route.shellModel.trim()
    const upstreamModel = route.upstreamModel.trim()
    if (!shellModel || !upstreamModel) continue

    const displayName = route.displayName.trim() || shellModel
    const contextWindow = positiveIntegerString(route.contextWindow)
    const levels = route.levels.map(level => level.trim()).filter(Boolean)
    const overrides = Object.fromEntries(
      claudeCodeEffortLevels
        .map(level => [level, route.overrides[level].trim()])
        .filter(([, value]) => Boolean(value))
    )
    const targetField = route.thinkingMode === 'budget'
      ? 'thinking.budget_tokens'
      : route.targetField.trim() || 'output_config.effort'
    const thinking = route.thinkingEnabled && levels.length > 0
      ? { mode: route.thinkingMode, target_field: targetField, levels, overrides }
      : undefined

    const payload: Record<string, unknown> = {
      shell_model: shellModel,
      upstream_model: upstreamModel,
      display_name: displayName
    }
    if (contextWindow) payload.context_window = Number(contextWindow)
    if (thinking) payload.thinking = thinking
    routePayload.push(payload)

    const catalogEntry: Record<string, unknown> = {
      role: 'route',
      display_name: displayName,
      request_model: shellModel,
      upstream_model: upstreamModel
    }
    if (contextWindow === '1000000') catalogEntry.supports_1m = true
    if (thinking) {
      catalogEntry.capabilities = ['thinking', 'effort']
      effortPayload[shellModel] = {
        target_field: targetField,
        values: buildClaudeCodeEffortMap(levels, overrides)
      }
    }
    catalogPayload.push(catalogEntry)
  }

  if (routePayload.length === 0) {
    delete extra.claude_code_routes
    delete extra.claude_code_model_catalog
    delete extra.claude_code_effort_mapping
    return
  }
  extra.claude_code_routes = routePayload
  extra.claude_code_model_catalog = catalogPayload
  if (Object.keys(effortPayload).length > 0) extra.claude_code_effort_mapping = effortPayload
  else delete extra.claude_code_effort_mapping
}

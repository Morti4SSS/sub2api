export interface ClaudeCodeCatalogForm {
  role: string
  displayName: string
  requestModel: string
  upstreamModel: string
  supports1m: boolean
  capabilities: string
}

export interface ClaudeCodeEffortForm {
  model: string
  targetField: string
  low: string
  medium: string
  high: string
  xhigh: string
  max: string
}

export const createEmptyClaudeCodeCatalogEntry = (): ClaudeCodeCatalogForm => ({
  role: '',
  displayName: '',
  requestModel: '',
  upstreamModel: '',
  supports1m: false,
  capabilities: ''
})

export const createEmptyClaudeCodeEffortEntry = (): ClaudeCodeEffortForm => ({
  model: '',
  targetField: 'output_config.effort',
  low: '',
  medium: '',
  high: '',
  xhigh: '',
  max: ''
})

export const buildClaudeCodeEffortSummary = (entry: ClaudeCodeEffortForm) => {
  const groups = new Map<string, string[]>()
  for (const level of ['low', 'medium', 'high', 'xhigh', 'max'] as const) {
    const mapped = entry[level].trim()
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

export const readClaudeCodeCatalog = (extra?: Record<string, unknown>): ClaudeCodeCatalogForm[] => {
  const raw = extra?.claude_code_model_catalog
  if (!Array.isArray(raw)) return [createEmptyClaudeCodeCatalogEntry()]
  const entries = raw
    .map((item) => {
      if (!item || typeof item !== 'object') return null
      const data = item as Record<string, unknown>
      return {
        role: stringValue(data.role),
        displayName: stringValue(data.display_name),
        requestModel: stringValue(data.request_model),
        upstreamModel: stringValue(data.upstream_model),
        supports1m: data.supports_1m === true,
        capabilities: stringListValue(data.capabilities).join(', ')
      }
    })
    .filter((item): item is ClaudeCodeCatalogForm => item !== null)
  return entries.length > 0 ? entries : [createEmptyClaudeCodeCatalogEntry()]
}

export const readClaudeCodeEffortMappings = (extra?: Record<string, unknown>): ClaudeCodeEffortForm[] => {
  const raw = extra?.claude_code_effort_mapping
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return [createEmptyClaudeCodeEffortEntry()]
  const entries = Object.entries(raw as Record<string, unknown>)
    .map(([model, item]) => {
      if (!item || typeof item !== 'object' || Array.isArray(item)) return null
      const data = item as Record<string, unknown>
      const values = data.values && typeof data.values === 'object' && !Array.isArray(data.values)
        ? data.values as Record<string, unknown>
        : {}
      return {
        model: model.trim(),
        targetField: stringValue(data.target_field) || 'output_config.effort',
        low: stringValue(values.low),
        medium: stringValue(values.medium),
        high: stringValue(values.high),
        xhigh: stringValue(values.xhigh),
        max: stringValue(values.max)
      }
    })
    .filter((item): item is ClaudeCodeEffortForm => item !== null)
  return entries.length > 0 ? entries : [createEmptyClaudeCodeEffortEntry()]
}

export const writeClaudeCodeConfigToExtra = (
  extra: Record<string, unknown>,
  catalog: ClaudeCodeCatalogForm[],
  effortMappings: ClaudeCodeEffortForm[]
) => {
  const catalogPayload = catalog
    .map((entry) => {
      const role = entry.role.trim()
      const displayName = entry.displayName.trim()
      const requestModel = entry.requestModel.trim()
      if (!role || !displayName || !requestModel) return null
      const payload: Record<string, unknown> = { role, display_name: displayName, request_model: requestModel }
      const upstreamModel = entry.upstreamModel.trim()
      if (upstreamModel) payload.upstream_model = upstreamModel
      if (entry.supports1m) payload.supports_1m = true
      const capabilities = entry.capabilities.split(',').map((item) => item.trim()).filter(Boolean)
      if (capabilities.length > 0) payload.capabilities = capabilities
      return payload
    })
    .filter((item): item is Record<string, unknown> => item !== null)

  if (catalogPayload.length > 0) extra.claude_code_model_catalog = catalogPayload
  else delete extra.claude_code_model_catalog

  const effortPayload: Record<string, unknown> = {}
  for (const [index, entry] of effortMappings.entries()) {
    const model = entry.model.trim() || catalog[index]?.requestModel.trim() || ''
    if (!model) continue
    const values: Record<string, string> = {}
    for (const key of ['low', 'medium', 'high', 'xhigh', 'max'] as const) {
      const value = entry[key].trim()
      if (value) values[key] = value
    }
    if (Object.keys(values).length === 0) continue
    effortPayload[model] = {
      target_field: entry.targetField.trim() || 'output_config.effort',
      values
    }
  }

  if (Object.keys(effortPayload).length > 0) extra.claude_code_effort_mapping = effortPayload
  else delete extra.claude_code_effort_mapping
}

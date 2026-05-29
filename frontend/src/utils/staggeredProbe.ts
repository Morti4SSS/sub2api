import type { AccountPlatform, ClaudeModel } from '@/types'

const DEFAULT_RECENT_HOURS = 24
const MAX_RECENT_HOURS = 720
const DEFAULT_INTERVAL_MINUTES = 0
const MAX_INTERVAL_MINUTES = 1440
const prioritizedOpenAIProbeModels = ['gpt-5.4', 'gpt-5.3', 'gpt-5.2', 'gpt-5.1', 'gpt-5']

export type ProbeDelayMode = 'fixed' | 'random'

export const DEFAULT_PROBE_PROMPTS = [
  'Answer with one short sentence: what is a variable?',
  'Translate hello into Chinese.',
  'Give one study tip in under five words.',
  'Answer only the result of 3+5.',
  'Explain what a function is in one sentence.',
  'Translate cat into Chinese.',
  'Give one simple time management tip.',
  'Explain what a cache is in one sentence.',
  'Answer only the result of 9-4.',
  'Translate good morning into Chinese.',
  'Explain why backups are useful in one sentence.',
  'Give one concise coding habit.',
]

export function normalizeProbeRecentHours(value: unknown): number {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed) || parsed <= 0) return DEFAULT_RECENT_HOURS
  return Math.min(MAX_RECENT_HOURS, parsed)
}

export function normalizeProbeIntervalMinutes(value: unknown): number {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed) || parsed < 0) return DEFAULT_INTERVAL_MINUTES
  return Math.min(MAX_INTERVAL_MINUTES, parsed)
}

export function normalizeProbeDelayMode(value: unknown): ProbeDelayMode {
  return value === 'fixed' ? 'fixed' : 'random'
}

export function normalizeProbeIntervalRangeMinutes(minValue: unknown, maxValue: unknown): { min: number; max: number } {
  const min = normalizeProbeIntervalMinutes(minValue)
  const max = Math.max(min, normalizeProbeIntervalMinutes(maxValue))
  return { min, max }
}

export function shouldSkipRecentAccount(
  lastUsedAt: string | null | undefined,
  recentHours: unknown,
  now: Date = new Date()
): boolean {
  return shouldSkipRecentTimestamp(lastUsedAt, recentHours, now)
}

export function shouldSkipRecentTimestamp(
  timestamp: string | null | undefined,
  recentHours: unknown,
  now: Date = new Date()
): boolean {
  if (!timestamp) return false
  const ts = Date.parse(timestamp)
  if (!Number.isFinite(ts)) return false
  const windowMs = normalizeProbeRecentHours(recentHours) * 60 * 60 * 1000
  return now.getTime() - ts < windowMs
}

export function pickProbePrompt(
  prompts: string[] = DEFAULT_PROBE_PROMPTS,
  previousPrompt = '',
  random: () => number = Math.random
): string {
  const pool = prompts.map(item => item.trim()).filter(Boolean)
  if (pool.length === 0) return DEFAULT_PROBE_PROMPTS[0]
  if (pool.length === 1) return pool[0]

  const candidates = pool.filter(item => item !== previousPrompt)
  const index = Math.floor(Math.min(1, Math.max(0, random())) * candidates.length)
  return candidates[Math.min(candidates.length - 1, Math.max(0, index))]
}

export function resolveProbeDelayMs(
  options: {
    mode: unknown
    fixedMinutes?: unknown
    minMinutes?: unknown
    maxMinutes?: unknown
  },
  random: () => number = Math.random
): number {
  if (normalizeProbeDelayMode(options.mode) === 'fixed') {
    return normalizeProbeIntervalMinutes(options.fixedMinutes) * 60 * 1000
  }
  const { min, max } = normalizeProbeIntervalRangeMinutes(options.minMinutes, options.maxMinutes)
  if (max === 0) return 0
  const ratio = Math.min(1, Math.max(0, random()))
  return Math.round((min + ratio * (max - min)) * 60 * 1000)
}

export function selectProbeModel(platform: AccountPlatform, models: ClaudeModel[]): string {
  if (platform !== 'openai') return ''
  const textModels = models.filter(model => {
    const id = model.id.toLowerCase()
    return id.includes('codex') || (id.startsWith('gpt-') && !id.startsWith('gpt-image-'))
  })
  if (textModels.length === 0) return ''

  const codexModel = textModels.find(model => model.id.toLowerCase().includes('codex'))
  if (codexModel) return codexModel.id

  for (const preferredID of prioritizedOpenAIProbeModels) {
    const exactModel = textModels.find(model => model.id.toLowerCase() === preferredID)
    if (exactModel) return exactModel.id
  }

  const gpt5Model = textModels.find(model => model.id.toLowerCase().startsWith('gpt-5'))
  return gpt5Model?.id || textModels[0].id
}

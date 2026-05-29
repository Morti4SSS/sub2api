const DEFAULT_BATCH_SIZE = 1
const MAX_BATCH_SIZE = 50
const DEFAULT_BATCH_DELAY_SECONDS = 0
const MAX_BATCH_DELAY_SECONDS = 600
const DEFAULT_BATCH_CONCURRENCY = 1
const MAX_BATCH_CONCURRENCY = 10

export type BatchDelayMode = 'fixed' | 'random'

export function normalizeBatchSize(value: unknown): number {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed)) return DEFAULT_BATCH_SIZE
  return Math.min(MAX_BATCH_SIZE, Math.max(1, parsed))
}

export function normalizeBatchDelaySeconds(value: unknown): number {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed)) return DEFAULT_BATCH_DELAY_SECONDS
  return Math.min(MAX_BATCH_DELAY_SECONDS, Math.max(0, parsed))
}

export function normalizeBatchDelayMode(value: unknown): BatchDelayMode {
  return value === 'random' ? 'random' : 'fixed'
}

export function normalizeBatchDelayRangeSeconds(minValue: unknown, maxValue: unknown): { min: number; max: number } {
  const min = normalizeBatchDelaySeconds(minValue)
  const max = Math.max(min, normalizeBatchDelaySeconds(maxValue))
  return { min, max }
}

export function resolveBatchDelaySeconds(
  options: {
    mode: unknown
    fixedSeconds?: unknown
    minSeconds?: unknown
    maxSeconds?: unknown
  },
  random: () => number = Math.random
): number {
  if (normalizeBatchDelayMode(options.mode) === 'fixed') {
    return normalizeBatchDelaySeconds(options.fixedSeconds)
  }
  const { min, max } = normalizeBatchDelayRangeSeconds(options.minSeconds, options.maxSeconds)
  if (max === 0) return 0
  const ratio = Math.min(1, Math.max(0, random()))
  return Math.round(min + ratio * (max - min))
}

export function normalizeBatchConcurrency(value: unknown): number {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed)) return DEFAULT_BATCH_CONCURRENCY
  return Math.min(MAX_BATCH_CONCURRENCY, Math.max(1, parsed))
}

export function chunkItems<T>(items: T[], batchSize: unknown): T[][] {
  const size = normalizeBatchSize(batchSize)
  const chunks: T[][] = []
  for (let index = 0; index < items.length; index += size) {
    chunks.push(items.slice(index, index + size))
  }
  return chunks
}

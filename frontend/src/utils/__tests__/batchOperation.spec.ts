import { describe, expect, it } from 'vitest'

import {
  chunkItems,
  normalizeBatchConcurrency,
  normalizeBatchDelayMode,
  normalizeBatchDelayRangeSeconds,
  normalizeBatchDelaySeconds,
  normalizeBatchSize,
  resolveBatchDelaySeconds,
} from '../batchOperation'

describe('batchOperation helpers', () => {
  it('normalizes batch size with safe bounds', () => {
    expect(normalizeBatchSize(0)).toBe(1)
    expect(normalizeBatchSize(-3)).toBe(1)
    expect(normalizeBatchSize('bad')).toBe(1)
    expect(normalizeBatchSize(3)).toBe(3)
    expect(normalizeBatchSize(999)).toBe(50)
  })

  it('normalizes group delay seconds with safe bounds', () => {
    expect(normalizeBatchDelaySeconds(-1)).toBe(0)
    expect(normalizeBatchDelaySeconds('bad')).toBe(0)
    expect(normalizeBatchDelaySeconds(30)).toBe(30)
    expect(normalizeBatchDelaySeconds(9999)).toBe(600)
  })

  it('normalizes batch delay mode and random delay range', () => {
    expect(normalizeBatchDelayMode('random')).toBe('random')
    expect(normalizeBatchDelayMode('fixed')).toBe('fixed')
    expect(normalizeBatchDelayMode('bad')).toBe('fixed')

    expect(normalizeBatchDelayRangeSeconds(-1, 'bad')).toEqual({ min: 0, max: 0 })
    expect(normalizeBatchDelayRangeSeconds(30, 10)).toEqual({ min: 30, max: 30 })
    expect(normalizeBatchDelayRangeSeconds(10, 30)).toEqual({ min: 10, max: 30 })
    expect(normalizeBatchDelayRangeSeconds(9999, 9999)).toEqual({ min: 600, max: 600 })
  })

  it('resolves fixed and random batch delay seconds', () => {
    expect(resolveBatchDelaySeconds({ mode: 'fixed', fixedSeconds: 12 }, () => 0.9)).toBe(12)
    expect(resolveBatchDelaySeconds({ mode: 'random', minSeconds: 10, maxSeconds: 20 }, () => 0)).toBe(10)
    expect(resolveBatchDelaySeconds({ mode: 'random', minSeconds: 10, maxSeconds: 20 }, () => 0.5)).toBe(15)
    expect(resolveBatchDelaySeconds({ mode: 'random', minSeconds: 10, maxSeconds: 20 }, () => 1)).toBe(20)
  })

  it('normalizes concurrency with safe bounds', () => {
    expect(normalizeBatchConcurrency(0)).toBe(1)
    expect(normalizeBatchConcurrency(-3)).toBe(1)
    expect(normalizeBatchConcurrency('bad')).toBe(1)
    expect(normalizeBatchConcurrency(3)).toBe(3)
    expect(normalizeBatchConcurrency(999)).toBe(10)
  })

  it('chunks items by normalized batch size', () => {
    expect(chunkItems([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]])
    expect(chunkItems([1, 2, 3], 0)).toEqual([[1], [2], [3]])
    expect(chunkItems([], 3)).toEqual([])
  })
})

import { describe, expect, it } from 'vitest'

import type { ClaudeModel } from '@/types'
import {
  pickProbePrompt,
  resolveProbeDelayMs,
  selectProbeModel,
  shouldSkipRecentAccount,
  shouldSkipRecentTimestamp,
  normalizeProbeDelayMode,
  normalizeProbeIntervalMinutes,
  normalizeProbeIntervalRangeMinutes,
  normalizeProbeRecentHours,
} from '../staggeredProbe'

describe('staggeredProbe helpers', () => {
  it('skips accounts used within the recent window', () => {
    const now = new Date('2026-05-17T12:00:00Z')

    expect(shouldSkipRecentAccount('2026-05-17T10:30:00Z', 2, now)).toBe(true)
    expect(shouldSkipRecentAccount('2026-05-17T09:30:00Z', 2, now)).toBe(false)
    expect(shouldSkipRecentAccount(null, 2, now)).toBe(false)
  })

  it('skips timestamps within the recent window', () => {
    const now = new Date('2026-05-17T12:00:00Z')

    expect(shouldSkipRecentTimestamp('2026-05-17T11:30:00Z', 2, now)).toBe(true)
    expect(shouldSkipRecentTimestamp('2026-05-17T09:30:00Z', 2, now)).toBe(false)
    expect(shouldSkipRecentTimestamp('invalid', 2, now)).toBe(false)
  })

  it('normalizes conservative probe settings', () => {
    expect(normalizeProbeRecentHours(0)).toBe(24)
    expect(normalizeProbeRecentHours(48)).toBe(48)
    expect(normalizeProbeRecentHours(9999)).toBe(720)

    expect(normalizeProbeIntervalMinutes(-1)).toBe(0)
    expect(normalizeProbeIntervalMinutes(10)).toBe(10)
    expect(normalizeProbeIntervalMinutes(9999)).toBe(1440)
  })

  it('normalizes fixed and random probe delay settings', () => {
    expect(normalizeProbeDelayMode('random')).toBe('random')
    expect(normalizeProbeDelayMode('fixed')).toBe('fixed')
    expect(normalizeProbeDelayMode('bad')).toBe('random')

    expect(normalizeProbeIntervalRangeMinutes(-1, 'bad')).toEqual({ min: 0, max: 0 })
    expect(normalizeProbeIntervalRangeMinutes(12, 3)).toEqual({ min: 12, max: 12 })
    expect(normalizeProbeIntervalRangeMinutes(3, 12)).toEqual({ min: 3, max: 12 })
  })

  it('resolves fixed and random probe delay milliseconds', () => {
    expect(resolveProbeDelayMs({ mode: 'fixed', fixedMinutes: 2 }, () => 0.9)).toBe(120000)
    expect(resolveProbeDelayMs({ mode: 'random', minMinutes: 3, maxMinutes: 15 }, () => 0)).toBe(180000)
    expect(resolveProbeDelayMs({ mode: 'random', minMinutes: 3, maxMinutes: 15 }, () => 0.5)).toBe(540000)
    expect(resolveProbeDelayMs({ mode: 'random', minMinutes: 3, maxMinutes: 15 }, () => 1)).toBe(900000)
  })

  it('picks prompt without repeating previous prompt when possible', () => {
    const prompts = ['a', 'b', 'c']

    expect(pickProbePrompt(prompts, 'a', () => 0)).toBe('b')
    expect(pickProbePrompt(['only'], 'only', () => 0)).toBe('only')
  })

  it('selects OpenAI probe model with Codex and GPT priority', () => {
    const models: ClaudeModel[] = [
      { id: 'gpt-4.1', type: 'model', display_name: 'GPT 4.1', created_at: '' },
      { id: 'gpt-5.4', type: 'model', display_name: 'GPT 5.4', created_at: '' },
      { id: 'codex-mini-latest', type: 'model', display_name: 'Codex Mini', created_at: '' },
    ]

    expect(selectProbeModel('openai', models)).toBe('codex-mini-latest')
    expect(selectProbeModel('openai', models.slice(0, 2))).toBe('gpt-5.4')
  })

  it('does not select probe models for non-OpenAI platforms', () => {
    const models: ClaudeModel[] = [
      { id: 'claude-sonnet-4-5', type: 'model', display_name: 'Sonnet', created_at: '' },
    ]

    expect(selectProbeModel('anthropic', models)).toBe('')
    expect(selectProbeModel('gemini', models)).toBe('')
    expect(selectProbeModel('antigravity', models)).toBe('')
    expect(selectProbeModel('openai', [])).toBe('')
  })

  it('skips OpenAI accounts when no GPT or Codex model is available', () => {
    const models: ClaudeModel[] = [
      { id: 'gpt-image-2', type: 'model', display_name: 'GPT Image', created_at: '' },
      { id: 'omni-moderation-latest', type: 'model', display_name: 'Moderation', created_at: '' },
    ]

    expect(selectProbeModel('openai', models)).toBe('')
  })
})

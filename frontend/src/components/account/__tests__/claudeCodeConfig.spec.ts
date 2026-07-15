import { describe, expect, it } from 'vitest'
import {
  buildClaudeCodeEffortMap,
  buildClaudeCodeEffortSummary,
  createEmptyClaudeCodeRoute,
  readClaudeCodeRoutes,
  readUpstreamModelsURL,
  writeClaudeCodeRoutesToExtra,
  writeUpstreamModelsURLToExtra,
  type ClaudeCodeRouteForm
} from '../claudeCodeConfig'

const route = (overrides: Partial<ClaudeCodeRouteForm> = {}): ClaudeCodeRouteForm => ({
  ...createEmptyClaudeCodeRoute(),
  shellModel: 'claude-opus-4-8',
  upstreamModel: 'glm-5.2',
  displayName: 'Strong model route',
  contextWindow: '1000000',
  thinkingEnabled: true,
  thinkingMode: 'levels',
  targetField: 'reasoning_effort',
  levels: ['off', 'on'],
  ...overrides
})

describe('claudeCodeConfig', () => {
  it('writes the route owner and narrow r2 compatibility projections', () => {
    const extra: Record<string, unknown> = {}

    writeClaudeCodeRoutesToExtra(extra, [route()])

    expect(extra.claude_code_routes).toEqual([
      {
        shell_model: 'claude-opus-4-8',
        upstream_model: 'glm-5.2',
        display_name: 'Strong model route',
        context_window: 1000000,
        thinking: {
          mode: 'levels',
          target_field: 'reasoning_effort',
          levels: ['off', 'on'],
          overrides: {}
        }
      }
    ])
    expect(extra.claude_code_model_catalog).toEqual([
      {
        role: 'route',
        display_name: 'Strong model route',
        request_model: 'claude-opus-4-8',
        upstream_model: 'glm-5.2',
        supports_1m: true,
        capabilities: ['thinking', 'effort']
      }
    ])
    expect(extra.claude_code_effort_mapping).toEqual({
      'claude-opus-4-8': {
        target_field: 'reasoning_effort',
        values: { low: 'off', medium: 'off', high: 'on', xhigh: 'on', max: 'on' }
      }
    })
  })

  it('reads the new owner without merging stale legacy fields', () => {
    const extra = {
      claude_code_routes: [{ shell_model: 'claude-opus-4-8', upstream_model: 'glm-5.2' }],
      claude_code_model_catalog: [{
        role: 'route',
        display_name: 'Legacy',
        request_model: 'claude-opus-4-8',
        upstream_model: 'legacy-model'
      }]
    }

    expect(readClaudeCodeRoutes(extra)[0].upstreamModel).toBe('glm-5.2')
  })

  it('migrates the legacy catalog and exact effort values into one route', () => {
    const extra = {
      claude_code_model_catalog: [{
        role: 'opus',
        display_name: 'Legacy route',
        request_model: 'claude-opus-4-8',
        upstream_model: 'legacy-model',
        supports_1m: true
      }],
      claude_code_effort_mapping: {
        'claude-opus-4-8': {
          target_field: 'reasoning_effort',
          values: { low: 'off', medium: 'off', high: 'on', xhigh: 'turbo', max: 'on' }
        }
      }
    }

    const migrated = readClaudeCodeRoutes(extra)[0]

    expect(migrated.contextWindow).toBe('1000000')
    expect(migrated.levels).toEqual(['off', 'on', 'turbo'])
    expect(migrated.overrides).toEqual({ low: 'off', medium: 'off', high: 'on', xhigh: 'turbo', max: 'on' })
  })

  it('folds ordered levels and applies per-level overrides', () => {
    expect(buildClaudeCodeEffortMap(['weak', 'middle', 'strong'], { xhigh: 'turbo' })).toEqual({
      low: 'weak',
      medium: 'middle',
      high: 'middle',
      xhigh: 'turbo',
      max: 'strong'
    })
    expect(buildClaudeCodeEffortSummary(['off', 'on'], {})).toBe('low/medium -> off; high/xhigh/max -> on')
  })

  it('returns one empty route when no configuration exists', () => {
    expect(readClaudeCodeRoutes({})).toEqual([createEmptyClaudeCodeRoute()])
  })

  it('does not persist thinking when the route toggle is off', () => {
    const extra: Record<string, unknown> = {}

    writeClaudeCodeRoutesToExtra(extra, [route({ thinkingEnabled: false })])

    expect((extra.claude_code_routes as Array<Record<string, unknown>>)[0]).not.toHaveProperty('thinking')
    expect(extra.claude_code_effort_mapping).toBeUndefined()
  })

  it('reads, writes, and clears the account-owned upstream model list URL', () => {
    const extra: Record<string, unknown> = { upstream_models_url: ' https://relay.example.com/catalog ' }

    expect(readUpstreamModelsURL(extra)).toBe('https://relay.example.com/catalog')
    writeUpstreamModelsURLToExtra(extra, ' https://new.example.com/models ')
    expect(extra.upstream_models_url).toBe('https://new.example.com/models')

    writeUpstreamModelsURLToExtra(extra, '   ')
    expect(extra).not.toHaveProperty('upstream_models_url')
  })
})

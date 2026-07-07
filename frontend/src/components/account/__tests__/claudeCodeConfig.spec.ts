import { describe, expect, it } from 'vitest'
import {
  buildClaudeCodeEffortSummary,
  createEmptyClaudeCodeCatalogEntry,
  readClaudeCodeCatalog,
  readClaudeCodeEffortMappings,
  writeClaudeCodeConfigToExtra
} from '../claudeCodeConfig'

describe('claudeCodeConfig', () => {
  it('reads and writes catalog entries', () => {
    const extra: Record<string, unknown> = {}
    writeClaudeCodeConfigToExtra(extra, [
      {
        role: 'sonnet',
        displayName: 'Relay Sonnet',
        requestModel: 'relay-sonnet',
        upstreamModel: 'upstream-sonnet',
        supports1m: true,
        capabilities: 'thinking, effort'
      }
    ], [])

    expect(extra.claude_code_model_catalog).toEqual([
      {
        role: 'sonnet',
        display_name: 'Relay Sonnet',
        request_model: 'relay-sonnet',
        upstream_model: 'upstream-sonnet',
        supports_1m: true,
        capabilities: ['thinking', 'effort']
      }
    ])
    expect(readClaudeCodeCatalog(extra)[0].requestModel).toBe('relay-sonnet')
  })

  it('summarizes folded effort mappings', () => {
    expect(buildClaudeCodeEffortSummary({
      model: 'relay-sonnet',
      targetField: 'reasoning_effort',
      low: 'low',
      medium: 'medium',
      high: 'high',
      xhigh: 'high',
      max: 'high'
    })).toBe('low -> low; medium -> medium; high/xhigh/max -> high')
  })

  it('returns one empty row for missing config', () => {
    expect(readClaudeCodeCatalog({})).toEqual([createEmptyClaudeCodeCatalogEntry()])
    expect(readClaudeCodeEffortMappings({})[0].targetField).toBe('output_config.effort')
  })
})

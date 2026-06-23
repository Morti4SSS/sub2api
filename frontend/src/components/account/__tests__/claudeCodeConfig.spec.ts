import { describe, expect, it } from 'vitest'
import {
  buildClaudeCodeEffortSummary,
  writeClaudeCodeConfigToExtra
} from '../claudeCodeConfig'

describe('claudeCodeConfig', () => {
  it('writes catalog and effort mapping into extra', () => {
    const extra: Record<string, unknown> = {
      quota_limit: 100
    }

    writeClaudeCodeConfigToExtra(
      extra,
      [
        {
          role: 'sonnet',
          displayName: 'glm-5.1',
          requestModel: 'glm-5.1',
          upstreamModel: 'provider-glm-5.1',
          supports1m: true,
          capabilities: 'effort, max_effort, thinking'
        }
      ],
      [
        {
          model: '',
          targetField: 'output_config.effort',
          low: '',
          medium: '',
          high: '',
          xhigh: 'max',
          max: ''
        }
      ]
    )

    expect(extra.quota_limit).toBe(100)
    expect(extra.claude_code_model_catalog).toEqual([
      {
        role: 'sonnet',
        display_name: 'glm-5.1',
        request_model: 'glm-5.1',
        upstream_model: 'provider-glm-5.1',
        supports_1m: true,
        capabilities: ['effort', 'max_effort', 'thinking']
      }
    ])
    expect(extra.claude_code_effort_mapping).toEqual({
      'glm-5.1': {
        target_field: 'output_config.effort',
        values: {
          xhigh: 'max'
        }
      }
    })
  })

  it('summarizes unsupported effort levels as folded mappings', () => {
    const summary = buildClaudeCodeEffortSummary({
      model: 'provider-model',
      targetField: 'output_config.effort',
      low: 'high',
      medium: 'high',
      high: 'high',
      xhigh: 'max',
      max: 'max'
    })

    expect(summary).toBe('low/medium/high -> high; xhigh/max -> max')
  })
})

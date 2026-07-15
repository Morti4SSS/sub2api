import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClaudeCodeConfigEditor from '../ClaudeCodeConfigEditor.vue'
import { createEmptyClaudeCodeRoute } from '../claudeCodeConfig'

const { getClaudeCodeOptionsMock, syncUpstreamModelsMock, syncUpstreamModelsPreviewMock } = vi.hoisted(() => ({
  getClaudeCodeOptionsMock: vi.fn(),
  syncUpstreamModelsMock: vi.fn(),
  syncUpstreamModelsPreviewMock: vi.fn()
}))

vi.mock('@/api/admin/accounts', () => ({
  getClaudeCodeOptions: getClaudeCodeOptionsMock,
  accountsAPI: {
    syncUpstreamModels: syncUpstreamModelsMock,
    syncUpstreamModelsPreview: syncUpstreamModelsPreviewMock
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${Object.values(params).join(',')}` : key
    })
  }
})

describe('ClaudeCodeConfigEditor', () => {
  beforeEach(() => {
    getClaudeCodeOptionsMock.mockReset()
    syncUpstreamModelsMock.mockReset()
    syncUpstreamModelsPreviewMock.mockReset()
  })

  it('loads canonical shells and keeps thinking controls on the same route row', async () => {
    getClaudeCodeOptionsMock.mockResolvedValue({
      models: [{ id: 'claude-opus-4-8', display_name: 'Claude Opus 4.8' }],
      effort_levels: ['low', 'medium', 'high', 'xhigh', 'max']
    })
    const routes = [createEmptyClaudeCodeRoute()]
    const wrapper = mount(ClaudeCodeConfigEditor, {
      props: { routes },
      global: {
        stubs: {
          Icon: { template: '<span />' }
        }
      }
    })

    await flushPromises()

    const shell = wrapper.get('[data-testid="claude-code-shell-0"]')
    expect(shell.findAll('option').map(option => option.attributes('value'))).toContain('claude-opus-4-8')
    expect(wrapper.get('[data-testid="claude-code-thinking-enabled-0"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="claude-code-effort-model-0"]').exists()).toBe(false)
  })

  it('syncs the current saved account into suggestions without replacing manual routes', async () => {
    getClaudeCodeOptionsMock.mockResolvedValue({ models: [], effort_levels: [] })
    syncUpstreamModelsMock.mockResolvedValue({ models: ['glm-5.2', 'deepseek-r1'] })
    const routes = [createEmptyClaudeCodeRoute()]
    routes[0].upstreamModel = 'manual-model'
    const wrapper = mount(ClaudeCodeConfigEditor, {
      props: {
        routes,
        accountId: 7,
        upstreamModelsUrl: 'https://relay.example.com/catalog?tenant=private'
      },
      global: { stubs: { Icon: { template: '<span />' } } }
    })

    await flushPromises()
    await wrapper.get('[data-testid="claude-code-sync-models"]').trigger('click')
    await flushPromises()

    expect(syncUpstreamModelsMock).toHaveBeenCalledWith(7, 'https://relay.example.com/catalog?tenant=private')
    expect(wrapper.get('[data-testid="claude-code-upstream-model-0"]').element).toHaveProperty('value', 'manual-model')
    expect(wrapper.findAll('#claude-code-upstream-models option').map(option => option.attributes('value'))).toEqual([
      'glm-5.2',
      'deepseek-r1'
    ])
    expect(wrapper.get('[data-testid="claude-code-sync-result"]').text()).toContain('2')
  })

  it('keeps manual input and renders safe diagnostics when sync fails', async () => {
    getClaudeCodeOptionsMock.mockResolvedValue({ models: [], effort_levels: [] })
    syncUpstreamModelsMock.mockRejectedValue({
      message: 'Upstream model list request failed with HTTP 403',
      reason: 'upstream',
      metadata: {
        request_url: 'https://relay.example.com/catalog',
        http_status: '403',
        content_type: 'text/html',
        response_shape: 'html'
      }
    })
    const routes = [createEmptyClaudeCodeRoute()]
    routes[0].upstreamModel = 'manual-model'
    const wrapper = mount(ClaudeCodeConfigEditor, {
      props: { routes, accountId: 7, upstreamModelsUrl: '' },
      global: { stubs: { Icon: { template: '<span />' } } }
    })

    await flushPromises()
    await wrapper.get('[data-testid="claude-code-sync-models"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="claude-code-upstream-model-0"]').element).toHaveProperty('value', 'manual-model')
    const diagnostics = wrapper.get('[data-testid="claude-code-sync-result"]').text()
    expect(diagnostics).toContain('https://relay.example.com/catalog')
    expect(diagnostics).toContain('403')
    expect(diagnostics).toContain('text/html')
    expect(diagnostics).toContain('html')
    expect(diagnostics).not.toContain('private')
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountTokenStatusCell from '../AccountTokenStatusCell.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const mountCell = (tokenStatus: any) => mount(AccountTokenStatusCell, {
  props: {
    account: {
      id: 1,
      name: 'acct',
      platform: 'openai',
      type: 'oauth',
      token_status: tokenStatus
    } as any
  }
})

describe('AccountTokenStatusCell', () => {
  it('renders a recorded background refresh and next window compactly', () => {
    const wrapper = mountCell({
      access_token: 'present',
      refresh_token: 'present',
      last_attempt_at: '2026-07-15T12:00:00Z',
      last_result: 'success',
      trigger: 'background',
      next_window_start: '2026-07-15T13:30:00Z',
      next_window_end: '2026-07-15T13:35:00Z'
    })

    expect(wrapper.text()).toContain('AT ok')
    expect(wrapper.text()).toContain('RT ok')
    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.success')
    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.background')
    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.lastAttempt')
    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.nextWindow')
  })

  it('renders the recorded failed manual refresh message in title', () => {
    const wrapper = mountCell({
      access_token: 'present',
      refresh_token: 'present',
      last_attempt_at: '2026-07-15T12:00:00Z',
      last_result: 'failed',
      trigger: 'manual',
      error: 'invalid_grant refresh_token=***'
    })

    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.failed')
    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.manual')
    expect(wrapper.attributes('title')).toContain('invalid_grant')
  })

  it('does not infer refresh success from a present refresh token', () => {
    const wrapper = mountCell({ access_token: 'present', refresh_token: 'present' })

    expect(wrapper.text()).toContain('admin.accounts.tokenStatus.noRecord')
    expect(wrapper.text()).not.toContain('admin.accounts.tokenStatus.success')
  })
})

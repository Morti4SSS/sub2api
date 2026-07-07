import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AccountTokenStatusCell from '../AccountTokenStatusCell.vue'

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
  it('renders automatic token refresh state compactly', () => {
    const wrapper = mountCell({ access_token: 'present', refresh_token: 'present', refresh_state: 'auto' })

    expect(wrapper.text()).toContain('AT ok')
    expect(wrapper.text()).toContain('RT ok')
    expect(wrapper.text()).toContain('auto')
  })

  it('renders failed refresh message in title', () => {
    const wrapper = mountCell({
      access_token: 'present',
      refresh_token: 'present',
      refresh_state: 'failed',
      message: 'token refresh retry exhausted: timeout'
    })

    expect(wrapper.text()).toContain('fail')
    expect(wrapper.attributes('title')).toContain('timeout')
  })
})

import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModelsMock } = vi.hoisted(() => ({
  getAvailableModelsMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels: getAvailableModelsMock
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: '' },
    options: { type: Array, default: () => [] },
    valueKey: { type: String, default: 'value' },
    labelKey: { type: String, default: 'label' }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option
        v-for="option in options"
        :key="option[valueKey]"
        :value="option[valueKey]"
      >
        {{ option[labelKey] }}
      </option>
    </select>
  `
})

const TextAreaStub = defineComponent({
  name: 'TextArea',
  props: {
    modelValue: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: `
    <textarea
      v-bind="$attrs"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  `
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI OAuth',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    credentials: {},
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

describe('AccountTestModal', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    getAvailableModelsMock.mockReset()
    getAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockResolvedValue({ done: true, value: undefined })
        })
      }
    } as any)
    localStorage.setItem('auth_token', 'test-token')
  })

  afterEach(() => {
    global.fetch = originalFetch
    localStorage.clear()
  })

  it('posts compact mode for OpenAI compact probe', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="compact-test-settings"]').exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(false)
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'gpt-5.4',
      mode: 'compact'
    })
  })

  it('uses a substantial default question instead of a greeting or probe word', async () => {
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: buildAccount() },
      global: {
        stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    const [, options] = (global.fetch as any).mock.calls[0]
    const prompt = JSON.parse(options.body).prompt
    expect(Array.from(prompt).length).toBeGreaterThanOrEqual(24)
    expect(['hi', 'hello', '你好', '您好', '嗨', '哈喽', 'ping', 'test']).not.toContain(prompt.toLowerCase())
  })

  it('blocks greetings and undersized prompts before sending the request', async () => {
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: buildAccount() },
      global: {
        stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    for (const prompt of ['hi', '你好', '请说明接口是否可用？']) {
      await wrapper.get('textarea').setValue(prompt)
      await (wrapper.vm as any).startTest()

      expect(global.fetch).not.toHaveBeenCalled()
      expect(wrapper.text()).toContain('admin.accounts.testPromptProbeRejected')
    }
  })

  it('sends custom prompt for text account tests', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 9,
          name: 'Claude Relay',
          platform: 'anthropic',
          type: 'apikey',
          status: 'active'
        } as any
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'claude-sonnet-4-5'
    const textarea = wrapper.find('textarea')
    expect(textarea.exists()).toBe(true)
    await textarea.setValue('  请简要说明本次模型调用是否成功，并给出判断结果的依据。  ')
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'claude-sonnet-4-5',
      prompt: '请简要说明本次模型调用是否成功，并给出判断结果的依据。'
    })
  })

  it('renders Chat Completions path status from test SSE', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      encoder.encode('data: {"type":"status","text":"已通过 /v1/chat/completions 验证"}\n\n'),
      encoder.encode('data: {"type":"test_complete","success":true}\n\n')
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(() => Promise.resolve(
            chunks.length > 0
              ? { done: false, value: chunks.shift() }
              : { done: true, value: undefined }
          ))
        })
      }
    } as any)

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain('已通过 /v1/chat/completions 验证')
  })

  it('renders fixed-account gateway diagnostics from test SSE', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      encoder.encode('data: {"type":"diagnostics","data":{"account_id":301,"account_name":"Claude relay A","client_identity":"claude_code_cli","gateway_path":"claude_messages","requested_model":"glm-5.2","upstream_model":"glm-5.2","passthrough":true,"upstream_http_status":200}}\n\n'),
      encoder.encode('data: {"type":"test_complete","success":true}\n\n')
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(() => Promise.resolve(
            chunks.length > 0
              ? { done: false, value: chunks.shift() }
              : { done: true, value: undefined }
          ))
        })
      }
    } as any)
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: buildAccount() },
      global: {
        stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain('Claude relay A (#301)')
    expect(wrapper.text()).toContain('claude_code_cli')
    expect(wrapper.text()).toContain('glm-5.2 -> glm-5.2')
    expect(wrapper.text()).toContain('HTTP 200')
  })

  it('renders sanitized upstream error diagnostics from a failed test', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      encoder.encode('data: {"type":"diagnostics","data":{"account_id":303,"account_name":"Claude relay failure","client_identity":"claude_code_cli","gateway_path":"claude_messages","requested_model":"glm-5.2","upstream_model":"glm-5.2","passthrough":true,"upstream_http_status":404,"upstream_error_code":"upstream_model_not_found","upstream_error_reason":"model glm-5.2 not found; api_key=***"}}\n\n'),
      encoder.encode('data: {"type":"error","error":"Connection test failed"}\n\n')
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(() => Promise.resolve(
            chunks.length > 0
              ? { done: false, value: chunks.shift() }
              : { done: true, value: undefined }
          ))
        })
      }
    } as any)
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: buildAccount({ platform: 'anthropic' }) },
      global: {
        stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'glm-5.2'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.testDiagnosticErrorCode: upstream_model_not_found')
    expect(wrapper.text()).toContain('admin.accounts.testDiagnosticErrorReason: model glm-5.2 not found; api_key=***')
  })
})

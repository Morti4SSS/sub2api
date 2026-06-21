import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModels, getById, copyToClipboard } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  getById: vi.fn(),
  copyToClipboard: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels,
      getById
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.imagePromptDefault': 'Generate a cute orange cat astronaut sticker on a clean pastel background.'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.imageReceived' && params?.count) {
          return `received-${params.count}`
        }
        return messages[key] || key
      }
    })
  }
})

function createStreamResponse(lines: string[]) {
  const encoder = new TextEncoder()
  const chunks = lines.map((line) => encoder.encode(line))
  let index = 0

  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn().mockImplementation(async () => {
          if (index < chunks.length) {
            return { done: false, value: chunks[index++] }
          }
          return { done: true, value: undefined }
        })
      })
    }
  } as Response
}

function mountModal() {
  return mount(AccountTestModal, {
    props: {
      show: false,
      account: {
        id: 42,
        name: 'Gemini Image Test',
        platform: 'gemini',
        type: 'apikey',
        status: 'active'
      }
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: { template: '<div class="select-stub"></div>' },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

function mountOpenAIModal() {
  return mount(AccountTestModal, {
    props: {
      show: false,
      account: {
        id: 42,
        name: 'OpenAI OAuth',
        platform: 'openai',
        type: 'oauth',
        status: 'active',
        credentials: {},
        extra: {}
      }
    } as any,
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: { template: '<div class="select-stub"></div>' },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        Icon: true
      }
    }
  })
}

describe('AccountTestModal', () => {
  beforeEach(() => {
    getById.mockReset()
    getAvailableModels.mockResolvedValue([
      { id: 'gemini-2.0-flash', display_name: 'Gemini 2.0 Flash' },
      { id: 'gemini-2.5-flash-image', display_name: 'Gemini 2.5 Flash Image' },
      { id: 'gemini-3.1-flash-image', display_name: 'Gemini 3.1 Flash Image' }
    ])
    copyToClipboard.mockReset()
    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn((key: string) => (key === 'auth_token' ? 'test-token' : null)),
        setItem: vi.fn(),
        removeItem: vi.fn(),
        clear: vi.fn()
      },
      configurable: true
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gemini-2.5-flash-image"}\n',
        'data: {"type":"image","image_url":"data:image/png;base64,QUJD","mime_type":"image/png"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('gemini 图片模型测试会携带提示词并渲染图片预览', async () => {
    const wrapper = mountModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    await promptInput.setValue('draw a tiny orange cat astronaut')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gemini-3.1-flash-image',
      prompt: 'draw a tiny orange cat astronaut'
    })

    const preview = wrapper.find('img[alt="test-image-1"]')
    expect(preview.exists()).toBe(true)
    expect(preview.attributes('src')).toBe('data:image/png;base64,QUJD')
  })

  it('OpenAI 文本测试会显示上次提示词并提交本次输入', async () => {
    const getStoredItem = globalThis.localStorage.getItem as any
    getStoredItem.mockImplementation((key: string) => {
      if (key === 'auth_token') return 'test-token'
      if (key === 'sub2api.accountTestPrompt.openai.text') return 'previous custom prompt'
      return null
    })
    getAvailableModels.mockResolvedValue([{ id: 'gpt-5.4', display_name: 'GPT-5.4' }])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"test_complete","success":true}\n'
      ])
    ) as any

    const wrapper = mountOpenAIModal()
    await wrapper.setProps({ show: true })
    await flushPromises()

    const promptInput = wrapper.find('textarea.textarea-stub')
    expect(promptInput.exists()).toBe(true)
    expect((promptInput.element as HTMLTextAreaElement).value).toBe('previous custom prompt')
    await promptInput.setValue('new reusable prompt')

    const buttons = wrapper.findAll('button')
    const startButton = buttons.find((button) => button.text().includes('admin.accounts.startTest'))
    expect(startButton).toBeTruthy()

    await startButton!.trigger('click')
    await flushPromises()
    await flushPromises()

    const [, request] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(request.body)).toEqual({
      model_id: 'gpt-5.4',
      prompt: 'new reusable prompt'
    })
    expect(globalThis.localStorage.setItem).toHaveBeenCalledWith(
      'sub2api.accountTestPrompt.openai.text',
      'new reusable prompt'
    )
  })

  it('401 token_invalidated 时会回拉账号状态并抛出更新事件', async () => {
    getAvailableModels.mockResolvedValue([{ id: 'gpt-5.4', display_name: 'GPT-5.4' }])
    getById.mockResolvedValue({
      id: 42,
      name: 'OpenAI OAuth',
      platform: 'openai',
      type: 'oauth',
      status: 'error',
      error_message: 'Authentication failed (401)',
      credentials: {},
      extra: {}
    })
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"error","error":"API returned 401: {\\"error\\":{\\"code\\":\\"token_invalidated\\"}}"}\n',
        'data: {"type":"test_complete","success":false,"error":"API returned 401: {\\"error\\":{\\"code\\":\\"token_invalidated\\"}}"}\n'
      ])
    ) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 42,
          name: 'OpenAI OAuth',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {}
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'default'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(42)
    const emitted = wrapper.emitted('account-status-updated')
    expect(emitted).toBeTruthy()
    expect(emitted?.[0]?.[0]?.status).toBe('error')
    expect(wrapper.text()).toContain('error')
  })

  it('普通错误不会触发账号状态回拉', async () => {
    getAvailableModels.mockResolvedValue([{ id: 'gpt-5.4', display_name: 'GPT-5.4' }])
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"error","error":"API returned 500: upstream failed"}\n',
        'data: {"type":"test_complete","success":false,"error":"API returned 500: upstream failed"}\n'
      ])
    ) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 42,
          name: 'OpenAI OAuth',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {}
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(getById).not.toHaveBeenCalled()
    expect(wrapper.emitted('account-status-updated')).toBeFalsy()
  })

  it('状态回拉失败时保留错误输出且不抛异常', async () => {
    getAvailableModels.mockResolvedValue([{ id: 'gpt-5.4', display_name: 'GPT-5.4' }])
    getById.mockRejectedValue(new Error('fetch latest failed'))
    global.fetch = vi.fn().mockResolvedValue(
      createStreamResponse([
        'data: {"type":"test_start","model":"gpt-5.4"}\n',
        'data: {"type":"error","error":"API returned 401: {\\"error\\":{\\"code\\":\\"token_invalidated\\"}}"}\n',
        'data: {"type":"test_complete","success":false,"error":"API returned 401: {\\"error\\":{\\"code\\":\\"token_invalidated\\"}}"}\n'
      ])
    ) as any

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          id: 42,
          name: 'OpenAI OAuth',
          platform: 'openai',
          type: 'oauth',
          status: 'active',
          credentials: {},
          extra: {}
        }
      } as any,
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: { template: '<div class="select-stub"></div>' },
          TextArea: {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
          },
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(42)
    expect(wrapper.emitted('account-status-updated')).toBeFalsy()
    expect(wrapper.text()).toContain('API returned 401')
  })
})

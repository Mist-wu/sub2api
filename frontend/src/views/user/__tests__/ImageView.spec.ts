import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ImageView from '../ImageView.vue'

const { generate, getHistory, getHistoryFile, showSuccess, showError } = vi.hoisted(() => ({
  generate: vi.fn(),
  getHistory: vi.fn(),
  getHistoryFile: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

const messages: Record<string, string> = {
  'image.title': 'Image',
  'image.description': 'Generate images',
  'image.promptLabel': 'Prompt',
  'image.promptPlaceholder': 'Describe it',
  'image.promptRequired': 'Enter a prompt first',
  'image.generate': 'Generate Image',
  'image.generating': 'Generating...',
  'image.currentResult': 'Current Result',
  'image.noResultTitle': 'No image yet',
  'image.noResultHint': 'Enter a prompt',
  'image.history': 'History',
  'image.historyEmpty': 'No history',
  'image.generatedAt': 'Generated at',
  'image.revisedPrompt': 'Revised prompt',
  'image.download': 'Download',
  'image.view': 'View',
  'image.generateSuccess': 'Image generated',
  'image.generateFailed': 'Image generation failed',
  'image.loadHistoryFailed': 'Failed to load history',
  'image.loadFileFailed': 'Failed to load image',
  'image.loadingTitle': 'Generating image',
  'image.elapsed': '{seconds}s elapsed',
}

vi.mock('@/api', () => ({
  imageAPI: {
    generate,
    getHistory,
    getHistoryFile,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showSuccess, showError }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        let value = messages[key] ?? key
        if (params) {
          for (const [name, replacement] of Object.entries(params)) {
            value = value.replace(`{${name}}`, String(replacement))
          }
        }
        return value
      },
      locale: { value: 'en-US' },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }

describe('ImageView', () => {
  beforeEach(() => {
    generate.mockReset()
    getHistory.mockReset()
    getHistoryFile.mockReset()
    showSuccess.mockReset()
    showError.mockReset()

    getHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:image'),
      revokeObjectURL: vi.fn(),
    })
  })

  it('disables generation when the prompt is empty', async () => {
    const wrapper = mount(ImageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.find('form').trigger('submit.prevent')
    expect(generate).not.toHaveBeenCalled()
  })

  it('renders loading and successful preview', async () => {
    generate.mockResolvedValue({
      id: 12,
      prompt: 'glass city',
      revised_prompt: 'cinematic glass city',
      model: 'gpt-image-2',
      mime_type: 'image/png',
      image_base64: 'iVBORw0KGgo=',
      created_at: '2026-04-28T00:00:00Z',
    })

    const wrapper = mount(ImageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })
    await flushPromises()

    await wrapper.find('textarea').setValue('glass city')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(generate).toHaveBeenCalledWith('glass city')
    expect(wrapper.find('img').attributes('src')).toContain('data:image/png;base64,iVBORw0KGgo=')
    expect(wrapper.text()).toContain('cinematic glass city')
    expect(showSuccess).toHaveBeenCalledWith('Image generated')
  })
})

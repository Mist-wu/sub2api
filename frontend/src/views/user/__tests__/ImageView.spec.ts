import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ImageView from '../ImageView.vue'
import { resetImageGenerationJobsForTest } from '@/composables/useImageGenerationJobs'

const { generate, getGenerationJob, getHistory, getHistoryFile, showSuccess, showError } = vi.hoisted(() => ({
  generate: vi.fn(),
  getGenerationJob: vi.fn(),
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
  'image.generateQueued': 'Image task started',
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
  'image.activeJobs': 'Tasks {active}/{max}',
  'image.concurrentHint': 'Up to {max} images',
  'image.concurrencyFull': 'Limit reached',
  'image.concurrencyFullMessage': 'Wait for one to finish',
  'image.statusSubmitting': 'Submitting',
  'image.statusRunning': 'Generating',
  'image.statusSucceeded': 'Done',
  'image.statusFailed': 'Failed',
  'image.statusHistory': 'History',
}

vi.mock('@/api', () => ({
  imageAPI: {
    generate,
    getGenerationJob,
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
    vi.useRealTimers()
    window.localStorage.clear()
    resetImageGenerationJobsForTest()
    generate.mockReset()
    getGenerationJob.mockReset()
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
    vi.useFakeTimers()
    generate.mockResolvedValue({
      job_id: 'job-12',
      prompt: 'glass city',
      status: 'running',
      created_at: '2026-04-28T00:00:00Z',
      started_at: '2026-04-28T00:00:00Z',
    })
    getGenerationJob.mockResolvedValue({
      job_id: 'job-12',
      prompt: 'glass city',
      status: 'succeeded',
      created_at: '2026-04-28T00:00:00Z',
      started_at: '2026-04-28T00:00:00Z',
      completed_at: '2026-04-28T00:00:10Z',
      result: {
        id: 12,
        prompt: 'glass city',
        revised_prompt: 'cinematic glass city',
        model: 'gpt-image-2',
        mime_type: 'image/png',
        image_base64: 'iVBORw0KGgo=',
        thumbnail_mime_type: 'image/jpeg',
        thumbnail_base64: 'thumb',
        created_at: '2026-04-28T00:00:10Z',
      },
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
    expect(wrapper.text()).toContain('Generating')
    await vi.advanceTimersByTimeAsync(2500)
    await flushPromises()

    expect(getGenerationJob).toHaveBeenCalledWith('job-12')
    expect(wrapper.find('img').attributes('src')).toContain('data:image/jpeg;base64,thumb')
    expect(wrapper.text()).toContain('cinematic glass city')
    expect(showSuccess).toHaveBeenCalledWith('Image task started')
  })
})

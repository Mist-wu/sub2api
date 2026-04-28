<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="card">
        <div class="p-6">
          <div class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="flex items-center gap-3">
              <div class="flex h-11 w-11 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="sparkles" size="lg" />
              </div>
              <div>
                <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('image.title') }}</h1>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('image.description') }}</p>
              </div>
            </div>
            <div class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-600 dark:border-dark-700 dark:text-dark-300">
              {{ t('image.activeJobs', { active: activeJobCount, max: maxConcurrentJobs }) }}
            </div>
          </div>

          <form class="space-y-4" @submit.prevent="handleGenerate">
            <TextArea
              id="image-prompt"
              v-model="prompt"
              :label="t('image.promptLabel')"
              :placeholder="t('image.promptPlaceholder')"
              :error="promptError"
              rows="5"
              required
            />

            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <button
                type="submit"
                class="btn btn-primary min-h-[44px] w-full sm:w-auto"
                :disabled="!canSubmit"
              >
                <Icon name="sparkles" size="sm" class="mr-2" />
                {{ activeJobCount >= maxConcurrentJobs ? t('image.concurrencyFull') : t('image.generate') }}
              </button>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('image.concurrentHint', { max: maxConcurrentJobs }) }}</p>
            </div>
          </form>

          <transition name="fade">
            <div
              v-if="errorMessage"
              class="mt-5 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
            >
              {{ errorMessage }}
            </div>
          </transition>
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
        <section class="card overflow-hidden">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-800">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('image.currentResult') }}</h2>
          </div>

          <div class="p-6">
            <div
              v-if="resultCards.length === 0"
              class="flex min-h-[420px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 text-center dark:border-dark-700 dark:bg-dark-900"
            >
              <Icon name="sparkles" size="xl" class="text-gray-400 dark:text-dark-500" />
              <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('image.noResultTitle') }}</h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('image.noResultHint') }}</p>
            </div>

            <div v-else class="space-y-4">
              <article
                v-for="card in resultCards"
                :key="card.key"
                class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
              >
                <div class="grid gap-4 sm:grid-cols-[180px_minmax(0,1fr)]">
                  <div class="aspect-square overflow-hidden rounded-lg border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-800">
                    <img
                      v-if="card.thumbnailSrc"
                      :src="card.thumbnailSrc"
                      :alt="card.prompt"
                      class="h-full w-full object-cover"
                    />
                    <div v-else-if="card.status === 'running' || card.status === 'submitting'" class="flex h-full w-full flex-col items-center justify-center gap-3 text-center">
                      <LoadingSpinner size="lg" />
                      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('image.loadingTitle') }}</span>
                    </div>
                    <div v-else class="flex h-full w-full items-center justify-center">
                      <Icon name="sparkles" size="lg" class="text-gray-400 dark:text-dark-500" />
                    </div>
                  </div>

                  <div class="min-w-0">
                    <div class="mb-3 flex flex-wrap items-center gap-2">
                      <span :class="statusClass(card.status)" class="rounded-full px-2.5 py-1 text-xs font-medium">
                        {{ statusText(card.status) }}
                      </span>
                      <span
                        v-if="card.status === 'running' || card.status === 'submitting'"
                        class="text-xs text-gray-500 dark:text-dark-400"
                      >
                        {{ t('image.elapsed', { seconds: card.elapsedSeconds }) }}
                      </span>
                    </div>

                    <p class="break-words text-sm font-medium text-gray-800 dark:text-dark-100">{{ card.prompt }}</p>
                    <p v-if="card.revisedPrompt" class="mt-3 break-words text-xs leading-5 text-gray-500 dark:text-dark-400">
                      {{ t('image.revisedPrompt') }}: {{ card.revisedPrompt }}
                    </p>
                    <p v-if="card.errorMessage" class="mt-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300">
                      {{ card.errorMessage }}
                    </p>
                    <p v-if="card.createdAt" class="mt-3 text-xs text-gray-500 dark:text-dark-400">
                      {{ t('image.generatedAt') }}: {{ formatDate(card.createdAt) }}
                    </p>

                    <div v-if="card.imageId" class="mt-4 flex flex-wrap gap-2">
                      <button class="btn btn-secondary btn-sm" type="button" @click="downloadImage(card.imageId, card.mimeType)">
                        <Icon name="download" size="sm" class="mr-1" />
                        {{ t('image.download') }}
                      </button>
                    </div>
                  </div>
                </div>
              </article>
            </div>
          </div>
        </section>

        <section class="card overflow-hidden">
          <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-800">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('image.history') }}</h2>
            <button class="btn btn-ghost btn-sm" type="button" :disabled="historyLoading" @click="loadHistory">
              <Icon name="refresh" size="sm" />
            </button>
          </div>

          <div class="max-h-[720px] overflow-y-auto p-4">
            <div v-if="historyLoading" class="space-y-3">
              <div v-for="index in 3" :key="index" class="h-24 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"></div>
            </div>
            <div v-else-if="historyItems.length === 0" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('image.historyEmpty') }}
            </div>
            <div v-else class="space-y-3">
              <article
                v-for="item in historyItems"
                :key="item.id"
                class="rounded-lg border border-gray-200 bg-white p-3 transition-colors hover:border-primary-300 dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-700"
              >
                <button class="flex w-full gap-3 text-left" type="button" @click="viewHistory(item)">
                  <div class="h-20 w-20 shrink-0 overflow-hidden rounded-md bg-gray-100 dark:bg-dark-800">
                    <img
                      v-if="thumbnailSrc(item)"
                      :src="thumbnailSrc(item)"
                      :alt="item.prompt"
                      class="h-full w-full object-cover"
                    />
                    <div v-else class="flex h-full w-full items-center justify-center">
                      <Icon name="sparkles" size="sm" class="text-gray-400 dark:text-dark-500" />
                    </div>
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="history-prompt text-sm font-medium text-gray-800 dark:text-dark-100">{{ item.prompt }}</p>
                    <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ formatDate(item.created_at) }}</p>
                  </div>
                </button>
                <div class="mt-3 flex justify-end gap-2">
                  <button class="btn btn-ghost btn-sm" type="button" @click="viewHistory(item)">
                    <Icon name="eye" size="sm" class="mr-1" />
                    {{ t('image.view') }}
                  </button>
                  <button class="btn btn-secondary btn-sm" type="button" @click="downloadImage(item.id, item.mime_type)">
                    <Icon name="download" size="sm" class="mr-1" />
                    {{ t('image.download') }}
                  </button>
                </div>
              </article>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TextArea from '@/components/common/TextArea.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { imageAPI, type UserImageGeneration, type UserImageHistoryItem } from '@/api'
import { useAppStore } from '@/stores'
import { useImageGenerationJobs, type ImageGenerationClientJob, type ImageGenerationClientJobStatus } from '@/composables/useImageGenerationJobs'

interface ResultCard {
  key: string
  imageId?: number
  prompt: string
  revisedPrompt?: string | null
  mimeType: string
  thumbnailSrc: string
  status: ImageGenerationClientJobStatus | 'history'
  errorMessage?: string
  elapsedSeconds: number
  createdAt: string
}

const { t, locale } = useI18n()
const appStore = useAppStore()
const { jobs, activeJobCount, lastCompletedAt, maxConcurrentJobs, startImageJob } = useImageGenerationJobs()

const prompt = ref('')
const promptError = ref('')
const errorMessage = ref('')
const historyItems = ref<UserImageHistoryItem[]>([])
const historyLoading = ref(false)
const selectedHistory = ref<UserImageHistoryItem | null>(null)

const trimmedPrompt = computed(() => prompt.value.trim())
const canSubmit = computed(() => Boolean(trimmedPrompt.value) && activeJobCount.value < maxConcurrentJobs)
const resultCards = computed<ResultCard[]>(() => {
  const cards = jobs.value.map(jobToResultCard)
  if (selectedHistory.value && !cards.some((card) => card.imageId === selectedHistory.value?.id)) {
    cards.unshift(historyToResultCard(selectedHistory.value))
  }
  return cards
})

onMounted(() => {
  void loadHistory()
})

watch(lastCompletedAt, (value, oldValue) => {
  if (value && value !== oldValue) {
    void loadHistory()
  }
})

async function handleGenerate() {
  promptError.value = ''
  errorMessage.value = ''
  if (!trimmedPrompt.value) {
    promptError.value = t('image.promptRequired')
    return
  }
  if (activeJobCount.value >= maxConcurrentJobs) {
    errorMessage.value = t('image.concurrencyFullMessage', { max: maxConcurrentJobs })
    return
  }

  try {
    await startImageJob(trimmedPrompt.value)
    prompt.value = ''
    appStore.showSuccess(t('image.generateQueued'))
  } catch (error) {
    const message = mapImageErrorMessage(error)
    errorMessage.value = message
    appStore.showError(message)
  }
}

async function loadHistory() {
  historyLoading.value = true
  try {
    const data = await imageAPI.getHistory(1, 20)
    historyItems.value = data.items || []
  } catch (error) {
    appStore.showError(getErrorMessage(error) || t('image.loadHistoryFailed'))
  } finally {
    historyLoading.value = false
  }
}

function viewHistory(item: UserImageHistoryItem) {
  selectedHistory.value = item
}

async function downloadImage(id: number, mimeType: string) {
  try {
    const blob = await imageAPI.getHistoryFile(id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `image-${id}${extensionForMimeType(mimeType)}`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch (error) {
    appStore.showError(getErrorMessage(error) || t('image.loadFileFailed'))
  }
}

function jobToResultCard(job: ImageGenerationClientJob): ResultCard {
  const result = job.result
  return {
    key: job.localId,
    imageId: result?.id,
    prompt: result?.prompt || job.prompt,
    revisedPrompt: result?.revised_prompt,
    mimeType: result?.mime_type || 'image/png',
    thumbnailSrc: result ? generationPreviewSrc(result) : '',
    status: job.status,
    errorMessage: job.errorMessage,
    elapsedSeconds: job.elapsedSeconds,
    createdAt: result?.created_at || job.completedAt || job.createdAt,
  }
}

function historyToResultCard(item: UserImageHistoryItem): ResultCard {
  return {
    key: `history-${item.id}`,
    imageId: item.id,
    prompt: item.prompt,
    revisedPrompt: item.revised_prompt,
    mimeType: item.mime_type,
    thumbnailSrc: thumbnailSrc(item),
    status: 'history',
    elapsedSeconds: 0,
    createdAt: item.created_at,
  }
}

function thumbnailSrc(item: Pick<UserImageHistoryItem, 'thumbnail_base64' | 'thumbnail_mime_type' | 'mime_type'>) {
  if (!item.thumbnail_base64) {
    return ''
  }
  return `data:${item.thumbnail_mime_type || item.mime_type || 'image/jpeg'};base64,${item.thumbnail_base64}`
}

function generationPreviewSrc(item: UserImageGeneration) {
  if (item.thumbnail_base64) {
    return `data:${item.thumbnail_mime_type || item.mime_type || 'image/jpeg'};base64,${item.thumbnail_base64}`
  }
  if (item.image_base64) {
    return `data:${item.mime_type || 'image/png'};base64,${item.image_base64}`
  }
  return ''
}

function statusText(status: ResultCard['status']) {
  switch (status) {
    case 'submitting':
      return t('image.statusSubmitting')
    case 'running':
      return t('image.statusRunning')
    case 'succeeded':
      return t('image.statusSucceeded')
    case 'failed':
      return t('image.statusFailed')
    default:
      return t('image.statusHistory')
  }
}

function statusClass(status: ResultCard['status']) {
  switch (status) {
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-300'
    case 'running':
    case 'submitting':
      return 'bg-primary-100 text-primary-700 dark:bg-primary-950/40 dark:text-primary-300'
    default:
      return 'bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-300'
  }
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function extensionForMimeType(mimeType: string) {
  switch (mimeType) {
    case 'image/jpeg':
    case 'image/jpg':
      return '.jpg'
    case 'image/webp':
      return '.webp'
    case 'image/gif':
      return '.gif'
    default:
      return '.png'
  }
}

function mapImageErrorMessage(error: unknown) {
  const message = getErrorMessage(error)
  if (message === 'IMAGE_CONCURRENCY_LIMIT_EXCEEDED') {
    return t('image.concurrencyFullMessage', { max: maxConcurrentJobs })
  }
  if (message === 'IMAGE_PROMPT_REQUIRED') {
    return t('image.promptRequired')
  }
  return message || t('image.generateFailed')
}

function getErrorMessage(error: unknown) {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = String((error as { message?: unknown }).message || '').trim()
    if (message) return message
  }
  return ''
}
</script>

<style scoped>
.history-prompt {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>

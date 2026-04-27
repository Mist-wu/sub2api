<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <div class="card">
        <div class="p-6">
          <div class="mb-5 flex items-center gap-3">
            <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
              <Icon name="sparkles" size="lg" />
            </div>
            <div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('image.title') }}</h1>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('image.description') }}</p>
            </div>
          </div>

          <form class="space-y-4" @submit.prevent="handleGenerate">
            <TextArea
              id="image-prompt"
              v-model="prompt"
              :label="t('image.promptLabel')"
              :placeholder="t('image.promptPlaceholder')"
              :disabled="generating"
              :error="promptError"
              rows="5"
              required
            />

            <button
              type="submit"
              class="btn btn-primary min-h-[44px] w-full sm:w-auto"
              :disabled="generating || !prompt.trim()"
            >
              <LoadingSpinner v-if="generating" size="sm" color="white" class="mr-2" />
              <Icon v-else name="sparkles" size="sm" class="mr-2" />
              {{ generating ? t('image.generating') : t('image.generate') }}
            </button>
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
              v-if="generating"
              class="flex min-h-[420px] flex-col items-center justify-center rounded-lg border border-dashed border-primary-200 bg-primary-50/60 text-center dark:border-primary-800/60 dark:bg-primary-950/20"
            >
              <LoadingSpinner size="xl" />
              <h3 class="mt-5 text-base font-semibold text-gray-900 dark:text-white">{{ t('image.loadingTitle') }}</h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('image.elapsed', { seconds: elapsedSeconds }) }}</p>
            </div>

            <div v-else-if="activePreview" class="space-y-4">
              <div class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900">
                <img
                  :src="activePreview.src"
                  :alt="activePreview.prompt"
                  class="max-h-[620px] w-full object-contain"
                />
              </div>
              <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div class="min-w-0">
                  <p class="break-words text-sm text-gray-700 dark:text-dark-200">{{ activePreview.prompt }}</p>
                  <p v-if="activePreview.revisedPrompt" class="mt-2 break-words text-xs text-gray-500 dark:text-dark-400">
                    {{ t('image.revisedPrompt') }}: {{ activePreview.revisedPrompt }}
                  </p>
                  <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                    {{ t('image.generatedAt') }}: {{ formatDate(activePreview.createdAt) }}
                  </p>
                </div>
                <button class="btn btn-secondary shrink-0" type="button" @click="downloadImage(activePreview.id, activePreview.mimeType)">
                  <Icon name="download" size="sm" class="mr-2" />
                  {{ t('image.download') }}
                </button>
              </div>
            </div>

            <div
              v-else
              class="flex min-h-[420px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 text-center dark:border-dark-700 dark:bg-dark-900"
            >
              <Icon name="sparkles" size="xl" class="text-gray-400 dark:text-dark-500" />
              <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('image.noResultTitle') }}</h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('image.noResultHint') }}</p>
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
                      v-if="historyImageUrls[item.id]"
                      :src="historyImageUrls[item.id]"
                      :alt="item.prompt"
                      class="h-full w-full object-cover"
                    />
                    <div v-else class="flex h-full w-full items-center justify-center">
                      <LoadingSpinner size="sm" color="secondary" />
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TextArea from '@/components/common/TextArea.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { imageAPI, type UserImageGeneration, type UserImageHistoryItem } from '@/api'
import { useAppStore } from '@/stores'

interface ImagePreview {
  id: number
  prompt: string
  revisedPrompt?: string | null
  mimeType: string
  src: string
  createdAt: string
}

const { t, locale } = useI18n()
const appStore = useAppStore()

const prompt = ref('')
const promptError = ref('')
const errorMessage = ref('')
const generating = ref(false)
const elapsedSeconds = ref(0)
const activePreview = ref<ImagePreview | null>(null)
const historyItems = ref<UserImageHistoryItem[]>([])
const historyImageUrls = ref<Record<number, string>>({})
const historyLoading = ref(false)

let timer: number | undefined

const trimmedPrompt = computed(() => prompt.value.trim())

onMounted(() => {
  void loadHistory()
})

onBeforeUnmount(() => {
  stopTimer()
  revokeHistoryUrls()
})

async function handleGenerate() {
  promptError.value = ''
  errorMessage.value = ''
  if (!trimmedPrompt.value) {
    promptError.value = t('image.promptRequired')
    return
  }

  generating.value = true
  startTimer()
  try {
    const result = await imageAPI.generate(trimmedPrompt.value)
    activePreview.value = previewFromGeneration(result)
    appStore.showSuccess(t('image.generateSuccess'))
    await loadHistory()
  } catch (error) {
    const message = getErrorMessage(error) || t('image.generateFailed')
    errorMessage.value = message
    appStore.showError(message)
  } finally {
    generating.value = false
    stopTimer()
  }
}

async function loadHistory() {
  historyLoading.value = true
  try {
    const data = await imageAPI.getHistory(1, 20)
    historyItems.value = data.items || []
    await Promise.allSettled(historyItems.value.map((item) => ensureHistoryImageUrl(item.id)))
  } catch (error) {
    appStore.showError(getErrorMessage(error) || t('image.loadHistoryFailed'))
  } finally {
    historyLoading.value = false
  }
}

async function viewHistory(item: UserImageHistoryItem) {
  try {
    const src = await ensureHistoryImageUrl(item.id)
    activePreview.value = {
      id: item.id,
      prompt: item.prompt,
      revisedPrompt: item.revised_prompt,
      mimeType: item.mime_type,
      src,
      createdAt: item.created_at
    }
  } catch (error) {
    appStore.showError(getErrorMessage(error) || t('image.loadFileFailed'))
  }
}

async function ensureHistoryImageUrl(id: number): Promise<string> {
  const existing = historyImageUrls.value[id]
  if (existing) return existing
  const blob = await imageAPI.getHistoryFile(id)
  const url = URL.createObjectURL(blob)
  historyImageUrls.value = { ...historyImageUrls.value, [id]: url }
  return url
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

function previewFromGeneration(result: UserImageGeneration): ImagePreview {
  return {
    id: result.id,
    prompt: result.prompt,
    revisedPrompt: result.revised_prompt,
    mimeType: result.mime_type,
    src: `data:${result.mime_type};base64,${result.image_base64}`,
    createdAt: result.created_at
  }
}

function startTimer() {
  stopTimer()
  elapsedSeconds.value = 0
  timer = window.setInterval(() => {
    elapsedSeconds.value += 1
  }, 1000)
}

function stopTimer() {
  if (timer !== undefined) {
    window.clearInterval(timer)
    timer = undefined
  }
}

function revokeHistoryUrls() {
  Object.values(historyImageUrls.value).forEach((url) => URL.revokeObjectURL(url))
  historyImageUrls.value = {}
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

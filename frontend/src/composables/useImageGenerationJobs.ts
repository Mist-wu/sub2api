import { computed, reactive } from 'vue'
import { imageAPI, type UserImageGeneration, type UserImageGenerationJob } from '@/api'

export const MAX_CONCURRENT_IMAGE_JOBS = 3

export type ImageGenerationClientJobStatus = 'submitting' | 'running' | 'succeeded' | 'failed'

export interface ImageGenerationClientJob {
  localId: string
  jobId?: string
  prompt: string
  status: ImageGenerationClientJobStatus
  result?: UserImageGeneration
  errorMessage?: string
  errorReason?: string
  createdAt: string
  startedAt?: string
  completedAt?: string
  elapsedSeconds: number
}

const STORAGE_KEY = 'sub2api:image-generation-jobs'
const POLL_INTERVAL_MS = 2500
const MAX_STORED_JOBS = 20

const state = reactive({
  hydrated: false,
  jobs: [] as ImageGenerationClientJob[],
  now: Date.now(),
  lastCompletedAt: 0,
})

let ticker: number | undefined
const pollingLocalIds = new Set<string>()

const activeJobCount = computed(() => state.jobs.filter((job) => isActiveStatus(job.status)).length)
const jobs = computed(() => state.jobs)
const lastCompletedAt = computed(() => state.lastCompletedAt)

export function useImageGenerationJobs() {
  hydrateJobs()

  return {
    jobs,
    activeJobCount,
    lastCompletedAt,
    maxConcurrentJobs: MAX_CONCURRENT_IMAGE_JOBS,
    startImageJob,
    removeImageJob,
    retryImageJobPolling,
  }
}

export async function startImageJob(prompt: string): Promise<ImageGenerationClientJob> {
  hydrateJobs()
  const normalizedPrompt = prompt.trim()
  if (!normalizedPrompt) {
    throw new Error('IMAGE_PROMPT_REQUIRED')
  }
  if (activeJobCount.value >= MAX_CONCURRENT_IMAGE_JOBS) {
    throw new Error('IMAGE_CONCURRENCY_LIMIT_EXCEEDED')
  }

  const now = new Date().toISOString()
  const localJob: ImageGenerationClientJob = {
    localId: createLocalJobId(),
    prompt: normalizedPrompt,
    status: 'submitting',
    createdAt: now,
    startedAt: now,
    elapsedSeconds: 0,
  }
  state.jobs.unshift(localJob)
  trimStoredJobs()
  ensureTicker()
  persistJobs()

  try {
    const remoteJob = await imageAPI.generate(normalizedPrompt)
    applyRemoteJob(localJob.localId, remoteJob)
    if (remoteJob.status === 'running') {
      void pollImageJob(localJob.localId)
    }
  } catch (error) {
    markJobFailed(localJob.localId, getErrorMessage(error) || 'IMAGE_GENERATION_FAILED')
    persistJobs()
    throw error
  }

  return localJob
}

export function retryImageJobPolling(localId: string) {
  hydrateJobs()
  const job = state.jobs.find((item) => item.localId === localId)
  if (!job || !job.jobId || !isActiveStatus(job.status)) {
    return
  }
  void pollImageJob(localId)
}

export function removeImageJob(localId: string) {
  hydrateJobs()
  const index = state.jobs.findIndex((job) => job.localId === localId)
  if (index >= 0) {
    state.jobs.splice(index, 1)
    persistJobs()
  }
}

export function resetImageGenerationJobsForTest() {
  state.hydrated = false
  state.jobs.splice(0, state.jobs.length)
  state.lastCompletedAt = 0
  pollingLocalIds.clear()
  stopTicker()
}

async function pollImageJob(localId: string) {
  if (pollingLocalIds.has(localId)) {
    return
  }
  pollingLocalIds.add(localId)

  let failures = 0
  try {
    while (true) {
      const job = state.jobs.find((item) => item.localId === localId)
      if (!job || !job.jobId || !isActiveStatus(job.status)) {
        return
      }

      await wait(POLL_INTERVAL_MS)

      try {
        const remoteJob = await imageAPI.getGenerationJob(job.jobId)
        failures = 0
        applyRemoteJob(localId, remoteJob)
        if (remoteJob.status !== 'running') {
          return
        }
      } catch (error) {
        failures += 1
        if (failures >= 3) {
          markJobFailed(localId, getErrorMessage(error) || 'IMAGE_POLL_FAILED')
          persistJobs()
          return
        }
      }
    }
  } finally {
    pollingLocalIds.delete(localId)
  }
}

function applyRemoteJob(localId: string, remoteJob: UserImageGenerationJob) {
  const job = state.jobs.find((item) => item.localId === localId)
  if (!job) {
    return
  }

  job.jobId = remoteJob.job_id
  job.prompt = remoteJob.prompt || job.prompt
  job.status = remoteJob.status === 'succeeded' ? 'succeeded' : remoteJob.status === 'failed' ? 'failed' : 'running'
  job.startedAt = remoteJob.started_at || job.startedAt
  job.completedAt = remoteJob.completed_at || job.completedAt
  job.errorMessage = remoteJob.error_message
  job.errorReason = remoteJob.error_reason
  if (remoteJob.result) {
    job.result = remoteJob.result
  }

  if (job.status === 'succeeded' || job.status === 'failed') {
    state.lastCompletedAt = Date.now()
  }
  trimStoredJobs()
  ensureTicker()
  persistJobs()
}

function markJobFailed(localId: string, message: string) {
  const job = state.jobs.find((item) => item.localId === localId)
  if (!job) {
    return
  }
  job.status = 'failed'
  job.errorMessage = message
  job.completedAt = new Date().toISOString()
  state.lastCompletedAt = Date.now()
  ensureTicker()
}

function hydrateJobs() {
  if (state.hydrated) {
    return
  }
  state.hydrated = true

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as ImageGenerationClientJob[]
      if (Array.isArray(parsed)) {
        state.jobs.splice(0, state.jobs.length, ...parsed.slice(0, MAX_STORED_JOBS))
      }
    }
  } catch {
    state.jobs.splice(0, state.jobs.length)
  }

  state.jobs.forEach((job) => {
    if (job.jobId && isActiveStatus(job.status)) {
      void pollImageJob(job.localId)
    }
  })
  ensureTicker()
}

function persistJobs() {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state.jobs.slice(0, MAX_STORED_JOBS)))
  } catch {
    // localStorage can fail in private mode; the in-memory jobs still keep the SPA route-switch case working.
  }
}

function trimStoredJobs() {
  if (state.jobs.length > MAX_STORED_JOBS) {
    state.jobs.splice(MAX_STORED_JOBS)
  }
}

function ensureTicker() {
  if (state.jobs.some((job) => isActiveStatus(job.status))) {
    if (ticker === undefined) {
      ticker = window.setInterval(() => {
        state.now = Date.now()
        state.jobs.forEach((job) => {
          if (isActiveStatus(job.status)) {
            const started = Date.parse(job.startedAt || job.createdAt)
            if (Number.isFinite(started)) {
              job.elapsedSeconds = Math.max(0, Math.floor((state.now - started) / 1000))
            }
          }
        })
      }, 1000)
    }
    return
  }
  stopTicker()
}

function stopTicker() {
  if (ticker !== undefined) {
    window.clearInterval(ticker)
    ticker = undefined
  }
}

function isActiveStatus(status: ImageGenerationClientJobStatus) {
  return status === 'submitting' || status === 'running'
}

function createLocalJobId() {
  const random = Math.random().toString(16).slice(2)
  return `local_${Date.now()}_${random}`
}

function wait(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function getErrorMessage(error: unknown) {
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message || '').trim()
  }
  return ''
}

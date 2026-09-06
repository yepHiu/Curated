import { computed, onBeforeUnmount, ref, shallowReactive } from 'vue'
import type { Movie } from '@/domain/movie/types'
import { captureCuratedFrameCandidate, exportCuratedFrameCandidate, saveCuratedFrameCandidate, type CuratedFrameCaptureCandidate, type SaveCuratedCaptureResult } from '@/lib/curated-frames/save-capture'
import { deleteCuratedFrame } from '@/lib/curated-frames/db'
import { i18n } from '@/i18n'
import { triggerDownloadBlob } from '@/lib/curated-frames/export-file'
import { formatFrameFilename } from '@/lib/curated-frames/capture'

export interface CaptureJob {
  key: string
  movie: Movie
  positionSec: number
  phase: 'capturing' | 'ready' | 'queued' | 'saving' | 'saved' | 'error' | 'export-error'
  preview: string
  error: string
  candidate?: CuratedFrameCaptureCandidate
  capture: Promise<void>
  completion?: Promise<SaveCuratedCaptureResult>
  bytes: number
  committed: boolean
  exporting: boolean
  originalBlob?: Blob
}

// Jobs own the movie snapshot, image and retry identity. Upload concurrency is
// one; admission is bounded before allocating a full-resolution canvas.
export function useCuratedCaptureQueue() {
  const jobs = ref<CaptureJob[]>([])
  const latest = computed(() => jobs.value[jobs.value.length - 1])
  const pendingCount = computed(() => jobs.value.filter(j => ['capturing', 'queued', 'saving'].includes(j.phase)).length)
  let tail: Promise<unknown> = Promise.resolve()
  let disposed = false
  const expiry = new Map<CaptureJob, ReturnType<typeof setTimeout>>()

  function dismiss(job: CaptureJob) {
    if (job.exporting || ['queued', 'saving'].includes(job.phase)) return
    clearTimeout(expiry.get(job))
    expiry.delete(job)
    jobs.value = jobs.value.filter(j => j !== job)
    if (job.preview) URL.revokeObjectURL(job.preview)
    job.preview = ''
    job.candidate = undefined
    job.originalBlob = undefined
    job.bytes = 0
  }

  function retain(job: CaptureJob) {
    if (disposed) return
    clearTimeout(expiry.get(job))
    expiry.set(job, setTimeout(() => dismiss(job), 180_000))
  }

  function prepare(video: HTMLVideoElement, movie: Movie, positionSec: number): CaptureJob | undefined {
    for (const job of [...jobs.value]) if (job.phase === 'saved' && !job.exporting) dismiss(job)
    const bytes = video.videoWidth * video.videoHeight * 4
    if (disposed || jobs.value.length >= 4 || jobs.value.reduce((n, j) => n + j.bytes, 0) + bytes > 128 * 1024 * 1024) return
    const job = shallowReactive<CaptureJob>({
      key: crypto.randomUUID(), movie: { ...movie, actors: [...movie.actors] }, positionSec,
      phase: 'capturing', preview: '', error: '', bytes, committed: false, exporting: false, capture: Promise.resolve(),
    })
    jobs.value.push(job)
    job.capture = captureCuratedFrameCandidate(video, { positionSecOverride: positionSec, onPreview: url => { job.preview = url } }).then(result => {
      if (disposed || !jobs.value.includes(job)) return
      if (!result.ok) { job.phase = 'error'; job.error = result.reason; job.bytes = 0; retain(job); return }
      job.candidate = result.candidate
      job.bytes = result.candidate.blob.size
      job.preview = URL.createObjectURL(result.candidate.blob)
      if (job.phase === 'capturing') job.phase = 'ready'
    }).catch(() => { job.phase = 'error'; job.bytes = 0; job.error = i18n.global.t('curated.captureBlobFail'); retain(job) })
    return job
  }

  async function retryExport(job: CaptureJob) {
    if (!job.candidate || job.exporting) return
    job.exporting = true
    try { await exportCuratedFrameCandidate(job.candidate, job.movie); job.phase = 'saved'; job.error = '' }
    catch { job.phase = 'export-error'; job.error = i18n.global.t('curated.captureExportFailed') }
    finally { job.exporting = false; retain(job) }
  }

  function submit(job: CaptureJob): Promise<SaveCuratedCaptureResult> {
    if (job.completion) return job.completion
    job.phase = 'queued'
    const run = async (): Promise<SaveCuratedCaptureResult> => {
      await job.capture
      if (disposed || !job.candidate) return { ok: false, reason: job.error || i18n.global.t('curated.captureNotReady') }
      job.phase = 'saving'
      const result = await saveCuratedFrameCandidate(job.candidate, job.movie, { skipExport: true })
      job.phase = result.ok ? 'saved' : 'error'
      job.error = result.ok ? '' : result.reason
      job.committed = result.ok
      retain(job)
      if (result.ok && !disposed) void retryExport(job)
      return result
    }
    const completion = tail.then(run).catch((): SaveCuratedCaptureResult => {
      job.phase = 'error'; job.error = i18n.global.t('curated.saveFailedApi'); retain(job)
      return { ok: false, reason: job.error }
    })
    job.completion = completion
    tail = completion
    void completion.then(result => { if (!result.ok) job.completion = undefined })
    return completion
  }

  async function undo(job: CaptureJob) {
    if (!job.committed || !job.candidate || job.exporting) return
    try { await deleteCuratedFrame(job.candidate.id); dismiss(job) }
    catch { job.error = i18n.global.t('curated.captureUndoFailed') }
  }

  function downloadOriginal(job: CaptureJob) {
    if (job.candidate) triggerDownloadBlob(job.originalBlob ?? job.candidate.blob, formatFrameFilename(job.movie.code, job.positionSec, job.candidate.capturedAt))
  }

  async function compressAndRetry(job: CaptureJob) {
    if (!job.candidate || job.committed || job.candidate.blob.size <= 12 * 1024 * 1024) return
    if (job.phase !== 'error') return
    job.phase = 'capturing'
    try {
      const bitmap = await createImageBitmap(job.candidate.blob)
      const canvas = document.createElement('canvas'); canvas.width = bitmap.width; canvas.height = bitmap.height
      const ctx = canvas.getContext('2d'); if (!ctx) { bitmap.close(); return }
      ctx.drawImage(bitmap, 0, 0); bitmap.close()
      const blob = await new Promise<Blob | null>(resolve => canvas.toBlob(resolve, 'image/jpeg', .9))
      canvas.width = canvas.height = 1
      if (!blob) throw new Error('encoding failed')
      job.originalBlob = job.candidate.blob
      job.candidate = { ...job.candidate, blob }
      job.bytes = blob.size + job.originalBlob.size
      await submit(job)
    } catch { job.phase = 'error'; job.error = i18n.global.t('curated.captureBlobFail') }
  }

  onBeforeUnmount(() => {
    disposed = true
    for (const timer of expiry.values()) clearTimeout(timer)
    expiry.clear()
    for (const job of jobs.value) { if (job.preview) URL.revokeObjectURL(job.preview) }
    jobs.value = []
  })
  return { jobs, latest, pendingCount, prepare, submit, retryExport, undo, dismiss, downloadOriginal, compressAndRetry }
}

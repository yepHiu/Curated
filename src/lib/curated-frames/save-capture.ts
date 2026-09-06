import type { Movie } from "@/domain/movie/types"
import { api } from "@/api/endpoints"
import { i18n } from "@/i18n"
import { captureVideoFrameToPng, formatFrameFilename } from "@/lib/curated-frames/capture"
import { getStoredDirectoryHandle, putCuratedFrame } from "@/lib/curated-frames/db"
import { triggerDownloadBlob, writeBlobToDirectory } from "@/lib/curated-frames/export-file"
import { bumpCuratedFramesRevision } from "@/lib/curated-frames/revision"
import { getCuratedFrameSaveMode } from "@/lib/curated-frames/settings-storage"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"

export type SaveCuratedCaptureResult =
  | { ok: true; id: string; positionSec: number; exportFailed?: boolean }
  | { ok: false; reason: string }

export type CuratedFrameCaptureCandidate = {
  id: string
  blob: Blob
  positionSec: number
  capturedAt: string
}

export type SaveCuratedCaptureOptions = {
  positionSecOverride?: number
  onPreview?: (url: string) => void
}

export function resolveCuratedCapturePositionSec(
  videoCurrentTime: number,
  positionSecOverride?: number,
): number {
  const explicit = Number(positionSecOverride)
  if (Number.isFinite(explicit) && explicit >= 0) {
    return explicit
  }
  return Number.isFinite(videoCurrentTime) && videoCurrentTime >= 0 ? videoCurrentTime : 0
}

export async function captureCuratedFrameCandidate(
  video: HTMLVideoElement,
  options: SaveCuratedCaptureOptions = {},
): Promise<{ ok: true; candidate: CuratedFrameCaptureCandidate } | { ok: false; reason: string }> {
  const positionSec = resolveCuratedCapturePositionSec(video.currentTime, options.positionSecOverride)
  const capturedAt = new Date().toISOString()
  const cap = await captureVideoFrameToPng(video, options.onPreview)
  if (!cap.ok) {
    return { ok: false, reason: cap.reason }
  }
  return {
    ok: true,
    candidate: {
      id: crypto.randomUUID(),
      blob: cap.blob,
      positionSec,
      capturedAt,
    },
  }
}

export async function saveCuratedFrameCandidate(
  candidate: CuratedFrameCaptureCandidate,
  movie: Movie,
  options: { skipExport?: boolean } = {},
): Promise<SaveCuratedCaptureResult> {
  const row = {
    id: candidate.id,
    movieId: movie.id,
    title: movie.title,
    code: movie.code,
    actors: [...movie.actors],
    positionSec: candidate.positionSec,
    capturedAt: candidate.capturedAt,
    tags: [] as string[],
    imageBlob: candidate.blob,
  }

  try {
    if (USE_WEB) {
      if (candidate.blob.size > 12 * 1024 * 1024) return { ok: false, reason: i18n.global.t('curated.captureTooLarge') }
      await api.createCuratedFrameUpload({
        id: row.id,
        movieId: row.movieId,
        title: row.title,
        code: row.code,
        actors: row.actors,
        positionSec: row.positionSec,
        capturedAt: row.capturedAt,
        tags: row.tags,
      }, candidate.blob)
      bumpCuratedFramesRevision()
    } else {
      await putCuratedFrame(row)
    }
  } catch (error) {
    void error
    return {
      ok: false,
      reason: USE_WEB
        ? i18n.global.t("curated.saveFailedApi")
        : i18n.global.t("curated.saveFailedIdb"),
    }
  }

  let exportFailed = false
  if (!options.skipExport) {
    try { await exportCuratedFrameCandidate(candidate, movie) } catch { exportFailed = true }
  }
  return { ok: true, id: row.id, positionSec: row.positionSec, ...(exportFailed ? { exportFailed } : {}) }
}

export async function exportCuratedFrameCandidate(candidate: CuratedFrameCaptureCandidate, movie: Movie): Promise<void> {
  const filename = formatFrameFilename(movie.code, candidate.positionSec, candidate.capturedAt).replace(/\.png$/, candidate.blob.type === 'image/jpeg' ? '.jpg' : '.png')
  const mode = getCuratedFrameSaveMode()
  if (mode === "download") {
    triggerDownloadBlob(candidate.blob, filename)
  }
  if (mode === "directory") {
    const dir = await getStoredDirectoryHandle()
    if (!dir) throw new Error(i18n.global.t('curated.exportDirectoryUnavailable'))
    await writeBlobToDirectory(dir, candidate.blob, filename)
  }
}

/**
 * 从 video 截帧：Web API 时 POST 后端 SQLite；否则写入 IndexedDB。按设置可额外下载或写入用户目录。
 */
export async function saveCuratedCaptureFromVideo(
  video: HTMLVideoElement,
  movie: Movie,
  options: SaveCuratedCaptureOptions = {},
): Promise<SaveCuratedCaptureResult> {
  const candidate = await captureCuratedFrameCandidate(video, options)
  if (!candidate.ok) return candidate
  return saveCuratedFrameCandidate(candidate.candidate, movie)
}

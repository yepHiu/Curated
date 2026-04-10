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
  | { ok: true }
  | { ok: false; reason: string }

export type SaveCuratedCaptureOptions = {
  positionSecOverride?: number
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

/**
 * 从 video 截帧：Web API 时 POST 后端 SQLite；否则写入 IndexedDB。按设置可额外下载或写入用户目录。
 */
export async function saveCuratedCaptureFromVideo(
  video: HTMLVideoElement,
  movie: Movie,
  options: SaveCuratedCaptureOptions = {},
): Promise<SaveCuratedCaptureResult> {
  const cap = await captureVideoFrameToPng(video)
  if (!cap.ok) {
    return { ok: false, reason: cap.reason }
  }

  const positionSec = resolveCuratedCapturePositionSec(
    video.currentTime,
    options.positionSecOverride,
  )
  const capturedAt = new Date().toISOString()
  const id = crypto.randomUUID()

  const row = {
    id,
    movieId: movie.id,
    title: movie.title,
    code: movie.code,
    actors: [...movie.actors],
    positionSec,
    capturedAt,
    tags: [] as string[],
    imageBlob: cap.blob,
  }

  try {
    if (USE_WEB) {
      await api.createCuratedFrameUpload({
        id: row.id,
        movieId: row.movieId,
        title: row.title,
        code: row.code,
        actors: row.actors,
        positionSec: row.positionSec,
        capturedAt: row.capturedAt,
        tags: row.tags,
      }, cap.blob)
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

  const filename = formatFrameFilename(movie.code, positionSec, capturedAt)
  const mode = getCuratedFrameSaveMode()

  if (mode === "download") {
    try {
      triggerDownloadBlob(cap.blob, filename)
    } catch {
      // 仍保留 IDB
    }
  }

  if (mode === "directory") {
    try {
      const dir = await getStoredDirectoryHandle()
      if (dir) {
        await writeBlobToDirectory(dir, cap.blob, filename)
      }
    } catch {
      // 权限或 API 失败时忽略，IDB 已保存
    }
  }

  return { ok: true }
}

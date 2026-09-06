import { i18n } from "@/i18n"

export type CaptureFrameResult =
  | { ok: true; blob: Blob }
  | { ok: false; reason: string }

/**
 * 将当前 video 帧绘制为 PNG。跨域无 CORS 时 canvas 会被污染导致失败。
 */
export async function captureVideoFrameToPng(video: HTMLVideoElement, onPreview?: (url: string) => void): Promise<CaptureFrameResult> {
  const w = video.videoWidth
  const h = video.videoHeight
  if (!w || !h) {
    return Promise.resolve({ ok: false, reason: i18n.global.t("curated.captureNotReady") })
  }

  const canvas = document.createElement("canvas")
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext("2d")
  if (!ctx) {
    return Promise.resolve({ ok: false, reason: i18n.global.t("curated.captureNoCtx") })
  }

  try {
    ctx.drawImage(video, 0, 0, w, h)
    if (onPreview) {
      const preview = document.createElement('canvas')
      preview.width = 192
      preview.height = Math.max(1, Math.round(192 * h / w))
      preview.getContext('2d')?.drawImage(canvas, 0, 0, preview.width, preview.height)
      onPreview(preview.toDataURL('image/jpeg', .8))
      preview.width = preview.height = 1
    }
  } catch {
    return Promise.resolve({
      ok: false,
      reason: i18n.global.t("curated.captureCors"),
    })
  }

  // Large frames benefit from isolating encoder work. Unsupported browsers and
  // worker failures retain the same frozen canvas and use the standard encoder.
  if (w * h >= 3840 * 2160 && typeof Worker !== 'undefined' && typeof OffscreenCanvas !== 'undefined' && typeof createImageBitmap === 'function') {
    try {
      const { encodeCaptureOffThread } = await import('./capture-worker')
      const blob = await encodeCaptureOffThread(canvas)
      canvas.width = canvas.height = 1
      return { ok: true, blob }
    } catch { /* fallback to the frozen canvas */ }
  }

  return new Promise((resolve) => {
    try { canvas.toBlob(
      (blob) => {
        if (!blob) {
          resolve({ ok: false, reason: i18n.global.t("curated.captureBlobFail") })
          return
        }
        resolve({ ok: true, blob })
      },
      "image/png",
    ) } catch { resolve({ ok: false, reason: i18n.global.t('curated.captureCors') }) }
  })
}

export function formatFrameFilename(code: string, positionSec: number, capturedAt: string): string {
  const safeCode = code.replace(/[/\\?%*:|"<>]/g, "_").slice(0, 80) || "frame"
  const t = Math.floor(positionSec)
  const iso = capturedAt.replace(/[:.]/g, "-").slice(0, 19)
  return `Curated_${safeCode}_${t}s_${iso}.png`
}

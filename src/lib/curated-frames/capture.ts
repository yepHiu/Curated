import { i18n } from "@/i18n"

export type CaptureFrameResult =
  | { ok: true; blob: Blob }
  | { ok: false; reason: string }

/**
 * 将当前 video 帧绘制为 PNG。跨域无 CORS 时 canvas 会被污染导致失败。
 */
export function captureVideoFrameToPng(video: HTMLVideoElement): Promise<CaptureFrameResult> {
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
  } catch {
    return Promise.resolve({
      ok: false,
      reason: i18n.global.t("curated.captureCors"),
    })
  }

  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          resolve({ ok: false, reason: i18n.global.t("curated.captureBlobFail") })
          return
        }
        resolve({ ok: true, blob })
      },
      "image/png",
      0.92,
    )
  })
}

export function formatFrameFilename(code: string, positionSec: number, capturedAt: string): string {
  const safeCode = code.replace(/[/\\?%*:|"<>]/g, "_").slice(0, 80) || "frame"
  const t = Math.floor(positionSec)
  const iso = capturedAt.replace(/[:.]/g, "-").slice(0, 19)
  return `Curated_${safeCode}_${t}s_${iso}.png`
}

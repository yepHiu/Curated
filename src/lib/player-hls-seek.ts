export const HLS_SEEK_REUSE_LEAD_SEC = 30
export const HLS_TRANSCODE_SEEK_REUSE_LEAD_SEC = 60
export const HLS_SEEK_CATCHUP_TIMEOUT_MS = 20_000
export const HLS_SEEK_CATCHUP_INTERVAL_MS = 200
export const HLS_STARTUP_BUFFER_SEC = 8
export const HLS_STARTUP_BUFFER_WAIT_MS = 4_000

type MediaTimeRanges = {
  length: number
  end(index: number): number
}

type MediaWrittenEndSource = {
  duration: number
  buffered?: MediaTimeRanges | null
}

type MediaCatchupSource = MediaWrittenEndSource & {
  addEventListener(type: string, listener: EventListener): void
  removeEventListener(type: string, listener: EventListener): void
}

export function getMediaWrittenEndSec(media: MediaWrittenEndSource | null | undefined): number {
  if (!media) return 0
  const duration = Number.isFinite(media.duration) && media.duration > 0 ? media.duration : 0
  let bufferedEnd = 0
  const ranges = media.buffered
  if (ranges && ranges.length > 0) {
    try {
      bufferedEnd = ranges.end(ranges.length - 1)
    } catch {
      bufferedEnd = 0
    }
  }
  const written = Math.max(duration, bufferedEnd)
  return Number.isFinite(written) && written > 0 ? written : 0
}

export function hlsSeekReuseLeadSec(sessionKind?: string | null): number {
  if ((sessionKind ?? "").trim().toLowerCase() === "transcode-hls") {
    return HLS_TRANSCODE_SEEK_REUSE_LEAD_SEC
  }
  return HLS_SEEK_REUSE_LEAD_SEC
}

export function shouldReuseHlsSessionForSeek(input: {
  forceSessionSwap?: boolean
  localTargetSec: number
  writtenEndSec: number
  reuseLeadSec?: number
}): boolean {
  if (input.forceSessionSwap) return false
  if (!Number.isFinite(input.localTargetSec) || input.localTargetSec < 0) return false
  const written = Number.isFinite(input.writtenEndSec) && input.writtenEndSec > 0 ? input.writtenEndSec : 0
  const lead = Math.max(0, input.reuseLeadSec ?? HLS_SEEK_REUSE_LEAD_SEC)
  if (written <= 0) {
    return input.localTargetSec <= lead
  }
  return input.localTargetSec <= written + lead
}

export function isPrematureHlsEndedEvent(input: {
  absoluteTimeSec: number
  totalDurationSec: number
  thresholdSec?: number
}): boolean {
  const total = input.totalDurationSec
  const absolute = input.absoluteTimeSec
  if (!Number.isFinite(total) || total <= 0) return false
  if (!Number.isFinite(absolute) || absolute < 0) return false
  const threshold = Math.max(0, input.thresholdSec ?? 1)
  return absolute < total - threshold
}

export async function waitForMediaWrittenEnd(
  media: MediaCatchupSource | null | undefined,
  targetSec: number,
  options: {
    timeoutMs?: number
    intervalMs?: number
    isAborted?: () => boolean
    now?: () => number
  } = {},
): Promise<boolean> {
  if (!media || !Number.isFinite(targetSec)) return false
  const readyAt = targetSec - 0.25
  if (getMediaWrittenEndSec(media) >= readyAt) return true

  const timeoutMs = Math.max(0, options.timeoutMs ?? HLS_SEEK_CATCHUP_TIMEOUT_MS)
  const intervalMs = Math.max(50, options.intervalMs ?? HLS_SEEK_CATCHUP_INTERVAL_MS)
  const now = options.now ?? Date.now
  const deadline = now() + timeoutMs

  return await new Promise((resolve) => {
    let settled = false
    const finish = (ok: boolean) => {
      if (settled) return
      settled = true
      cleanup()
      resolve(ok)
    }
    const onUpdate = () => {
      if (options.isAborted?.()) {
        finish(false)
        return
      }
      if (getMediaWrittenEndSec(media) >= readyAt) {
        finish(true)
      }
    }
    const timer = window.setInterval(() => {
      if (options.isAborted?.() || now() > deadline) {
        finish(false)
        return
      }
      onUpdate()
    }, intervalMs)
    const cleanup = () => {
      window.clearInterval(timer)
      media.removeEventListener("durationchange", onUpdate)
      media.removeEventListener("progress", onUpdate)
      media.removeEventListener("loadedmetadata", onUpdate)
    }
    media.addEventListener("durationchange", onUpdate)
    media.addEventListener("progress", onUpdate)
    media.addEventListener("loadedmetadata", onUpdate)
  })
}

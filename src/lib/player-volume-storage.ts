/**
 * 播放器音量（0–100）与静音：浏览器 localStorage 全局记忆，跨影片与刷新保留。
 * 兼容旧版仅存储数字字符串的格式。
 */

const STORAGE_KEY = "jav-library-player-volume-v1"

const DEFAULT_PERCENT = 100

export interface PlayerAudioPrefs {
  volumePercent: number
  muted: boolean
}

const DEFAULT_PREFS: PlayerAudioPrefs = {
  volumePercent: DEFAULT_PERCENT,
  muted: false,
}

function clampPercent(n: number): number {
  if (!Number.isFinite(n)) return DEFAULT_PERCENT
  return Math.max(0, Math.min(100, Math.round(n)))
}

function normalizePrefs(p: PlayerAudioPrefs): PlayerAudioPrefs {
  const volumePercent = clampPercent(p.volumePercent)
  const muted = Boolean(p.muted) && volumePercent > 0
  return { volumePercent, muted }
}

export function getPlayerAudioPrefs(): PlayerAudioPrefs {
  if (typeof localStorage === "undefined") {
    return { ...DEFAULT_PREFS }
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === null || raw === "") {
      return { ...DEFAULT_PREFS }
    }
    try {
      const parsed = JSON.parse(raw) as unknown
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        const o = parsed as Record<string, unknown>
        const vpRaw =
          typeof o.volumePercent === "number"
            ? o.volumePercent
            : typeof o.v === "number"
              ? o.v
              : NaN
        const mutedRaw =
          typeof o.muted === "boolean" ? o.muted : typeof o.m === "boolean" ? o.m : false
        if (Number.isFinite(vpRaw)) {
          return normalizePrefs({
            volumePercent: vpRaw as number,
            muted: mutedRaw,
          })
        }
      }
    } catch {
      /* 非 JSON，按旧版纯数字处理 */
    }
    const legacy = Number(raw)
    if (Number.isFinite(legacy)) {
      return normalizePrefs({ volumePercent: legacy, muted: false })
    }
  } catch {
    /* ignore */
  }
  return { ...DEFAULT_PREFS }
}

export function savePlayerAudioPrefs(prefs: PlayerAudioPrefs): void {
  if (typeof localStorage === "undefined") {
    return
  }
  const n = normalizePrefs(prefs)
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ volumePercent: n.volumePercent, muted: n.muted }),
    )
  } catch {
    // quota / 隐私模式等
  }
}

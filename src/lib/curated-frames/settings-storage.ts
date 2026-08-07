import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"
import {
  DEFAULT_CURATED_CAPTURE_KEY_CODE,
  normalizeCuratedCaptureKeyCode,
} from "@/lib/player-shortcuts"

const MODE_KEY = "jav-curated-frames-save-mode"
const CAPTURE_KEY_CODE_KEY = "jav-curated-capture-key-code"
const EXPORT_MODE_KEY = "jav-curated-frame-export-mode"
const CAPTURE_FEEDBACK_SOUND_KEY = "jav-curated-capture-feedback-sound-v1"

export type CuratedFrameExportModePreference = "raw" | "watermarked"

export function getCuratedFrameSaveMode(): CuratedFrameSaveMode {
  if (typeof localStorage === "undefined") return "app"
  const v = localStorage.getItem(MODE_KEY)
  if (v === "download" || v === "directory") return v
  return "app"
}

export function setCuratedFrameSaveMode(mode: CuratedFrameSaveMode) {
  if (typeof localStorage === "undefined") return
  localStorage.setItem(MODE_KEY, mode)
}

export function getCuratedCaptureKeyCode(): string {
  if (typeof localStorage === "undefined") return DEFAULT_CURATED_CAPTURE_KEY_CODE
  return normalizeCuratedCaptureKeyCode(localStorage.getItem(CAPTURE_KEY_CODE_KEY))
}

export function setCuratedCaptureKeyCode(code: string) {
  if (typeof localStorage === "undefined") return
  const normalized = normalizeCuratedCaptureKeyCode(code)
  if (normalized === DEFAULT_CURATED_CAPTURE_KEY_CODE) {
    localStorage.removeItem(CAPTURE_KEY_CODE_KEY)
    return
  }
  localStorage.setItem(CAPTURE_KEY_CODE_KEY, normalized)
}

export function resetCuratedCaptureKeyCode() {
  if (typeof localStorage === "undefined") return
  localStorage.removeItem(CAPTURE_KEY_CODE_KEY)
}

export function getCuratedFrameExportMode(): CuratedFrameExportModePreference {
  if (typeof localStorage === "undefined") return "raw"
  return localStorage.getItem(EXPORT_MODE_KEY) === "watermarked" ? "watermarked" : "raw"
}

export function setCuratedFrameExportMode(mode: CuratedFrameExportModePreference) {
  if (typeof localStorage === "undefined") return
  if (mode === "raw") {
    localStorage.removeItem(EXPORT_MODE_KEY)
    return
  }
  localStorage.setItem(EXPORT_MODE_KEY, mode)
}

export function getCuratedCaptureFeedbackSoundEnabled(): boolean {
  if (typeof localStorage === "undefined") return true
  return localStorage.getItem(CAPTURE_FEEDBACK_SOUND_KEY) !== "off"
}

export function setCuratedCaptureFeedbackSoundEnabled(enabled: boolean) {
  if (typeof localStorage === "undefined") return
  if (enabled) {
    localStorage.removeItem(CAPTURE_FEEDBACK_SOUND_KEY)
    return
  }
  localStorage.setItem(CAPTURE_FEEDBACK_SOUND_KEY, "off")
}

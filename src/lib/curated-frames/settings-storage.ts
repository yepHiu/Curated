import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"
import {
  DEFAULT_CURATED_CAPTURE_KEY_CODE,
  normalizeCuratedCaptureKeyCode,
} from "@/lib/player-shortcuts"

const MODE_KEY = "jav-curated-frames-save-mode"
const CAPTURE_KEY_CODE_KEY = "jav-curated-capture-key-code"

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

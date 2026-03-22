import type { CuratedFrameSaveMode } from "@/domain/curated-frame/types"

const MODE_KEY = "jav-curated-frames-save-mode"

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

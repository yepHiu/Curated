export const DEFAULT_CURATED_CAPTURE_KEY_CODE = "KeyC"

const RESERVED_CURATED_CAPTURE_KEY_CODES = new Set([
  "Space",
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "ArrowDown",
  "Escape",
  "KeyJ",
  "KeyK",
  "KeyL",
  "KeyM",
  "KeyF",
  "KeyP",
])

export function isCuratedCaptureKeyReserved(code: string): boolean {
  return RESERVED_CURATED_CAPTURE_KEY_CODES.has(code)
}

export function isSupportedCuratedCaptureKeyCode(code: string): boolean {
  if (/^Key[A-Z]$/.test(code)) return true
  if (/^Digit[0-9]$/.test(code)) return true
  if (/^F(?:[1-9]|1[0-2])$/.test(code)) return true
  if (code === "PageUp" || code === "PageDown") return true
  return false
}

export function normalizeCuratedCaptureKeyCode(
  code: string | null | undefined,
): string {
  const trimmed = typeof code === "string" ? code.trim() : ""
  if (!trimmed) return DEFAULT_CURATED_CAPTURE_KEY_CODE
  if (!isSupportedCuratedCaptureKeyCode(trimmed)) return DEFAULT_CURATED_CAPTURE_KEY_CODE
  if (isCuratedCaptureKeyReserved(trimmed)) return DEFAULT_CURATED_CAPTURE_KEY_CODE
  return trimmed
}

export function formatCuratedCaptureKeyLabel(code: string): string {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3)
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  return code
}

export function shouldIgnoreGlobalPlaybackHotkeysForTarget(
  target: EventTarget | null,
): boolean {
  if (!(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true
  if (target.isContentEditable || target.contentEditable === "true") return true
  if (target.closest('[data-slot="slider"]')) return true
  return false
}

export function shouldBlurPlaybackSliderAfterCommit(
  activeElement: Element | null,
  sliderRoot: HTMLElement | null,
): boolean {
  if (!(activeElement instanceof HTMLElement)) return false
  if (!sliderRoot) return false
  return activeElement === sliderRoot || sliderRoot.contains(activeElement)
}

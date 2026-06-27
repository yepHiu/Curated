import type {
  ComicFitMode,
  ComicReaderMode,
  ComicReaderSettings,
  ComicReadingDirection,
} from "@/domain/comic/types"

export type ComicReaderPreferenceLike = Partial<{
  comicId: string
  mode: ComicReaderMode
  fit: ComicFitMode
  direction: ComicReadingDirection
  updatedAt: string
}>

export interface TemporaryStitch {
  anchorPageIndex: number
  adjacentPageIndex: number
}

const temporaryStitches = new Map<string, TemporaryStitch>()

export function resolveComicReaderPreferences(
  defaults: ComicReaderSettings,
  perBook?: ComicReaderPreferenceLike | null,
): ComicReaderSettings {
  return {
    mode: perBook?.mode ?? defaults.mode,
    fit: perBook?.fit ?? defaults.fit,
    direction: perBook?.direction ?? defaults.direction,
  }
}

export function resolveReaderKeyStep(
  key: string,
  direction: ComicReadingDirection,
): -1 | 0 | 1 {
  if (key === " " || key === "Space" || key === "Spacebar") {
    return 1
  }
  if (key === "ArrowRight") {
    return direction === "rtl" ? -1 : 1
  }
  if (key === "ArrowLeft") {
    return direction === "rtl" ? 1 : -1
  }
  return 0
}

export function clampComicPageIndex(pageIndex: number, pageCount: number): number {
  const max = Math.max(0, pageCount - 1)
  if (!Number.isFinite(pageIndex)) return 0
  return Math.min(max, Math.max(0, Math.floor(pageIndex)))
}

export function resolveStitchPairDisplayOrder(
  anchorPageIndex: number,
  adjacentPageIndex: number,
  direction: ComicReadingDirection,
): [number, number] {
  const pair = [anchorPageIndex, adjacentPageIndex].sort((a, b) => a - b) as [number, number]
  return direction === "rtl" ? [pair[1], pair[0]] : pair
}

export function setTemporaryStitch(comicId: string, stitch: TemporaryStitch): void {
  const id = comicId.trim()
  if (!id) return
  temporaryStitches.set(id, {
    anchorPageIndex: stitch.anchorPageIndex,
    adjacentPageIndex: stitch.adjacentPageIndex,
  })
}

export function getTemporaryStitch(comicId: string): TemporaryStitch | undefined {
  const stitch = temporaryStitches.get(comicId.trim())
  return stitch ? { ...stitch } : undefined
}

export function clearTemporaryStitch(comicId: string): void {
  temporaryStitches.delete(comicId.trim())
}

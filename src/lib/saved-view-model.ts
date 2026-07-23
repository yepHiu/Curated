import type { SavedViewFiltersV1, SavedViewMode, SavedViewPlayState, SavedViewTab } from "@/api/types"
import { normalizeLibraryResolutionFilter } from "@/lib/library-query"

const modes = new Set<SavedViewMode>(["library", "favorites", "recent", "tags", "trash"])
const tabs = new Set<SavedViewTab>(["all", "new", "top-rated"])
const playStates = new Set<SavedViewPlayState>(["all", "unwatched", "in-progress", "completed"])

export function normalizeSavedViewName(value: string): string {
  const name = value.trim()
  if (!name || Array.from(name).length > 40) {
    throw new Error("saved view name must contain 1 to 40 characters")
  }
  return name
}

function normalizeText(value: string | undefined): string | undefined {
  const normalized = value?.trim() ?? ""
  if (Array.from(normalized).length > 200) {
    throw new Error("saved view filter text is too long")
  }
  return normalized || undefined
}

export function normalizeSavedViewFiltersV1(input: SavedViewFiltersV1): SavedViewFiltersV1 {
  if (input.schemaVersion !== 1) {
    throw new Error("unsupported saved view schemaVersion")
  }
  const mode = input.mode ?? "library"
  const tab = input.tab ?? "all"
  const playState = input.playState ?? "all"
  if (!modes.has(mode) || !tabs.has(tab) || !playStates.has(playState)) {
    throw new Error("invalid saved view filter")
  }
  if (
    input.userRating !== undefined &&
    (!Number.isFinite(input.userRating) || input.userRating < 0 || input.userRating > 5)
  ) {
    throw new Error("saved view userRating must be between 0 and 5")
  }
  if (
    input.addedWithinDays !== undefined &&
    (!Number.isInteger(input.addedWithinDays) || input.addedWithinDays < 1 || input.addedWithinDays > 3650)
  ) {
    throw new Error("saved view addedWithinDays must be between 1 and 3650")
  }
  if (mode === "trash") {
    return { schemaVersion: 1, mode: "trash", tab: "all", playState: "all" }
  }
  return {
    schemaVersion: 1,
    mode,
    q: normalizeText(input.q),
    tag: normalizeText(input.tag),
    actor: normalizeText(input.actor),
    studio: normalizeText(input.studio),
    tab,
    playState,
    userRating: input.userRating,
    resolution: normalizeLibraryResolutionFilter(input.resolution ?? "") || undefined,
    addedWithinDays: input.addedWithinDays,
  }
}

export function normalizedSavedViewNameKey(value: string): string {
  return normalizeSavedViewName(value).toLocaleLowerCase()
}

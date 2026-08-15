import type {
  SavedViewFiltersV1,
  SavedViewMode,
  SavedViewPlayState,
  SavedViewSort,
  SavedViewTab,
} from "@/api/types"
import {
  normalizeLibraryCatalogFilter,
  normalizeLibraryResolutionFilter,
  normalizeLibraryRuntimeFilter,
  normalizeLibrarySortFilter,
  normalizeLibraryYearFilter,
  serializeLibraryTagFilters,
  parseLibraryTagFilterText,
} from "@/lib/library-query"
import { librarySortKeyFromTab } from "@/lib/movie-sort"

const modes = new Set<SavedViewMode>(["library", "favorites", "recent", "tags", "trash"])
const tabs = new Set<SavedViewTab>(["all", "new", "top-rated"])
const playStates = new Set<SavedViewPlayState>(["all", "unwatched", "in-progress", "completed"])
const sorts = new Set<SavedViewSort>([
  "added",
  "release",
  "rating",
  "code",
  "actor",
  "studio",
  "year",
])

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
  const year = normalizeLibraryYearFilter(input.year ?? "")
  const runtime = normalizeLibraryRuntimeFilter(input.runtime ?? "")
  const catalog = normalizeLibraryCatalogFilter(input.catalog ?? "")
  if (input.year?.trim() && !year) {
    throw new Error("saved view year must be unknown or a year from 1800 to 3000")
  }
  if (input.runtime?.trim() && !runtime) {
    throw new Error("invalid saved view runtime")
  }
  if (input.catalog?.trim() && !catalog) {
    throw new Error("invalid saved view catalog")
  }
  const sortRaw = input.sort?.trim() ?? ""
  const sort = sortRaw ? normalizeLibrarySortFilter(sortRaw) : ""
  if (sortRaw && !sort) {
    throw new Error("invalid saved view sort")
  }
  if (mode === "trash") {
    return { schemaVersion: 1, mode: "trash", tab: "all", playState: "all" }
  }
  const unrated = input.unrated === true
  const resolvedSort = (sort || librarySortKeyFromTab(tab)) as SavedViewSort
  if (!sorts.has(resolvedSort)) {
    throw new Error("invalid saved view sort")
  }
  return {
    schemaVersion: 1,
    mode,
    q: normalizeText(input.q),
    tag: normalizeText(serializeLibraryTagFilters(parseLibraryTagFilterText(input.tag))),
    actor: normalizeText(serializeLibraryTagFilters(parseLibraryTagFilterText(input.actor))),
    studio: normalizeText(serializeLibraryTagFilters(parseLibraryTagFilterText(input.studio))),
    tab,
    sort: resolvedSort === "added" ? undefined : resolvedSort,
    playState,
    userRating: unrated ? undefined : input.userRating,
    unrated: unrated || undefined,
    resolution: normalizeLibraryResolutionFilter(input.resolution ?? "") || undefined,
    addedWithinDays: input.addedWithinDays,
    year: year || undefined,
    runtime: runtime || undefined,
    catalog: catalog || undefined,
  }
}

export function normalizedSavedViewNameKey(value: string): string {
  return normalizeSavedViewName(value).toLocaleLowerCase()
}

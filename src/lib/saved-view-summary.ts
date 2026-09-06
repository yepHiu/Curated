import type { SavedViewFiltersV1 } from "@/api/types"
import { parseLibraryTagFilterText } from "@/lib/library-query"

export type SavedViewSummaryTranslate = (
  key: string,
  values?: Record<string, unknown>,
) => string

/** Human-readable Saved View filter summary, matching the library bookmark menu. */
export function summarizeSavedViewFilters(
  filters: SavedViewFiltersV1,
  t: SavedViewSummaryTranslate,
): string {
  const entries: string[] = []
  if (filters.q) entries.push(t("library.savedViewSummarySearch", { value: filters.q }))
  const actors = parseLibraryTagFilterText(filters.actor)
  if (actors.length > 0) {
    entries.push(t("library.savedViewSummaryActor", { value: actors.join(" · ") }))
  }
  if (filters.tag) {
    const tags = parseLibraryTagFilterText(filters.tag)
    if (tags.length > 0) {
      entries.push(t("library.savedViewSummaryTag", { value: tags.join(" · ") }))
    }
  }
  const studios = parseLibraryTagFilterText(filters.studio)
  if (studios.length > 0) {
    entries.push(t("library.savedViewSummaryStudio", { value: studios.join(" · ") }))
  }
  if (filters.playState && filters.playState !== "all") {
    entries.push(t(`library.savedViewPlay.${filters.playState}`))
  }
  if (filters.unrated) {
    entries.push(t("library.savedViewUnrated"))
  } else if (filters.userRating !== undefined) {
    entries.push(t("library.savedViewRatingAtLeast", { value: filters.userRating }))
  }
  if (filters.resolution) entries.push(filters.resolution.toUpperCase())
  if (filters.addedWithinDays) {
    entries.push(t("library.savedViewAddedDays", { days: filters.addedWithinDays }))
  }
  if (filters.year === "unknown") {
    entries.push(t("library.savedViewYearUnknown"))
  } else if (filters.year) {
    entries.push(filters.year)
  }
  if (filters.runtime) {
    entries.push(t(`library.savedViewRuntimeValue.${filters.runtime}`))
  }
  if (filters.catalog) {
    entries.push(t(`library.savedViewCatalogValue.${filters.catalog}`))
  }
  if (filters.sort && filters.sort !== "added") {
    entries.push(t(`library.savedViewSortValue.${filters.sort}`))
  }
  return entries.join(" · ") || t("library.savedViewAllLibrary")
}

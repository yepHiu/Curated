import { getCurrentLocalDayKey } from "./current-local-day-key"

export type MovieReleaseStatus = "released" | "unreleased"

export function getMovieReleaseStatus(
  releaseDate: string | undefined,
  today: string = getCurrentLocalDayKey(),
): MovieReleaseStatus | undefined {
  const date = releaseDate?.trim() ?? ""
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return undefined
  const parsed = new Date(`${date}T00:00:00Z`)
  // Reject impossible days that Date normalizes, such as February 30.
  if (!Number.isFinite(parsed.getTime()) || parsed.toISOString().slice(0, 10) !== date) {
    return undefined
  }
  return date <= today ? "released" : "unreleased"
}

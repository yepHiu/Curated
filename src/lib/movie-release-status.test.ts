import { describe, expect, it } from "vitest"
import { getCurrentLocalDayKey } from "./current-local-day-key"
import { getMovieReleaseStatus } from "./movie-release-status"

describe("movie release status", () => {
  it("uses the system's local calendar date", () => {
    expect(getCurrentLocalDayKey(new Date(2026, 8, 27, 0, 1))).toBe("2026-09-27")
    expect(getCurrentLocalDayKey(new Date(2026, 8, 27, 23, 59))).toBe("2026-09-27")
  })

  it.each([
    ["2026-09-26", "released"],
    ["2026-09-27", "released"],
    ["2026-09-28", "unreleased"],
    ["2025-12-31", "released"],
    ["2027-01-01", "unreleased"],
    [" 2026-09-27 ", "released"],
    ["2024-02-29", "released"],
  ])("compares %s with today", (date, expected) => {
    expect(getMovieReleaseStatus(date, "2026-09-27")).toBe(expected)
  })

  it.each([undefined, "", "   ", "2026", "2026-9-27", "2026-09-27T00:00:00Z", "2026-02-29", "2026-02-30", "2026-13-01", "2026-09-00"])("does not infer a status for %s", (date) => {
    expect(getMovieReleaseStatus(date, "2026-09-27")).toBeUndefined()
  })
})

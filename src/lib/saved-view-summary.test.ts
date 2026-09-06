import { describe, expect, it } from "vitest"
import { summarizeSavedViewFilters } from "@/lib/saved-view-summary"

const t = (key: string, values?: Record<string, unknown>) => {
  if (!values) return key
  return `${key}:${JSON.stringify(values)}`
}

describe("summarizeSavedViewFilters", () => {
  it("describes play state, actor, and tag filters in product language", () => {
    expect(
      summarizeSavedViewFilters(
        {
          schemaVersion: 1,
          playState: "unwatched",
          actor: "Ada",
          tag: "轻松",
        },
        t,
      ),
    ).toBe(
      [
        'library.savedViewSummaryActor:{"value":"Ada"}',
        'library.savedViewSummaryTag:{"value":"轻松"}',
        "library.savedViewPlay.unwatched",
      ].join(" · "),
    )
  })

  it("falls back to the whole-library label when no filters are set", () => {
    expect(summarizeSavedViewFilters({ schemaVersion: 1 }, t)).toBe("library.savedViewAllLibrary")
  })
})

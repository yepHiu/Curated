import { describe, expect, it } from "vitest"
import {
  buildCuratedFrameDialogNavigationEntries,
  findAdjacentCuratedFrameDialogEntry,
} from "./dialog-navigation"

function item(id: string) {
  return { row: { id } }
}

describe("curated frame dialog navigation", () => {
  it("uses the timeline order for previous and next entries", () => {
    const entries = buildCuratedFrameDialogNavigationEntries({
      mainTab: "timeline",
      listItems: [item("newest"), item("middle"), item("oldest")],
      actorGroups: [],
      movieGroups: [],
    })

    expect(findAdjacentCuratedFrameDialogEntry(entries, "middle", null, "previous")?.item.row.id).toBe(
      "newest",
    )
    expect(findAdjacentCuratedFrameDialogEntry(entries, "middle", null, "next")?.item.row.id).toBe(
      "oldest",
    )
  })

  it("uses the clicked actor group instance when actor sorting duplicates a frame", () => {
    const shared = item("shared")
    const entries = buildCuratedFrameDialogNavigationEntries({
      mainTab: "actors",
      listItems: [],
      actorGroups: [
        ["Actor A", [item("a-1"), shared]],
        ["Actor B", [shared, item("b-2")]],
      ],
      movieGroups: [],
    })

    expect(findAdjacentCuratedFrameDialogEntry(entries, "shared", "Actor A", "previous")?.item.row.id).toBe(
      "a-1",
    )
    expect(findAdjacentCuratedFrameDialogEntry(entries, "shared", "Actor B", "next")?.item.row.id).toBe(
      "b-2",
    )
  })

  it("uses the visible movie-group order and stops at boundaries", () => {
    const entries = buildCuratedFrameDialogNavigationEntries({
      mainTab: "movies",
      listItems: [],
      actorGroups: [],
      movieGroups: [
        { items: [item("movie-a-1"), item("movie-a-2")] },
        { items: [item("movie-b-1")] },
      ],
    })

    expect(findAdjacentCuratedFrameDialogEntry(entries, "movie-a-2", null, "next")?.item.row.id).toBe(
      "movie-b-1",
    )
    expect(findAdjacentCuratedFrameDialogEntry(entries, "movie-a-1", null, "previous")).toBeNull()
    expect(findAdjacentCuratedFrameDialogEntry(entries, "movie-b-1", null, "next")).toBeNull()
  })
})

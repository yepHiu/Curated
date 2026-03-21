import { describe, expect, it } from "vitest"
import {
  buildBrowseRouteTarget,
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryTabQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  mergeLibraryQuery,
} from "@/lib/library-query"

describe("library query helpers", () => {
  it("normalizes browse query values", () => {
    const query = {
      from: "favorites",
      q: "MKB",
      selected: "mkb-100",
      tab: "top-rated",
    }

    expect(getBrowseSourceMode(query)).toBe("favorites")
    expect(getLibrarySearchQuery(query)).toBe("MKB")
    expect(getSelectedMovieQuery(query)).toBe("mkb-100")
    expect(getLibraryTabQuery(query)).toBe("top-rated")
  })

  it("merges browse query patches and clears empty values", () => {
    const merged = mergeLibraryQuery(
      {
        q: "Rin",
        selected: "mkb-100",
        tab: "new",
      },
      {
        q: "",
        selected: undefined,
        tab: "all",
      },
    )

    expect(merged).toEqual({
    })
  })

  it("preserves browse context when building navigation targets", () => {
    const browseTarget = buildBrowseRouteTarget("recent", {
      q: "Mina",
      selected: "sld-101",
      tab: "new",
    })

    const movieQuery = buildMovieRouteQuery(
      {
        q: "Mina",
        selected: "sld-101",
        tab: "new",
      },
      "recent",
      "nva-102",
    )

    expect(browseTarget).toEqual({
      name: "recent",
      query: {
        q: "Mina",
        selected: "sld-101",
        tab: "new",
      },
    })

    expect(movieQuery).toEqual({
      from: "recent",
      q: "Mina",
      selected: "nva-102",
      tab: "new",
    })
  })

  it("maps removed or unknown tab query to all", () => {
    expect(getLibraryTabQuery({ tab: "favorites" })).toBe("all")
    expect(getLibraryTabQuery({ tab: "unknown" })).toBe("all")
  })

  it("reads exact tag filter and merges tag patch", () => {
    expect(getLibraryTagExactQuery({ tag: "4K" })).toBe("4K")
    expect(getLibraryTagExactQuery({})).toBe("")

    const merged = mergeLibraryQuery({ q: "foo", tag: "old" }, { tag: "new", q: undefined })
    expect(merged.tag).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("reads exact actor filter and merges actor patch", () => {
    expect(getLibraryActorExactQuery({ actor: "Mina" })).toBe("Mina")
    expect(getLibraryActorExactQuery({})).toBe("")

    const merged = mergeLibraryQuery(
      { q: "foo", actor: "old" },
      { actor: "new", q: undefined },
    )
    expect(merged.actor).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("preserves actor in browse and movie route helpers", () => {
    const q = { q: "x", actor: "Lead A", tab: "new" as const }
    expect(buildBrowseRouteTarget("library", q)).toEqual({
      name: "library",
      query: { q: "x", actor: "Lead A", tab: "new" },
    })
    expect(buildMovieRouteQuery(q, "library", "id-1")).toEqual({
      from: "library",
      q: "x",
      actor: "Lead A",
      selected: "id-1",
      tab: "new",
    })
  })
})

import { describe, expect, it } from "vitest"
import {
  buildBrowseRouteTarget,
  buildClearLibraryActorFilterQuery,
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryStudioExactQuery,
  getLibraryTabQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  mergeLibraryQuery,
  isLibraryBrowseRoute,
  resolveLibraryMode,
} from "@/lib/library-query"

describe("library query helpers", () => {
  it("resolveLibraryMode falls back to path when name is missing", () => {
    expect(
      resolveLibraryMode({
        name: undefined,
        path: "/trash",
      }),
    ).toBe("trash")
    expect(
      resolveLibraryMode({
        name: undefined,
        path: "/library",
      }),
    ).toBe("library")
    expect(
      resolveLibraryMode({
        name: "trash",
        path: "/trash",
      }),
    ).toBe("trash")
  })

  it("resolveLibraryMode prefers path when name and path disagree (e.g. stale name during navigation)", () => {
    expect(
      resolveLibraryMode({
        name: "library",
        path: "/trash",
      }),
    ).toBe("trash")
  })

  it("isLibraryBrowseRoute is true for trash path even when name is missing", () => {
    expect(
      isLibraryBrowseRoute({
        name: undefined,
        path: "/trash",
      }),
    ).toBe(true)
    expect(
      isLibraryBrowseRoute({
        name: undefined,
        path: "/settings",
      }),
    ).toBe(false)
  })

  it("normalizes browse query values", () => {
    const query = {
      from: "favorites",
      q: "MKB",
      selected: "mkb-100",
      tab: "top-rated",
    }

    expect(getBrowseSourceMode({ from: "trash" })).toBe("trash")
    expect(getBrowseSourceMode(query)).toBe("favorites")
    expect(getLibrarySearchQuery(query)).toBe("MKB")
    expect(getSelectedMovieQuery(query)).toBe("mkb-100")
    expect(getLibraryTabQuery(query)).toBe("top-rated")
  })

  it("reads selected movie id from first string when query repeats key", () => {
    expect(getSelectedMovieQuery({ selected: ["a", "b"] })).toBe("a")
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

  it("clears equivalent q when clearing actor filter", () => {
    expect(
      buildClearLibraryActorFilterQuery(
        { actor: "Mina", q: " mina ", selected: "movie-1", tab: "new" },
        "Mina",
      ),
    ).toEqual({ tab: "new" })
  })

  it("keeps unrelated q when clearing actor filter", () => {
    expect(
      buildClearLibraryActorFilterQuery(
        { actor: "Mina", q: "studio keyword", selected: "movie-1", tab: "new" },
        "Mina",
      ),
    ).toEqual({ q: "studio keyword", tab: "new" })
  })

  it("reads exact studio filter and merges studio patch", () => {
    expect(getLibraryStudioExactQuery({ studio: "Foo" })).toBe("Foo")
    expect(getLibraryStudioExactQuery({})).toBe("")

    const merged = mergeLibraryQuery(
      { q: "foo", studio: "old" },
      { studio: "new", q: undefined },
    )
    expect(merged.studio).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("preserves studio in browse and movie route helpers", () => {
    const q = { q: "x", studio: "ACME", tab: "new" as const }
    expect(buildBrowseRouteTarget("library", q)).toEqual({
      name: "library",
      query: { q: "x", studio: "ACME", tab: "new" },
    })
    expect(buildMovieRouteQuery(q, "library", "id-1")).toEqual({
      from: "library",
      q: "x",
      studio: "ACME",
      selected: "id-1",
      tab: "new",
    })
  })
})

import { describe, expect, it } from "vitest"
import {
  buildBrowseRouteTarget,
  buildSavedViewFiltersV1,
  buildSavedViewRouteTarget,
  buildClearLibraryActorFilterQuery,
  getDetailBrowseTargetMode,
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibraryActorExactFilters,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryAddedWithinDaysQuery,
  getLibraryPlayStateQuery,
  getLibraryResolutionQuery,
  getLibrarySortQuery,
  getLibrarySortQueryForRoute,
  getLibraryUserRatingQuery,
  getLibraryUnratedQuery,
  getLibraryYearQuery,
  getLibraryRuntimeQuery,
  getLibraryCatalogQuery,
  getLibraryStudioExactFilters,
  getLibraryStudioExactQuery,
  getLibraryTabQuery,
  getLibraryTagExactQuery,
  getCuratedFrameTagQuery,
  getCuratedFrameTagFilters,
  getLibraryTagExactFilters,
  serializeLibraryTagFilters,
  serializeCuratedFrameTagFilters,
  mergeLibraryQuery,
  isLibraryBrowseRoute,
  mergeCuratedFramesQuery,
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

  it("does not treat the root home path as a library browse route", () => {
    expect(
      isLibraryBrowseRoute({
        name: "home",
        path: "/",
      }),
    ).toBe(false)
  })

  it("normalizes browse query values", () => {
    const query = {
      browse: "favorites",
      q: "MKB",
      selected: "mkb-100",
      tab: "top-rated",
    }

    expect(getBrowseSourceMode({ from: "trash" })).toBe("trash")
    expect(getBrowseSourceMode({ browse: "recent" })).toBe("recent")
    expect(getBrowseSourceMode(query)).toBe("favorites")
    expect(getLibrarySearchQuery(query)).toBe("MKB")
    expect(getLibraryTabQuery(query)).toBe("top-rated")
  })

  it("maps retired tags browse onto the library route", () => {
    expect(resolveLibraryMode({ name: "tags", path: "/tags" })).toBe("library")
    expect(getBrowseSourceMode({ browse: "tags" })).toBe("library")
    expect(getBrowseSourceMode({ from: "tags" })).toBe("library")
    expect(getDetailBrowseTargetMode("tags", "tag")).toBe("library")
    expect(getDetailBrowseTargetMode("tags", "actor")).toBe("library")
    expect(getDetailBrowseTargetMode("tags", "studio")).toBe("library")
    expect(getDetailBrowseTargetMode("favorites", "actor")).toBe("favorites")
    expect(
      buildSavedViewRouteTarget({
        schemaVersion: 1,
        mode: "tags",
        tag: "Drama",
      }),
    ).toEqual({
      name: "library",
      query: {
        tag: "Drama",
      },
    })
    expect(buildBrowseRouteTarget("tags", { tag: "Drama" })).toEqual({
      name: "library",
      query: { tag: "Drama" },
    })
    expect(buildSavedViewFiltersV1("tags", { tag: "Drama" }).mode).toBe("library")
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
      autoplay: "1",
      t: "12",
      back: "detail",
      browse: "favorites",
    })

    const movieQuery = buildMovieRouteQuery(
      {
        q: "Mina",
        selected: "sld-101",
        tab: "new",
        autoplay: "1",
        t: "12",
        back: "detail",
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
      browse: "recent",
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
    expect(getLibraryTagExactQuery({ tag: ["first", "ignored"] })).toBe("first")
    expect(getLibraryTagExactFilters({ tag: "4K,fav,4k" })).toEqual(["4K", "fav"])
    expect(serializeLibraryTagFilters([" fav ", "4K", "FAV"])).toBe("fav,4K")

    const merged = mergeLibraryQuery({ q: "foo", tag: "old" }, { tag: "new", q: undefined })
    expect(merged.tag).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("reads exact actor filter and merges actor patch", () => {
    expect(getLibraryActorExactQuery({ actor: "Mina" })).toBe("Mina")
    expect(getLibraryActorExactQuery({})).toBe("")
    expect(getLibraryActorExactFilters({ actor: "Mina,Lead,mina" })).toEqual(["Mina", "Lead"])
    expect(serializeLibraryTagFilters([" Mina ", "Lead", "MINA"])).toBe("Mina,Lead")

    const merged = mergeLibraryQuery(
      { q: "foo", actor: "old" },
      { actor: "new", q: undefined },
    )
    expect(merged.actor).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("preserves comma-separated actors in browse and movie route helpers", () => {
    const q = { q: "x", actor: "Lead A,Lead B", tab: "new" as const }
    expect(buildBrowseRouteTarget("library", q)).toEqual({
      name: "library",
      query: { q: "x", actor: "Lead A,Lead B", tab: "new" },
    })
    expect(buildMovieRouteQuery(q, "library", "id-1")).toEqual({
      browse: "library",
      q: "x",
      actor: "Lead A,Lead B",
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
    expect(getLibraryStudioExactFilters({ studio: "Foo,Bar,foo" })).toEqual(["Foo", "Bar"])

    const merged = mergeLibraryQuery(
      { q: "foo", studio: "old" },
      { studio: "new", q: undefined },
    )
    expect(merged.studio).toBe("new")
    expect(merged.q).toBeUndefined()
  })

  it("preserves comma-separated studios in browse and movie route helpers", () => {
    const q = { q: "x", studio: "ACME,Other", tab: "new" as const }
    expect(buildBrowseRouteTarget("library", q)).toEqual({
      name: "library",
      query: { q: "x", studio: "ACME,Other", tab: "new" },
    })
    expect(buildMovieRouteQuery(q, "library", "id-1")).toEqual({
      browse: "library",
      q: "x",
      studio: "ACME,Other",
      selected: "id-1",
      tab: "new",
    })
  })

  it("normalizes advanced filters and preserves them through browse navigation", () => {
    const query = {
      playState: "unwatched",
      userRating: "5",
      resolution: "2160P",
      addedWithinDays: "30",
      year: "2024",
      runtime: "long",
      catalog: "unscraped",
      unrated: "1",
      selected: "movie-1",
      autoplay: "1",
      t: "42",
    }
    expect(getLibraryPlayStateQuery(query)).toBe("unwatched")
    expect(getLibraryUserRatingQuery(query)).toBe(5)
    expect(getLibraryUnratedQuery(query)).toBe(true)
    expect(getLibraryResolutionQuery(query)).toBe("4k")
    expect(getLibraryAddedWithinDaysQuery(query)).toBe(30)
    expect(getLibraryYearQuery(query)).toBe("2024")
    expect(getLibraryRuntimeQuery(query)).toBe("long")
    expect(getLibraryCatalogQuery(query)).toBe("unscraped")
    expect(buildBrowseRouteTarget("favorites", query)).toEqual({
      name: "favorites",
      query: {
        playState: "unwatched",
        unrated: "1",
        resolution: "4k",
        addedWithinDays: "30",
        year: "2024",
        runtime: "long",
        catalog: "unscraped",
        selected: "movie-1",
      },
    })
  })

  it("builds a versioned Saved View without transient navigation state", () => {
    const filters = buildSavedViewFiltersV1("library", {
      q: "Mina",
      actor: "Mina",
      tab: "top-rated",
      playState: "unwatched",
      userRating: "5",
      resolution: "2160p",
      addedWithinDays: "90",
      selected: "movie-1",
      from: "detail",
      browse: "favorites",
      back: "library",
      autoplay: "1",
      t: "12",
    })
    expect(filters).toEqual({
      schemaVersion: 1,
      mode: "library",
      q: "Mina",
      tag: undefined,
      actor: "Mina",
      studio: undefined,
      tab: "top-rated",
      sort: "rating",
      playState: "unwatched",
      userRating: 5,
      unrated: undefined,
      resolution: "4k",
      addedWithinDays: 90,
      year: undefined,
      runtime: undefined,
      catalog: undefined,
    })
    expect(buildSavedViewRouteTarget(filters)).toEqual({
      name: "library",
      query: {
        q: "Mina",
        actor: "Mina",
        tab: "top-rated",
        playState: "unwatched",
        userRating: "5",
        resolution: "4k",
        addedWithinDays: "90",
      },
    })
  })

  it("reads explicit library sort and falls back to tab mapping", () => {
    expect(getLibrarySortQuery({ sort: "code" })).toBe("code")
    expect(getLibrarySortQuery({ tab: "new" })).toBe("release")
    expect(getLibrarySortQuery({ tab: "top-rated" })).toBe("rating")
    expect(getLibrarySortQuery({})).toBe("added")
    expect(getLibrarySortQueryForRoute({ tab: "new" })).toBeUndefined()
    expect(getLibrarySortQueryForRoute({ sort: "code" })).toBe("code")
  })

  it("reads and merges curated frame tag filter separately from curated search", () => {
    expect(getCuratedFrameTagQuery({ cft: "pose", cfq: "abc" })).toBe("pose")
    expect(getCuratedFrameTagQuery({})).toBe("")
    expect(getCuratedFrameTagFilters({ cft: "pose,close-up,pose" })).toEqual(["pose", "close-up"])
    expect(serializeCuratedFrameTagFilters([" pose ", "close-up", "POSE"])).toBe("pose,close-up")

    const merged = mergeCuratedFramesQuery(
      { cfq: "abc", cft: "old" },
      { cft: "new" },
    )
    expect(merged).toEqual({ cfq: "abc", cft: "new" })

    const cleared = mergeCuratedFramesQuery(merged, { cft: "" })
    expect(cleared).toEqual({ cfq: "abc" })
  })
})

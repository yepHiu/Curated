import { describe, expect, it } from "vitest"
import {
  buildBrowseRouteTarget,
  buildMovieRouteQuery,
  getBrowseSourceMode,
  getLibrarySearchQuery,
  getLibraryTabQuery,
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
        tab: "favorites",
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
      tab: "favorites",
    })

    const movieQuery = buildMovieRouteQuery(
      {
        q: "Mina",
        selected: "sld-101",
        tab: "favorites",
      },
      "recent",
      "nva-102",
    )

    expect(browseTarget).toEqual({
      name: "recent",
      query: {
        q: "Mina",
        selected: "sld-101",
        tab: "favorites",
      },
    })

    expect(movieQuery).toEqual({
      from: "recent",
      q: "Mina",
      selected: "nva-102",
      tab: "favorites",
    })
  })
})

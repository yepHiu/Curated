import { describe, expect, it } from "vitest"
import { buildLibraryBrowseScrollKey } from "@/lib/library-scroll-key"

describe("buildLibraryBrowseScrollKey", () => {
  it("omits the selected movie from the cache key", () => {
    expect(
      buildLibraryBrowseScrollKey({
        name: "library",
        query: {
          q: "abc",
          selected: "movie-1",
          tab: "new",
        },
      } as never),
    ).toBe("library?q=abc&tab=new")
  })

  it("sorts query keys for stable cache keys", () => {
    expect(
      buildLibraryBrowseScrollKey({
        name: "favorites",
        query: {
          tab: "top-rated",
          actor: "Aoi",
        },
      } as never),
    ).toBe("favorites?actor=Aoi&tab=top-rated")
  })
})

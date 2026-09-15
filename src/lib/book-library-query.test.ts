import { describe, expect, it } from "vitest"
import {
  comicLibraryHasConstraints,
  isLegacyFavoriteSort,
  parseBookLibrarySort,
  parseComicLibraryBrowse,
  parsePhotoLibraryBrowse,
  patchBookLibraryQuery,
  photoLibraryHasConstraints,
} from "./book-library-query"

describe("book library query", () => {
  it("treats fileName as the only non-default sort and ignores favorite as a sort value", () => {
    expect(parseBookLibrarySort("fileName")).toBe("fileName")
    expect(parseBookLibrarySort("addedAt")).toBe("addedAt")
    expect(parseBookLibrarySort("favorite")).toBe("addedAt")
    expect(isLegacyFavoriteSort({ sort: "favorite" })).toBe(true)
  })

  it("maps legacy sort=favorite onto the favorite filter while keeping imported-time order", () => {
    expect(parseComicLibraryBrowse({ sort: "favorite", q: " rain " })).toEqual({
      q: "rain",
      tag: "",
      favorite: true,
      readStatus: "all",
      sort: "addedAt",
    })
  })

  it("parses comic favorite and read-status filters from dedicated query keys", () => {
    expect(
      parseComicLibraryBrowse({
        tag: "作者:青井",
        favorite: "1",
        readStatus: "reading",
        sort: "fileName",
      }),
    ).toEqual({
      q: "",
      tag: "作者:青井",
      favorite: true,
      readStatus: "reading",
      sort: "fileName",
    })
  })

  it("canonicalizes book library query patches and drops abandoned keys", () => {
    expect(
      patchBookLibraryQuery(
        { q: "old", filter: "unread", sort: "favorite", favorite: "1" },
        { q: "", favorite: true, sort: "addedAt", readStatus: "all" },
      ),
    ).toEqual({
      favorite: "1",
    })
    expect(
      patchBookLibraryQuery({ tag: "keep" }, { q: "night", sort: "fileName", tag: "作者:甲" }),
    ).toEqual({
      q: "night",
      tag: "作者:甲",
      sort: "fileName",
    })
  })

  it("parses photo browse state without inventing favorite or read-status filters", () => {
    expect(parsePhotoLibraryBrowse({ q: "beach", tag: "summer", sort: "favorite" })).toEqual({
      q: "beach",
      tag: "summer",
      sort: "addedAt",
    })
    expect(photoLibraryHasConstraints(parsePhotoLibraryBrowse({ tag: "summer" }))).toBe(true)
    expect(comicLibraryHasConstraints(parseComicLibraryBrowse({}))).toBe(false)
  })
})

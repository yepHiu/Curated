import { afterEach, describe, expect, it, vi } from "vitest"
import { httpClient } from "./http-client"
import { comicApi } from "./comic-endpoints"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("comicApi", () => {
  it("adds comic library paths through the comic path endpoint", async () => {
    const result = {
      id: "comic-library-1",
      path: "D:/Comics",
      title: "Comics",
      firstLibraryScanPending: true,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(result)

    await expect(
      comicApi.addComicLibraryPath({ path: "D:/Comics", title: "Comics" }),
    ).resolves.toEqual(result)

    expect(post).toHaveBeenCalledWith("/library/comics/paths", {
      path: "D:/Comics",
      title: "Comics",
    })
  })

  it("patches only comic settings through the shared settings endpoint", async () => {
    const settings = {
      comicLibraryEnabled: true,
      defaultComicImportLibraryPathId: "comic-library-1",
    }
    const patch = vi.spyOn(httpClient, "patch").mockResolvedValueOnce(settings)

    await expect(
      comicApi.patchComicSettings({
        comicLibraryEnabled: true,
        defaultComicImportLibraryPathId: "comic-library-1",
      }),
    ).resolves.toEqual(settings)

    expect(patch).toHaveBeenCalledWith("/settings", {
      comicLibraryEnabled: true,
      defaultComicImportLibraryPathId: "comic-library-1",
    })
  })

  it("starts comic scans through the independent comic scan endpoint", async () => {
    const task = {
      taskId: "scan-comics-1",
      type: "scan.comics",
      status: "running",
      createdAt: "2026-06-28T00:00:00Z",
      progress: 0,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(task)

    await expect(comicApi.startComicScan()).resolves.toEqual(task)

    expect(post).toHaveBeenCalledWith("/library/comics/scans", {})
  })

  it("can restrict comic scans to selected comic library paths", async () => {
    const task = {
      taskId: "scan-comics-path-1",
      type: "scan.comics",
      status: "running",
      createdAt: "2026-06-28T00:00:00Z",
      progress: 0,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(task)

    await expect(comicApi.startComicScan({ paths: ["D:/Comics"] })).resolves.toEqual(task)

    expect(post).toHaveBeenCalledWith("/library/comics/scans", { paths: ["D:/Comics"] })
  })

  it("lists and patches comics through independent comic collection endpoints", async () => {
    const page = { items: [], total: 0, limit: 50, offset: 0 }
    const list = vi.spyOn(httpClient, "get").mockResolvedValueOnce(page)
    const detail = {
      id: "comic-1",
      title: "Comic",
      tags: [],
      isFavorite: true,
      readStatus: "unread",
      pageCount: 1,
      currentPageIndex: 0,
      sourceFileName: "comic.cbz",
      location: "D:/Comics/comic.cbz",
      addedAt: "2026-06-28T00:00:00Z",
      updatedAt: "2026-06-28T00:00:00Z",
      pages: [],
    }
    const patch = vi.spyOn(httpClient, "patch").mockResolvedValueOnce(detail)

    await expect(comicApi.listComics({ q: "Comic", favorite: true, limit: 50 })).resolves.toEqual(page)
    await expect(comicApi.patchComic("comic-1", { favorite: true })).resolves.toEqual(detail)

    expect(list).toHaveBeenCalledWith("/library/comics", {
      q: "Comic",
      favorite: "true",
      limit: 50,
    })
    expect(patch).toHaveBeenCalledWith("/library/comics/comic-1", { favorite: true })
  })

  it("uses comic book subresource routes for pages, progress, and preferences", async () => {
    const get = vi.spyOn(httpClient, "get")
    const put = vi.spyOn(httpClient, "put")
    get
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce({ comicId: "comic-1", pageIndex: 1, completed: false })
      .mockResolvedValueOnce({ comicId: "comic-1", mode: "scroll", fit: "width", direction: "rtl" })
    put
      .mockResolvedValueOnce({ comicId: "comic-1", pageIndex: 2, completed: false })
      .mockResolvedValueOnce({ comicId: "comic-1", mode: "page", fit: "contain", direction: "rtl" })

    await comicApi.listComicPages("comic-1")
    await comicApi.getComicProgress("comic-1")
    await comicApi.putComicProgress("comic-1", { pageIndex: 2, completed: false })
    await comicApi.getComicPreferences("comic-1")
    await comicApi.putComicPreferences("comic-1", { mode: "page", fit: "contain", direction: "rtl" })

    expect(get).toHaveBeenNthCalledWith(1, "/library/comics/books/comic-1/pages")
    expect(get).toHaveBeenNthCalledWith(2, "/library/comics/books/comic-1/progress")
    expect(put).toHaveBeenNthCalledWith(1, "/library/comics/books/comic-1/progress", {
      pageIndex: 2,
      completed: false,
    })
    expect(get).toHaveBeenNthCalledWith(3, "/library/comics/books/comic-1/preferences")
    expect(put).toHaveBeenNthCalledWith(2, "/library/comics/books/comic-1/preferences", {
      mode: "page",
      fit: "contain",
      direction: "rtl",
    })
  })

  it("uses comic-only cache endpoints", async () => {
    const status = { maxBytes: 1024, usedBytes: 10, entryCount: 1 }
    const get = vi.spyOn(httpClient, "get").mockResolvedValueOnce(status)
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(status)

    await expect(comicApi.getComicCacheStatus()).resolves.toEqual(status)
    await expect(comicApi.cleanupComicCache()).resolves.toEqual(status)

    expect(get).toHaveBeenCalledWith("/library/comics/cache/status")
    expect(post).toHaveBeenCalledWith("/library/comics/cache/cleanup", {})
  })
})

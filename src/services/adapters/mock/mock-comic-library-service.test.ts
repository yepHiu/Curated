import { beforeEach, describe, expect, it, vi } from "vitest"

async function freshMockComicService() {
  vi.resetModules()
  return await import("./mock-comic-library-service")
}

beforeEach(() => {
  localStorage.clear()
  vi.resetModules()
})

describe("mockComicLibraryService", () => {
  it("returns sample comics for the comic wall", async () => {
    const { mockComicLibraryService } = await freshMockComicService()

    expect(mockComicLibraryService.comicsLoaded.value).toBe(true)
    expect(mockComicLibraryService.comics.value.length).toBeGreaterThanOrEqual(3)
    expect(mockComicLibraryService.comics.value[0]).toMatchObject({
      id: expect.any(String),
      title: expect.any(String),
      pageCount: expect.any(Number),
      sourceFileName: expect.stringMatching(/\.(cbz|zip)$/),
    })
  })

  it("stores favorite, rating, progress, and reading preferences in comic localStorage", async () => {
    const [{ MOCK_COMIC_PREFS_KEY }, { mockComicLibraryService }] = await Promise.all([
      import("@/lib/mock-comic-prefs-storage"),
      freshMockComicService(),
    ])
    const comicId = mockComicLibraryService.comics.value[0]?.id
    expect(comicId).toBeTruthy()
    if (!comicId) return

    await mockComicLibraryService.patchComic(comicId, {
      favorite: true,
      rating: 4.5,
      tags: ["author:manual", "stitched-spread"],
    })
    await mockComicLibraryService.saveComicProgress(comicId, 3, false)
    await mockComicLibraryService.saveComicPreferences(comicId, {
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })

    const raw = localStorage.getItem(MOCK_COMIC_PREFS_KEY)
    expect(raw).toContain(comicId)
    expect(raw).not.toContain("jav-library-movie-prefs")

    const { mockComicLibraryService: reloadedService } = await freshMockComicService()
    expect(reloadedService.getComicById(comicId)).toMatchObject({
      isFavorite: true,
      rating: 4.5,
      tags: ["author:manual", "stitched-spread"],
      currentPageIndex: 3,
      readStatus: "reading",
    })
    await expect(reloadedService.getComicProgress(comicId)).resolves.toMatchObject({
      comicId,
      pageIndex: 3,
      completed: false,
    })
    await expect(reloadedService.getComicPreferences(comicId)).resolves.toMatchObject({
      comicId,
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })
  })

  it("rejects real scan and file reveal actions in mock mode", async () => {
    const { mockComicLibraryService } = await freshMockComicService()

    await expect(mockComicLibraryService.scanComics()).rejects.toMatchObject({
      status: 501,
    })
    await expect(mockComicLibraryService.revealComicSource("comic-1")).rejects.toMatchObject({
      status: 501,
    })
  })
})

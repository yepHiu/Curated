import { beforeEach, describe, expect, it, vi } from "vitest"
import type {
  ComicBookDetailDTO,
  ComicBookListItemDTO,
  ComicBooksPageDTO,
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  SettingsDTO,
} from "@/api/types"

const comicApiMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  patchComicSettings: vi.fn(),
  listComicLibraryPaths: vi.fn(),
  addComicLibraryPath: vi.fn(),
  updateComicLibraryPathTitle: vi.fn(),
  deleteComicLibraryPath: vi.fn(),
  startComicScan: vi.fn(),
  listComics: vi.fn(),
  getComic: vi.fn(),
  patchComic: vi.fn(),
  deleteComic: vi.fn(),
  revealComicSource: vi.fn(),
  listComicPages: vi.fn(),
  getComicProgress: vi.fn(),
  putComicProgress: vi.fn(),
  deleteComicProgress: vi.fn(),
  getComicPreferences: vi.fn(),
  putComicPreferences: vi.fn(),
  getComicCacheStatus: vi.fn(),
  cleanupComicCache: vi.fn(),
}))

const movieApiMocks = vi.hoisted(() => ({
  listMovies: vi.fn(),
  patchMovie: vi.fn(),
  getMovie: vi.fn(),
}))

vi.mock("@/api/comic-endpoints", () => ({
  comicApi: comicApiMocks,
}))

vi.mock("@/api/endpoints", () => ({
  api: movieApiMocks,
}))

function comicListItem(id: string, overrides: Partial<ComicBookListItemDTO> = {}): ComicBookListItemDTO {
  return {
    id,
    title: `Comic ${id}`,
    tags: ["author:manual"],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 3,
    currentPageIndex: 0,
    coverUrl: `/api/library/comics/books/${id}/pages/0/thumbnail`,
    sourceFileName: `${id}.cbz`,
    location: `D:/Comics/${id}.cbz`,
    addedAt: "2026-06-28T00:00:00Z",
    updatedAt: "2026-06-28T00:00:00Z",
    ...overrides,
  }
}

function comicDetail(id: string, overrides: Partial<ComicBookDetailDTO> = {}): ComicBookDetailDTO {
  return {
    ...comicListItem(id),
    pages: [
      {
        comicId: id,
        index: 0,
        entryPath: "001.jpg",
        fileName: "001.jpg",
        imageExt: ".jpg",
        imageUrl: `/api/library/comics/books/${id}/pages/0/image`,
        thumbUrl: `/api/library/comics/books/${id}/pages/0/thumbnail`,
      },
    ],
    ...overrides,
  }
}

function comicsPage(items: ComicBookListItemDTO[]): ComicBooksPageDTO {
  return {
    items,
    total: items.length,
    limit: 500,
    offset: 0,
  }
}

function settingsDto(overrides: Partial<SettingsDTO> = {}): SettingsDTO {
  return {
    libraryPaths: [],
    comicLibraryEnabled: true,
    comicLibraryPaths: [
      {
        id: "comic-path-1",
        path: "D:/Comics",
        title: "Comics",
        firstLibraryScanPending: false,
      },
    ],
    defaultComicImportLibraryPathId: "comic-path-1",
    comicReader: {
      mode: "page",
      fit: "contain",
      direction: "rtl",
    },
    comicCache: {
      maxBytes: 2 * 1024 * 1024 * 1024,
    },
    player: {
      hardwareDecode: true,
      nativePlayerEnabled: false,
      streamPushEnabled: true,
      preferNativePlayer: false,
      seekForwardStepSec: 10,
      seekBackwardStepSec: 10,
    },
    organizeLibrary: true,
    autoLibraryWatch: true,
    autoActorProfileScrape: false,
    autoDownloadUpdates: false,
    launchAtLogin: false,
    launchAtLoginSupported: false,
    curatedFrameExportFormat: "jpg",
    metadataMovieProvider: "",
    metadataMovieProviders: [],
    metadataMovieProviderChain: [],
    proxy: { enabled: false },
    backendLog: {
      logDir: "",
      logLevel: "info",
    },
    ...overrides,
  }
}

beforeEach(() => {
  vi.resetModules()
  for (const mock of Object.values(comicApiMocks)) {
    mock.mockReset()
  }
  for (const mock of Object.values(movieApiMocks)) {
    mock.mockReset()
  }
})

describe("webComicLibraryService", () => {
  it("loads comics through comic endpoints and never calls movie endpoints", async () => {
    comicApiMocks.listComics.mockResolvedValueOnce(comicsPage([comicListItem("comic-1")]))

    const { webComicLibraryService } = await import("./web-comic-library-service")
    await webComicLibraryService.reloadComicsFromApi({ q: "manual" })

    expect(comicApiMocks.listComics).toHaveBeenCalledWith(
      expect.objectContaining({
        q: "manual",
        limit: 500,
        offset: 0,
      }),
    )
    expect(webComicLibraryService.comics.value.map((comic) => comic.id)).toEqual(["comic-1"])
    expect(movieApiMocks.listMovies).not.toHaveBeenCalled()
    expect(movieApiMocks.getMovie).not.toHaveBeenCalled()
    expect(movieApiMocks.patchMovie).not.toHaveBeenCalled()
  })

  it("patches comic metadata through the comic patch endpoint and updates the cache", async () => {
    comicApiMocks.listComics.mockResolvedValueOnce(comicsPage([comicListItem("comic-1")]))
    comicApiMocks.patchComic.mockResolvedValueOnce(
      comicDetail("comic-1", {
        title: "Patched title",
        tags: ["author:manual", "favorite"],
        rating: 4.5,
        isFavorite: true,
      }),
    )

    const { webComicLibraryService } = await import("./web-comic-library-service")
    await webComicLibraryService.reloadComicsFromApi()
    const updated = await webComicLibraryService.patchComic(" comic-1 ", {
      title: "Patched title",
      tags: ["author:manual", "favorite"],
      favorite: true,
      rating: 4.5,
    })

    expect(comicApiMocks.patchComic).toHaveBeenCalledWith("comic-1", {
      title: "Patched title",
      tags: ["author:manual", "favorite"],
      favorite: true,
      ratingSet: true,
      rating: 4.5,
    })
    expect(updated?.title).toBe("Patched title")
    expect(webComicLibraryService.getComicById("comic-1")?.isFavorite).toBe(true)
    expect(movieApiMocks.patchMovie).not.toHaveBeenCalled()
  })

  it("syncs progress and reading preferences through comic book subresource endpoints", async () => {
    const progress: ComicReadingProgressDTO = {
      comicId: "comic-1",
      pageIndex: 2,
      completed: false,
      updatedAt: "2026-06-28T00:00:00Z",
    }
    const prefs: ComicReadingPreferencesDTO = {
      comicId: "comic-1",
      mode: "scroll",
      fit: "width",
      direction: "rtl",
      updatedAt: "2026-06-28T00:00:00Z",
    }
    comicApiMocks.putComicProgress.mockResolvedValueOnce(progress)
    comicApiMocks.putComicPreferences.mockResolvedValueOnce(prefs)

    const { webComicLibraryService } = await import("./web-comic-library-service")
    await expect(webComicLibraryService.saveComicProgress(" comic-1 ", 2, false)).resolves.toEqual(progress)
    await expect(
      webComicLibraryService.saveComicPreferences(" comic-1 ", {
        mode: "scroll",
        fit: "width",
        direction: "rtl",
      }),
    ).resolves.toEqual(prefs)

    expect(comicApiMocks.putComicProgress).toHaveBeenCalledWith("comic-1", {
      pageIndex: 2,
      completed: false,
    })
    expect(comicApiMocks.putComicPreferences).toHaveBeenCalledWith("comic-1", {
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })
  })

  it("loads comic settings through the comic settings facade", async () => {
    comicApiMocks.getSettings.mockResolvedValueOnce(settingsDto())

    const { webComicLibraryService } = await import("./web-comic-library-service")
    await webComicLibraryService.refreshSettings()

    expect(webComicLibraryService.comicLibraryEnabled.value).toBe(true)
    expect(webComicLibraryService.comicLibraryPaths.value).toEqual([
      {
        id: "comic-path-1",
        path: "D:/Comics",
        title: "Comics",
        firstLibraryScanPending: false,
      },
    ])
    expect(webComicLibraryService.defaultComicImportLibraryPathId.value).toBe("comic-path-1")
    expect(webComicLibraryService.comicReader.value.direction).toBe("rtl")
  })
})

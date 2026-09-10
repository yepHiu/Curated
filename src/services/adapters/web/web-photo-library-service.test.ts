import { beforeEach, describe, expect, it, vi } from "vitest"
import type { SettingsDTO } from "@/api/types"

const photoApiMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  patchPhotoSettings: vi.fn(),
  addPhotoLibraryPath: vi.fn(),
  updatePhotoLibraryPathTitle: vi.fn(),
  deletePhotoLibraryPath: vi.fn(),
  startPhotoScan: vi.fn(),
  listPhotos: vi.fn(),
  getPhoto: vi.fn(),
  importPhotos: vi.fn(),
}))

vi.mock("@/api/photo-endpoints", () => ({
  photoApi: photoApiMocks,
}))

function settingsDto(overrides: Partial<SettingsDTO> = {}): SettingsDTO {
  return {
    libraryPaths: [],
    backupDirectory: "",
    comicLibraryEnabled: false,
    autoComicLibraryWatch: true,
    comicLibraryPaths: [],
    comicReader: {
      mode: "page",
      fit: "contain",
      direction: "ltr",
    },
    comicCache: {
      maxBytes: 2 * 1024 * 1024 * 1024,
    },
    photoLibraryEnabled: false,
    autoPhotoLibraryWatch: true,
    photoLibraryPaths: [],
    photoViewer: {
      mode: "page",
      fit: "contain",
      direction: "ltr",
    },
    photoCache: {
      maxBytes: 5 * 1024 * 1024 * 1024,
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
    curatedFrameExportMode: "raw",
    metadataMovieProvider: "",
    metadataMovieProviders: [],
    metadataMovieProviderChain: [],
    proxy: { enabled: false },
    aiProvider: { kind: "openai-compatible", baseUrl: "", model: "" },
    backendLog: {
      logDir: "",
      logLevel: "info",
    },
    ...overrides,
  }
}

beforeEach(() => {
  vi.resetModules()
  for (const mock of Object.values(photoApiMocks)) {
    mock.mockReset()
  }
})

describe("webPhotoLibraryService", () => {
  it("rejects disabled photo uploads and delegates enabled ones", async () => {
    const { webPhotoLibraryService: service } = await import("./web-photo-library-service")
    photoApiMocks.getSettings.mockResolvedValueOnce(settingsDto())
    await service.refreshSettings()
    const files = [new File(["zip"], "photos.zip")]
    await expect(service.importPhotos(files)).rejects.toThrow()
    expect(photoApiMocks.importPhotos).not.toHaveBeenCalled()
    photoApiMocks.getSettings.mockResolvedValueOnce(settingsDto({photoLibraryEnabled: true}))
    await service.refreshSettings()
    photoApiMocks.importPhotos.mockResolvedValueOnce({ taskId: "photo-import" })
    const options = { onUploadProgress: vi.fn() }
    await expect(service.importPhotos(files, options)).resolves.toEqual({taskId: "photo-import"})
    expect(photoApiMocks.importPhotos).toHaveBeenCalledWith(files, options)
  })

  it("loads photo settings from independent photo fields in settings", async () => {
    photoApiMocks.getSettings.mockResolvedValueOnce(
      settingsDto({
        photoLibraryEnabled: true,
        autoPhotoLibraryWatch: false,
        photoLibraryPaths: [
          {
            id: "photo-path-1",
            path: "D:/Photos",
            title: "Photos",
            firstLibraryScanPending: false,
          },
        ],
        defaultPhotoImportLibraryPathId: "photo-path-1",
        photoViewer: {
          mode: "scroll",
          fit: "width",
          direction: "rtl",
        },
        photoCache: {
          maxBytes: 1024,
        },
      }),
    )

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.refreshSettings()

    expect(webPhotoLibraryService.photoLibraryEnabled.value).toBe(true)
    expect(webPhotoLibraryService.autoPhotoLibraryWatch.value).toBe(false)
    expect(webPhotoLibraryService.photoLibraryPaths.value).toEqual([
      {
        id: "photo-path-1",
        path: "D:/Photos",
        title: "Photos",
        firstLibraryScanPending: false,
      },
    ])
    expect(webPhotoLibraryService.defaultPhotoImportLibraryPathId.value).toBe("photo-path-1")
    expect(webPhotoLibraryService.photoViewer.value).toEqual({
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })
    expect(webPhotoLibraryService.photoCache.value).toEqual({ maxBytes: 1024 })
  })

  it("patches automatic photo library watch through photo settings fields", async () => {
    photoApiMocks.patchPhotoSettings.mockResolvedValueOnce(
      settingsDto({ autoPhotoLibraryWatch: false }),
    )

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.setAutoPhotoLibraryWatch(false)

    expect(photoApiMocks.patchPhotoSettings).toHaveBeenCalledWith({
      autoPhotoLibraryWatch: false,
    })
    expect(webPhotoLibraryService.autoPhotoLibraryWatch.value).toBe(false)
  })

  it("adds, updates, and removes photo library paths through photo path endpoints", async () => {
    photoApiMocks.addPhotoLibraryPath.mockResolvedValueOnce({
      id: "photo-path-1",
      path: "D:/Photos",
      title: "Photos",
      firstLibraryScanPending: true,
    })
    photoApiMocks.updatePhotoLibraryPathTitle.mockResolvedValueOnce({
      id: "photo-path-1",
      path: "D:/Photos",
      title: "Photo Shelf",
      firstLibraryScanPending: true,
    })
    photoApiMocks.deletePhotoLibraryPath.mockResolvedValueOnce(undefined)

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.addPhotoLibraryPath(" D:/Photos ", " Photos ")
    await webPhotoLibraryService.updatePhotoLibraryPathTitle(" photo-path-1 ", " Photo Shelf ")
    await webPhotoLibraryService.removePhotoLibraryPath(" photo-path-1 ")

    expect(photoApiMocks.addPhotoLibraryPath).toHaveBeenCalledWith({
      path: "D:/Photos",
      title: "Photos",
    })
    expect(photoApiMocks.updatePhotoLibraryPathTitle).toHaveBeenCalledWith("photo-path-1", {
      title: "Photo Shelf",
    })
    expect(photoApiMocks.deletePhotoLibraryPath).toHaveBeenCalledWith("photo-path-1")
    expect(webPhotoLibraryService.photoLibraryPaths.value).toEqual([])
  })

  it("starts photo scans through the photo scan endpoint", async () => {
    const task = {
      taskId: "task-photo-scan",
      type: "scan.photos",
      status: "running",
      progress: 0,
      createdAt: "2026-07-05T00:00:00Z",
    }
    photoApiMocks.startPhotoScan.mockResolvedValueOnce(task)

    const { webPhotoLibraryService } = await import("./web-photo-library-service")

    await expect(webPhotoLibraryService.scanPhotos([" D:/Photos "])).resolves.toEqual(task)
    expect(photoApiMocks.startPhotoScan).toHaveBeenCalledWith({ paths: ["D:/Photos"] })
  })

  it("loads photo books from the photo list endpoint", async () => {
    photoApiMocks.listPhotos.mockResolvedValueOnce({
      items: [
        {
          id: "photo-1",
          title: "Photo One",
          tags: ["portrait"],
          rating: null,
          isFavorite: false,
          pageCount: 2,
          currentPageIndex: 0,
          coverUrl: "/api/library/photos/books/photo-1/pages/0/thumbnail",
          sourceFileName: "Photo One.cbz",
          location: "D:/Photos/Photo One.cbz",
          addedAt: "2026-07-05",
          updatedAt: "2026-07-05T00:00:00Z",
        },
      ],
      total: 1,
      limit: 50,
      offset: 0,
    })

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.reloadPhotosFromApi({ q: "Photo", limit: 50 })

    expect(photoApiMocks.listPhotos).toHaveBeenCalledWith({ q: "Photo", limit: 50 })
    expect(webPhotoLibraryService.photos.value).toEqual([
      {
        id: "photo-1",
        title: "Photo One",
        tags: ["portrait"],
        rating: null,
        isFavorite: false,
        pageCount: 2,
        currentPageIndex: 0,
        coverUrl: "/api/library/photos/books/photo-1/pages/0/thumbnail",
        sourceFileName: "Photo One.cbz",
        location: "D:/Photos/Photo One.cbz",
        addedAt: "2026-07-05",
        updatedAt: "2026-07-05T00:00:00Z",
      },
    ])
  })

  it("patches photo viewer and cache settings without using comic fields", async () => {
    photoApiMocks.patchPhotoSettings
      .mockResolvedValueOnce(
        settingsDto({
          photoViewer: {
            mode: "scroll",
            fit: "width",
            direction: "rtl",
          },
        }),
      )
      .mockResolvedValueOnce(
        settingsDto({
          photoViewer: {
            mode: "scroll",
            fit: "width",
            direction: "rtl",
          },
          photoCache: {
            maxBytes: 2048,
          },
        }),
      )

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.patchPhotoViewer({ mode: "scroll", fit: "width" })
    await webPhotoLibraryService.patchPhotoCache({ maxBytes: 2048 })

    expect(photoApiMocks.patchPhotoSettings).toHaveBeenNthCalledWith(1, {
      photoViewer: {
        mode: "scroll",
        fit: "width",
        direction: "ltr",
      },
    })
    expect(photoApiMocks.patchPhotoSettings).toHaveBeenNthCalledWith(2, {
      photoCache: {
        maxBytes: 2048,
      },
    })
    expect(webPhotoLibraryService.photoViewer.value).toEqual({
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })
    expect(webPhotoLibraryService.photoCache.value).toEqual({ maxBytes: 2048 })
  })
})

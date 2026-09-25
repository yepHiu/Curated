import { beforeEach, describe, expect, it, vi } from "vitest"
import type { PhotoBookListItemDTO, SettingsDTO } from "@/api/types"

const photoApiMocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  patchPhotoSettings: vi.fn(),
  addPhotoLibraryPath: vi.fn(),
  updatePhotoLibraryPathTitle: vi.fn(),
  deletePhotoLibraryPath: vi.fn(),
  startPhotoScan: vi.fn(),
  listPhotos: vi.fn(),
  getPhoto: vi.fn(),
  replacePhotoTags: vi.fn(),
  patchPhoto: vi.fn(),
  deletePhoto: vi.fn(),
  getPhotoComment: vi.fn(),
  putPhotoComment: vi.fn(),
  importPhotos: vi.fn(),
}))

vi.mock("@/api/photo-endpoints", () => ({
  photoApi: photoApiMocks,
}))

/** 构造写真列表 DTO，供分页与暖页测试复用。 */
function photoListItem(id: string, overrides: Partial<PhotoBookListItemDTO> = {}): PhotoBookListItemDTO {
  return {
    id,
    title: `Photo ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 1,
    currentPageIndex: 0,
    coverUrl: `/api/library/photos/books/${id}/pages/0/thumbnail`,
    sourceFileName: `${id}.cbz`,
    location: `D:/Photos/${id}.cbz`,
    addedAt: "2026-07-05",
    updatedAt: "2026-07-05T00:00:00Z",
    ...overrides,
  }
}

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
    lanEnabled: false,
    lanListening: false,
    lanAccessUrls: [],
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
  it("updates favorites and removes only the deleted photo from the cache", async () => {
    const { webPhotoLibraryService: service } = await import("./web-photo-library-service")
    photoApiMocks.listPhotos.mockResolvedValueOnce({ items: [photoListItem("photo-1"), photoListItem("photo-2")], total: 2, limit: 500, offset: 0 })
    await service.reloadPhotosFromApi()
    photoApiMocks.patchPhoto.mockResolvedValueOnce({ ...photoListItem("photo-1"), isFavorite: true, pages: [] })
    await service.patchPhoto("photo-1", { favorite: true })
    expect(photoApiMocks.patchPhoto).toHaveBeenCalledWith("photo-1", { favorite: true })
    expect(service.getPhotoById("photo-1")?.isFavorite).toBe(true)
    photoApiMocks.deletePhoto.mockResolvedValueOnce(undefined)
    await service.deletePhoto("photo-1")
    expect(service.getPhotoById("photo-1")).toBeUndefined()
    expect(service.getPhotoById("photo-2")).toBeDefined()
  })
  it("updates the shared photo cache only after tags save successfully", async () => {
    const { webPhotoLibraryService: service } = await import('./web-photo-library-service')
    photoApiMocks.replacePhotoTags.mockResolvedValueOnce({ id: 'photo-1', title: 'Photo', tags: ['landscape'], pages: [] })
    const saved = await service.replacePhotoTags(' photo-1 ', ['landscape'])
    expect(photoApiMocks.replacePhotoTags).toHaveBeenCalledWith('photo-1', ['landscape'])
    expect(service.getPhotoById('photo-1')).toEqual(saved)
    photoApiMocks.replacePhotoTags.mockRejectedValueOnce(new Error('Save failed'))
    await expect(service.replacePhotoTags('photo-1', ['new'])).rejects.toThrow('Save failed')
    expect(service.getPhotoById('photo-1')!.tags).toEqual(['landscape'])
  })

  it("reads and saves photo comments through the photo book endpoint", async () => {
    const { webPhotoLibraryService: service } = await import("./web-photo-library-service")
    photoApiMocks.getPhotoComment.mockResolvedValueOnce({ body: "note", updatedAt: "t" })
    photoApiMocks.putPhotoComment.mockResolvedValueOnce({ body: "saved", updatedAt: "t2" })
    await expect(service.getPhotoComment(" photo-1 ")).resolves.toEqual({ body: "note", updatedAt: "t" })
    await expect(service.putPhotoComment(" photo-1 ", { body: "saved" })).resolves.toEqual({
      body: "saved",
      updatedAt: "t2",
    })
    expect(photoApiMocks.getPhotoComment).toHaveBeenCalledWith("photo-1")
    expect(photoApiMocks.putPhotoComment).toHaveBeenCalledWith("photo-1", { body: "saved" })
  })

  it("patches a photo rating through the photo book endpoint and updates the cache", async () => {
    const { webPhotoLibraryService: service } = await import("./web-photo-library-service")
    photoApiMocks.listPhotos.mockResolvedValueOnce({
      items: [photoListItem("photo-1")],
      total: 1,
      limit: 500,
      offset: 0,
    })
    await service.reloadPhotosFromApi()
    photoApiMocks.patchPhoto.mockResolvedValueOnce({
      ...photoListItem("photo-1", { rating: 3.5 }),
      pages: [],
    })
    await expect(service.patchPhoto(" photo-1 ", { rating: 3.5 })).resolves.toMatchObject({
      id: "photo-1",
      rating: 3.5,
    })
    expect(photoApiMocks.patchPhoto).toHaveBeenCalledWith("photo-1", {
      ratingSet: true,
      rating: 3.5,
    })
    expect(service.getPhotoById("photo-1")?.rating).toBe(3.5)
    photoApiMocks.patchPhoto.mockResolvedValueOnce({
      ...photoListItem("photo-1", { title: "展示写真" }),
      pages: [],
    })
    await expect(service.patchPhoto("photo-1", { title: "展示写真" })).resolves.toMatchObject({
      id: "photo-1",
      title: "展示写真",
    })
    expect(photoApiMocks.patchPhoto).toHaveBeenLastCalledWith("photo-1", {
      title: "展示写真",
    })
    photoApiMocks.patchPhoto.mockResolvedValueOnce({
      ...photoListItem("photo-1", { rating: null }),
      pages: [],
    })
    await service.patchPhoto("photo-1", { rating: null })
    expect(photoApiMocks.patchPhoto).toHaveBeenLastCalledWith("photo-1", {
      ratingSet: true,
      ratingClear: true,
    })
  })
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

    expect(photoApiMocks.listPhotos).toHaveBeenCalledWith({ q: "Photo", limit: 50, offset: 0 })
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

  it("pages through the full photo list when reload is called without an explicit limit", async () => {
    const firstPage = Array.from({ length: 500 }, (_, index) => photoListItem(`photo-${index + 1}`))
    const secondPage = Array.from({ length: 100 }, (_, index) => photoListItem(`photo-${index + 501}`))
    photoApiMocks.listPhotos
      .mockResolvedValueOnce({
        items: firstPage,
        total: 600,
        limit: 500,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: secondPage,
        total: 600,
        limit: 500,
        offset: 500,
      })

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.reloadPhotosFromApi()

    expect(photoApiMocks.listPhotos).toHaveBeenCalledTimes(2)
    expect(photoApiMocks.listPhotos).toHaveBeenNthCalledWith(1, { limit: 500, offset: 0 })
    expect(photoApiMocks.listPhotos).toHaveBeenNthCalledWith(2, { limit: 500, offset: 500 })
    expect(webPhotoLibraryService.photos.value).toHaveLength(600)
  })

  it("skips settings and list requests when the photo library is already warm", async () => {
    photoApiMocks.getSettings.mockResolvedValue(settingsDto())
    photoApiMocks.listPhotos.mockResolvedValue({
      items: [photoListItem("photo-1")],
      total: 1,
      limit: 500,
      offset: 0,
    })

    const { webPhotoLibraryService } = await import("./web-photo-library-service")
    await webPhotoLibraryService.ensurePhotosLoaded()
    await webPhotoLibraryService.ensurePhotosLoaded()

    expect(photoApiMocks.getSettings).toHaveBeenCalledTimes(1)
    expect(photoApiMocks.listPhotos).toHaveBeenCalledTimes(1)
  })
})

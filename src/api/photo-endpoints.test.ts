import { afterEach, describe, expect, it, vi } from "vitest"
import { httpClient } from "./http-client"
import { photoApi } from "./photo-endpoints"

afterEach(() => {
  vi.restoreAllMocks()
})

describe("photoApi", () => {
  it("adds photo library paths through the photo path endpoint", async () => {
    const result = {
      id: "photo-library-1",
      path: "D:/Photos",
      title: "Photos",
      firstLibraryScanPending: true,
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(result)

    await expect(
      photoApi.addPhotoLibraryPath({ path: "D:/Photos", title: "Photos" }),
    ).resolves.toEqual(result)

    expect(post).toHaveBeenCalledWith("/library/photos/paths", {
      path: "D:/Photos",
      title: "Photos",
    })
  })

  it("lists, patches, and deletes photo library paths through independent photo endpoints", async () => {
    const paths = [
      {
        id: "photo-library-1",
        path: "D:/Photos",
        title: "Photos",
        firstLibraryScanPending: true,
      },
    ]
    const get = vi.spyOn(httpClient, "get").mockResolvedValueOnce(paths)
    const patch = vi.spyOn(httpClient, "patch").mockResolvedValueOnce({
      ...paths[0],
      title: "Photo Shelf",
    })
    const del = vi.spyOn(httpClient, "delete").mockResolvedValueOnce(undefined)

    await expect(photoApi.listPhotoLibraryPaths()).resolves.toEqual(paths)
    await expect(
      photoApi.updatePhotoLibraryPathTitle("photo-library-1", { title: "Photo Shelf" }),
    ).resolves.toEqual({ ...paths[0], title: "Photo Shelf" })
    await expect(photoApi.deletePhotoLibraryPath("photo-library-1")).resolves.toBeUndefined()

    expect(get).toHaveBeenCalledWith("/library/photos/paths")
    expect(patch).toHaveBeenCalledWith("/library/photos/paths/photo-library-1", {
      title: "Photo Shelf",
    })
    expect(del).toHaveBeenCalledWith("/library/photos/paths/photo-library-1")
  })

  it("patches only photo settings through the shared settings endpoint", async () => {
    const settings = {
      photoLibraryEnabled: true,
      defaultPhotoImportLibraryPathId: "photo-library-1",
    }
    const patch = vi.spyOn(httpClient, "patch").mockResolvedValueOnce(settings)

    await expect(
      photoApi.patchPhotoSettings({
        photoLibraryEnabled: true,
        defaultPhotoImportLibraryPathId: "photo-library-1",
      }),
    ).resolves.toEqual(settings)

    expect(patch).toHaveBeenCalledWith("/settings", {
      photoLibraryEnabled: true,
      defaultPhotoImportLibraryPathId: "photo-library-1",
    })
  })

  it("starts photo scans through the independent photo scan endpoint", async () => {
    const task = {
      taskId: "task-photo-scan",
      type: "scan.photos",
      status: "running",
      progress: 0,
      createdAt: "2026-07-05T00:00:00Z",
    }
    const post = vi.spyOn(httpClient, "post").mockResolvedValueOnce(task)

    await expect(photoApi.startPhotoScan({ paths: ["D:/Photos"] })).resolves.toEqual(task)

    expect(post).toHaveBeenCalledWith("/library/photos/scans", { paths: ["D:/Photos"] })
  })

  it("loads photo books through independent photo endpoints", async () => {
    const page = {
      items: [
        {
          id: "photo-1",
          title: "Photo One",
          tags: [],
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
    }
    const get = vi
      .spyOn(httpClient, "get")
      .mockResolvedValueOnce(page)
      .mockResolvedValueOnce({ ...page.items[0], pages: [] })

    await expect(photoApi.listPhotos({ q: "Photo", limit: 50 })).resolves.toEqual(page)
    await expect(photoApi.getPhoto("photo-1")).resolves.toEqual({ ...page.items[0], pages: [] })

    expect(get).toHaveBeenNthCalledWith(1, "/library/photos", {
      q: "Photo",
      tag: undefined,
      favorite: undefined,
      limit: 50,
      offset: undefined,
    })
    expect(get).toHaveBeenNthCalledWith(2, "/library/photos/photo-1")
  })
})

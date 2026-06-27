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
})

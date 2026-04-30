import { afterEach, describe, expect, it, vi } from "vitest"
import type { CuratedFrameRecord } from "@/domain/curated-frame/types"

function frame(overrides: Partial<CuratedFrameRecord> = {}): CuratedFrameRecord {
  return {
    id: "frame-1",
    movieId: "movie-1",
    title: "Movie title",
    code: "ABC-123",
    actors: ["Mina"],
    positionSec: 12.5,
    capturedAt: "2026-04-30T12:00:00.000Z",
    tags: ["tag-a"],
    ...overrides,
  }
}

function makeApiMock() {
  return {
    listCuratedFrames: vi.fn(),
    patchCuratedFrameTags: vi.fn(),
    deleteCuratedFrame: vi.fn(),
    getCuratedFrameStats: vi.fn(),
    listCuratedFrameTags: vi.fn(),
  }
}

async function importDb(opts?: { useWebApi?: boolean; api?: ReturnType<typeof makeApiMock> }) {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", opts?.useWebApi ? "true" : "false")
  const api = opts?.api ?? makeApiMock()
  const bumpCuratedFramesRevision = vi.fn()
  vi.doMock("@/api/endpoints", () => ({ api }))
  vi.doMock("@/lib/curated-frames/revision", () => ({ bumpCuratedFramesRevision }))
  return {
    api,
    bumpCuratedFramesRevision,
    db: await import("@/lib/curated-frames/db"),
  }
}

afterEach(() => {
  vi.resetModules()
  vi.clearAllMocks()
  vi.unstubAllEnvs()
})

describe("curated frames db local guards", () => {
  it("rejects local puts without an image blob before opening IndexedDB", async () => {
    const { db } = await importDb({ useWebApi: false })

    await expect(db.putCuratedFrame(frame())).rejects.toThrow("本地模式需要 imageBlob")
  })
})

describe("curated frames db web api mode", () => {
  it("lists paged frames through the API and maps DTO rows for display", async () => {
    const api = makeApiMock()
    api.listCuratedFrames.mockResolvedValueOnce({
      items: [frame({ id: "frame-1" })],
      total: 1,
      limit: 20,
      offset: 10,
    })
    const { db } = await importDb({ useWebApi: true, api })

    await expect(db.listCuratedFramesPage({ q: "mina", limit: 20, offset: 10 })).resolves.toEqual({
      items: [frame({ id: "frame-1" })],
      total: 1,
      limit: 20,
      offset: 10,
    })
    expect(api.listCuratedFrames).toHaveBeenCalledWith({ q: "mina", limit: 20, offset: 10 })
  })

  it("updates tags through the API and bumps the curated frame revision", async () => {
    const api = makeApiMock()
    api.patchCuratedFrameTags.mockResolvedValueOnce(undefined)
    const { bumpCuratedFramesRevision, db } = await importDb({ useWebApi: true, api })

    await db.updateCuratedFrameTags("frame-1", ["a", "b"])

    expect(api.patchCuratedFrameTags).toHaveBeenCalledWith("frame-1", { tags: ["a", "b"] })
    expect(bumpCuratedFramesRevision).toHaveBeenCalledTimes(1)
  })

  it("deletes frames through the API and bumps the curated frame revision", async () => {
    const api = makeApiMock()
    api.deleteCuratedFrame.mockResolvedValueOnce(undefined)
    const { bumpCuratedFramesRevision, db } = await importDb({ useWebApi: true, api })

    await db.deleteCuratedFrame("frame-1")

    expect(api.deleteCuratedFrame).toHaveBeenCalledWith("frame-1")
    expect(bumpCuratedFramesRevision).toHaveBeenCalledTimes(1)
  })

  it("reads frame totals from API stats", async () => {
    const api = makeApiMock()
    api.getCuratedFrameStats.mockResolvedValueOnce({ total: 12 })
    const { db } = await importDb({ useWebApi: true, api })

    await expect(db.countCuratedFrames()).resolves.toBe(12)
  })

  it("loads tag suggestions and count facets from the API", async () => {
    const api = makeApiMock()
    api.listCuratedFrameTags
      .mockResolvedValueOnce({
        items: [
          { name: "tag-b", count: 1 },
          { name: "tag-a", count: 2 },
        ],
      })
      .mockResolvedValueOnce({
        items: [
          { name: "tag-b", count: 1 },
          { name: "tag-a", count: 2 },
        ],
      })
    const { db } = await importDb({ useWebApi: true, api })

    await expect(db.listCuratedFrameTagSuggestions()).resolves.toEqual(["tag-b", "tag-a"])
    await expect(db.listCuratedFrameTagFacets("zh-CN")).resolves.toEqual([
      { name: "tag-a", count: 2 },
      { name: "tag-b", count: 1 },
    ])
  })
})

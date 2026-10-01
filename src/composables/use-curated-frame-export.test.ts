import { beforeEach, describe, expect, it, vi } from "vitest"
import { useCuratedFrameExport } from "./use-curated-frame-export"
import type { CuratedFrameDialogItem } from "@/lib/curated-frames/dialog-navigation"

const { rawExport, watermarkedExport, download } = vi.hoisted(() => ({ rawExport: vi.fn(), watermarkedExport: vi.fn(), download: vi.fn() }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ exportCuratedFrames: rawExport }) }))
vi.mock("@/lib/curated-frames/watermarked-export", () => ({ buildWatermarkedCuratedFramesExport: watermarkedExport }))
vi.mock("@/lib/curated-frames/export-file", () => ({ triggerDownloadBlob: download }))

beforeEach(() => {
  vi.clearAllMocks()
})

describe("shared curated-frame export", () => {
  it("preserves actor-specific raw export parameters and downloads the returned filename", async () => {
    const blob = new Blob(["raw"])
    rawExport.mockResolvedValue({ blob, filename: "frames.zip" })
    await useCuratedFrameExport().exportFrames([], ["a", "b"], "Actor A", "png", "raw")
    expect(rawExport).toHaveBeenCalledWith({ ids: ["a", "b"], actorName: "Actor A", format: "png" })
    expect(download).toHaveBeenCalledWith(blob, "frames.zip")
    expect(watermarkedExport).not.toHaveBeenCalled()
  })

  it("uses full originals for Web watermarks and the existing Blob URL for Mock", async () => {
    const base = { movieId: "m", title: "Movie", code: "CODE", actors: [], tags: [], positionSec: 1, capturedAt: "2026-10-01" }
    const items: CuratedFrameDialogItem[] = [
      { row: { ...base, id: "web" }, url: "thumbnail-web" },
      { row: { ...base, id: "mock", imageBlob: new Blob(["original"]) }, url: "blob:mock-original" },
    ]
    watermarkedExport.mockResolvedValue({ blob: new Blob(["watermarked"]), filename: "watermarked.zip" })
    await useCuratedFrameExport().exportFrames(items, ["web", "mock"], undefined, "jpg", "watermarked")
    expect(watermarkedExport).toHaveBeenCalledWith([
      { row: items[0]!.row, imageUrl: expect.stringContaining("/web/image") },
      { row: items[1]!.row, imageUrl: "blob:mock-original" },
    ], "jpg")
    expect(rawExport).not.toHaveBeenCalled()
  })
})

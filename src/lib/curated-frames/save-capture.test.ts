import { afterEach, describe, expect, it, vi } from "vitest"

describe("resolveCuratedCapturePositionSec", () => {
  afterEach(() => {
    vi.resetModules()
  })

  it("prefers an explicit absolute playback position when provided", async () => {
    const { resolveCuratedCapturePositionSec } = await import("@/lib/curated-frames/save-capture")
    expect(resolveCuratedCapturePositionSec(12.25, 145.875)).toBe(145.875)
  })

  it("falls back to the video local currentTime when no override is provided", async () => {
    const { resolveCuratedCapturePositionSec } = await import("@/lib/curated-frames/save-capture")
    expect(resolveCuratedCapturePositionSec(12.25)).toBe(12.25)
  })
})

describe("saveCuratedCaptureFromVideo", () => {
  afterEach(() => {
    vi.resetModules()
    vi.restoreAllMocks()
    vi.unstubAllEnvs()
  })

  it("does not block local capture when a nearby curated frame already exists", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")
    const blob = new Blob(["png"], { type: "image/png" })
    const findNearbyCuratedFrame = vi.fn().mockResolvedValue({ id: "near-frame" })
    const putCuratedFrame = vi.fn().mockResolvedValue(undefined)

    vi.doMock("@/api/endpoints", () => ({
      api: {
        createCuratedFrameUpload: vi.fn(),
      },
    }))
    vi.doMock("@/i18n", () => ({
      i18n: {
        global: {
          t: (key: string) => key,
        },
      },
    }))
    vi.doMock("@/lib/curated-frames/capture", () => ({
      captureVideoFrameToPng: vi.fn().mockResolvedValue({ ok: true, blob }),
      formatFrameFilename: vi.fn(() => "frame.png"),
    }))
    vi.doMock("@/lib/curated-frames/db", () => ({
      findNearbyCuratedFrame,
      getStoredDirectoryHandle: vi.fn().mockResolvedValue(null),
      putCuratedFrame,
    }))
    vi.doMock("@/lib/curated-frames/export-file", () => ({
      triggerDownloadBlob: vi.fn(),
      writeBlobToDirectory: vi.fn(),
    }))
    vi.doMock("@/lib/curated-frames/revision", () => ({
      bumpCuratedFramesRevision: vi.fn(),
    }))
    vi.doMock("@/lib/curated-frames/settings-storage", () => ({
      getCuratedFrameSaveMode: vi.fn(() => "app"),
    }))

    const { saveCuratedCaptureFromVideo } = await import("@/lib/curated-frames/save-capture")
    const result = await saveCuratedCaptureFromVideo(
      { currentTime: 42 } as HTMLVideoElement,
      { id: "movie-1", title: "Title", code: "CODE", actors: ["Mina"] } as never,
    )

    expect(result).toEqual({ ok: true })
    expect(findNearbyCuratedFrame).not.toHaveBeenCalled()
    expect(putCuratedFrame).toHaveBeenCalled()
  })
})

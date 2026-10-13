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
    vi.unstubAllGlobals()
  })

  it.each(["false", "true"])("captures and saves without randomUUID (Web API %s), including nearby frames", async (webApi) => {
    const source = globalThis.crypto
    vi.stubGlobal("crypto", { getRandomValues: source.getRandomValues.bind(source) })
    vi.stubEnv("VITE_USE_WEB_API", webApi)
    const blob = new Blob(["png"], { type: "image/png" })
    const findNearbyCuratedFrame = vi.fn().mockResolvedValue({ id: "near-frame" })
    const putCuratedFrame = vi.fn().mockResolvedValue(undefined)
    const createCuratedFrameUpload = vi.fn().mockResolvedValue(undefined)

    vi.doMock("@/api/endpoints", () => ({
      api: {
        createCuratedFrameUpload,
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

    expect(result).toMatchObject({ ok: true, id: expect.any(String) })
    if (result.ok) expect(result.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    expect(findNearbyCuratedFrame).not.toHaveBeenCalled()
    if (webApi === "true") {
      expect(createCuratedFrameUpload).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ id: result.ok ? result.id : "", movieId: "movie-1", positionSec: 42 }), blob)
      expect(putCuratedFrame).not.toHaveBeenCalled()
    } else {
      expect(putCuratedFrame).toHaveBeenCalled()
      expect(createCuratedFrameUpload).not.toHaveBeenCalled()
    }
  })
})

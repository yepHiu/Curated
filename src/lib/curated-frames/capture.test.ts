import { afterEach, describe, expect, it, vi } from "vitest"
import { captureVideoFrameToPng, formatFrameFilename } from "@/lib/curated-frames/capture"

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

function videoSize(width: number, height: number): HTMLVideoElement {
  return {
    videoWidth: width,
    videoHeight: height,
  } as HTMLVideoElement
}

function mockCanvas(options: {
  ctx: CanvasRenderingContext2D | null
  blob?: Blob | null
}) {
  const originalCreateElement = document.createElement.bind(document)
  const canvas = {
    width: 0,
    height: 0,
    getContext: vi.fn(() => options.ctx),
    toBlob: vi.fn((callback: BlobCallback) => {
      callback(options.blob ?? null)
    }),
  } as unknown as HTMLCanvasElement

  vi.spyOn(document, "createElement").mockImplementation(
    ((tagName: string, elementOptions?: ElementCreationOptions) => {
      if (tagName.toLowerCase() === "canvas") {
        return canvas
      }
      return originalCreateElement(tagName, elementOptions)
    }) as typeof document.createElement,
  )

  return canvas
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe("captureVideoFrameToPng", () => {
  it("reports not-ready videos before touching canvas APIs", async () => {
    await expect(captureVideoFrameToPng(videoSize(0, 720))).resolves.toEqual({
      ok: false,
      reason: "curated.captureNotReady",
    })
  })

  it("reports missing 2d canvas contexts", async () => {
    mockCanvas({ ctx: null })

    await expect(captureVideoFrameToPng(videoSize(1280, 720))).resolves.toEqual({
      ok: false,
      reason: "curated.captureNoCtx",
    })
  })

  it("reports CORS canvas failures when drawing the video throws", async () => {
    mockCanvas({
      ctx: {
        drawImage: vi.fn(() => {
          throw new DOMException("tainted canvas", "SecurityError")
        }),
      } as unknown as CanvasRenderingContext2D,
    })

    await expect(captureVideoFrameToPng(videoSize(1280, 720))).resolves.toEqual({
      ok: false,
      reason: "curated.captureCors",
    })
  })

  it("reports blob conversion failures", async () => {
    mockCanvas({
      ctx: { drawImage: vi.fn() } as unknown as CanvasRenderingContext2D,
      blob: null,
    })

    await expect(captureVideoFrameToPng(videoSize(1280, 720))).resolves.toEqual({
      ok: false,
      reason: "curated.captureBlobFail",
    })
  })

  it("returns a png blob when canvas capture succeeds", async () => {
    const blob = new Blob(["png"], { type: "image/png" })
    const drawImage = vi.fn()
    const canvas = mockCanvas({
      ctx: { drawImage } as unknown as CanvasRenderingContext2D,
      blob,
    })

    await expect(captureVideoFrameToPng(videoSize(1280, 720))).resolves.toEqual({
      ok: true,
      blob,
    })
    expect(canvas.width).toBe(1280)
    expect(canvas.height).toBe(720)
    expect(drawImage).toHaveBeenCalledTimes(1)
  })
})

describe("formatFrameFilename", () => {
  it("sanitizes unsafe filename characters and floors the playback position", () => {
    expect(
      formatFrameFilename(
        'AB/12\\34?%*:|"<>',
        42.9,
        "2026-04-30T12:34:56.789Z",
      ),
    ).toBe("Curated_AB_12_34_________42s_2026-04-30T12-34-56.png")
  })
})

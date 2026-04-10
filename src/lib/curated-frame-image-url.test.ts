import { describe, expect, it } from "vitest"
import { curatedFrameImageUrl, curatedFrameThumbnailUrl } from "@/lib/curated-frame-image-url"

describe("curated frame image urls", () => {
  it("builds same-origin image and thumbnail URLs", () => {
    expect(curatedFrameImageUrl("frame 1")).toContain("/api/curated-frames/frame%201/image")
    expect(curatedFrameThumbnailUrl("frame 1")).toContain("/api/curated-frames/frame%201/thumbnail")
  })
})

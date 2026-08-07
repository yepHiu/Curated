import { describe, expect, it } from "vitest"
import {
  buildStoredZip,
  buildWatermarkedBandLayout,
  buildWatermarkedFrameFilename,
} from "./watermarked-export"

describe("watermarked export helpers", () => {
  it("keeps the added band close to ten percent of the source height", () => {
    const layout = buildWatermarkedBandLayout(1080)

    expect(layout.bandHeight).toBe(108)
    expect(layout.bandHeight / 1080).toBeCloseTo(0.1, 2)
    expect(layout.padding).toBeGreaterThan(0)
  })

  it("builds a safe, descriptive filename", () => {
    expect(
      buildWatermarkedFrameFilename(
        { code: "ABC/123", positionSec: 12.9, id: "frame-123456789" },
        "png",
      ),
    ).toBe("curated-watermarked-ABC_123-12s-frame-12.png")
  })

  it("builds a valid stored zip with local, central, and end signatures", async () => {
    const blob = buildStoredZip([
      { name: "one.txt", data: new Uint8Array([1, 2, 3]) },
      { name: "two.txt", data: new Uint8Array([4, 5]) },
    ])
    const bytes = new Uint8Array(await blob.arrayBuffer())
    const view = new DataView(bytes.buffer)

    expect(blob.type).toBe("application/zip")
    expect(view.getUint32(0, true)).toBe(0x04034b50)
    const endOffset = bytes.length - 22
    expect(view.getUint32(endOffset, true)).toBe(0x06054b50)
    const centralOffset = view.getUint32(endOffset + 16, true)
    expect(view.getUint32(centralOffset, true)).toBe(0x02014b50)
  })
})

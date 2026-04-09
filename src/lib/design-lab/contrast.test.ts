import { describe, expect, it } from "vitest"
import { getContrastRatio } from "@/lib/design-lab/contrast"

describe("getContrastRatio", () => {
  it("returns 21 for black on white", () => {
    expect(getContrastRatio("#000000", "#ffffff")).toBe(21)
  })

  it("returns 1 for identical colors", () => {
    expect(getContrastRatio("#fe628e", "#fe628e")).toBe(1)
  })
})

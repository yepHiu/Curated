import { beforeEach, describe, expect, it } from "vitest"
import type { ComicReaderSettings } from "@/domain/comic/types"
import {
  clearTemporaryStitch,
  getTemporaryStitch,
  resolveComicReaderPreferences,
  resolveReaderKeyStep,
  resolveStitchPairDisplayOrder,
  setTemporaryStitch,
} from "./comic-reader-controls"

const defaults: ComicReaderSettings = {
  mode: "page",
  fit: "contain",
  direction: "ltr",
}

beforeEach(() => {
  clearTemporaryStitch("comic-1")
})

describe("comic reader controls", () => {
  it("uses global defaults when no per-book preference exists", () => {
    expect(resolveComicReaderPreferences(defaults)).toEqual(defaults)
  })

  it("uses per-book preference overrides over global defaults", () => {
    expect(
      resolveComicReaderPreferences(defaults, {
        comicId: "comic-1",
        mode: "scroll",
        fit: "width",
        direction: "rtl",
      }),
    ).toEqual({
      mode: "scroll",
      fit: "width",
      direction: "rtl",
    })
  })

  it("maps arrow keys by reading direction and lets space advance", () => {
    expect(resolveReaderKeyStep("ArrowRight", "ltr")).toBe(1)
    expect(resolveReaderKeyStep("ArrowLeft", "ltr")).toBe(-1)
    expect(resolveReaderKeyStep("ArrowRight", "rtl")).toBe(-1)
    expect(resolveReaderKeyStep("ArrowLeft", "rtl")).toBe(1)
    expect(resolveReaderKeyStep(" ", "rtl")).toBe(1)
    expect(resolveReaderKeyStep("Space", "ltr")).toBe(1)
  })

  it("orders stitched current and next pages by visual reading direction", () => {
    expect(resolveStitchPairDisplayOrder(4, 5, "ltr")).toEqual([4, 5])
    expect(resolveStitchPairDisplayOrder(4, 5, "rtl")).toEqual([5, 4])
  })

  it("stores temporary stitch state per comic session", () => {
    setTemporaryStitch("comic-1", { anchorPageIndex: 4, adjacentPageIndex: 5 })

    expect(getTemporaryStitch("comic-1")).toEqual({
      anchorPageIndex: 4,
      adjacentPageIndex: 5,
    })

    clearTemporaryStitch("comic-1")

    expect(getTemporaryStitch("comic-1")).toBeUndefined()
  })
})

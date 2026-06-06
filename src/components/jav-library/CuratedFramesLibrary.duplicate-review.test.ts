import { describe, expect, it } from "vitest"
import curatedFramesLibrarySource from "./CuratedFramesLibrary.vue?raw"

describe("CuratedFramesLibrary duplicate review summary", () => {
  it("does not render the duplicate review summary alert above the frame library", () => {
    expect(curatedFramesLibrarySource).not.toContain("curated.duplicateReviewTitle")
    expect(curatedFramesLibrarySource).not.toContain("curated.duplicateReviewBody")
  })
})

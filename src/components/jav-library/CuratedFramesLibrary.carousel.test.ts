import { describe, expect, it } from "vitest"
import curatedFramesLibrarySource from "./CuratedFramesLibrary.vue?raw"

describe("CuratedFramesLibrary dialog carousel", () => {
  it("uses the shadcn-vue Carousel primitives for the dialog image pane", () => {
    expect(curatedFramesLibrarySource).toContain("@/components/ui/carousel")
    expect(curatedFramesLibrarySource).toContain("<Carousel")
    expect(curatedFramesLibrarySource).toContain("<CarouselContent")
    expect(curatedFramesLibrarySource).toContain("<CarouselItem")
    expect(curatedFramesLibrarySource).toContain("entry in dialogNavigationEntries")
    expect(curatedFramesLibrarySource).toContain("@init-api=\"onDialogCarouselInit\"")
  })
})

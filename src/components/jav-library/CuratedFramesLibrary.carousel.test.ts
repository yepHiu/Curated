import { describe, expect, it } from "vitest"
import curatedFramesLibrarySource from "./CuratedFramesLibrary.vue?raw"
import dialogSource from "./CuratedFrameDetailDialog.vue?raw"

describe("CuratedFramesLibrary dialog carousel", () => {
  it("uses the shadcn-vue Carousel primitives for the dialog image pane", () => {
    expect(curatedFramesLibrarySource).toContain("<CuratedFrameDetailDialog")
    expect(dialogSource).toContain("@/components/ui/carousel")
    expect(dialogSource).toContain("<Carousel")
    expect(dialogSource).toContain("<CarouselContent")
    expect(dialogSource).toContain("<CarouselItem")
    expect(dialogSource).toContain("entry in dialogNavigationEntries")
    expect(dialogSource).toContain("@init-api=\"onDialogCarouselInit\"")
  })
})

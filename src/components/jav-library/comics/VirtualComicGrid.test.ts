import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import VirtualComicGrid from "./VirtualComicGrid.vue"

vi.mock("@vueuse/core", () => ({
  useResizeObserver: vi.fn(),
  useMediaQuery: vi.fn(() => ({ value: false, __v_isRef: true })),
}))

vi.mock("vue-virtual-scroller", () => ({
  DynamicScroller: {
    props: ["items"],
    template: `
      <div data-dynamic-scroller data-comic-grid-scroller class="h-full min-h-0 overflow-y-auto pr-2">
        <template v-for="(item, index) in ((items || []).length <= 4 ? items : (items || []).slice(0, 2))" :key="item.id ?? index">
          <slot :item="item" :index="index" :active="true" />
        </template>
      </div>
    `,
  },
  DynamicScrollerItem: {
    template: "<div data-dynamic-scroller-item><slot /></div>",
  },
}))

vi.mock("@/components/jav-library/comics/ComicCard.vue", () => ({
  default: {
    name: "ComicCard",
    props: ["comic", "batchMode", "batchChecked"],
    emits: ["toggleBatchSelect"],
    template:
      "<article data-comic-card :data-batch-mode=\"batchMode ? 'true' : 'false'\" :data-batch-checked=\"batchChecked ? 'true' : 'false'\" @click=\"$emit('toggleBatchSelect', comic.id)\">{{ comic.id }}</article>",
  },
}))

/** 构造墙测用漫画条目。 */
function makeComic(id: string): ComicBook {
  return {
    id,
    title: `Comic ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 12,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Comics/${id}.cbz`,
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
  }
}

describe("VirtualComicGrid", () => {
  it("uses the same grid track and gap variables as the movie poster grid", () => {
    const wrapper = mount(VirtualComicGrid, {
      props: {
        comics: [makeComic("c1")],
      },
    })

    const style = wrapper.get("[data-virtual-comic-grid]").attributes("style")

    expect(style).toContain(
      "grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))",
    )
    expect(style).toContain("column-gap: var(--movie-grid-gap)")
    expect(style).toContain("row-gap: var(--movie-grid-gap)")
    expect(style).toContain("padding-bottom: var(--movie-grid-gap)")
  })

  it("centers comic cards inside the same max-width frame as the movie grid", () => {
    const wrapper = mount(VirtualComicGrid, {
      props: {
        comics: [makeComic("c1")],
      },
    })

    const cardShell = wrapper.get("[data-comic-card-shell]")
    const cardFrame = wrapper.get("[data-comic-card-frame]")

    expect(cardShell.classes()).toEqual(expect.arrayContaining(["flex", "min-w-0", "justify-center"]))
    expect(cardFrame.classes()).toEqual(expect.arrayContaining(["w-full", "min-w-0"]))
    expect(cardFrame.attributes("style")).toContain("max-width: min(100%, var(--movie-card-max-width))")
  })

  it("uses the same inner scroll gutter as the movie poster scroller", () => {
    const wrapper = mount(VirtualComicGrid, {
      props: {
        comics: [makeComic("c1")],
      },
    })

    expect(wrapper.get("[data-comic-grid-scroller]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0", "overflow-y-auto", "pr-2"]),
    )
  })

  it("passes batch selection state into comic cards and forwards toggles", async () => {
    const wrapper = mount(VirtualComicGrid, {
      props: {
        comics: [makeComic("c1"), makeComic("c2")],
        batchMode: true,
        batchSelectedIds: ["c2"],
      },
    })

    const cards = wrapper.findAll("[data-comic-card]")
    expect(cards[0]?.attributes("data-batch-mode")).toBe("true")
    expect(cards[0]?.attributes("data-batch-checked")).toBe("false")
    expect(cards[1]?.attributes("data-batch-checked")).toBe("true")

    await cards[0]!.trigger("click")

    expect(wrapper.emitted("toggleBatchSelect")).toEqual([["c1"]])
  })

  it("keeps the active card DOM far smaller than a 300-book list", () => {
    const comics = Array.from({ length: 300 }, (_, index) => makeComic(`c${index + 1}`))
    const wrapper = mount(VirtualComicGrid, {
      props: { comics },
    })

    expect(wrapper.findAll("[data-comic-card]").length).toBeLessThan(80)
    expect(wrapper.findAll("[data-comic-card]").length).toBeGreaterThan(0)
  })
})

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import VirtualPhotoGrid from "./VirtualPhotoGrid.vue"

vi.mock("@vueuse/core", () => ({
  useResizeObserver: vi.fn(),
  useMediaQuery: vi.fn(() => ({ value: false, __v_isRef: true })),
}))

vi.mock("vue-virtual-scroller", () => ({
  DynamicScroller: {
    props: ["items"],
    template: `
      <div data-dynamic-scroller data-photo-grid-scroller class="h-full min-h-0 overflow-y-auto pr-2">
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

vi.mock("@/components/jav-library/photos/PhotoCard.vue", () => ({
  default: {
    name: "PhotoCard",
    props: ["photo"],
    template: "<article data-photo-card>{{ photo.id }}</article>",
  },
}))

/** 构造墙测用写真条目。 */
function makePhoto(id: string): PhotoBook {
  return {
    id,
    title: `Photo ${id}`,
    tags: [],
    rating: null,
    isFavorite: false,
    pageCount: 12,
    currentPageIndex: 0,
    sourceFileName: `${id}.cbz`,
    location: `D:/Photos/${id}.cbz`,
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
  }
}

describe("VirtualPhotoGrid", () => {
  it("uses the same grid track and gap variables as the movie poster grid", () => {
    const wrapper = mount(VirtualPhotoGrid, {
      props: {
        photos: [makePhoto("p1")],
      },
    })

    const style = wrapper.get("[data-virtual-photo-grid]").attributes("style")

    expect(style).toContain(
      "grid-template-columns: repeat(auto-fill, minmax(min(100%, var(--movie-grid-min-track)), 1fr))",
    )
    expect(style).toContain("column-gap: var(--movie-grid-gap)")
    expect(style).toContain("row-gap: var(--movie-grid-gap)")
    expect(style).toContain("padding-bottom: var(--movie-grid-gap)")
  })

  it("centers photo cards inside the same max-width frame as the movie grid", () => {
    const wrapper = mount(VirtualPhotoGrid, {
      props: {
        photos: [makePhoto("p1")],
      },
    })

    const cardShell = wrapper.get("[data-photo-card-shell]")
    const cardFrame = wrapper.get("[data-photo-card-frame]")

    expect(cardShell.classes()).toEqual(expect.arrayContaining(["flex", "min-w-0", "justify-center"]))
    expect(cardFrame.classes()).toEqual(expect.arrayContaining(["w-full", "min-w-0"]))
    expect(cardFrame.attributes("style")).toContain("max-width: min(100%, var(--movie-card-max-width))")
  })

  it("uses the same inner scroll gutter as the movie poster scroller", () => {
    const wrapper = mount(VirtualPhotoGrid, {
      props: {
        photos: [makePhoto("p1")],
      },
    })

    expect(wrapper.get("[data-photo-grid-scroller]").classes()).toEqual(
      expect.arrayContaining(["h-full", "min-h-0", "overflow-y-auto", "pr-2"]),
    )
  })

  it("keeps the active card DOM far smaller than a 300-book list", () => {
    const photos = Array.from({ length: 300 }, (_, index) => makePhoto(`p${index + 1}`))
    const wrapper = mount(VirtualPhotoGrid, {
      props: { photos },
    })

    expect(wrapper.findAll("[data-photo-card]").length).toBeLessThan(80)
    expect(wrapper.findAll("[data-photo-card]").length).toBeGreaterThan(0)
  })
})

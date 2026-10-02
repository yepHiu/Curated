import { mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { Movie } from "@/domain/movie/types"

import VirtualMovieMasonry from "./VirtualMovieMasonry.vue"
import { estimateVirtualMovieChunkHeight } from "@/lib/library-virtual-scroll"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"

const mediaQueryMatches = vi.hoisted(() => ({ value: false, __v_isRef: true }))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@vueuse/core", () => ({
  useResizeObserver: vi.fn(),
  useMediaQuery: vi.fn(() => mediaQueryMatches),
}))

vi.mock("lucide-vue-next", () => ({
  Inbox: { template: "<span />" },
  SearchX: { template: "<span />" },
  ChevronUp: { template: "<span />" },
}))

vi.mock("vue-virtual-scroller", () => ({
  DynamicScroller: {
    props: ["items"],
    template: `
      <div data-dynamic-scroller>
        <div data-dynamic-scroller-before><slot name="before" /></div>
        <template v-for="(item, index) in items" :key="item.id ?? index">
          <slot :item="item" :index="index" :active="true" />
        </template>
        <div data-dynamic-scroller-after><slot name="after" /></div>
      </div>
    `,
  },
  DynamicScrollerItem: {
    template: "<div data-dynamic-scroller-item><slot /></div>",
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { template: "<div data-empty-card><slot /></div>" },
  CardDescription: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
}))

vi.mock("@/components/jav-library/MovieCard.vue", () => ({
  default: {
    props: ["movie"],
    template: "<article data-movie-card>{{ movie.id }}</article>",
  },
}))

vi.mock("@/composables/use-library-scroll-preserve", () => ({
  useLibraryScrollPreserve: () => ({
    scrollTop: ref(0),
    scrollToTop: vi.fn(),
  }),
}))

vi.mock("@/lib/library-virtual-scroll", () => ({
  estimateVirtualMovieChunkHeight: vi.fn(() => 320),
  getVirtualMovieFocusChunkIndex: vi.fn(() => 0),
  resolveVirtualMoviePosterLoadPolicy: vi.fn(() => ({
    loading: "lazy",
    fetchPriority: "auto",
  })),
}))

vi.mock("@/lib/movie-grid-template", () => ({
  buildMovieGridChunkStyle: vi.fn(() => ({})),
}))

function makeMovie(id: string): Movie {
  return {
    id,
    code: id,
    title: `Movie ${id}`,
    studio: "",
    actors: [],
    tags: [],
    userTags: [],
    runtimeMinutes: 0,
    rating: 0,
    summary: "",
    isFavorite: false,
    addedAt: "",
    location: "",
    resolution: "",
    year: 0,
    tone: "",
    coverClass: "",
  }
}

describe("VirtualMovieMasonry", () => {
  beforeEach(() => {
    mediaQueryMatches.value = false
    vi.mocked(estimateVirtualMovieChunkHeight).mockClear()
    vi.mocked(buildMovieGridChunkStyle).mockClear()
  })

  it("renders header slot inside the dynamic scroller before movie items", () => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: {
        movies: [makeMovie("m1")],
      },
      slots: {
        header: "<div data-masonry-header>Actor Profile</div>",
      },
    })

    const html = wrapper.html()
    const scrollerIndex = html.indexOf("data-dynamic-scroller")
    const headerIndex = html.indexOf("data-masonry-header")
    const movieCardIndex = html.indexOf("data-movie-card")

    expect(scrollerIndex).toBeGreaterThanOrEqual(0)
    expect(wrapper.find("[data-dynamic-scroller-before] [data-masonry-header]").exists()).toBe(true)
    expect(headerIndex).toBeGreaterThan(scrollerIndex)
    expect(movieCardIndex).toBeGreaterThan(headerIndex)
  })

  it("renders header slot above empty state when there are no movies", () => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: {
        movies: [],
        emptyTitle: "Nothing here",
      },
      slots: {
        header: "<div data-masonry-header>Actor Profile</div>",
      },
    })

    const html = wrapper.html()
    const headerIndex = html.indexOf("data-masonry-header")
    const emptyCardIndex = html.indexOf("data-media-empty-state")

    expect(wrapper.find("[data-masonry-header]").exists()).toBe(true)
    expect(emptyCardIndex).toBeGreaterThan(headerIndex)
    expect(html).toContain("Nothing here")
  })

  it.each([true, false])("keeps the footer inside the scroll area (has movies: %s)", (hasMovies) => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies: hasMovies ? [makeMovie('m1')] : [] },
      slots: { footer: '<section data-frames-footer>Frames</section>' },
    })
    const html = wrapper.html()
    expect(html.indexOf('data-frames-footer')).toBeGreaterThan(html.indexOf(hasMovies ? 'data-movie-card' : 'data-media-empty-state'))
    if (hasMovies) expect(wrapper.find('[data-dynamic-scroller-after] [data-frames-footer]').exists()).toBe(true)
    else expect(wrapper.find('.overflow-y-auto [data-frames-footer]').exists()).toBe(true)
  })

  it("uses default density estimates when the Retina desktop media query does not match", () => {
    mount(VirtualMovieMasonry, {
      props: {
        movies: [makeMovie("m1")],
      },
    })

    expect(vi.mocked(buildMovieGridChunkStyle)).toHaveBeenCalledWith({
      gap: "var(--movie-grid-gap)",
      minTrackWidth: "var(--movie-grid-min-track)",
    })
    expect(vi.mocked(estimateVirtualMovieChunkHeight)).toHaveBeenCalledWith(
      expect.objectContaining({
        gapPx: 20,
      }),
    )
  })

  it("returns home only after upward overscroll at the top", async () => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies: [makeMovie("m1")], returnHomeOnOverscroll: true },
    })
    const scroller = wrapper.get("[data-movie-scroll-region]").element as HTMLElement
    await wrapper.vm.$nextTick()
    scroller.scrollTop = 300
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120 }))
    expect(wrapper.emitted("returnHome")).toBeUndefined()

    scroller.scrollTop = 0
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -50 }))
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -35 }))
    expect(wrapper.emitted("returnHome")).toHaveLength(1)
    wrapper.unmount()
  })

  it("ignores zoom, horizontal gestures and separated small wheel movements", async () => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies: [makeMovie("m1")], returnHomeOnOverscroll: true },
    })
    await wrapper.vm.$nextTick()
    const scroller = wrapper.get("[data-movie-scroll-region]").element
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120, ctrlKey: true }))
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120, deltaX: 200 }))
    let currentTime = 1000
    const now = vi.spyOn(Date, "now").mockImplementation(() => currentTime)
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -50 }))
    currentTime = 1500
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -50 }))
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    now.mockRestore()
    wrapper.unmount()
  })

  it.each([false, true])("requires an enabled return gesture, including an empty list (empty: %s)", async (empty) => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies: empty ? [] : [makeMovie("m1")] },
    })
    await wrapper.vm.$nextTick()
    const scroller = wrapper.get("[data-movie-scroll-region]").element
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120 }))
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    await wrapper.setProps({ returnHomeOnOverscroll: true })
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120 }))
    expect(wrapper.emitted("returnHome")).toHaveLength(1)
    wrapper.unmount()
  })

  it("returns home on a downward finger swipe starting at the top, once per gesture", async () => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies: [makeMovie("m1")], returnHomeOnOverscroll: true },
    })
    await wrapper.vm.$nextTick()
    const scroller = wrapper.get("[data-movie-scroll-region]").element as HTMLElement
    function touch(type: string, x: number, y: number) {
      scroller.dispatchEvent(new TouchEvent(type, {
        touches: [{ clientX: x, clientY: y } as Touch],
      }))
    }
    scroller.scrollTop = 200
    touch("touchstart", 100, 100)
    scroller.scrollTop = 0
    touch("touchmove", 100, 210)
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    touch("touchstart", 100, 100)
    touch("touchmove", 250, 190)
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    touch("touchstart", 100, 100)
    touch("touchmove", 100, 150)
    expect(wrapper.emitted("returnHome")).toBeUndefined()
    touch("touchmove", 100, 185)
    touch("touchmove", 100, 220)
    expect(wrapper.emitted("returnHome")).toHaveLength(1)
    wrapper.unmount()
  })

  it("uses compact density estimates when the Retina desktop media query matches", () => {
    mediaQueryMatches.value = true

    mount(VirtualMovieMasonry, {
      props: {
        movies: [makeMovie("m1")],
      },
    })

    expect(vi.mocked(estimateVirtualMovieChunkHeight)).toHaveBeenCalledWith(
      expect.objectContaining({
        gapPx: 16,
      }),
    )
  })
})

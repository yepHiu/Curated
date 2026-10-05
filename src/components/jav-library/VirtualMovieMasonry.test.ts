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

  it.each([
    { name: "movie list", movies: [makeMovie("m1")], slots: undefined },
    { name: "empty list with header", movies: [], slots: { header: "<h2>Movies</h2>" } },
    { name: "empty list", movies: [], slots: undefined },
  ])("keeps wheel and touch scrolling within the $name", async ({ movies, slots }) => {
    const wrapper = mount(VirtualMovieMasonry, {
      props: { movies },
      slots,
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
    scroller.dispatchEvent(new WheelEvent("wheel", { deltaY: -120 }))
    scroller.dispatchEvent(new TouchEvent("touchstart", {
      touches: [{ clientX: 100, clientY: 100 } as Touch],
    }))
    scroller.dispatchEvent(new TouchEvent("touchmove", {
      touches: [{ clientX: 100, clientY: 220 } as Touch],
    }))
    expect(wrapper.emitted("returnHome")).toBeUndefined()
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

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
    const emptyCardIndex = html.indexOf("data-empty-card")

    expect(wrapper.find("[data-masonry-header]").exists()).toBe(true)
    expect(emptyCardIndex).toBeGreaterThan(headerIndex)
    expect(html).toContain("Nothing here")
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

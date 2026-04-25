import { mount } from "@vue/test-utils"
import { nextTick, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import DetailPage from "./DetailPage.vue"

function makeMovie(overrides: Partial<Movie> = {}): Movie {
  return {
    id: "movie-1",
    title: "Movie 1",
    code: "CODE-1",
    studio: "Studio",
    actors: ["Actor A"],
    tags: ["meta-a"],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4.5,
    metadataRating: 4.5,
    userRating: undefined,
    summary: "Summary",
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: "D:/Library/movie-1.mp4",
    resolution: "1080p",
    year: 2026,
    releaseDate: "2026-04-01",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    coverUrl: "",
    thumbUrl: "",
    previewImages: [],
    actorAvatarUrls: {},
    ...overrides,
  }
}

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/composables/use-related-visible-count", () => ({
  useRelatedVisibleCount: () => ({
    visibleCount: ref(6),
  }),
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<section><slot /></section>" },
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<p><slot /></p>" },
  CardHeader: { name: "CardHeader", template: "<header><slot /></header>" },
  CardTitle: { name: "CardTitle", template: "<h2><slot /></h2>" },
}))

vi.mock("@/components/jav-library/DetailPanel.vue", () => ({
  default: { name: "DetailPanel", template: "<div data-detail-panel />" },
}))

vi.mock("@/components/jav-library/MovieGrid.vue", () => ({
  default: { name: "MovieGrid", template: "<div data-movie-grid />" },
}))

vi.mock("@/components/jav-library/PreviewImageViewer.vue", () => ({
  default: {
    name: "PreviewImageViewer",
    props: ["open", "images", "initialIndex", "movieCode"],
    template: "<div data-preview-image-viewer :data-open=\"String(open)\" :data-index=\"String(initialIndex ?? '')\" />",
  },
}))

vi.mock("@/components/jav-library/MovieCommentSection.vue", () => ({
  default: { name: "MovieCommentSection", template: "<div data-movie-comment-section />" },
}))

vi.mock("@/components/jav-library/MediaStill.vue", () => ({
  default: {
    name: "MediaStill",
    props: ["src", "alt", "fit", "loading", "fetchPriority"],
    emits: ["load", "error"],
    template: "<div class='media-still-stub' :data-src='src' />",
  },
}))

describe("DetailPage", () => {
  it("keeps a stable fallback ratio until preview dimensions load, then adapts to the image ratio", async () => {
    const wrapper = mount(DetailPage, {
      props: {
        movie: makeMovie({
          previewImages: ["https://example.com/portrait.jpg"],
        }),
        relatedMovies: [],
      },
    })

    const previewCard = wrapper.get('[data-preview-gallery-item="0"]')

    expect(previewCard.attributes("data-aspect-ratio")).toBe("1.7778")
    expect(previewCard.classes()).toContain("h-40")
    expect(previewCard.classes()).toContain("sm:h-44")
    expect(previewCard.classes()).toContain("xl:h-48")
    expect(previewCard.classes()).toContain("rounded-[1rem]")
    expect(previewCard.classes()).not.toContain("h-36")
    expect(previewCard.classes()).not.toContain("rounded-[1.25rem]")
    expect(wrapper.getComponent({ name: "MediaStill" }).props("fit")).toBe("cover")

    wrapper.getComponent({ name: "MediaStill" }).vm.$emit("load", {
      naturalWidth: 720,
      naturalHeight: 1280,
    })
    await nextTick()

    expect(previewCard.attributes("data-aspect-ratio")).toBe("0.5625")
    expect(previewCard.classes()).not.toContain("aspect-[16/9]")
  })
})

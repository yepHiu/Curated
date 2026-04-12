import { mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { ref } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import HomeHeroCarousel from "./HomeHeroCarousel.vue"

function makeMovie(id: string, overrides: Partial<Movie> = {}): Movie {
  return {
    id,
    title: `A Very Long Movie Title ${id} That Should Be Truncated In The Hero Frame`,
    code: `CODE-${id}`,
    studio: "Studio A",
    actors: ["Actor A"],
    tags: ["tag-a"],
    userTags: [],
    runtimeMinutes: 120,
    rating: 4.0,
    metadataRating: 4.0,
    userRating: undefined,
    summary: `Summary ${id}`,
    isFavorite: false,
    addedAt: "2026-04-01T00:00:00.000Z",
    location: `D:/Library/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    releaseDate: "2026-04-01",
    tone: "from-primary/35 via-primary/10 to-card",
    coverClass: "aspect-[4/5.6]",
    coverUrl: `https://example.com/${id}.jpg`,
    ...overrides,
  }
}

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/jav-library/MediaStill.vue", () => ({
  default: {
    name: "MediaStill",
    props: ["src", "alt"],
    template: "<div class='media-still-stub' />",
  },
}))

describe("HomeHeroCarousel", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("renders a sliding track with previous and next preview frames plus in-frame metadata", async () => {
    const wrapper = mount(HomeHeroCarousel, {
      props: {
        movies: [makeMovie("m1"), makeMovie("m2"), makeMovie("m3"), makeMovie("m4")],
        autoplayMs: 12000,
        manualTransitionMs: 420,
        autoplayTransitionMs: 760,
      },
    })

    expect(wrapper.get("[data-home-hero-shell]").classes()).toContain("px-0")
    expect(wrapper.get("[data-home-hero-frame]").classes()).not.toContain("rounded-[2rem]")
    expect(wrapper.get("[data-home-hero-frame]").classes()).not.toContain("bg-card/35")
    expect(wrapper.get("[data-home-hero-progress-rail]").classes()).toContain("mx-auto")
    expect(wrapper.get("[data-home-hero-progress-rail]").classes()).toContain("max-w-[54rem]")
    expect(wrapper.get("[data-home-hero-track]").classes()).toContain("transition-transform")
    expect(wrapper.find('[data-hero-slide-clone="head"]').text()).toContain("CODE-m4")
    expect(wrapper.find('[data-hero-slide-clone="tail"]').text()).toContain("CODE-m1")
    expect(wrapper.find('[data-hero-slide-state="prev"]').exists()).toBe(true)
    expect(wrapper.find('[data-hero-slide-state="active"]').exists()).toBe(true)
    expect(wrapper.find('[data-hero-slide-state="next"]').exists()).toBe(true)
    expect(wrapper.find("[data-hero-slide-overlay]").exists()).toBe(true)
    expect(wrapper.get("[data-hero-slide-overlay]").classes()).toContain("bg-[linear-gradient(180deg,rgba(8,10,14,0.04)_0%,rgba(8,10,14,0.22)_58%,rgba(8,10,14,0.72)_100%)]")
    expect(wrapper.get('[data-hero-slide-state="active"]').classes()).toContain("shadow-foreground/10")
    expect(wrapper.get('[data-hero-slide-state="active"]').classes()).not.toContain("shadow-[0_24px_64px_rgba(0,0,0,0.34)]")
    expect(wrapper.get('[data-hero-slide-state="prev"]').classes()).toContain("brightness-75")
    expect(wrapper.get('[data-hero-slide-state="prev"]').classes()).toContain("saturate-[0.82]")
    expect(wrapper.get("[data-hero-slide-title]").classes()).toContain("truncate")
    expect(wrapper.get("[data-hero-slide-code]").classes()).toContain("bg-black/44")
    expect(wrapper.get("[data-hero-slide-code]").classes()).toContain("text-white")
    expect(wrapper.get("[data-hero-slide-code]").classes()).toContain("shadow-lg")
    expect(wrapper.get('[data-hero-slide-state="active"]').text()).toContain("CODE-m1")
    expect(wrapper.get("[data-home-hero-track]").attributes("style")).toContain("transition-duration: 760ms")

    await wrapper.findAll("[data-hero-progress-item]")[1]!.trigger("click")

    expect(wrapper.get('[data-hero-slide-state="active"]').text()).toContain("CODE-m2")
    expect(wrapper.get("[data-home-hero-track]").attributes("style")).toContain("transition-duration: 420ms")
  })

  it("keeps the wrapped head slide visually aligned while snapping back from the tail clone", async () => {
    vi.useFakeTimers()

    const wrapper = mount(HomeHeroCarousel, {
      props: {
        movies: [makeMovie("m1"), makeMovie("m2"), makeMovie("m3"), makeMovie("m4")],
        autoplayMs: 12000,
      },
    })

    await wrapper.findAll("[data-hero-progress-item]")[3]!.trigger("click")
    await wrapper.findAll("[data-hero-progress-item]")[0]!.trigger("click")
    await nextTick()

    const wrappedActiveSlides = wrapper
      .findAll('[data-hero-slide-state="active"]')
      .filter((slide) => slide.text().includes("CODE-m1"))
    expect(wrappedActiveSlides).toHaveLength(2)

    vi.advanceTimersByTime(740)
    await nextTick()

    const snappedActiveSlides = wrapper
      .findAll('[data-hero-slide-state="active"]')
      .filter((slide) => slide.text().includes("CODE-m1"))
    expect(snappedActiveSlides).toHaveLength(2)
  })
})

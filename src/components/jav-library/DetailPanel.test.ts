import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import DetailPanel from "./DetailPanel.vue"
import type { Movie } from "@/domain/movie/types"
import { useExperimentalAgent } from "@/lib/experimental-agent"

const runAction = vi.hoisted(() => vi.fn())

vi.mock("@/services/ai-service", () => ({
  useAIService: () => ({ runAction }),
}))

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
    locale: ref("zh-CN"),
  }),
}))

vi.mock("@vueuse/core", () => ({
  useFocusWithin: () => ({
    focused: ref(true),
  }),
  onClickOutside: vi.fn(),
}))

vi.mock("@/components/ui/avatar", () => ({
  Avatar: { name: "Avatar", template: "<div><slot /></div>" },
  AvatarFallback: { name: "AvatarFallback", template: "<div><slot /></div>" },
  AvatarImage: { name: "AvatarImage", template: "<img />" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: {
    name: "Badge",
    props: ["as", "variant"],
    template: '<component :is="as || \'div\'" v-bind="$attrs" :data-variant="variant"><slot /></component>',
  },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { name: "Card", template: "<div><slot /></div>" },
  CardContent: {
    name: "CardContent",
    props: ["class"],
    template: '<div data-card-content :class="$props.class"><slot /></div>',
  },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuItem: {
    name: "DropdownMenuItem",
    emits: ["click"],
    template: '<button @click="$emit(\'click\', $event)"><slot /></button>',
  },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/separator", () => ({
  Separator: { name: "Separator", template: "<hr />" },
}))

vi.mock("@/components/jav-library/MediaStill.vue", () => ({
  default: { name: "MediaStill", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieDeleteConfirmDialog.vue", () => ({
  default: {
    name: "MovieDeleteConfirmDialog",
    props: ["variant"],
    emits: ["confirm", "update:open"],
    template: '<button :data-delete-variant="variant" @click="$emit(\'confirm\')" />',
  },
}))

vi.mock("@/components/jav-library/MovieEditDialog.vue", () => ({
  default: { name: "MovieEditDialog", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieMetadataRefreshConfirmDialog.vue", () => ({
  default: {
    name: "MovieMetadataRefreshConfirmDialog",
    props: ["open", "movieTitle"],
    emits: ["update:open", "confirm"],
    template:
      '<div v-if="open" data-refresh-confirm><button type="button" @click="$emit(\'confirm\')">Confirm</button></div>',
  },
}))

vi.mock("@/components/jav-library/MovieRatingStars.vue", () => ({
  default: {
    name: "MovieRatingStars",
    props: ["modelValue"],
    emits: ["commit"],
    template: '<button data-rating-stars @click="$emit(\'commit\', 3.5)" />',
  },
}))

vi.mock("@/components/jav-library/ExpandableText.vue", () => ({
  default: { name: "ExpandableText", props: ["text"], template: "<div data-summary-text>{{ text }}</div>" },
}))

describe("DetailPanel", () => {
  it("translates title and synopsis in place and can restore their originals", async () => {
    useExperimentalAgent().setEnabled(true)
    runAction.mockImplementation(async (name: string) => ({ proposedText: name === "translate_title" ? "译文标题" : "译文简介" }))
    try {
      const wrapper = mount(DetailPanel, { props: { movie: makeMovie() } })
      await wrapper.get("[data-translate-title]").trigger("click")
      await wrapper.get("[data-translate-summary]").trigger("click")
      await flushPromises()

      expect(runAction).toHaveBeenCalledWith("translate_title", { movieId: "movie-1", body: "Movie 1", locale: "zh-CN" }, expect.any(AbortSignal))
      expect(runAction).toHaveBeenCalledWith("translate_summary", { movieId: "movie-1", body: "Summary", locale: "zh-CN" }, expect.any(AbortSignal))
      expect(wrapper.get("[data-detail-title]").text()).toBe("译文标题")
      expect(wrapper.get("[data-summary-text]").text()).toBe("译文简介")

      await wrapper.get("[data-translate-title]").trigger("click")
      await wrapper.get("[data-translate-summary]").trigger("click")
      expect(wrapper.get("[data-detail-title]").text()).toBe("Movie 1")
      expect(wrapper.get("[data-summary-text]").text()).toBe("Summary")
      wrapper.unmount()
    } finally {
      useExperimentalAgent().setEnabled(false)
      runAction.mockReset()
    }
  })

  it("does not show a previous movie's late translation and handles an unchanged result", async () => {
    useExperimentalAgent().setEnabled(true)
    let finish!: (value: { proposedText: string }) => void
    runAction.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
      .mockResolvedValueOnce({ noop: true })
    try {
      const wrapper = mount(DetailPanel, { props: { movie: makeMovie() } })
      await wrapper.get("[data-translate-title]").trigger("click")
      await wrapper.setProps({ movie: makeMovie({ id: "movie-2", title: "Movie 2" }) })
      finish({ proposedText: "Old translation" })
      await flushPromises()
      expect(wrapper.get("[data-detail-title]").text()).toBe("Movie 2")

      await wrapper.get("[data-translate-summary]").trigger("click")
      await flushPromises()
      expect(wrapper.get("[data-summary-text]").text()).toBe("Summary")
      expect(wrapper.text()).toContain("detailPanel.movieAiTranslateSummaryNoop")
      wrapper.unmount()
    } finally {
      useExperimentalAgent().setEnabled(false)
      runAction.mockReset()
    }
  })

  it("keeps the original text and shows an inline error when translation fails", async () => {
    useExperimentalAgent().setEnabled(true)
    runAction.mockRejectedValueOnce(new Error("Translation unavailable"))
    try {
      const wrapper = mount(DetailPanel, { props: { movie: makeMovie() } })
      await wrapper.get("[data-translate-title]").trigger("click")
      await flushPromises()
      expect(wrapper.get("[data-detail-title]").text()).toBe("Movie 1")
      expect(wrapper.text()).toContain("Translation unavailable")
      wrapper.unmount()
    } finally {
      useExperimentalAgent().setEnabled(false)
      runAction.mockReset()
    }
  })

  it("shows a named source link directly after the metadata provider", () => {
    const sourceUrl = "https://javdb.com/v/test"
    const wrapper = mount(DetailPanel, { props: { movie: makeMovie({ metadataProvider: "JavBus" }), readOnly: true, sourceUrl } })
    const link = wrapper.get("[data-source-page-link]")
    expect(link.text()).toBe("JAVDB")
    expect(link.attributes("href")).toBe(sourceUrl)
    expect(link.attributes("target")).toBe("_blank")
    expect(link.attributes("rel")).toBe("noopener noreferrer")
    expect(wrapper.get("[data-metadata-provider]").element.nextElementSibling?.querySelector("a")).toBe(link.element)
    expect(wrapper.text()).not.toContain(sourceUrl)
  })

  it.each([undefined, "", "javascript:alert(1)"])("hides unavailable or unsafe source %s", (sourceUrl) => {
    const wrapper = mount(DetailPanel, { props: { movie: makeMovie(), sourceUrl } })
    expect(wrapper.find("[data-source-page-link]").exists()).toBe(false)
  })

  it("keeps metadata browsing while hiding local movie operations in read-only mode", () => {
    const wrapper = mount(DetailPanel, {
      props: { movie: makeMovie({ rating: 0 }), readOnly: true },
    })
    expect(wrapper.text()).toContain('Movie 1')
    expect(wrapper.text()).toContain('detailPanel.cast')
    expect(wrapper.findComponent({ name: 'DropdownMenu' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'MovieEditDialog' }).exists()).toBe(false)
    expect(wrapper.find('[data-rating-stars]').exists()).toBe(false)
    expect(wrapper.find('[data-translate-title]').exists()).toBe(false)
    expect(wrapper.find('[data-translate-summary]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('detailPanel.myTags')
    expect(wrapper.text()).not.toContain('detailPanel.play')
    expect(wrapper.find('[aria-label="detailPanel.ariaRemoveNfoTag"]').exists()).toBe(false)
  })

  it("renders the cover code badge as an external JAVDB search link", () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ code: "JUR-681" }),
      },
    })

    const link = wrapper.findAll("a").find((candidate) => candidate.text() === "JUR-681")

    expect(link).toBeDefined()
    expect(link!.attributes("href")).toBe("https://javdb.com/search?q=JUR-681&f=all")
    expect(link!.attributes("target")).toBe("_blank")
    expect(link!.attributes("rel")).toBe("noopener noreferrer")
    expect(link!.attributes("data-variant")).toBe("outline")
    expect(link!.attributes("class")).not.toContain("hover:text-primary")
    expect(link!.attributes("class")).toContain("select-none")
  })

  it("shows the scraped metadata provider after year and resolution", () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ metadataProvider: "JavBus" }),
      },
    })

    const provider = wrapper.get("[data-metadata-provider]")
    expect(provider.text()).toContain("JavBus")
    expect(wrapper.text()).toContain("2026")
    expect(wrapper.text()).toContain("1080p")
  })

  it("hides the metadata provider when the movie has not been scraped", () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ metadataProvider: "  " }),
      },
    })

    expect(wrapper.find("[data-metadata-provider]").exists()).toBe(false)
  })

  it("places the compact rating card below tags in the info column", () => {
    // 评分卡跟在我的标签后面，固定 250px 宽，不再贴在封面列。
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ userTags: ["mine"] }),
      },
    })
    const info = wrapper.get("[data-detail-info-column]")
    const rating = info.get("[data-detail-rating-card]")
    expect(
      wrapper.get("[data-detail-media-column]").find("[data-detail-rating-card]").exists(),
    ).toBe(false)
    expect(rating.classes()).toEqual(expect.arrayContaining(["w-[250px]", "max-w-full"]))
    const infoHtml = info.html()
    expect(infoHtml.indexOf("detailPanel.myTags")).toBeLessThan(
      infoHtml.indexOf("data-detail-rating-card"),
    )
  })

  it("emits user rating updates from the rating stars", async () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie(),
      },
    })

    await wrapper.get("[data-rating-stars]").trigger("click")

    expect(wrapper.emitted("updateUserRating")).toEqual([
      [
        {
          movieId: "movie-1",
          value: 3.5,
        },
      ],
    ])
  })

  it("clears the local user rating override", async () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ userRating: 4 }),
      },
    })

    const clearButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("detailPanel.clearLocalRating"))

    expect(clearButton).toBeDefined()
    await clearButton!.trigger("click")

    expect(wrapper.emitted("updateUserRating")).toEqual([
      [
        {
          movieId: "movie-1",
          value: null,
        },
      ],
    ])
  })

  it("emits trash and permanent delete confirmations", async () => {
    const activeWrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie(),
      },
    })

    await activeWrapper.get('[data-delete-variant="trash"]').trigger("click")
    expect(activeWrapper.emitted("deleteMovie")).toEqual([["movie-1"]])

    const trashedWrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ trashedAt: "2026-04-02T00:00:00.000Z" }),
      },
    })

    await trashedWrapper.get('[data-delete-variant="permanent"]').trigger("click")
    expect(trashedWrapper.emitted("deleteMoviePermanently")).toEqual([["movie-1"]])
  })

  it("emits restore for trashed movies", async () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie({ trashedAt: "2026-04-02T00:00:00.000Z" }),
      },
    })

    const restoreButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("detailPanel.restoreMovie"))

    expect(restoreButton).toBeDefined()
    await restoreButton!.trigger("click")

    expect(wrapper.emitted("restoreMovie")).toEqual([["movie-1"]])
  })

  it("confirms single-movie metadata refresh before emitting the action", async () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie(),
      },
    })

    const refreshItem = wrapper
      .findAll("button")
      .find((button) => button.text().includes("detailPanel.refreshMetadata"))
    expect(refreshItem).toBeDefined()

    await refreshItem!.trigger("click")
    expect(wrapper.find("[data-refresh-confirm]").exists()).toBe(true)
    expect(wrapper.emitted("refreshMetadata")).toBeUndefined()

    await wrapper.get("[data-refresh-confirm] button").trigger("click")
    expect(wrapper.emitted("refreshMetadata")).toEqual([["movie-1"]])
  })

  it("adds a suggested user tag immediately when clicked", async () => {
    const wrapper = mount(DetailPanel, {
      props: {
        movie: makeMovie(),
        userTagSuggestions: ["alpha", "beta"],
      },
    })

    const addButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("common.add"))

    expect(addButton).toBeDefined()

    await addButton!.trigger("click")
    await wrapper.get('input[role="combobox"]').setValue("alp")

    const suggestion = wrapper
      .findAll('[role="option"]')
      .find((option) => option.text() === "alpha")

    expect(suggestion).toBeDefined()

    await suggestion!.trigger("mousedown")

    expect(wrapper.emitted("updateUserTags")).toEqual([
      [
        {
          movieId: "movie-1",
          tags: ["alpha"],
        },
      ],
    ])
  })
})

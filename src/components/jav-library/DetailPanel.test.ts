import { mount } from "@vue/test-utils"
import { ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import DetailPanel from "./DetailPanel.vue"
import type { Movie } from "@/domain/movie/types"

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
  CardContent: { name: "CardContent", template: "<div><slot /></div>" },
  CardDescription: { name: "CardDescription", template: "<div><slot /></div>" },
  CardTitle: { name: "CardTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuItem: { name: "DropdownMenuItem", template: "<button><slot /></button>" },
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

vi.mock("@/components/jav-library/MovieRatingStars.vue", () => ({
  default: {
    name: "MovieRatingStars",
    props: ["modelValue"],
    emits: ["commit"],
    template: '<button data-rating-stars @click="$emit(\'commit\', 3.5)" />',
  },
}))

vi.mock("@/components/jav-library/ExpandableText.vue", () => ({
  default: { name: "ExpandableText", template: "<div />" },
}))

describe("DetailPanel", () => {
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

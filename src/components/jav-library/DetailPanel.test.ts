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
  Badge: { name: "Badge", template: "<div><slot /></div>" },
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
  default: { name: "MovieDeleteConfirmDialog", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieEditDialog.vue", () => ({
  default: { name: "MovieEditDialog", template: "<div />" },
}))

vi.mock("@/components/jav-library/MovieRatingStars.vue", () => ({
  default: { name: "MovieRatingStars", template: "<div />" },
}))

vi.mock("@/components/jav-library/ExpandableText.vue", () => ({
  default: { name: "ExpandableText", template: "<div />" },
}))

describe("DetailPanel", () => {
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

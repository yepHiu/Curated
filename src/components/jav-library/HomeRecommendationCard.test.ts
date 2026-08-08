import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import HomeRecommendationCard from "./HomeRecommendationCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

const movie: Movie = {
  id: "m01",
  title: "Movie 1",
  code: "M01",
  studio: "Studio A",
  actors: ["Actor A"],
  tags: ["Tag A"],
  userTags: [],
  runtimeMinutes: 120,
  rating: 4.5,
  metadataRating: 4.5,
  isFavorite: false,
  addedAt: "2026-07-20T00:00:00Z",
  location: "D:/m01.mp4",
  resolution: "1080p",
  year: 2026,
  summary: "Summary",
  tone: "",
  coverClass: "",
}

const SlotStub = { template: "<div><slot /></div>" }
const ButtonStub = {
  inheritAttrs: false,
  props: ["disabled"],
  emits: ["click"],
  template: '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
}

function mountCard(withFeedback = false) {
  return mount(HomeRecommendationCard, {
    props: {
      entry: {
        movie,
        score: 45,
        reasons: [{ code: "shared_actor", entityType: "actor", entityValue: "Actor A" }],
      },
      feedback: withFeedback
        ? [{
            id: "feedback-1",
            action: "less",
            targetType: "actor",
            targetValue: "Actor A",
            sourceMovieId: "m01",
            createdAt: "2026-07-21T00:00:00Z",
            updatedAt: "2026-07-21T00:00:00Z",
          }]
        : [],
    },
    global: {
      stubs: {
        MovieCard: { props: ["movie"], template: "<div>{{ movie.title }}</div>" },
        Badge: SlotStub,
        Button: ButtonStub,
        DropdownMenu: SlotStub,
        DropdownMenuContent: SlotStub,
        DropdownMenuGroup: SlotStub,
        DropdownMenuItem: ButtonStub,
        DropdownMenuLabel: SlotStub,
        DropdownMenuSeparator: SlotStub,
        DropdownMenuSub: SlotStub,
        DropdownMenuSubContent: SlotStub,
        DropdownMenuSubTrigger: SlotStub,
        DropdownMenuTrigger: SlotStub,
        Clock3: true,
        MoreHorizontal: true,
        RotateCcw: true,
        Tags: true,
        ThumbsDown: true,
        UserRound: true,
      },
    },
  })
}

function buttonContaining(wrapper: ReturnType<typeof mountCard>, text: string) {
  const button = wrapper.findAll("button").find((candidate) => candidate.text().includes(text))
  if (!button) throw new Error(`button not found: ${text}`)
  return button
}

describe("HomeRecommendationCard", () => {
  it("keeps recommendation tags and the overflow action on one row", () => {
    const wrapper = mountCard()
    const tags = wrapper.get("[data-home-recommendation-tags]")
    const actions = wrapper.get("[data-home-recommendation-actions]")

    expect(tags.element.parentElement).toBe(actions.element.parentElement)
    expect(wrapper.get("[data-home-recommendation-meta]").classes()).toContain("items-center")
  })

  it("renders translated reason codes and emits explicit feedback targets", async () => {
    const wrapper = mountCard()
    expect(wrapper.text()).toContain("home.recommendationReason.shared_actor")

    await buttonContaining(wrapper, "home.recommendationNotInterested").trigger("click")
    expect(wrapper.emitted("submitFeedback")?.[0]).toEqual([{
      action: "not_interested",
      targetType: "movie",
      targetValue: "m01",
      sourceMovieId: "m01",
    }])
  })

  it("offers an immediate undo for active feedback", async () => {
    const wrapper = mountCard(true)
    await buttonContaining(wrapper, "home.recommendationUndo").trigger("click")
    expect(wrapper.emitted("deleteFeedback")?.[0]).toEqual(["feedback-1"])
  })
})

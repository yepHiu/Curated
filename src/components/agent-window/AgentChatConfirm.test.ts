import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { Movie } from "@/domain/movie/types"
import AgentChatConfirm from "./AgentChatConfirm.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) => {
      if (!values) return key
      return `${key}:${JSON.stringify(values)}`
    },
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button type='button'><slot /></button>" },
}))

function movie(id: string, title: string): Movie {
  return {
    id,
    title,
    code: id,
    studio: "Studio",
    actors: ["Ada"],
    tags: [],
    userTags: [],
    runtimeMinutes: 100,
    rating: 4,
    summary: "",
    isFavorite: false,
    addedAt: "2026-07-01T00:00:00Z",
    location: `D:/${id}.mp4`,
    resolution: "1080p",
    year: 2026,
    tone: "",
    coverClass: "",
  }
}

const libraryMovies = Array.from({ length: 6 }, (_, index) => movie(`m${index + 1}`, `Title ${index + 1}`))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: { value: libraryMovies },
    trashedMovies: { value: [] },
  }),
}))

const confirmEntry = {
  id: "c1",
  kind: "confirm" as const,
  name: "create_saved_view",
  confirmToken: "cfm_1",
  changes: [{
    path: "savedView.filters",
    before: "",
    after: { schemaVersion: 1, playState: "unwatched", actor: "Ada" },
  }],
  arguments: { name: "未看完", filters: { schemaVersion: 1 } },
  sessionId: "ses_1",
  status: "pending" as const,
}

describe("AgentChatConfirm", () => {
  it("describes a new bookmark in product language", () => {
    const wrapper = mount(AgentChatConfirm, {
      props: { entry: confirmEntry },
    })
    const text = wrapper.text()
    expect(wrapper.find("[data-agent-confirm-narrative]").exists()).toBe(true)
    expect(text).toContain("agentWindow.confirmTitleCreateView")
    expect(text).toContain("未看完")
    expect(text).toContain("library.savedViewPlay.unwatched")
    expect(text).toContain("Ada")
    expect(text).toContain("agentWindow.confirmApplyCreateView")
    expect(text).not.toContain("[object Object]")
    expect(text).not.toContain("agentWindow.confirmBefore")
    expect(text).not.toContain("agentWindow.confirmEmpty")
    const discard = wrapper.find("[data-agent-confirm-discard]")
    const apply = wrapper.find("[data-agent-confirm-apply]")
    expect(discard.attributes("variant")).toBe("outline")
    expect(apply.attributes("variant")).toBeUndefined()
    expect(discard.classes().join(" ")).not.toContain("rounded-full")
    expect(apply.classes().join(" ")).not.toContain("rounded-full")
    expect(
      wrapper.find("[data-agent-confirm-discard]").element.compareDocumentPosition(
        wrapper.find("[data-agent-confirm-apply]").element,
      ) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it("previews matching movies and folds the rest", async () => {
    const wrapper = mount(AgentChatConfirm, {
      props: { entry: confirmEntry },
    })
    expect(wrapper.find("[data-agent-confirm-movies]").text()).toContain("6")
    expect(wrapper.findAll("[data-agent-movie-card]")).toHaveLength(4)
    expect(wrapper.find("[data-agent-movie-card=\"m1\"]").exists()).toBe(true)
    expect(wrapper.find("[data-agent-movie-card=\"m5\"]").exists()).toBe(false)
    expect(wrapper.find("[data-agent-confirm-show-more]").text()).toContain("2")

    await wrapper.find("[data-agent-confirm-show-more]").trigger("click")
    expect(wrapper.findAll("[data-agent-movie-card]")).toHaveLength(6)
    expect(wrapper.find("[data-agent-confirm-show-less]").exists()).toBe(true)
  })
})

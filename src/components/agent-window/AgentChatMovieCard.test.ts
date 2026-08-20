import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import AgentChatMovieCard from "./AgentChatMovieCard.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe("AgentChatMovieCard", () => {
  it("emits open with the movie id", async () => {
    const wrapper = mount(AgentChatMovieCard, {
      props: {
        movie: { movieId: "m1", title: "Hello", code: "ABC-123", reason: "轻松" },
      },
    })
    await wrapper.get('[data-agent-movie-card="m1"]').trigger("click")
    expect(wrapper.emitted("open")?.[0]).toEqual(["m1"])
  })
})

import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

import AgentChatComposer from "./AgentChatComposer.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: {
      value: [
        {
          id: "m1",
          title: "Hello",
          code: "ABC-001",
          actors: ["Ada"],
          tags: ["轻松"],
          userTags: [],
        },
        {
          id: "m2",
          title: "World",
          code: "ABC-002",
          actors: ["Bea"],
          tags: ["剧情"],
          userTags: [],
        },
        {
          id: "m3",
          title: "Extra",
          code: "ABC-003",
          actors: ["Cara"],
          tags: ["纪录"],
          userTags: [],
        },
      ],
    },
    listActors: async () => ({
      actors: [{ name: "Ada", movieCount: 1 }, { name: "Bea", movieCount: 1 }, { name: "Cara", movieCount: 1 }],
      total: 3,
    }),
  }),
}))

vi.mock("@/services/comic-library-service", () => ({
  useComicLibraryService: () => ({
    comicLibraryEnabled: { value: false },
    comics: { value: [] },
  }),
}))

vi.mock("@/services/photo-library-service", () => ({
  usePhotoLibraryService: () => ({
    photoLibraryEnabled: { value: false },
    photos: { value: [] },
  }),
}))

describe("AgentChatComposer mentions", () => {
  it("opens the picker when the draft contains @", async () => {
    const wrapper = mount(AgentChatComposer, {
      props: {
        streaming: false,
        modelValue: "",
        mentions: [],
      },
    })
    const input = wrapper.find("[data-agent-window-input]")
    await input.setValue("@")
    const el = input.element as HTMLTextAreaElement
    el.setSelectionRange(1, 1)
    await input.trigger("input")
    await flushPromises()

    expect(wrapper.find("[data-agent-mention-open]").attributes("data-agent-mention-open")).toBe("true")
    expect(wrapper.find("[data-agent-mention-picker]").exists()).toBe(true)
    expect(wrapper.find('[data-agent-mention-item="movie:m1"]').exists()).toBe(true)
    expect(wrapper.find('[data-agent-mention-item="movie:m2"]').exists()).toBe(true)
    expect(wrapper.find('[data-agent-mention-item="movie:m3"]').exists()).toBe(false)
    expect(wrapper.find("[data-agent-window-composer]").classes()).toContain("bg-muted/40")
    expect(wrapper.find("[data-agent-mention-picker]").classes()).toContain("bg-popover")
  })
})

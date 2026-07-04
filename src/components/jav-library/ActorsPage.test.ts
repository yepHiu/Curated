import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

const routeState = vi.hoisted(() => ({
  query: {
    actorTag: "Hidden Actor Tag",
  } as Record<string, unknown>,
}))

const listActors = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.tag != null ? `${key}:${params.tag}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    replace: vi.fn(),
  }),
}))

vi.mock("lucide-vue-next", () => ({
  X: { name: "X", template: "<span />" },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    movies: { value: [] },
    listActors,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: '<button data-button @click="$emit(\'click\', $event)"><slot /></button>',
  },
}))

vi.mock("@/components/jav-library/ActorLibraryCard.vue", () => ({
  default: {
    props: ["actor"],
    template: '<article data-actor-card>{{ actor.name }}</article>',
  },
}))

describe("ActorsPage", () => {
  it("does not expose actor-tag filtering UI or pass actorTag to the list query", async () => {
    listActors.mockResolvedValueOnce({
      total: 1,
      actors: [
        {
          name: "Alpha Star",
          movieCount: 12,
          userTags: ["Hidden Actor Tag"],
        },
      ],
    })
    const { default: ActorsPage } = await import("./ActorsPage.vue")

    const wrapper = mount(ActorsPage)
    await flushPromises()

    expect(listActors).toHaveBeenCalledWith(
      expect.not.objectContaining({ actorTag: "Hidden Actor Tag" }),
    )
    expect(wrapper.text()).toContain("Alpha Star")
    expect(wrapper.text()).not.toContain("actors.filteredByTag")
    expect(wrapper.text()).not.toContain("Hidden Actor Tag")
    expect(wrapper.text()).not.toContain("actors.clearTagFilter")
  })
})

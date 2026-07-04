import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ActorListItemDTO } from "@/api/types"

const routerPush = vi.fn()

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n != null ? `${key}:${params.n}` : key,
  }),
}))

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: routerPush,
  }),
}))

vi.mock("@/lib/library-query", () => ({
  mergeLibraryQuery: vi.fn((_query, patch) => patch),
}))

vi.mock("@/components/ui/avatar", () => ({
  Avatar: {
    props: ["class"],
    template: '<div data-avatar :class="$props.class"><slot /></div>',
  },
  AvatarFallback: {
    props: ["class"],
    template: '<div data-avatar-fallback :class="$props.class"><slot /></div>',
  },
  AvatarImage: {
    props: ["src", "alt", "class"],
    template: '<img data-avatar-image :src="src" :alt="alt" :class="$props.class" />',
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: {
    props: ["class"],
    template: '<div data-card :class="$props.class"><slot /></div>',
  },
  CardContent: {
    props: ["class"],
    template: '<div data-card-content :class="$props.class"><slot /></div>',
  },
  CardDescription: {
    props: ["class"],
    template: '<div data-card-description :class="$props.class"><slot /></div>',
  },
  CardHeader: {
    props: ["class"],
    template: '<div data-card-header :class="$props.class"><slot /></div>',
  },
  CardTitle: {
    props: ["class"],
    template: '<div data-card-title :class="$props.class"><slot /></div>',
  },
}))

function actor(overrides: Partial<ActorListItemDTO> = {}): ActorListItemDTO {
  return {
    name: "Alpha Star",
    avatarUrl: "https://example.com/alpha.jpg",
    movieCount: 12,
    userTags: ["Hidden Actor Tag"],
    ...overrides,
  }
}

describe("ActorLibraryCard", () => {
  it("hides actor user tags and tag editing controls", async () => {
    const { default: ActorLibraryCard } = await import("./ActorLibraryCard.vue")

    const wrapper = mount(ActorLibraryCard, {
      props: {
        actor: actor(),
      },
    })

    expect(wrapper.text()).toContain("Alpha Star")
    expect(wrapper.text()).toContain("actors.movieCount:12")
    expect(wrapper.text()).not.toContain("Hidden Actor Tag")
    expect(wrapper.text()).not.toContain("common.add")
    expect(wrapper.findAll("[data-badge]")).toHaveLength(0)
  })

  it("uses a larger portrait-forward avatar treatment", async () => {
    const { default: ActorLibraryCard } = await import("./ActorLibraryCard.vue")

    const wrapper = mount(ActorLibraryCard, {
      props: {
        actor: actor(),
      },
    })

    const avatar = wrapper.get("[data-avatar]")
    expect(avatar.classes()).toContain("size-24")
    expect(avatar.classes()).toContain("rounded-2xl")
    expect(wrapper.get("[data-avatar-image]").attributes("alt")).toBe("Alpha Star")
  })
})

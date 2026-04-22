import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

const getActorProfile = vi.fn()
const patchActorExternalLinks = vi.fn()

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

vi.mock("@vueuse/core", () => ({
  useFocusWithin: () => ({ focused: ref(true) }),
  useResizeObserver: vi.fn(),
  onClickOutside: vi.fn(),
}))

vi.mock("lucide-vue-next", () => ({
  Plus: { name: "Plus", template: "<span />" },
  X: { name: "X", template: "<span />" },
}))

vi.mock("@/api/endpoints", () => ({
  api: {
    getActorProfile,
    patchActorExternalLinks,
    scrapeActorProfile: vi.fn(),
    getTaskStatus: vi.fn(),
  },
}))

vi.mock("@/api/http-client", () => ({
  HttpClientError: class HttpClientError extends Error {
    status: number
    apiError?: { code: string; message: string; retryable: boolean }

    constructor(
      status: number,
      apiError?: { code: string; message: string; retryable: boolean },
    ) {
      super(apiError?.message ?? `HTTP ${status}`)
      this.status = status
      this.apiError = apiError
    }
  },
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => ({
    patchActorUserTags: vi.fn(),
  }),
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

vi.mock("@/composables/use-user-tag-suggest-keyboard", () => ({
  useUserTagSuggestKeyboard: () => ({
    highlightIndex: ref(-1),
    onTagSuggestKeydown: vi.fn(),
  }),
}))

vi.mock("@/lib/actors-route-query", () => ({
  mergeActorsQuery: vi.fn(() => ({})),
}))

vi.mock("@/lib/user-tag-suggestions", () => ({
  filterUserTagSuggestions: vi.fn(() => []),
}))

vi.mock("@/components/ui/avatar", () => ({
  Avatar: { template: "<div><slot /></div>" },
  AvatarFallback: { template: "<div><slot /></div>" },
  AvatarImage: { template: "<img />" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    emits: ["click"],
    template: "<button @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/card", () => ({
  Card: { template: "<div><slot /></div>" },
  CardContent: { template: "<div><slot /></div>" },
  CardHeader: { template: "<div><slot /></div>" },
  CardTitle: { template: "<div><slot /></div>" },
}))

async function mountComponent() {
  vi.resetModules()
  vi.stubEnv("VITE_USE_WEB_API", "true")
  const mod = await import("./ActorProfileCard.vue")
  return mount(mod.default, {
    props: {
      actorName: "Alpha Star",
      userTagSuggestions: [],
    },
    global: {
      stubs: {
        Teleport: false,
      },
    },
  })
}

describe("ActorProfileCard", () => {
  beforeEach(() => {
    getActorProfile.mockReset()
    patchActorExternalLinks.mockReset()
  })

  it("renders one saved external link and replaces it with a new valid url", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: ["https://example.com/a"],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })
    patchActorExternalLinks.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: ["https://example.com/b"],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })

    const wrapper = await mountComponent()
    await flushPromises()

    expect(wrapper.text()).toContain("https://example.com/a")

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("https://example.com/b")
    await wrapper.get("[data-actor-external-link-save]").trigger("click")
    await flushPromises()

    expect(patchActorExternalLinks).toHaveBeenCalledWith("Alpha Star", ["https://example.com/b"])
    expect(wrapper.text()).not.toContain("https://example.com/a")
    expect(wrapper.text()).toContain("https://example.com/b")
    expect(wrapper.find("[data-actor-external-link-remove]").exists()).toBe(false)
  })

  it("shows an inline error for invalid external links", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: [],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })

    const wrapper = await mountComponent()
    await flushPromises()

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("ftp://example.com")
    await wrapper.get("[data-actor-external-link-save]").trigger("click")

    expect(wrapper.text()).toContain("library.actorExternalLinksInvalid")
    expect(patchActorExternalLinks).not.toHaveBeenCalled()
  })

  it("closes the external link editor from the inline cancel control", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: [],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })

    const wrapper = await mountComponent()
    await flushPromises()

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    expect(wrapper.find("[data-actor-external-link-input]").exists()).toBe(true)

    await wrapper.get("[data-actor-external-link-cancel]").trigger("click")
    expect(wrapper.find("[data-actor-external-link-input]").exists()).toBe(false)
  })

  it("shows a friendly message instead of raw HTTP 404 when saving fails with not found", async () => {
    const { HttpClientError } = await import("@/api/http-client")

    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: [],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })
    patchActorExternalLinks.mockRejectedValue(new HttpClientError(404))

    const wrapper = await mountComponent()
    await flushPromises()

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("https://example.com/b")
    await wrapper.get("[data-actor-external-link-save]").trigger("click")
    await flushPromises()

    expect(wrapper.text()).toContain("library.actorExternalLinksUnsupported")
    expect(wrapper.text()).not.toContain("HTTP 404")
  })

  it("shows actor not found only for structured actor-not-found responses", async () => {
    const { HttpClientError } = await import("@/api/http-client")

    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: [],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })
    patchActorExternalLinks.mockRejectedValue(
      new HttpClientError(404, {
        code: "COMMON_NOT_FOUND",
        message: "actor not found",
        retryable: false,
      }),
    )

    const wrapper = await mountComponent()
    await flushPromises()

    await wrapper.get("[data-actor-external-link-add]").trigger("click")
    await wrapper.get("[data-actor-external-link-input]").setValue("https://example.com/b")
    await wrapper.get("[data-actor-external-link-save]").trigger("click")
    await flushPromises()

    expect(wrapper.text()).toContain("library.actorProfileNotFound")
  })

  it("renders the external links heading only once when a link exists", async () => {
    getActorProfile.mockResolvedValue({
      name: "Alpha Star",
      externalLinks: ["https://example.com/a"],
      userTags: [],
      summary: "Bio",
      avatarUrl: "https://example.com/avatar.jpg",
    })

    const wrapper = await mountComponent()
    await flushPromises()

    const text = wrapper.text()
    const count = text.split("library.actorExternalLinks").length - 1

    expect(count).toBe(1)
    expect(text).toContain("https://example.com/a")
  })
})

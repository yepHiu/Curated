import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import WishlistDetailView from "./WishlistDetailView.vue"

const service = vi.hoisted(() => ({ get: vi.fn(), assetUrl: (path: string) => `/protected${path}` }))
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ wishlist: service }) }))
vi.mock("vue-router", () => ({ useRoute: () => ({ params: { id: "wish-1" } }), useRouter: () => ({ push: vi.fn() }) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("wishlist shared movie details", () => {
  it("passes wishlist metadata and protected images to the existing detail page", async () => {
    service.get.mockResolvedValue({
      id: "wish-1", code: "TEST-001", createdAt: "2026-09-23", enrichmentState: "ready",
      metadata: { title: "Wishlist title", summary: "Summary", actors: ["Actor"], tags: ["Tag"], studio: "Studio", releaseDate: "2026-01-02", runtimeMinutes: 90, provider: "Provider" },
      assets: [
        { role: "preview_image", url: "/preview", thumbnailUrl: "/preview-thumb" },
        { role: "cover", url: "/cover", thumbnailUrl: "/cover-thumb" },
      ],
    })
    const wrapper = mount(WishlistDetailView, {
      global: { stubs: { DetailPage: { name: "DetailPage", props: { movie: Object, readOnly: Boolean }, template: '<div data-shared-detail />' } } },
    })
    await flushPromises()
    const detail = wrapper.getComponent({ name: "DetailPage" })
    expect(service.get).toHaveBeenCalledWith("wish-1")
    expect(detail.props("readOnly")).toBe(true)
    expect(detail.props("movie")).toMatchObject({
      id: "wish-1", title: "Wishlist title", actors: ["Actor"], tags: ["Tag"],
      coverUrl: "/protected/cover", thumbUrl: "/protected/cover-thumb",
      previewImages: ["/protected/preview"], location: "", year: 2026,
    })
    expect(wrapper.find("input").exists()).toBe(false)
    wrapper.unmount()
  })
})

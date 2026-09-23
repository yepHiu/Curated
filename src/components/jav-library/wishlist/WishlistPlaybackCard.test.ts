import { flushPromises, mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import WishlistPlaybackCard from "./WishlistPlaybackCard.vue"

const playback = vi.hoisted(() => vi.fn())
vi.mock("@/services/library-service", () => ({ useLibraryService: () => ({ wishlist: { playback, integrationsAvailable: true } }) }))
vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("wishlist playback card", () => {
  it("checks only on demand and links only confirmed matches", async () => {
    playback.mockResolvedValue({ results: [
      { site: "Jable", status: "available", url: "https://jable.tv/videos/ssis-001/" },
      { site: "MISSAV", status: "unknown" },
    ] })
    const wrapper = mount(WishlistPlaybackCard, { props: { itemId: "wish-1", code: "SSIS-001" } })
    expect(playback).not.toHaveBeenCalled()
    await wrapper.get("button").trigger("click")
    await flushPromises()
    expect(playback).toHaveBeenCalledWith("wish-1")
    expect(wrapper.findAll("a")).toHaveLength(1)
    expect(wrapper.get("a").attributes("href")).toBe("https://jable.tv/videos/ssis-001/")
    expect(wrapper.text()).toContain("wishlist.playback.unknown")
    wrapper.unmount()
  })
})

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
      { site: "MISSAV", status: "blocked", url: "https://missav.ws/ssis-001/" },
    ] })
    const wrapper = mount(WishlistPlaybackCard, { props: { itemId: "wish-1" } })
    expect(playback).not.toHaveBeenCalled()
    expect(wrapper.get("button").attributes("aria-label")).toBe("wishlist.playback.check")
    expect(wrapper.get("button").text()).toBe("")
    await wrapper.get("button").trigger("click")
    await flushPromises()
    expect(playback).toHaveBeenCalledWith("wish-1")
    expect(wrapper.get("button").attributes("aria-label")).toBe("wishlist.playback.recheck")
    expect(wrapper.get("button").text()).toBe("")
    expect(wrapper.findAll("a")).toHaveLength(2)
    expect(wrapper.findAll("a")[0].attributes("href")).toBe("https://jable.tv/videos/ssis-001/")
    expect(wrapper.findAll("a")[1].attributes("href")).toBe("https://missav.ws/ssis-001/")
    expect(wrapper.text()).toContain("wishlist.playback.blocked")
    expect(wrapper.findAll("a")[0].text()).toContain("wishlist.playback.watch")
    expect(wrapper.findAll("a")[1].text()).toBe("")
    expect(wrapper.findAll("a")[1].attributes("aria-label")).toBe("wishlist.playback.verify")
    wrapper.unmount()
  })
})

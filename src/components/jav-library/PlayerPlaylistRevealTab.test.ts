import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import PlayerPlaylistRevealTab from "./PlayerPlaylistRevealTab.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("PlayerPlaylistRevealTab", () => {
  it("keeps the tab hidden until the hot zone is hovered", async () => {
    const wrapper = mount(PlayerPlaylistRevealTab, {
      props: { visible: false },
    })

    expect(wrapper.find("[data-player-playlist-tab]").exists()).toBe(false)

    await wrapper.get("[data-player-playlist-hotzone]").trigger("mouseenter")
    expect(wrapper.emitted("enter")).toHaveLength(1)

    await wrapper.setProps({ visible: true })
    expect(wrapper.get("[data-player-playlist-tab]").exists()).toBe(true)
    expect(wrapper.get("[data-player-playlist-tab]").classes()).toEqual(
      expect.arrayContaining(["h-16", "w-6"]),
    )

    await wrapper.get("[data-player-playlist-tab]").trigger("click")
    expect(wrapper.emitted("open")).toHaveLength(1)
  })

  it("keeps a larger close handle visible while the drawer is open", async () => {
    const wrapper = mount(PlayerPlaylistRevealTab, {
      props: { expanded: true, visible: true },
    })

    expect(wrapper.find("[data-player-playlist-hotzone]").exists()).toBe(false)
    const tab = wrapper.get("[data-player-playlist-tab]")
    expect(tab.classes()).toEqual(expect.arrayContaining(["h-16", "w-6"]))
    await tab.trigger("click")
    expect(wrapper.emitted("close")).toHaveLength(1)
  })
})

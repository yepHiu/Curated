import { mount } from "@vue/test-utils"
import { nextTick, ref } from "vue"
import { describe, expect, it } from "vitest"
import PlayerButton from "./PlayerButton.vue"

describe("PlayerButton", () => {
  it("keeps the hint inside the fullscreen surface and updates it with the action", async () => {
    const surface = document.createElement("div")
    document.body.append(surface)
    const wrapper = mount(PlayerButton, {
      props: { ariaLabel: "Play" },
      attrs: { "data-player-capture": "", disabled: false },
      global: { provide: { playerTooltipPortal: ref(surface) } },
      attachTo: surface,
    })
    try {
      await wrapper.get("button").trigger("focus")
      await nextTick()
      expect(surface.querySelector('[role="tooltip"]')?.textContent).toContain("Play")
      await wrapper.setProps({ ariaLabel: "Pause" })
      expect(surface.querySelector('[role="tooltip"]')?.textContent).toContain("Pause")
      expect(wrapper.get("button").attributes()).toHaveProperty("data-player-capture")
      expect(wrapper.get("button").attributes("title")).toBeUndefined()
      await wrapper.get("button").trigger("click")
      expect(wrapper.emitted("click")).toHaveLength(1)
    } finally {
      wrapper.unmount()
      surface.remove()
    }
  })
})

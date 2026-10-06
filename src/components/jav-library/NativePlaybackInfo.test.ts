import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import NativePlaybackInfo from "./NativePlaybackInfo.vue"

describe("native playback info interactions", () => {
  it("supports copying, saving, pending feedback and Escape without playback key bubbling", async () => {
    const wrapper = mount(NativePlaybackInfo, { props: { lang: "zh-CN", busy: false, feedback: "" }, attachTo: document.body })
    try {
      expect(document.activeElement).toBe(wrapper.element)
      const buttons = wrapper.findAll("button")
      await buttons[0]!.trigger("click")
      await buttons[1]!.trigger("click")
      expect(wrapper.emitted("copy")).toHaveLength(1)
      expect(wrapper.emitted("save")).toHaveLength(1)
      await wrapper.setProps({ busy: true, feedback: "播放信息已复制" })
      expect(buttons[1]!.attributes("disabled")).toBeDefined()
      expect(wrapper.get('[role="status"]').text()).toBe("播放信息已复制")
      await wrapper.trigger("keydown", { key: "Escape" })
      expect(wrapper.emitted("close")).toHaveLength(1)
    } finally { wrapper.unmount() }
  })
})

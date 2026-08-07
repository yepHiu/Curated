import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFramesBatchActionBar from "./CuratedFramesBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("CuratedFramesBatchActionBar", () => {
  it("renders raw and watermarked export buttons", async () => {
    const wrapper = mount(CuratedFramesBatchActionBar, {
      props: {
        selectedCount: 2,
        exportBusy: false,
        deleteBusy: false,
        showSelectVisible: true,
        useWebApi: true,
        exportError: "",
      },
    })

    const buttons = wrapper.findAll("button")
    const exportButtons = buttons.filter((button) => button.text().includes("curated.export"))

    expect(exportButtons).toHaveLength(2)
    await exportButtons[1]!.trigger("click")
    expect(wrapper.emitted("exportWatermarked")).toHaveLength(1)
    expect(wrapper.text()).not.toContain("curated.exportWebp")
    expect(wrapper.text()).not.toContain("curated.exportPng")
  })

  it("keeps raw export disabled without Web API while allowing watermarked export", async () => {
    const wrapper = mount(CuratedFramesBatchActionBar, {
      props: {
        selectedCount: 2,
        exportBusy: false,
        deleteBusy: false,
        showSelectVisible: true,
        useWebApi: false,
        exportError: "",
      },
    })

    const exportButtons = wrapper.findAll("button").filter((button) => button.text().includes("curated.export"))
    expect(exportButtons[0]?.element.disabled).toBe(true)
    expect(exportButtons[1]?.element.disabled).toBe(false)
    await exportButtons[1]!.trigger("click")
    expect(wrapper.emitted("exportWatermarked")).toHaveLength(1)
  })
})

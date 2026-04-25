import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFramesBatchActionBar from "./CuratedFramesBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("CuratedFramesBatchActionBar", () => {
  it("renders a single export button", () => {
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

    expect(exportButtons).toHaveLength(1)
    expect(wrapper.text()).not.toContain("curated.exportWebp")
    expect(wrapper.text()).not.toContain("curated.exportPng")
  })
})

import { shallowMount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFramesBatchActionBar from "./CuratedFramesBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

describe("CuratedFramesBatchActionBar layout", () => {
  it("aligns flush with the content bottom instead of preserving rounded shell corners", () => {
    const wrapper = shallowMount(CuratedFramesBatchActionBar, {
      props: {
        selectedCount: 2,
        exportBusy: false,
        deleteBusy: false,
        showSelectVisible: true,
        useWebApi: true,
        exportError: "",
      },
    })

    const toolbar = wrapper.get('[role="toolbar"]')
    expect(toolbar.classes().join(" ")).not.toContain("rounded-b-[calc")
  })
})

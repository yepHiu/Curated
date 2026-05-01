import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameLibraryToolbar from "./CuratedFrameLibraryToolbar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("@/components/ui/tabs", () => ({
  TabsList: {
    name: "TabsList",
    template: "<div data-tabs-list><slot /></div>",
  },
  TabsTrigger: {
    name: "TabsTrigger",
    props: ["value"],
    template: "<button type='button' data-tabs-trigger><slot /></button>",
  },
}))

describe("CuratedFrameLibraryToolbar", () => {
  it("renders tab triggers, page summary, and enter batch action", async () => {
    const wrapper = mount(CuratedFrameLibraryToolbar, {
      props: {
        shownCount: 12,
        totalRows: 34,
        batchMode: false,
        showSelectVisible: true,
        selectVisibleDisabled: false,
      },
    })

    expect(wrapper.findAll("[data-tabs-trigger]")).toHaveLength(3)
    expect(wrapper.text()).toContain("curated.tabTimeline")
    expect(wrapper.text()).toContain("curated.pageSummary")
    expect(wrapper.text()).toContain('"shown":12')
    expect(wrapper.text()).toContain('"total":34')

    await wrapper.get("button:not([data-tabs-trigger])").trigger("click")

    expect(wrapper.emitted("enterBatchMode")).toEqual([[]])
  })

  it("renders batch actions and forwards toolbar events", async () => {
    const wrapper = mount(CuratedFrameLibraryToolbar, {
      props: {
        shownCount: 0,
        totalRows: 0,
        batchMode: true,
        showSelectVisible: true,
        selectVisibleDisabled: false,
      },
    })
    const actionButtons = wrapper
      .findAll("button")
      .filter((button) => button.attributes("data-tabs-trigger") === undefined)

    expect(actionButtons).toHaveLength(2)
    expect(actionButtons[0]!.attributes("disabled")).toBeUndefined()

    await actionButtons[0]!.trigger("click")
    await actionButtons[1]!.trigger("click")

    expect(wrapper.emitted("selectVisible")).toEqual([[]])
    expect(wrapper.emitted("exitBatchMode")).toEqual([[]])
  })
})

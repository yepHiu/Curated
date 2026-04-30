import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsLibraryPathToolbar from "./SettingsLibraryPathToolbar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  CheckSquare: { name: "CheckSquare", template: "<span />" },
  ListChecks: { name: "ListChecks", template: "<span />" },
  Sparkles: { name: "Sparkles", template: "<span />" },
  X: { name: "X", template: "<span />" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

const baseProps = {
  batchMode: false,
  libraryPathsCount: 2,
  hasMetadataPathSelection: false,
  metadataRefreshBusy: false,
}

describe("SettingsLibraryPathToolbar", () => {
  it("renders library path heading and enters batch mode", async () => {
    const wrapper = mount(SettingsLibraryPathToolbar, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.libraryPaths")
    expect(wrapper.text()).toContain("library.batchManage")

    await wrapper.get("[data-library-paths-enter-batch]").trigger("click")

    expect(wrapper.emitted("enterBatchMode")).toHaveLength(1)
  })

  it("renders batch actions and emits toolbar commands", async () => {
    const wrapper = mount(SettingsLibraryPathToolbar, {
      props: {
        ...baseProps,
        batchMode: true,
        hasMetadataPathSelection: true,
      },
    })

    await wrapper.get("[data-library-paths-select-all]").trigger("click")
    await wrapper.get("[data-library-paths-clear-selection]").trigger("click")
    await wrapper.get("[data-library-paths-refresh-metadata]").trigger("click")
    await wrapper.get("[data-library-paths-exit-batch]").trigger("click")

    expect(wrapper.emitted("selectAll")).toHaveLength(1)
    expect(wrapper.emitted("clearSelection")).toHaveLength(1)
    expect(wrapper.emitted("refreshMetadata")).toHaveLength(1)
    expect(wrapper.emitted("exitBatchMode")).toHaveLength(1)
  })

  it("disables unavailable batch actions", () => {
    const wrapper = mount(SettingsLibraryPathToolbar, {
      props: {
        ...baseProps,
        batchMode: true,
        libraryPathsCount: 0,
        hasMetadataPathSelection: false,
        metadataRefreshBusy: true,
      },
    })

    expect(wrapper.get("[data-library-paths-select-all]").attributes("disabled")).toBeDefined()
    expect(wrapper.get("[data-library-paths-refresh-metadata]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("settings.submitting")
  })
})

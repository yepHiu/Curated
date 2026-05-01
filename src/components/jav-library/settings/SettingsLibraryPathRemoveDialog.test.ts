import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsLibraryPathRemoveDialog from "./SettingsLibraryPathRemoveDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.title ? `${key}:${params.title}` : key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: "<button :disabled=\"$attrs.disabled\" @click=\"$emit('click', $event)\"><slot /></button>",
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    emits: ["update:open"],
    template: "<div class=\"dialog-stub\"><slot /></div>",
  },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<div><slot /></div>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<div><slot /></div>" },
  DialogTitle: { name: "DialogTitle", template: "<h2><slot /></h2>" },
}))

const pendingPath = {
  id: "library-a",
  title: "Primary archive",
  path: "D:/Media/JAV/Main",
}

describe("SettingsLibraryPathRemoveDialog", () => {
  it("renders pending path details and emits confirm", async () => {
    const wrapper = mount(SettingsLibraryPathRemoveDialog, {
      props: {
        open: true,
        pending: pendingPath,
        busy: false,
        contentClass: "dialog-content",
      },
    })

    expect(wrapper.text()).toContain("settings.removePathConfirmTitle")
    expect(wrapper.text()).toContain("settings.removePathConfirmDesc:Primary archive")
    expect(wrapper.text()).toContain("D:/Media/JAV/Main")

    await wrapper.get("[data-remove-path-confirm]").trigger("click")

    expect(wrapper.emitted("confirm")).toHaveLength(1)
  })

  it("disables confirm while busy or without pending path", () => {
    const busyWrapper = mount(SettingsLibraryPathRemoveDialog, {
      props: {
        open: true,
        pending: pendingPath,
        busy: true,
        contentClass: "dialog-content",
      },
    })
    const emptyWrapper = mount(SettingsLibraryPathRemoveDialog, {
      props: {
        open: true,
        pending: null,
        busy: false,
        contentClass: "dialog-content",
      },
    })

    expect(busyWrapper.get("[data-remove-path-confirm]").attributes("disabled")).toBeDefined()
    expect(busyWrapper.text()).toContain("settings.removePathConfirmWorking")
    expect(emptyWrapper.get("[data-remove-path-confirm]").attributes("disabled")).toBeDefined()
  })

  it("forwards dialog open updates", () => {
    const wrapper = mount(SettingsLibraryPathRemoveDialog, {
      props: {
        open: true,
        pending: pendingPath,
        busy: false,
        contentClass: "dialog-content",
      },
    })

    wrapper.getComponent({ name: "Dialog" }).vm.$emit("update:open", false)

    expect(wrapper.emitted("update:open")).toEqual([[false]])
  })
})

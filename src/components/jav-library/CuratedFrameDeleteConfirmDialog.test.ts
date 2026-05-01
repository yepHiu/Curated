import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import CuratedFrameDeleteConfirmDialog from "./CuratedFrameDeleteConfirmDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    emits: ["update:open"],
    template: "<div data-dialog><slot /></div>",
  },
  DialogClose: {
    name: "DialogClose",
    props: ["asChild"],
    template: "<span data-dialog-close><slot /></span>",
  },
  DialogContent: {
    name: "DialogContent",
    template: "<section data-dialog-content><slot /></section>",
  },
  DialogDescription: {
    name: "DialogDescription",
    template: "<p data-dialog-description><slot /></p>",
  },
  DialogFooter: {
    name: "DialogFooter",
    template: "<footer data-dialog-footer><slot /></footer>",
  },
  DialogHeader: {
    name: "DialogHeader",
    template: "<header data-dialog-header><slot /></header>",
  },
  DialogTitle: {
    name: "DialogTitle",
    template: "<h2 data-dialog-title><slot /></h2>",
  },
}))

describe("CuratedFrameDeleteConfirmDialog", () => {
  it("renders delete confirmation content and forwards confirm", async () => {
    const wrapper = mount(CuratedFrameDeleteConfirmDialog, {
      props: {
        open: true,
        label: "ABC-001",
        error: "Delete failed",
        busy: false,
      },
    })
    const buttons = wrapper.findAll("button")

    expect(wrapper.text()).toContain("curated.deleteCard")
    expect(wrapper.text()).toContain("curated.deleteConfirm")
    expect(wrapper.text()).toContain("ABC-001")
    expect(wrapper.text()).toContain("Delete failed")

    await buttons[1]!.trigger("click")

    expect(wrapper.emitted("confirm")).toEqual([[]])
  })

  it("disables actions and shows working text while busy", () => {
    const wrapper = mount(CuratedFrameDeleteConfirmDialog, {
      props: {
        open: true,
        label: "ABC-001",
        error: "",
        busy: true,
      },
    })
    const buttons = wrapper.findAll("button")

    expect(buttons[0]!.attributes("disabled")).toBeDefined()
    expect(buttons[1]!.attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("curated.deleteWorking")
  })
})

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import ComicDeleteConfirmDialog from "./ComicDeleteConfirmDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    emits: ["click"],
    template: '<button v-bind="$attrs" @click="$emit(\'click\', $event)"><slot /></button>',
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    emits: ["update:open"],
    template: '<div v-if="open"><slot /></div>',
  },
  DialogClose: {
    name: "DialogClose",
    template: "<span><slot /></span>",
  },
  DialogContent: {
    name: "DialogContent",
    template: "<section><slot /></section>",
  },
  DialogDescription: {
    name: "DialogDescription",
    template: "<p><slot /></p>",
  },
  DialogFooter: {
    name: "DialogFooter",
    template: "<footer><slot /></footer>",
  },
  DialogHeader: {
    name: "DialogHeader",
    template: "<header><slot /></header>",
  },
  DialogTitle: {
    name: "DialogTitle",
    template: "<h2><slot /></h2>",
  },
}))

describe("ComicDeleteConfirmDialog", () => {
  it("emits confirm and closes when the destructive action is accepted", async () => {
    const wrapper = mount(ComicDeleteConfirmDialog, {
      props: {
        open: true,
      },
    })

    await wrapper.get("[data-comic-delete-confirm]").trigger("click")

    expect(wrapper.emitted("confirm")).toEqual([[]])
    expect(wrapper.emitted("update:open")).toEqual([[false]])
  })
})

import { mount, shallowMount } from "@vue/test-utils"
import { ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import LibraryBatchActionBar from "./LibraryBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: { name: "Dialog", template: "<div><slot /></div>" },
  DialogClose: { name: "DialogClose", template: "<div><slot /></div>" },
  DialogContent: { name: "DialogContent", template: "<div><slot /></div>" },
  DialogDescription: { name: "DialogDescription", template: "<div><slot /></div>" },
  DialogFooter: { name: "DialogFooter", template: "<div><slot /></div>" },
  DialogHeader: { name: "DialogHeader", template: "<div><slot /></div>" },
  DialogTitle: { name: "DialogTitle", template: "<div><slot /></div>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<input :value=\"modelValue\" @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

vi.mock("@vueuse/core", () => ({
  useFocusWithin: () => ({
    focused: ref(true),
  }),
}))

describe("LibraryBatchActionBar layout", () => {
  it("aligns flush with the content bottom instead of preserving rounded shell corners", () => {
    const wrapper = shallowMount(LibraryBatchActionBar, {
      props: {
        mode: "library",
        selectedCount: 2,
        useWebApi: true,
        scrapeProgress: null,
        scrapeBusy: false,
        operationBusy: false,
      },
    })

    const toolbar = wrapper.get('[role="toolbar"]')
    expect(toolbar.classes().join(" ")).not.toContain("rounded-b-[calc")
  })

  it("supports fuzzy suggestion selection for batch tag append", async () => {
    const wrapper = mount(LibraryBatchActionBar, {
      props: {
        mode: "library",
        selectedCount: 2,
        useWebApi: true,
        scrapeProgress: null,
        scrapeBusy: false,
        operationBusy: false,
        userTagSuggestions: ["alpha", "beta", "gamma"],
      },
    })

    const openDialogButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("library.batchAddTag"))

    expect(openDialogButton).toBeDefined()

    await openDialogButton!.trigger("click")
    await wrapper.get('input[role="combobox"]').setValue("alp")

    const suggestion = wrapper
      .findAll('[role="option"]')
      .find((option) => option.text() === "alpha")

    expect(suggestion).toBeDefined()

    await suggestion!.trigger("mousedown")

    expect(wrapper.emitted("addUserTag")).toEqual([["alpha"]])
  })
})

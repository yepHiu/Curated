import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import ComicBatchActionBar from "./ComicBatchActionBar.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/components/ui/button", () => ({
  Button: {
    name: "Button",
    props: ["disabled"],
    emits: ["click"],
    template:
      '<button v-bind="$attrs" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
  },
}))

vi.mock("@/components/ui/dialog", () => ({
  Dialog: {
    name: "Dialog",
    props: ["open"],
    emits: ["update:open"],
    template: '<div data-dialog><slot /></div>',
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

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue", "placeholder"],
    emits: ["update:modelValue"],
    template:
      '<input v-bind="$attrs" :placeholder="placeholder" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

describe("ComicBatchActionBar", () => {
  it("uses the library batch toolbar shape and emits comic batch actions", async () => {
    const wrapper = mount(ComicBatchActionBar, {
      props: {
        selectedCount: 2,
        operationBusy: false,
      },
    })

    expect(wrapper.get('[role="toolbar"]').attributes("aria-label")).toBe(
      "comics.batchToolbarAria",
    )
    expect(wrapper.text()).toContain("comics.batchSelected:{\"n\":2}")

    await wrapper.get("[data-comic-batch-add-favorite]").trigger("click")
    await wrapper.get("[data-comic-batch-remove-favorite]").trigger("click")
    await wrapper.get("[data-comic-batch-select-visible]").trigger("click")
    await wrapper.get("[data-comic-batch-clear-selection]").trigger("click")
    await wrapper.get("[data-comic-batch-exit]").trigger("click")

    expect(wrapper.emitted("addFavorite")).toHaveLength(1)
    expect(wrapper.emitted("removeFavorite")).toHaveLength(1)
    expect(wrapper.emitted("selectAllVisible")).toHaveLength(1)
    expect(wrapper.emitted("clearSelection")).toHaveLength(1)
    expect(wrapper.emitted("exit")).toHaveLength(1)
  })

  it("submits tags and confirms deletion from the comic batch toolbar", async () => {
    const wrapper = mount(ComicBatchActionBar, {
      props: {
        selectedCount: 2,
        operationBusy: false,
      },
    })

    await wrapper.get("[data-comic-batch-open-tag]").trigger("click")
    await wrapper.get("[data-comic-batch-tag-input]").setValue("artist:alpha")
    await wrapper.get("[data-comic-batch-submit-tag]").trigger("click")
    await wrapper.get("[data-comic-batch-open-delete]").trigger("click")
    await wrapper.get("[data-comic-batch-confirm-delete]").trigger("click")

    expect(wrapper.emitted("addTag")).toEqual([["artist:alpha"]])
    expect(wrapper.emitted("deleteComics")).toHaveLength(1)
  })
})

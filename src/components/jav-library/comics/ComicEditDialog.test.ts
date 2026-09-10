import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import ComicEditDialog from "./ComicEditDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: {
    name: "Badge",
    props: ["as", "variant"],
    template: '<component :is="as || \'span\'" v-bind="$attrs"><slot /></component>',
  },
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
    template: '<div v-if="open" data-dialog-open><slot /></div>',
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
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      '<input v-bind="$attrs" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
}))

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  return {
    id: "comic-1",
    title: "Original Title",
    tags: ["author:alpha"],
    rating: 3,
    isFavorite: false,
    readStatus: "reading",
    pageCount: 12,
    currentPageIndex: 2,
    sourceFileName: "original.cbz",
    location: "D:/Comics/original.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("ComicEditDialog", () => {
  it("opens with current comic values and saves a comic patch", async () => {
    const patchComic = vi.fn((_patch, done: (err?: unknown) => void) => done())
    const wrapper = mount(ComicEditDialog, {
      props: {
        comic: makeComic(),
        patchComic,
        open: true,
      },
    })

    expect(wrapper.get("[data-comic-title-input]").element).toHaveProperty(
      "value",
      "Original Title",
    )
    expect(wrapper.get("[data-comic-tags-input]").element).toHaveProperty(
      "value",
      "author:alpha",
    )
    expect(wrapper.get("[data-comic-rating-input]").element).toHaveProperty("value", "3")

    await wrapper.get("[data-comic-title-input]").setValue("Edited Title")
    await wrapper.get("[data-comic-tags-input]").setValue("author:alpha, series:rain")
    await wrapper.get("[data-comic-rating-input]").setValue("4.5")
    await wrapper.get("[data-comic-favorite-toggle]").trigger("click")
    await wrapper.get("[data-comic-save]").trigger("click")

    expect(patchComic).toHaveBeenCalledWith(
      {
        title: "Edited Title",
        tags: ["author:alpha", "series:rain"],
        rating: 4.5,
        favorite: true,
      },
      expect.any(Function),
    )
    expect(wrapper.emitted("update:open")?.at(-1)).toEqual([false])
  })

  it("keeps the dialog open and shows an error when save fails", async () => {
    const patchComic = vi.fn((_patch, done: (err?: unknown) => void) => {
      done(new Error("save failed"))
    })
    const wrapper = mount(ComicEditDialog, {
      props: {
        comic: makeComic(),
        patchComic,
        open: true,
      },
    })

    await wrapper.get("[data-comic-save]").trigger("click")

    expect(wrapper.text()).toContain("save failed")
    expect(wrapper.emitted("update:open")).toBeUndefined()
  })

  it("offers the MVP tag prefix shortcuts inside the edit dialog", () => {
    const wrapper = mount(ComicEditDialog, {
      props: {
        comic: makeComic(),
        patchComic: vi.fn(),
        open: true,
      },
    })

    expect(wrapper.text()).toContain("author:")
    expect(wrapper.text()).toContain("series:")
    expect(wrapper.text()).toContain("volume:")
    expect(wrapper.text()).toContain("circle:")
  })
})

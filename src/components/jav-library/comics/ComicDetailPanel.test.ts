import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { ComicBook } from "@/domain/comic/types"
import ComicDetailPanel from "./ComicDetailPanel.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
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

vi.mock("@/components/ui/card", () => ({
  Card: {
    name: "Card",
    props: ["class"],
    template: "<section data-card :class='$props.class'><slot /></section>",
  },
  CardContent: {
    name: "CardContent",
    props: ["class"],
    template: "<div data-card-content :class='$props.class'><slot /></div>",
  },
  CardDescription: {
    name: "CardDescription",
    props: ["class"],
    template: "<p :class='$props.class'><slot /></p>",
  },
  CardTitle: {
    name: "CardTitle",
    props: ["class"],
    template: "<h2 :class='$props.class'><slot /></h2>",
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

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuItem: {
    name: "DropdownMenuItem",
    emits: ["click"],
    template:
      '<button v-bind="$attrs" type="button" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div><slot /></div>" },
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
    id: "comic-detail-1",
    title: "Original Title",
    tags: ["author:alpha", "series:rain"],
    rating: 3,
    isFavorite: false,
    readStatus: "unread",
    pageCount: 16,
    currentPageIndex: 0,
    sourceFileName: "original.cbz",
    location: "D:/Comics/original.cbz",
    addedAt: "2026-06-01T00:00:00.000Z",
    updatedAt: "2026-06-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("ComicDetailPanel", () => {
  it("renders the detail page in read-only mode with actions in the more menu", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic({ coverUrl: "https://example.com/detail-cover.jpg" }),
      },
    })

    expect(wrapper.text()).toContain("Original Title")
    expect(wrapper.text()).toContain("author:alpha")
    expect(wrapper.text()).toContain("series:rain")
    expect(wrapper.get("[data-comic-detail-cover]").attributes("src")).toBe(
      "https://example.com/detail-cover.jpg",
    )
    expect(wrapper.get("[data-comic-detail-rating-card]").text()).toContain("3")
    expect(wrapper.find("[data-comic-more-actions]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-edit-action]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-reveal-source]").exists()).toBe(true)
    expect(wrapper.find("[data-comic-delete-action]").exists()).toBe(true)

    expect(wrapper.find("[data-comic-title-input]").exists()).toBe(false)
    expect(wrapper.find("[data-comic-tags-input]").exists()).toBe(false)
    expect(wrapper.find("[data-comic-rating-input]").exists()).toBe(false)
    expect(wrapper.find("[data-comic-favorite-toggle]").exists()).toBe(false)
    expect(wrapper.find("[data-comic-save]").exists()).toBe(false)
  })

  it("omits source metadata from the visible detail body", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.text()).not.toContain("comics.sourceFile")
    expect(wrapper.text()).not.toContain("comics.sourceLocation")
    expect(wrapper.text()).not.toContain("D:/Comics/original.cbz")
  })

  it("opens the edit dialog from the more menu and forwards edited fields", async () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    await wrapper.get("[data-comic-edit-action]").trigger("click")
    await wrapper.get("[data-comic-title-input]").setValue("Edited Title")
    await wrapper.get("[data-comic-tags-input]").setValue("author:alpha, series:rain, volume:1")
    await wrapper.get("[data-comic-rating-input]").setValue("4.5")
    await wrapper.get("[data-comic-favorite-toggle]").trigger("click")
    await wrapper.get("[data-comic-save]").trigger("click")

    const patchCall = wrapper.emitted("patch")?.[0]
    expect(patchCall?.[0]).toEqual({
      title: "Edited Title",
      tags: ["author:alpha", "series:rain", "volume:1"],
      rating: 4.5,
      favorite: true,
    })
    expect(typeof patchCall?.[1]).toBe("function")
  })

  it("emits reader, reveal, and confirmed delete actions", async () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    await wrapper.get("[data-comic-start-reading]").trigger("click")
    expect(wrapper.emitted("startReading")?.[0]).toEqual([0])

    await wrapper.get("[data-comic-reveal-source]").trigger("click")
    expect(wrapper.emitted("revealSource")?.[0]).toEqual(["comic-detail-1"])

    await wrapper.get("[data-comic-delete-action]").trigger("click")
    await wrapper.get("[data-comic-delete-confirm]").trigger("click")
    expect(wrapper.emitted("deleteComic")?.[0]).toEqual(["comic-detail-1"])
  })

  it("uses the shared detail shell with the narrower media column", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.get("[data-comic-detail-panel]").classes()).toEqual(
      expect.arrayContaining(["rounded-3xl", "bg-card/85"]),
    )
    expect(wrapper.get("[data-comic-detail-content]").classes()).toEqual(
      expect.arrayContaining([
        "lg:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]",
        "xl:grid-cols-[minmax(0,28rem)_minmax(0,1fr)]",
      ]),
    )
    expect(wrapper.get("[data-comic-detail-media-column]").classes()).toEqual(
      expect.arrayContaining([
        "lg:max-w-[min(100%,24rem)]",
        "xl:max-w-[min(100%,28rem)]",
      ]),
    )
  })
})

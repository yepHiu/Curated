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
    expect(wrapper.text()).toContain("original.cbz")
    expect(wrapper.get("[data-comic-detail-cover]").attributes("src")).toBe(
      "https://example.com/detail-cover.jpg",
    )
    expect(wrapper.find("[data-comic-detail-rating-card]").exists()).toBe(false)
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

  it("omits the cover metadata card with rating, progress, and favorite state", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic({ rating: null, pageCount: 32, currentPageIndex: 0 }),
      },
    })

    expect(wrapper.find("[data-comic-detail-rating-card]").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("comics.detailRatingLabel")
    expect(wrapper.text()).not.toContain("comics.noRating")
    expect(wrapper.text()).not.toContain("comics.pageCount")
    expect(wrapper.text()).not.toContain("comics.favoriteOff")
    expect(wrapper.text()).not.toContain("1 / 32")
  })

  it("shows source metadata in the detail body", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.text()).not.toContain("comics.sourceFile")
    expect(wrapper.text()).not.toContain("comics.sourceLocation")
    expect(wrapper.text()).toContain("D:/Comics/original.cbz")
    expect(wrapper.text()).toContain("original.cbz")
  })

  it("renders the detail cover without fixed-ratio cropping so wide and tall pages can adapt", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic({ coverUrl: "https://example.com/wide-cover.jpg" }),
      },
    })

    const frame = wrapper.get("[data-comic-detail-cover-frame]")
    const cover = wrapper.get("[data-comic-detail-cover]")

    expect(frame.classes()).toEqual(
      expect.arrayContaining(["w-fit", "max-w-full", "max-h-[min(56vh,24rem)]"]),
    )
    expect(frame.classes()).not.toContain("aspect-[358/537]")
    expect(cover.classes()).toEqual(
      expect.arrayContaining([
        "block",
        "h-auto",
        "w-auto",
        "max-h-[min(56vh,24rem)]",
        "max-w-full",
        "object-contain",
      ]),
    )
    expect(cover.classes()).not.toEqual(
      expect.arrayContaining(["absolute", "inset-0", "h-full", "w-full", "object-cover"]),
    )
  })

  it("edits comic tags inline like detail tag chips", async () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    await wrapper.get("[data-comic-add-tag]").trigger("click")
    await wrapper.get("[data-comic-new-tag-input]").setValue("volume:1")
    await wrapper.get("[data-comic-add-tag]").trigger("click")

    let patchCall = wrapper.emitted("patch")?.[0]
    expect(patchCall?.[0]).toEqual({
      tags: ["author:alpha", "series:rain", "volume:1"],
    })
    expect(typeof patchCall?.[1]).toBe("function")

    await wrapper.get("[data-comic-remove-tag='author:alpha']").trigger("click")
    patchCall = wrapper.emitted("patch")?.[1]
    expect(patchCall?.[0]).toEqual({
      tags: ["series:rain"],
    })
    expect(typeof patchCall?.[1]).toBe("function")
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

  it("uses a left-aligned detail shell with the cover and title top-aligned", () => {
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
        "relative",
        "items-start",
        "justify-items-start",
        "lg:justify-start",
        "lg:grid-cols-[fit-content(18rem)_minmax(0,1fr)]",
        "xl:grid-cols-[fit-content(20rem)_minmax(0,1fr)]",
      ]),
    )
    expect(wrapper.get("[data-comic-detail-content]").classes()).not.toContain("items-center")
    expect(wrapper.get("[data-comic-detail-content]").classes()).not.toContain("lg:justify-center")
    expect(wrapper.get("[data-comic-detail-content]").classes()).not.toContain("pr-14")
    expect(wrapper.get("[data-comic-detail-content]").classes()).not.toContain("sm:pr-16")
    expect(wrapper.get("[data-comic-more-actions-zone]").classes()).toEqual(
      expect.arrayContaining(["absolute", "right-4", "top-4", "sm:right-6", "sm:top-6"]),
    )
    expect(
      wrapper.get("[data-comic-detail-info-column]").find("[data-comic-more-actions]").exists(),
    ).toBe(false)
    expect(wrapper.get("[data-comic-detail-media-column]").classes()).toEqual(
      expect.arrayContaining([
        "w-fit",
        "max-w-full",
      ]),
    )
    expect(wrapper.get("[data-comic-detail-media-column]").classes()).not.toContain("lg:mx-auto")
    expect(wrapper.get("[data-comic-detail-cover-frame]").classes()).not.toContain("mx-auto")
    expect(wrapper.get("[data-comic-detail-info-column]").classes()).toEqual(
      expect.arrayContaining(["justify-start", "gap-4"])
    )
    expect(wrapper.get("[data-comic-detail-info-column]").classes()).not.toContain("justify-center")
    expect(wrapper.get("[data-comic-detail-info-column]").classes()).not.toContain("lg:py-2")
    expect(wrapper.get("[data-comic-detail-title]").classes()).toEqual(
      expect.arrayContaining(["pr-12", "text-2xl", "sm:pr-14", "sm:text-3xl"]),
    )
    expect(wrapper.get("[data-comic-detail-title]").classes()).toContain("sm:text-3xl")
  })
})

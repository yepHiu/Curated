import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoDetailPanel from "./PhotoDetailPanel.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: {
    name: "Badge",
    props: ["as", "asChild", "variant"],
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
  CardTitle: {
    name: "CardTitle",
    props: ["class"],
    template: "<h2 :class='$props.class'><slot /></h2>",
  },
}))

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: { name: "DropdownMenu", template: "<div><slot /></div>" },
  DropdownMenuContent: { name: "DropdownMenuContent", template: "<div><slot /></div>" },
  DropdownMenuGroup: { name: "DropdownMenuGroup", template: "<div><slot /></div>" },
  DropdownMenuItem: {
    name: "DropdownMenuItem",
    props: ["disabled", "variant"],
    emits: ["click"],
    template:
      '<button v-bind="$attrs" type="button" :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>',
  },
  DropdownMenuTrigger: { name: "DropdownMenuTrigger", template: "<div><slot /></div>" },
}))

function makePhoto(overrides: Partial<PhotoBook> = {}): PhotoBook {
  return {
    id: "photo-detail-1",
    title: "Summer Frame",
    tags: ["portrait", "outdoor"],
    rating: 4.5,
    isFavorite: false,
    pageCount: 16,
    currentPageIndex: 2,
    coverUrl: "https://example.com/photo-cover.jpg",
    sourceFileName: "summer-frame.cbz",
    location: "D:/Photos/summer-frame.cbz",
    addedAt: "2026-07-01T00:00:00.000Z",
    updatedAt: "2026-07-01T00:00:00.000Z",
    ...overrides,
  }
}

describe("PhotoDetailPanel", () => {
  it("shows the photo title, tags, rating, cover, and browse action", () => {
    const wrapper = mount(PhotoDetailPanel, {
      props: {
        photo: makePhoto(),
      },
    })

    expect(wrapper.text()).toContain("Summer Frame")
    expect(wrapper.text()).toContain("portrait")
    expect(wrapper.text()).toContain("outdoor")
    expect(wrapper.text()).toContain("photos.detailRatingLabel")
    expect(wrapper.text()).toContain("4.5")
    expect(wrapper.text()).toContain("bookBrowser.continueAt")
    expect(wrapper.get("[data-photo-detail-cover]").attributes("src")).toBe(
      "https://example.com/photo-cover.jpg",
    )
    expect(wrapper.text()).toContain("summer-frame.cbz")
    expect(wrapper.text()).toContain("D:/Photos/summer-frame.cbz")
  })

  it("only offers supported photo browsing actions", async () => {
    const wrapper = mount(PhotoDetailPanel, { props: { photo: makePhoto() } })
    expect(wrapper.find("[data-photo-more-actions]").exists()).toBe(false)
    await wrapper.get("[data-photo-start-browsing]").trigger("click")
    expect(wrapper.emitted("startBrowsing")?.[0]).toEqual([2])
  })

  it("uses the same left-aligned cover and top-aligned title structure as comic detail", () => {
    const wrapper = mount(PhotoDetailPanel, {
      props: {
        photo: makePhoto(),
      },
    })

    expect(wrapper.get("[data-photo-detail-content]").classes()).toEqual(
      expect.arrayContaining([
        "relative",
        "items-start",
        "justify-items-start",
        "lg:justify-start",
        "lg:grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)]",
        "xl:grid-cols-[minmax(13rem,20rem)_minmax(0,1fr)]",
      ]),
    )
    expect(wrapper.get("[data-photo-detail-cover]").classes()).toEqual(
      expect.arrayContaining([
        "block",
        "h-auto",
        "w-auto",
        "max-h-[min(56vh,24rem)]",
        "max-w-full",
        "object-contain",
      ]),
    )
  })
})

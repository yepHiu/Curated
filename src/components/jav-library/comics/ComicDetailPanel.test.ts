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
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", props: ["disabled"], template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/input", () => ({
  Input: {
    name: "Input",
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template: "<input :value='modelValue' @input=\"$emit('update:modelValue', $event.target.value)\" />",
  },
}))

function makeComic(overrides: Partial<ComicBook> = {}): ComicBook {
  return {
    id: "comic-detail-1",
    title: "Original Title",
    tags: ["作者:青井"],
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
  it("emits title, tags, rating, and favorite edits", async () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    await wrapper.get("[data-comic-title-input]").setValue("Edited Title")
    await wrapper.get("[data-comic-tags-input]").setValue("作者:青井, 系列:雨庭")
    await wrapper.get("[data-comic-rating-input]").setValue("4.5")
    await wrapper.get("[data-comic-favorite-toggle]").trigger("click")
    await wrapper.get("[data-comic-save]").trigger("click")

    expect(wrapper.emitted("patch")?.[0]?.[0]).toEqual({
      title: "Edited Title",
      tags: ["作者:青井", "系列:雨庭"],
      rating: 4.5,
      favorite: true,
    })
  })

  it("suggests author, series, volume, and circle tag prefixes", () => {
    const wrapper = mount(ComicDetailPanel, {
      props: {
        comic: makeComic(),
      },
    })

    expect(wrapper.text()).toContain("作者:")
    expect(wrapper.text()).toContain("系列:")
    expect(wrapper.text()).toContain("卷:")
    expect(wrapper.text()).toContain("社团:")
  })
})

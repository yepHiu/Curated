import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoDetailPanel from "./PhotoDetailPanel.vue"
import BookMediaInfoDialog from "../books/BookMediaInfoDialog.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "zh-CN" },
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key,
  }),
}))

vi.mock("@/services/ai-service", () => ({
  useAIService: () => ({ runAction: vi.fn(), confirmTool: vi.fn() }),
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

vi.mock("@/components/jav-library/books/BookRatingCard.vue", () => ({
  default: {
    name: "BookRatingCard",
    props: ["rating", "disabled"],
    emits: ["commit"],
    template:
      '<section data-photo-detail-rating data-book-rating-card><button data-commit-rating type="button" @click="$emit(\'commit\', 3.5)" /></section>',
  },
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
  it("adds trimmed tags, blocks duplicate submits and retains the draft for retry", async () => {
    const wrapper = mount(PhotoDetailPanel, { props: { photo: makePhoto() } })
    await wrapper.get('[data-detail-add-tag]').trigger('click')
    const input = wrapper.get('[data-detail-new-tag-input]')
    await input.setValue('  landscape  ')
    await input.trigger('keydown', { key: 'Enter', isComposing: true })
    expect(wrapper.emitted('addTag')).toBeUndefined()
    await input.trigger('keydown', { key: 'Enter' })
    await input.trigger('keydown', { key: 'Enter' })
    expect(wrapper.emitted('addTag')).toHaveLength(1)
    const [tag, done] = wrapper.emitted('addTag')![0] as [string, (error?: unknown) => void]
    expect(tag).toBe('landscape')
    done(new Error('Unable to save'))
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="alert"]').text()).toBe('Unable to save')
    expect((input.element as HTMLInputElement).value).toBe('  landscape  ')
    await wrapper.get('[data-detail-add-tag]').trigger('click')
    const retryDone = wrapper.emitted('addTag')![1]![1] as () => void
    await wrapper.setProps({ photo: makePhoto({ tags: ['portrait', 'outdoor', 'landscape'] }) })
    retryDone()
    await wrapper.vm.$nextTick()
    expect((wrapper.get('[data-detail-new-tag-input]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain('landscape')
  })

  it("ignores duplicates and validates Unicode tag length", async () => {
    const wrapper = mount(PhotoDetailPanel, { props: { photo: makePhoto() } })
    await wrapper.get('[data-detail-add-tag]').trigger('click')
    await wrapper.get('[data-detail-new-tag-input]').setValue('portrait')
    await wrapper.get('[data-detail-add-tag]').trigger('click')
    expect(wrapper.emitted('addTag')).toBeUndefined()
    expect((wrapper.get('[data-detail-new-tag-input]').element as HTMLInputElement).value).toBe('')
    await wrapper.get('[data-detail-new-tag-input]').setValue('界'.repeat(65))
    await wrapper.get('[data-detail-add-tag]').trigger('click')
    expect(wrapper.get('[role="alert"]').text()).toContain('curated.tagMaxRunes')
    expect(wrapper.emitted('addTag')).toBeUndefined()
  })
  it("shows the photo title, tags, rating, cover, and browse action", () => {
    const wrapper = mount(PhotoDetailPanel, {
      props: {
        photo: makePhoto(),
      },
    })

    expect(wrapper.text()).toContain("Summer Frame")
    expect(wrapper.text()).toContain("portrait")
    expect(wrapper.text()).toContain("outdoor")
    expect(wrapper.find("[data-photo-detail-rating]").exists()).toBe(true)
    expect(
      wrapper.get("[data-photo-detail-info-column]").find("[data-photo-detail-rating]").exists(),
    ).toBe(true)
    expect(
      wrapper.get("[data-photo-detail-media-column]").find("[data-photo-detail-rating]").exists(),
    ).toBe(false)
    expect(wrapper.text()).toContain("bookBrowser.continueAt")
    expect(wrapper.get("[data-photo-detail-cover]").attributes("src")).toBe(
      "https://example.com/photo-cover.jpg",
    )
    expect(wrapper.text()).not.toContain("summer-frame.cbz")
    expect(wrapper.text()).not.toContain("D:/Photos/summer-frame.cbz")
  })

  it("emits a photo rating update from the rating card", async () => {
    // 标签下方的评分卡把半星选择交给详情页写入。
    const wrapper = mount(PhotoDetailPanel, { props: { photo: makePhoto() } })
    await wrapper.get("[data-commit-rating]").trigger("click")
    expect(wrapper.emitted("updateRating")).toEqual([[3.5]])
  })

  it("only offers supported photo browsing actions", async () => {
    const wrapper = mount(PhotoDetailPanel, { props: { photo: makePhoto() } })
    expect(wrapper.find("[data-photo-more-actions]").exists()).toBe(true)
    expect(wrapper.find("[data-photo-edit-action]").exists()).toBe(false)
    const dialog = wrapper.getComponent(BookMediaInfoDialog)
    expect(dialog.props("open")).toBe(false)
    await wrapper.get("[data-photo-media-info]").trigger("click")
    expect(dialog.props("open")).toBe(true)
    expect(dialog.props("book").location).toBe("D:/Photos/summer-frame.cbz")
    await wrapper.setProps({photo: makePhoto({id: "another-photo"})})
    expect(dialog.props("open")).toBe(false)
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
        "lg:grid-cols-[fit-content(18rem)_minmax(0,1fr)]",
        "xl:grid-cols-[fit-content(20rem)_minmax(0,1fr)]",
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
    // 评分卡跟在标签后面，不跟在封面下面。
    const info = wrapper.get("[data-photo-detail-info-column]")
    const rating = info.get("[data-photo-detail-rating]")
    expect(
      wrapper.get("[data-photo-detail-media-column]").find("[data-photo-detail-rating]").exists(),
    ).toBe(false)
    expect(
      Boolean(
        info.get("[data-photo-detail-tags]").element.compareDocumentPosition(rating.element) &
          Node.DOCUMENT_POSITION_FOLLOWING,
      ),
    ).toBe(true)
  })
})

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import BookRatingCard from "./BookRatingCard.vue"

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

vi.mock("@/components/jav-library/MovieRatingStars.vue", () => ({
  default: {
    name: "MovieRatingStars",
    props: ["modelValue", "disabled"],
    emits: ["commit"],
    template: '<button data-rating-stars type="button" @click="$emit(\'commit\', 3.5)" />',
  },
}))

describe("BookRatingCard", () => {
  it("shows an unrated label until a local score exists", () => {
    // 未评时显示占位文案，并禁用清除按钮。
    const wrapper = mount(BookRatingCard, { props: { rating: null } })
    expect(wrapper.text()).toContain("detailPanel.rating")
    expect(wrapper.text()).toContain("detailPanel.unrated")
    expect(wrapper.get("[data-book-rating-clear]").attributes("disabled")).toBeDefined()
  })

  it("keeps the card at a fixed 250px width", () => {
    // 评分卡固定 250px，避免在详情信息列里拉满整行。
    const wrapper = mount(BookRatingCard, { props: { rating: 4 } })
    expect(wrapper.get("[data-book-rating-card]").classes()).toEqual(
      expect.arrayContaining(["w-[250px]", "max-w-full"]),
    )
  })

  it("commits a half-star score and can clear it", async () => {
    // 半星写入后仍可清除为未评。
    const wrapper = mount(BookRatingCard, { props: { rating: 4 } })
    expect(wrapper.text()).toContain('detailPanel.combined:{"n":"4.0"}')
    await wrapper.get("[data-rating-stars]").trigger("click")
    expect(wrapper.emitted("commit")).toEqual([[3.5]])
    await wrapper.get("[data-book-rating-clear]").trigger("click")
    expect(wrapper.emitted("commit")).toEqual([[3.5], [null]])
  })
})

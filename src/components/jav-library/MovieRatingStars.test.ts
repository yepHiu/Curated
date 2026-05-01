import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params && "s" in params ? `${key}:${String(params.s)}` : key,
  }),
}))

async function mountRating(modelValue = 2.5) {
  const { default: MovieRatingStars } = await import("./MovieRatingStars.vue")
  return mount(MovieRatingStars, {
    props: {
      modelValue,
    },
  })
}

describe("MovieRatingStars", () => {
  it("renders rating aria labels from locale keys", async () => {
    const wrapper = await mountRating()

    expect(wrapper.get('[role="group"]').attributes("aria-label")).toBe("rating.ariaLabel")

    const buttons = wrapper.findAll("button")
    expect(buttons).toHaveLength(10)
    expect(buttons[0]?.attributes("aria-label")).toBe("rating.score:0.5")
    expect(buttons[1]?.attributes("aria-label")).toBe("rating.score:1")
    expect(buttons[8]?.attributes("aria-label")).toBe("rating.score:4.5")
    expect(buttons[9]?.attributes("aria-label")).toBe("rating.score:5")
  })
})

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

async function mountExpandableText(props: {
  text: string
  collapsedLines?: number
  forceExpandable?: boolean
  expandLabel?: string
  collapseLabel?: string
}) {
  const { default: ExpandableText } = await import("./ExpandableText.vue")
  return mount(ExpandableText, { props })
}

describe("ExpandableText", () => {
  it("does not render a toggle for short text", async () => {
    const wrapper = await mountExpandableText({
      text: "Short summary.",
      collapsedLines: 5,
    })

    expect(wrapper.text()).toContain("Short summary.")
    expect(wrapper.find("[data-expandable-toggle]").exists()).toBe(false)
  })

  it("expands and collapses long text", async () => {
    const wrapper = await mountExpandableText({
      text: "Long summary ".repeat(80),
      collapsedLines: 5,
      forceExpandable: true,
    })

    expect(wrapper.get("[data-expandable-toggle]").text()).toBe("movie.expandSummary")
    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-toggle]").text()).toBe("movie.collapseSummary")
    expect(wrapper.get("[data-expandable-content]").classes()).not.toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")
  })

  it("uses explicit expand and collapse labels when provided", async () => {
    const wrapper = await mountExpandableText({
      text: "Long summary ".repeat(80),
      forceExpandable: true,
      expandLabel: "Show more",
      collapseLabel: "Show less",
    })

    expect(wrapper.get("[data-expandable-toggle]").text()).toBe("Show more")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-toggle]").text()).toBe("Show less")
  })
})

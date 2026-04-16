import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"

import ExpandableText from "./ExpandableText.vue"

describe("ExpandableText", () => {
  it("does not render a toggle for short text", () => {
    const wrapper = mount(ExpandableText, {
      props: { text: "Short summary.", collapsedLines: 5 },
    })

    expect(wrapper.text()).toContain("Short summary.")
    expect(wrapper.find("[data-expandable-toggle]").exists()).toBe(false)
  })

  it("expands and collapses long text", async () => {
    const wrapper = mount(ExpandableText, {
      props: {
        text: "Long summary ".repeat(80),
        collapsedLines: 5,
        forceExpandable: true,
      },
    })

    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-content]").classes()).not.toContain("line-clamp-5")

    await wrapper.get("[data-expandable-toggle]").trigger("click")
    expect(wrapper.get("[data-expandable-content]").classes()).toContain("line-clamp-5")
  })
})

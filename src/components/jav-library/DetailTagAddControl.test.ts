import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import DetailTagAddControl from "./DetailTagAddControl.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
})

function render(tags: string[] = [], suggestions: string[] = []) {
  const wrapper = mount(DetailTagAddControl, {
    attachTo: document.body,
    props: { tags, suggestions, saveErrorMessage: "Save failed" },
  })
  wrappers.push(wrapper)
  return wrapper
}

describe("DetailTagAddControl", () => {
  it("focuses the inline field, cancels with Escape and closes on an outside click", async () => {
    const wrapper = render()
    await wrapper.get("[data-detail-add-tag]").trigger("click")
    const input = wrapper.get("input")
    expect(document.activeElement).toBe(input.element)
    await input.setValue("draft")
    await input.trigger("keydown", { key: "Escape", isComposing: true })
    expect(wrapper.find("input").exists()).toBe(true)
    await input.trigger("keydown", { key: "Escape" })
    expect(wrapper.find("input").exists()).toBe(false)
    await wrapper.get("[data-detail-add-tag]").trigger("click")
    expect((wrapper.get("input").element as HTMLInputElement).value).toBe("")
    await new Promise((resolve) => setTimeout(resolve, 0))
    document.body.dispatchEvent(new Event("pointerdown", { bubbles: true }))
    document.body.dispatchEvent(new MouseEvent("click", { bubbles: true }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find("input").exists()).toBe(false)
    expect(wrapper.emitted("add")).toBeUndefined()
  })

  it("keeps movie suggestions keyboard accessible and excludes existing tags", async () => {
    const wrapper = render(["alpha"], ["alpha", "alpine", "beta"])
    await wrapper.get("[data-detail-add-tag]").trigger("click")
    const input = wrapper.get("input")
    await input.setValue("al")
    expect(wrapper.findAll('[role="option"]').map((option) => option.text())).toEqual(["alpine"])
    const option = wrapper.get('[role="option"]')
    ;(option.element as HTMLElement).scrollIntoView = vi.fn()
    await input.trigger("keydown", { key: "ArrowDown" })
    expect(input.attributes("aria-activedescendant")).toBe(option.attributes("id"))
    await input.trigger("keydown", { key: "Enter" })
    expect(wrapper.emitted("add")![0]![0]).toBe("alpine")
  })

  it("validates the count and Unicode characters without splitting surrogate pairs", async () => {
    const wrapper = render(Array.from({ length: 64 }, (_, index) => `tag-${index}`))
    await wrapper.get("[data-detail-add-tag]").trigger("click")
    const input = wrapper.get("input")
    await input.setValue("new tag")
    await input.trigger("keydown", { key: "Enter" })
    expect(wrapper.get('[role="alert"]').text()).toBe("curated.tagMaxCount")
    expect(wrapper.emitted("add")).toBeUndefined()
    await wrapper.setProps({ tags: [] })
    await input.setValue("😀".repeat(65))
    await input.trigger("keydown", { key: "Enter" })
    expect(wrapper.get('[role="alert"]').text()).toBe("curated.tagMaxRunes")
    await input.setValue("😀".repeat(64))
    await input.trigger("keydown", { key: "Enter" })
    expect(wrapper.emitted("add")![0]![0]).toBe("😀".repeat(64))
  })
})

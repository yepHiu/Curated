import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, describe, expect, it } from "vitest"
import SettingsHint from "./SettingsHint.vue"

const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  document.body.innerHTML = ""
})
function render() {
  const wrapper = mount(SettingsHint, {
    props: { text: "Only applies to this setting" },
    slots: { default: '<p>Setting title</p>' },
    attachTo: document.body,
  })
  wrappers.push(wrapper)
  return wrapper
}

describe("settings explanatory tooltip", () => {
  it("hides prose by default and reveals it on title focus", async () => {
    const wrapper = render()
    expect(document.body.textContent).not.toContain("Only applies")
    const title = wrapper.get('[data-settings-hint-trigger]')
    expect(title.attributes("tabindex")).toBe("0")
    await title.trigger("focus")
    await flushPromises()
    expect(document.body.querySelector('[role="tooltip"]')?.textContent).toContain("Only applies")
    await title.trigger("keydown", { key: "Escape" })
    await flushPromises()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
  })
  it("preserves keyboard activation on existing action buttons", () => {
    const wrapper = mount(SettingsHint, { props: { text: "Action help" }, slots: { default: '<button type="button">Run</button>' } })
    wrappers.push(wrapper)
    const event = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true })
    wrapper.get("button").element.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(false)
  })
  it("supports click to reveal and keyboard dismissal without changing the setting", async () => {
    const wrapper = render()
    const title = wrapper.get('[data-settings-hint-trigger]')
    await title.trigger("click")
    await flushPromises()
    expect(document.body.querySelector('[role="tooltip"]')).not.toBeNull()
    await title.trigger("keydown", { key: "Enter" })
    await flushPromises()
    expect(document.body.querySelector('[role="tooltip"]')).toBeNull()
    await title.trigger("keydown", { key: " " })
    await flushPromises()
    expect(document.body.querySelector('[role="tooltip"]')).not.toBeNull()
  })
})

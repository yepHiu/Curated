import { flushPromises, mount } from "@vue/test-utils"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { DropdownMenuItem } from "@/components/ui/dropdown-menu"
import DesktopReloadMenu from "./DesktopReloadMenu.vue"

let wrapper: ReturnType<typeof mount>

beforeEach(() => {
  vi.stubGlobal("ResizeObserver", class { observe() {} unobserve() {} disconnect() {} })
  wrapper = mount(DesktopReloadMenu, { attachTo: document.body, props: { label: "重新加载" } })
})

afterEach(() => {
  wrapper.unmount()
  document.body.innerHTML = ""
  window.getSelection()?.removeAllRanges()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

async function rightClick(target: Element = document.body) {
  const event = new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 200, clientY: 120 })
  target.dispatchEvent(event)
  await flushPromises()
  return event
}

function menu() {
  return document.querySelector('[role="menu"][data-state="open"]')
}

it("opens on blank space and reloads the current page when selected", async () => {
  expect((await rightClick()).defaultPrevented).toBe(true)
  expect(menu()?.textContent).toContain("重新加载")
  const reload = vi.fn()
  const originalWindow = window
  vi.stubGlobal("window", { location: { reload } })
  wrapper.findComponent(DropdownMenuItem).vm.$emit("select", new Event("select"))
  vi.stubGlobal("window", originalWindow)
  expect(reload).toHaveBeenCalledOnce()
})

it.each([
  '<input />', '<textarea></textarea>', '<button><span>Action</span></button>',
  '<a href="/movies"><span>Movie</span></a>', '<div contenteditable="true"><span>Edit</span></div>',
  '<video></video>', '<img alt="Cover" />', '<div role="slider"></div>',
  '<div data-slot="context-menu-trigger"><span>Card</span></div>',
  '<div data-desktop-reload-exclude><span>Custom action</span></div>',
])("preserves right-click behavior for %s", async (html) => {
  const container = document.createElement("div")
  container.innerHTML = html
  document.body.append(container)
  expect((await rightClick(container.querySelector("span") ?? container.firstElementChild!)).defaultPrevented).toBe(false)
  expect(menu()).toBeNull()
})

it("lets an existing business context menu take priority", async () => {
  const card = document.createElement("div")
  card.addEventListener("contextmenu", event => event.preventDefault())
  document.body.append(card)
  await rightClick(card)
  expect(menu()).toBeNull()
})

it("preserves selected text", async () => {
  const text = document.createElement("p")
  text.textContent = "Selected title"
  document.body.append(text)
  const range = document.createRange()
  range.selectNodeContents(text)
  window.getSelection()?.addRange(range)
  expect((await rightClick(text)).defaultPrevented).toBe(false)
  expect(menu()).toBeNull()
})

it.each(["blur", "resize", "scroll", "Escape"])("closes on %s", async (action) => {
  await rightClick()
  expect(menu()).not.toBeNull()
  if (action === "Escape") document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  else if (action === "scroll") document.dispatchEvent(new Event("scroll"))
  else window.dispatchEvent(new Event(action))
  await flushPromises()
  expect(menu()).toBeNull()
})

it("removes the global listener on unmount", async () => {
  wrapper.unmount()
  expect((await rightClick()).defaultPrevented).toBe(false)
  expect(menu()).toBeNull()
})

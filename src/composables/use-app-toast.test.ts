import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { mount, type VueWrapper } from "@vue/test-utils"
import { nextTick } from "vue"
import Toaster from "@/components/ui/sonner/Sonner.vue"
import { pushAppToast, pushAppToastLoading } from "./use-app-toast"

vi.mock("@/composables/use-notification-center", () => ({ useNotificationCenter: vi.fn() }))

let wrapper: VueWrapper

async function settle() {
  await nextTick()
  await nextTick()
  await nextTick()
}

async function elapse(ms: number) {
  await vi.advanceTimersByTimeAsync(ms)
  await settle()
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.spyOn(document, "hidden", "get").mockReturnValue(false)
  wrapper = mount(Toaster, { props: { theme: "light" }, attachTo: document.body })
})

afterEach(() => {
  wrapper.unmount()
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe("app toast auto-close", () => {
  it.each(["default", "success", "warning", "destructive"] as const)("auto-closes %s toasts", async (variant) => {
    pushAppToast("result", { variant })
    await settle()
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(true)
    await elapse(4400)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(true)
    await elapse(400)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(false)
  })

  it.each([Infinity, -Infinity, NaN, 0, -1, 2_147_483_648])("auto-closes a toast with invalid duration %s", async (durationMs) => {
    pushAppToast("invalid duration", { durationMs })
    await settle()
    await elapse(4800)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(false)
  })

  it("auto-closes progress while the task is still running and allows manual dismissal", async () => {
    pushAppToastLoading("scraping")
    await settle()
    expect(wrapper.find(".animate-spin").exists()).toBe(true)
    expect(wrapper.find("[data-close-button]").exists()).toBe(true)
    await elapse(4800)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(false)
  })

  it.each([1000, 5000])("auto-closes a result arriving after %s ms and clears its spinner", async (delay) => {
    const id = pushAppToastLoading("scraping")
    await settle()
    await elapse(delay)
    pushAppToast("scraped", { id, variant: "success" })
    await settle()
    expect(wrapper.text()).toContain("scraped")
    expect(wrapper.find(".animate-spin").exists()).toBe(false)
    await elapse(4800)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(false)
  })

  it("respects a finite custom duration and keeps the action available until closing", async () => {
    pushAppToast("load failed", { durationMs: 9000, action: { label: "Reload", onClick: vi.fn() } })
    await settle()
    await elapse(4800)
    expect(wrapper.text()).toContain("Reload")
    await elapse(4500)
    expect(wrapper.find("[data-sonner-toast]").exists()).toBe(false)
  })
})

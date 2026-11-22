import { mount } from "@vue/test-utils"
import { defineComponent, h, nextTick, ref } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import App from "./App.vue"

vi.mock("@/composables/use-theme", () => ({
  useTheme: () => ({ resolvedMode: ref("dark") }),
}))

vi.mock("@/i18n", () => ({
  syncHtmlLang: vi.fn(),
}))

vi.mock("@/lib/locale-storage", () => ({
  persistLocale: vi.fn(),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string) => key,
  }),
}))

vi.mock("vue-router", () => ({
  RouterView: defineComponent({
    name: "ThrowingRouterView",
    setup() {
      throw new Error("route render failed")
    },
    render: () => h("div"),
  }),
}))

afterEach(() => {
  vi.restoreAllMocks()
})

describe("App", () => {
  it("renders a fault state when the routed subtree throws", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)

    const wrapper = mount(App)
    await nextTick()

    expect(wrapper.get("[data-app-fault]").text()).toContain("app.faultTitle")
    expect(wrapper.text()).toContain("app.faultDescription")
    expect(wrapper.findComponent({ name: "Toaster" }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: "DesktopReloadMenu" }).exists()).toBe(false)
    wrapper.unmount()
  })

  it("keeps desktop reload available even when the routed page fails", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined)
    window.javLibrary = {}
    const wrapper = mount(App)
    await nextTick()
    expect(wrapper.findComponent({ name: "DesktopReloadMenu" }).exists()).toBe(true)
    wrapper.unmount()
    delete window.javLibrary
  })
})

import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsAboutSection from "./SettingsAboutSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("./SettingsAppUpdateSection.vue", () => ({
  default: {
    name: "SettingsAppUpdateSection",
    props: ["backendVersionDisplay"],
    template: "<div data-app-update>{{ backendVersionDisplay }}</div>",
  },
}))

vi.mock("./SettingsHomepageDevTools.vue", () => ({
  default: {
    name: "SettingsHomepageDevTools",
    emits: ["refreshed"],
    template: "<button data-homepage-refresh @click=\"$emit('refreshed')\">refresh</button>",
  },
}))

const baseProps = {
  isViteDev: true,
  useWebApi: false,
  viteMode: "test",
  aboutHealth: null,
  aboutHealthLoading: false,
  aboutHealthError: "",
  backendVersionDisplay: "mock",
  backendVersionStatus: "default" as const,
}

describe("SettingsAboutSection", () => {
  it("renders mock data mode and frontend build details in dev mode", () => {
    const wrapper = mount(SettingsAboutSection, {
      props: baseProps,
    })

    expect(wrapper.text()).toContain("settings.aboutCardTitle")
    expect(wrapper.text()).toContain("settings.aboutVersionMock")
    expect(wrapper.text()).toContain("settings.aboutDataModeMock")
    expect(wrapper.text()).toContain('settings.aboutFrontendBuildDev:{"mode":"test"}')
  })

  it("renders web app update status and emits refreshHealth from homepage dev tools", async () => {
    const wrapper = mount(SettingsAboutSection, {
      props: {
        ...baseProps,
        useWebApi: true,
        backendVersionDisplay: "20260501.010203",
      },
    })

    expect(wrapper.get("[data-app-update]").text()).toBe("20260501.010203")

    await wrapper.get("[data-homepage-refresh]").trigger("click")

    expect(wrapper.emitted("refreshHealth")).toHaveLength(1)
  })
})

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
    props: ["backendVersionDisplay", "backendBuildStamp"],
    template: "<div data-app-update>{{ backendVersionDisplay }} {{ backendBuildStamp }}</div>",
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

  it("renders web app update status and the update-preferences slot", () => {
    const wrapper = mount(SettingsAboutSection, {
      props: {
        ...baseProps,
        useWebApi: true,
        backendVersionDisplay: "1.5.7",
        aboutHealth: { name: "curated", version: "1.5.7", buildStamp: "20260501.010203", channel: "release", transport: "http", databasePath: "test.db" },
      },
      slots: { updates: "<div data-auto-update>Auto update</div>" },
    })

    expect(wrapper.get("[data-app-update]").text()).toBe("1.5.7 20260501.010203")
    expect(wrapper.get("[data-auto-update]").text()).toBe("Auto update")
  })

  it.each([true, false])("shows the concise license section in dev=%s", (isViteDev) => {
    const wrapper = mount(SettingsAboutSection, {
      props: { ...baseProps, isViteDev },
    })

    expect(wrapper.text()).toContain("settings.aboutCopyrightValue")
    expect(wrapper.text()).toContain("settings.aboutRepositoryValue")
    expect(wrapper.text()).toContain("settings.aboutUsageLicensesTitle")
    expect(wrapper.text()).toContain("settings.aboutLicenseValue")
    expect(wrapper.text()).toContain("settings.aboutHarmonyFontNotice")
    expect(wrapper.findAll('a[href*="LICENSE"]')).toHaveLength(5)
    expect(wrapper.get("details summary").text()).toContain("settings.aboutThirdPartyTitle")
    expect(wrapper.findAll("details li")).toHaveLength(15)
    expect(wrapper.text()).toContain("FFmpeg")
    expect(wrapper.get('details a[href*="ThirdParty_NOTICES"]').text()).toBe(
      "settings.aboutThirdPartyNoticesLink",
    )
  })
})

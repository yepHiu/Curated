import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsOverviewSection from "./SettingsOverviewSection.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe("SettingsOverviewSection", () => {
  it("renders dashboard stats", () => {
    const wrapper = mount(SettingsOverviewSection, {
      props: {
        dashboardStats: [
          { labelKey: "settings.statMovies", value: "180", detailKey: "settings.statMoviesHint" },
          { labelKey: "settings.statFrames", value: "42" },
        ],
      },
    })

    expect(wrapper.text()).toContain("settings.navOverview")
    expect(wrapper.text()).toContain("settings.statMovies")
    expect(wrapper.text()).toContain("settings.statMoviesHint")
    expect(wrapper.text()).toContain("180")
    expect(wrapper.text()).toContain("42")
  })
})

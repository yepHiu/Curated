import { mount } from "@vue/test-utils"
import { describe, expect, it, vi } from "vitest"
import SettingsWatchTimeHeatmap from "./SettingsWatchTimeHeatmap.vue"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: { value: "en" },
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

vi.mock("lucide-vue-next", () => ({
  CalendarDays: { name: "CalendarDays", template: "<span />" },
}))

describe("SettingsWatchTimeHeatmap", () => {
  it("renders the title and summary metrics without the heatmap grid", () => {
    const wrapper = mount(SettingsWatchTimeHeatmap, {
      props: {
        days: [
          { dayKey: "2026-04-30", watchedSec: 3_600 },
          { dayKey: "2026-05-01", watchedSec: 7_200 },
        ],
        today: new Date(2026, 4, 1),
      },
    })

    expect(wrapper.text()).toContain("settings.watchTimeTitle")
    expect(wrapper.text()).toContain("settings.watchTimeThisWeek")
    expect(wrapper.text()).toContain("settings.watchTimePastThreeMonths")
    expect(wrapper.text()).toContain("3h")
    expect(wrapper.find(".watch-time-canvas").exists()).toBe(false)
    expect(wrapper.find(".watch-time-grid").exists()).toBe(false)
    expect(wrapper.find(".watch-time-legend-cell").exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="watch-time-heatmap-cell"]')).toHaveLength(0)
  })

  it("renders loading, error, and empty states", () => {
    const loading = mount(SettingsWatchTimeHeatmap, {
      props: { days: [], loading: true, today: new Date(2026, 4, 1) },
    })
    expect(loading.text()).toContain("settings.watchTimeLoading")

    const error = mount(SettingsWatchTimeHeatmap, {
      props: {
        days: [],
        error: "network failed",
        today: new Date(2026, 4, 1),
      },
    })
    expect(error.text()).toContain("network failed")

    const empty = mount(SettingsWatchTimeHeatmap, {
      props: { days: [], today: new Date(2026, 4, 1) },
    })
    expect(empty.text()).toContain("settings.watchTimeEmpty")
  })
})

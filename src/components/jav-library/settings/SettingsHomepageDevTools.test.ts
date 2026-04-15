import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsHomepageDevTools from "./SettingsHomepageDevTools.vue"

const refreshHomepageDailyRecommendationsMock = vi.hoisted(() => vi.fn())
const pushAppToastMock = vi.hoisted(() => vi.fn())

vi.mock("@/api/endpoints", () => ({
  api: {
    refreshHomepageDailyRecommendations: refreshHomepageDailyRecommendationsMock,
  },
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: pushAppToastMock,
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

describe("SettingsHomepageDevTools", () => {
  beforeEach(() => {
    refreshHomepageDailyRecommendationsMock.mockReset()
    pushAppToastMock.mockReset()
  })

  it("calls the refresh endpoint and emits refreshed on success", async () => {
    refreshHomepageDailyRecommendationsMock.mockResolvedValue({
      dateUtc: "2026-04-16",
      generatedAt: "2026-04-16T00:00:00Z",
      generationVersion: "v3",
      heroMovieIds: ["m1"],
      recommendationMovieIds: ["m2"],
    })

    const wrapper = mount(SettingsHomepageDevTools)
    await wrapper.get('[data-homepage-dev-refresh]').trigger("click")

    expect(refreshHomepageDailyRecommendationsMock).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted("refreshed")?.[0]?.[0]).toMatchObject({
      dateUtc: "2026-04-16",
      generationVersion: "v3",
    })
    expect(pushAppToastMock).toHaveBeenCalledWith(
      'settings.aboutHomepageRefreshSuccess:{"date":"2026-04-16"}',
      expect.objectContaining({ variant: "success" }),
    )
  })

  it("shows an error toast when refresh fails", async () => {
    refreshHomepageDailyRecommendationsMock.mockRejectedValue(new Error("boom"))

    const wrapper = mount(SettingsHomepageDevTools)
    await wrapper.get('[data-homepage-dev-refresh]').trigger("click")

    expect(refreshHomepageDailyRecommendationsMock).toHaveBeenCalledTimes(1)
    expect(pushAppToastMock).toHaveBeenCalledWith(
      "boom",
      expect.objectContaining({ variant: "destructive" }),
    )
    expect(wrapper.emitted("refreshed")).toBeFalsy()
  })
})

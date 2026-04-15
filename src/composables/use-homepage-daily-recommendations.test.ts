import { flushPromises } from "@vue/test-utils"
import { effectScope, ref } from "vue"
import { describe, expect, it, vi } from "vitest"
import { useHomepageDailyRecommendations } from "./use-homepage-daily-recommendations"
import type { HomepageDailyRecommendationsDTO } from "@/api/types"

function makeSnapshot(
  dateUtc: string,
  heroMovieIds: string[],
  recommendationMovieIds: string[],
): HomepageDailyRecommendationsDTO {
  return {
    dateUtc,
    generatedAt: `${dateUtc}T00:00:01Z`,
    generationVersion: "v1",
    heroMovieIds,
    recommendationMovieIds,
  }
}

describe("useHomepageDailyRecommendations", () => {
  it("loads immediately and refreshes when the UTC day key changes", async () => {
    const utcDayKey = ref("2026-04-15")
    const getHomepageDailyRecommendations = vi.fn()
      .mockResolvedValueOnce(makeSnapshot("2026-04-15", ["m1"], ["m2"]))
      .mockResolvedValueOnce(makeSnapshot("2026-04-16", ["m3"], ["m4"]))

    const scope = effectScope()
    const state = scope.run(() => useHomepageDailyRecommendations({
      utcDayKey,
      libraryService: {
        getHomepageDailyRecommendations,
      },
    }))

    expect(state).toBeTruthy()
    await flushPromises()

    expect(getHomepageDailyRecommendations).toHaveBeenCalledTimes(1)
    expect(state?.snapshot.value).toEqual(makeSnapshot("2026-04-15", ["m1"], ["m2"]))

    utcDayKey.value = "2026-04-16"
    await flushPromises()

    expect(getHomepageDailyRecommendations).toHaveBeenCalledTimes(2)
    expect(state?.snapshot.value).toEqual(makeSnapshot("2026-04-16", ["m3"], ["m4"]))

    scope.stop()
  })

  it("keeps the previous snapshot when a later refresh fails", async () => {
    const utcDayKey = ref("2026-04-15")
    const refreshError = new Error("refresh failed")
    const getHomepageDailyRecommendations = vi.fn()
      .mockResolvedValueOnce(makeSnapshot("2026-04-15", ["m1"], ["m2"]))
      .mockRejectedValueOnce(refreshError)

    const scope = effectScope()
    const state = scope.run(() => useHomepageDailyRecommendations({
      utcDayKey,
      libraryService: {
        getHomepageDailyRecommendations,
      },
    }))

    expect(state).toBeTruthy()
    await flushPromises()

    utcDayKey.value = "2026-04-16"
    await flushPromises()

    expect(getHomepageDailyRecommendations).toHaveBeenCalledTimes(2)
    expect(state?.snapshot.value).toEqual(makeSnapshot("2026-04-15", ["m1"], ["m2"]))
    expect(state?.error.value).toBe(refreshError)

    scope.stop()
  })
})

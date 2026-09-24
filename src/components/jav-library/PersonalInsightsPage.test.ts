import { flushPromises, mount } from "@vue/test-utils"
import { ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"
import type {
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
} from "@/api/types"
import PersonalInsightsPage from "./PersonalInsightsPage.vue"

const serviceMocks = vi.hoisted(() => ({
  getPersonalInsightsOverview: vi.fn(),
  getPersonalInsightsBreakdown: vi.fn(),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => serviceMocks,
}))

vi.mock("@/services/ai-service", () => ({
  useAIService: () => ({
    runAction: vi.fn(),
  }),
}))

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("en"),
    t: (key: string, params?: Record<string, unknown>) => {
      if (!params) return key
      return `${key} ${Object.values(params).join(" ")}`
    },
  }),
}))

function overview(
  range: PersonalInsightsRange = "30d",
  overrides: Partial<PersonalInsightsOverviewDTO> = {},
): PersonalInsightsOverviewDTO {
  return {
    range,
    from: "2026-06-23",
    to: "2026-07-22",
    timezone: "UTC",
    generatedAt: "2026-07-21T16:30:00Z",
    dataSince: "2026-06-01",
    watchedSeconds: 7200,
    startedMovies: 2,
    completedMovies: 1,
    completionRate: 0.5,
    completionThreshold: 0.9,
    ratedMovies: 1,
    averageUserRating: 4.5,
    ...overrides,
  }
}

function breakdown(
  dimension: PersonalInsightsDimension,
  range: PersonalInsightsRange = "30d",
): PersonalInsightsBreakdownDTO {
  return {
    range,
    dimension,
    from: "2026-06-23",
    to: "2026-07-22",
    timezone: "UTC",
    generatedAt: "2026-07-21T16:30:00Z",
    dataSince: "2026-06-01",
    totalWatchedSeconds: 7200,
    attribution: "full-per-entity",
    items: [{ name: `${dimension} A`, watchedSeconds: 3600, movieCount: 1, shareOfTotal: 0.5 }],
    limit: 10,
  }
}

beforeEach(() => {
  serviceMocks.getPersonalInsightsOverview.mockReset()
  serviceMocks.getPersonalInsightsBreakdown.mockReset()
  serviceMocks.getPersonalInsightsOverview.mockResolvedValue(overview())
  serviceMocks.getPersonalInsightsBreakdown.mockImplementation(
    ({ dimension, range }: { dimension: PersonalInsightsDimension; range: PersonalInsightsRange }) =>
      Promise.resolve(breakdown(dimension, range)),
  )
})

describe("PersonalInsightsPage", () => {
  it("loads one overview and three bounded breakdowns with semantic page structure", async () => {
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    expect(wrapper.findAll("h1")).toHaveLength(1)
    expect(wrapper.findAll("[data-insights-metric]")).toHaveLength(6)
    expect(wrapper.findAll("[data-insights-breakdown]")).toHaveLength(3)
    expect(wrapper.get('[data-insights-metric="completion-rate"]').text()).toContain("50%")
    expect(wrapper.text()).not.toContain("NaN")
    expect(wrapper.text()).not.toContain("Infinity")
    expect(serviceMocks.getPersonalInsightsOverview).toHaveBeenCalledWith({
      range: "30d",
      timezone: expect.any(String),
    })
    expect(serviceMocks.getPersonalInsightsBreakdown).toHaveBeenCalledTimes(3)
    expect(serviceMocks.getPersonalInsightsBreakdown).toHaveBeenCalledWith(
      expect.objectContaining({ dimension: "actor", limit: 10 }),
    )
  })

  it("renders unavailable ratios as an em dash and explains the empty denominator", async () => {
    serviceMocks.getPersonalInsightsOverview.mockResolvedValue(overview("30d", {
      watchedSeconds: 0,
      startedMovies: 0,
      completedMovies: 0,
      completionRate: null,
      ratedMovies: 0,
      averageUserRating: null,
      dataSince: null,
    }))
    serviceMocks.getPersonalInsightsBreakdown.mockImplementation(
      ({ dimension }: { dimension: PersonalInsightsDimension }) =>
        Promise.resolve({ ...breakdown(dimension), totalWatchedSeconds: 0, items: [], dataSince: null }),
    )

    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    expect(wrapper.get('[data-insights-metric="completion-rate"]').text()).toContain("—")
    expect(wrapper.get('[data-insights-metric="average-rating"]').text()).toContain("—")
    expect(wrapper.text()).toContain("insights.noCompletionDenominator")
    expect(wrapper.text()).toContain("insights.emptyTitle")
    expect(wrapper.text()).not.toContain("NaN")
    expect(wrapper.text()).not.toContain("Infinity")
  })

  it("shows five ranking items by default and expands one dimension at a time", async () => {
    serviceMocks.getPersonalInsightsBreakdown.mockImplementation(
      ({ dimension }: { dimension: PersonalInsightsDimension }) => Promise.resolve({
        ...breakdown(dimension),
        items: Array.from({ length: 7 }, (_, index) => ({
          name: `${dimension} ${index + 1}`,
          watchedSeconds: 600 - index * 60,
          movieCount: 1,
          shareOfTotal: (600 - index * 60) / 7200,
        })),
      }),
    )

    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    const actor = wrapper.get('[data-insights-breakdown="actor"]')
    const studio = wrapper.get('[data-insights-breakdown="studio"]')
    expect(actor.findAll("[data-slot='progress']")).toHaveLength(5)
    expect(studio.findAll("[data-slot='progress']")).toHaveLength(5)

    await actor.get('[data-insights-expand="actor"]').trigger("click")
    expect(actor.findAll("[data-slot='progress']")).toHaveLength(7)
    expect(actor.get('[data-insights-expand="actor"]').attributes("aria-expanded")).toBe("true")
    expect(studio.findAll("[data-slot='progress']")).toHaveLength(5)

    await actor.get('[data-insights-expand="actor"]').trigger("click")
    expect(actor.findAll("[data-slot='progress']")).toHaveLength(5)
  })

  it("does not let a slower previous range overwrite a newer selection", async () => {
    let resolveFirst: ((value: PersonalInsightsOverviewDTO) => void) | undefined
    serviceMocks.getPersonalInsightsOverview.mockImplementation(
      ({ range }: { range: PersonalInsightsRange }) => {
        if (range === "30d") {
          return new Promise<PersonalInsightsOverviewDTO>((resolve) => { resolveFirst = resolve })
        }
        return Promise.resolve(overview(range, { startedMovies: 2, completedMovies: 0, completionRate: 0 }))
      },
    )
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    await wrapper.get('input[value="90d"]').setValue(true)
    await flushPromises()
    expect(wrapper.get('[data-insights-metric="started"]').text()).toContain("2")

    resolveFirst?.(overview("30d", { startedMovies: 99, completedMovies: 0, completionRate: 0 }))
    await flushPromises()
    expect(wrapper.get('[data-insights-metric="started"]').text()).not.toContain("99")
    expect(serviceMocks.getPersonalInsightsOverview).toHaveBeenLastCalledWith({
      range: "90d",
      timezone: expect.any(String),
    })
  })

  it("keeps the previous range visibly labelled until the selected range loads", async () => {
    let resolveNext: ((value: PersonalInsightsOverviewDTO) => void) | undefined
    serviceMocks.getPersonalInsightsOverview.mockImplementation(
      ({ range }: { range: PersonalInsightsRange }) => range === "90d"
        ? new Promise<PersonalInsightsOverviewDTO>((resolve) => { resolveNext = resolve })
        : Promise.resolve(overview(range)),
    )
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    await wrapper.get('input[value="90d"]').setValue(true)
    await flushPromises()
    expect(wrapper.get("[data-insights-update-status]").text()).toContain("insights.showingPreviousRange")
    expect(wrapper.get('[data-insights-metric="started"]').text()).toContain("2")

    resolveNext?.(overview("90d", { startedMovies: 9, completedMovies: 3, completionRate: 1 / 3 }))
    await flushPromises()
    expect(wrapper.find("[data-insights-update-status]").exists()).toBe(false)
    expect(wrapper.get('[data-insights-metric="started"]').text()).toContain("9")
  })

  it("keeps a successful overview and other rankings when one dimension fails, then retries it", async () => {
    let actorFails = true
    serviceMocks.getPersonalInsightsBreakdown.mockImplementation(
      ({ dimension, range }: { dimension: PersonalInsightsDimension; range: PersonalInsightsRange }) => {
        if (dimension === "actor" && actorFails) return Promise.reject(new Error("actor unavailable"))
        return Promise.resolve(breakdown(dimension, range))
      },
    )
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    expect(wrapper.findAll("[data-insights-metric]")).toHaveLength(6)
    expect(wrapper.get('[data-insights-breakdown="studio"]').text()).toContain("studio A")
    expect(wrapper.get('[data-insights-breakdown="actor"]').text()).toContain("insights.breakdownError")

    actorFails = false
    await wrapper.get('[data-insights-retry-breakdown="actor"]').trigger("click")
    await flushPromises()
    expect(wrapper.get('[data-insights-breakdown="actor"]').text()).toContain("actor A")
    expect(wrapper.find('[data-insights-retry-breakdown="actor"]').exists()).toBe(false)
    expect(serviceMocks.getPersonalInsightsBreakdown).toHaveBeenLastCalledWith(
      expect.objectContaining({ dimension: "actor", range: "30d", limit: 10 }),
    )
  })

  it("shows a grounded fact only when enough watch data supports it", async () => {
    serviceMocks.getPersonalInsightsOverview.mockResolvedValue(overview("30d", {
      watchedSeconds: 7200,
      startedMovies: 6,
    }))
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()
    expect(wrapper.get("[data-insights-fact]").text()).toContain("actor A")

    serviceMocks.getPersonalInsightsOverview.mockResolvedValue(overview("90d", {
      watchedSeconds: 600,
      startedMovies: 2,
    }))
    await wrapper.get('input[value="90d"]').setValue(true)
    await flushPromises()
    expect(wrapper.find("[data-insights-fact]").exists()).toBe(false)
  })

  it("preserves the prior range with an explicit retry if the selected overview fails", async () => {
    serviceMocks.getPersonalInsightsOverview.mockImplementation(
      ({ range }: { range: PersonalInsightsRange }) => range === "90d"
        ? Promise.reject(new Error("offline"))
        : Promise.resolve(overview(range)),
    )
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()

    await wrapper.get('input[value="90d"]').setValue(true)
    await flushPromises()
    expect(wrapper.get("[data-insights-update-status]").text()).toContain("insights.previousRangeFailed")
    expect(wrapper.findAll("[data-insights-metric]")).toHaveLength(6)
    expect(wrapper.find("[data-insights-retry-overview]").exists()).toBe(true)
  })

  it("offers a retry after a failed aggregate request", async () => {
    serviceMocks.getPersonalInsightsOverview.mockRejectedValueOnce(new Error("offline"))
    const wrapper = mount(PersonalInsightsPage)
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)

    serviceMocks.getPersonalInsightsOverview.mockResolvedValueOnce(overview())
    await wrapper.get("button").trigger("click")
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.findAll("[data-insights-metric]")).toHaveLength(6)
  })
})

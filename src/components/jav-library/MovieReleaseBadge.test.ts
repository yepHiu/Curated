import { mount } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import MovieReleaseBadge from "./MovieReleaseBadge.vue"

vi.mock("vue-i18n", () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe("MovieReleaseBadge", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 27, 23, 59, 30))
  })
  afterEach(() => vi.useRealTimers())

  it("changes state when metadata changes and hides unknown dates", async () => {
    const wrapper = mount(MovieReleaseBadge, { props: { releaseDate: "2026-09-28" } })
    expect(wrapper.get('[data-release-status="unreleased"]').text()).toBe("detailPanel.releaseStatus.unreleased")
    await wrapper.setProps({ releaseDate: "2026-09-27" })
    expect(wrapper.get('[data-release-status="released"]').text()).toBe("detailPanel.releaseStatus.released")
    await wrapper.setProps({ releaseDate: "2026-02-30" })
    expect(wrapper.find("[data-release-status]").exists()).toBe(false)
    wrapper.unmount()
  })

  it("updates all badges across local midnight using one shared clock", async () => {
    const first = mount(MovieReleaseBadge, { props: { releaseDate: "2026-09-28" } })
    const second = mount(MovieReleaseBadge, { props: { releaseDate: "2026-09-28" } })
    expect(vi.getTimerCount()).toBe(1)
    await vi.advanceTimersByTimeAsync(60_000)
    expect(first.get("[data-release-status]").attributes("data-release-status")).toBe("released")
    expect(second.get("[data-release-status]").attributes("data-release-status")).toBe("released")
    first.unmount()
    expect(vi.getTimerCount()).toBe(1)
    second.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it.each(["focus", "visibilitychange"])("refreshes immediately on %s after a system date change", async (event) => {
    const wrapper = mount(MovieReleaseBadge, { props: { releaseDate: "2026-09-28" } })
    vi.setSystemTime(new Date(2026, 8, 28, 0, 0))
    const target = event === "focus" ? window : document
    target.dispatchEvent(new Event(event))
    await wrapper.vm.$nextTick()
    expect(wrapper.get("[data-release-status]").attributes("data-release-status")).toBe("released")
    wrapper.unmount()
  })
})

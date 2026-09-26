import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import DevEnvironmentBadge from "./DevEnvironmentBadge.vue"

afterEach(() => vi.unstubAllEnvs())

describe("DevEnvironmentBadge", () => {
  it("does not show developer entries in production", () => {
    vi.stubEnv("DEV", false)
    expect(mount(DevEnvironmentBadge).find("button").exists()).toBe(false)
  })

  it("opens debug tools even when the performance bar is visible", async () => {
    const wrapper = mount(DevEnvironmentBadge)
    await wrapper.get('[aria-label="Open debug tools"]').trigger("click")
    expect(wrapper.emitted("showDebugTools")).toHaveLength(1)
  })

  it("keeps the dev watermark visible without the perf restore action by default", () => {
    const wrapper = mount(DevEnvironmentBadge)

    expect(wrapper.text()).toContain("dev")
    expect(wrapper.text()).not.toContain("perf")
  })

  it("shows a compact perf restore action beside the dev watermark when the monitor is hidden", async () => {
    const wrapper = mount(DevEnvironmentBadge, {
      props: {
        showPerfRestore: true,
      },
    })

    expect(wrapper.text()).toContain("dev")
    expect(wrapper.text()).toContain("perf")

    await wrapper.get('[aria-label="Show performance monitor"]').trigger("click")

    expect(wrapper.emitted("showPerformanceMonitor")).toHaveLength(1)
  })
})

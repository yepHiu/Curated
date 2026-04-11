import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import DevEnvironmentBadge from "./DevEnvironmentBadge.vue"

describe("DevEnvironmentBadge", () => {
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

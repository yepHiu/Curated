import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import DevEnvironmentBadge from "./DevEnvironmentBadge.vue"

afterEach(() => vi.unstubAllEnvs())

describe("DevEnvironmentBadge", () => {
  it("does not show developer entries in production", () => {
    vi.stubEnv("DEV", false)
    expect(mount(DevEnvironmentBadge).find("button").exists()).toBe(false)
  })

  it("opens debug tools", async () => {
    const wrapper = mount(DevEnvironmentBadge)
    await wrapper.get('[aria-label="Open debug tools"]').trigger("click")
    expect(wrapper.emitted("showDebugTools")).toHaveLength(1)
  })

  it("keeps the dev watermark without a separate perf entry", () => {
    const wrapper = mount(DevEnvironmentBadge)

    expect(wrapper.text()).toContain("dev")
    expect(wrapper.text()).not.toContain("perf")
  })
})

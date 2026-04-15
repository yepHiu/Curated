import { mount } from "@vue/test-utils"
import { defineComponent } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"
import { getCurrentUtcDayKey, useCurrentUtcDayKey } from "./current-utc-day-key"

describe("current UTC day key", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("formats the current day in UTC", () => {
    expect(getCurrentUtcDayKey(new Date("2026-04-15T23:59:59+08:00"))).toBe("2026-04-15")
    expect(getCurrentUtcDayKey(new Date("2026-04-15T23:59:59-08:00"))).toBe("2026-04-16")
  })

  it("updates after crossing into a new UTC day", async () => {
    vi.useFakeTimers()

    let current = new Date("2026-04-15T23:59:30.000Z")
    const Probe = defineComponent({
      setup() {
        return useCurrentUtcDayKey({
          pollIntervalMs: 1_000,
          now: () => current,
        })
      },
      template: "<div>{{ dayKey }}</div>",
    })

    const wrapper = mount(Probe)
    expect(wrapper.text()).toBe("2026-04-15")

    current = new Date("2026-04-16T00:00:05.000Z")
    vi.advanceTimersByTime(1_000)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toBe("2026-04-16")
  })
})

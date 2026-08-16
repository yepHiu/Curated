import { mount } from "@vue/test-utils"
import { afterEach, describe, expect, it, vi } from "vitest"
import type { FrameMarkerInput } from "@/lib/player-frame-markers"

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const messages: Record<string, string> = {
        "player.frameMarkerAria": "Jump to curated frame at {time}",
        "player.frameMarkerClusterAria": "Jump to the earliest of {count} curated frames ({time})",
        "player.frameMarkerClusterTip": "{count} curated frames · earliest {time}",
      }
      let text = messages[key] ?? key
      for (const [name, value] of Object.entries(params ?? {})) {
        text = text.split(`{${name}}`).join(String(value))
      }
      return text
    },
  }),
}))

function markers(...positionsSec: number[]): FrameMarkerInput[] {
  return positionsSec.map((positionSec, index) => ({ id: `f${index}`, positionSec }))
}

async function mountComponent(props?: { markers?: FrameMarkerInput[]; durationSec?: number }) {
  const mod = await import("./PlayerProgressFrameMarkers.vue")
  return mount(mod.default, {
    attachTo: document.body,
    props: {
      markers: props?.markers ?? markers(100, 103, 107, 111, 400),
      durationSec: props?.durationSec ?? 600,
    },
  })
}

/** jsdom 无 ResizeObserver；需要模拟"已测得宽度 600px"时安装该桩（1s = 1px） */
function stubMeasuredWidth(widthPx: number) {
  const observers: Array<{ callback: ResizeObserverCallback }> = []
  const fake = class {
    constructor(callback: ResizeObserverCallback) {
      observers.push({ callback })
    }
    observe() {
      for (const observer of observers) {
        observer.callback(
          [{ contentRect: { width: widthPx } } as unknown as ResizeObserverEntry],
          this as unknown as ResizeObserver,
        )
      }
    }
    unobserve() {}
    disconnect() {}
    takeRecords() {
      return []
    }
  }
  vi.stubGlobal("ResizeObserver", fake)
}

afterEach(() => {
  vi.unstubAllGlobals()
  document.body.innerHTML = ""
})

describe("PlayerProgressFrameMarkers", () => {
  it("renders one unmerged tick per marker before width is measured", async () => {
    const wrapper = await mountComponent()

    const ticks = wrapper.findAll("[data-frame-marker]")
    expect(ticks).toHaveLength(5)
    expect(ticks.every((tick) => tick.attributes("data-frame-marker") === "1")).toBe(true)
    expect(wrapper.text()).not.toContain("×")
  })

  it("merges dense markers into a single counted cluster once width is known", async () => {
    stubMeasuredWidth(600)
    const wrapper = await mountComponent()

    const ticks = wrapper.findAll("[data-frame-marker]")
    expect(ticks).toHaveLength(2)
    expect(ticks[0].attributes("data-frame-marker")).toBe("4")
    expect(ticks[0].text()).toContain("×4")
    expect(ticks[1].attributes("data-frame-marker")).toBe("1")
  })

  it("emits seek with the earliest frame of the clicked cluster", async () => {
    stubMeasuredWidth(600)
    const wrapper = await mountComponent()

    await wrapper.find("[data-frame-marker]").trigger("click")
    expect(wrapper.emitted("seek")).toEqual([[100]])
  })

  it("keeps the layer click-through while ticks stay interactive", async () => {
    const wrapper = await mountComponent()

    expect(wrapper.get("[data-slot='player-progress-frame-markers']").classes()).toContain(
      "pointer-events-none",
    )
    expect(wrapper.get("[data-frame-marker]").classes()).toContain("pointer-events-auto")
  })

  it("labels single and clustered ticks for screen readers", async () => {
    stubMeasuredWidth(600)
    const wrapper = await mountComponent()

    const ticks = wrapper.findAll("[data-frame-marker]")
    expect(ticks[0].attributes("aria-label")).toBe(
      "Jump to the earliest of 4 curated frames (01:40)",
    )
    expect(ticks[1].attributes("aria-label")).toBe("Jump to curated frame at 06:40")
  })

  it("shows a hover tooltip and hides it on leave", async () => {
    stubMeasuredWidth(600)
    const wrapper = await mountComponent()

    await wrapper.find("[data-frame-marker]").trigger("mouseenter")
    const tooltip = wrapper.get("[role='status']")
    expect(tooltip.text()).toBe("4 curated frames · earliest 01:40")

    await wrapper.find("[data-frame-marker]").trigger("mouseleave")
    expect(wrapper.find("[role='status']").exists()).toBe(false)
  })

  it("renders nothing interactive when duration is unusable", async () => {
    stubMeasuredWidth(600)
    const wrapper = await mountComponent({ durationSec: 0 })

    expect(wrapper.findAll("[data-frame-marker]")).toHaveLength(0)
  })
})

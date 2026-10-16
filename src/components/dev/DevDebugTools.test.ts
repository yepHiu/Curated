import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { useDevPerformanceMonitor } from "@/composables/use-dev-performance-monitor"
import DevDebugTools from "./DevDebugTools.vue"

vi.mock("@/composables/use-dev-performance-monitor", async () => {
  const { ref } = await vi.importActual<typeof import("vue")>("vue")

  return {
    useDevPerformanceMonitor: vi.fn(() => ({
      useWebApi: false,
      paused: ref(false),
      frontendSnapshot: ref({
        routeName: "library",
        fps: 60,
        longTaskCount30s: 0,
        memoryUsedMB: 128,
        lastRouteChangeMs: 12,
        video: {
          available: false,
          estimatedFps: null,
          waitingCount30s: 0,
          droppedFrames: null,
          totalFrames: null,
          droppedFrameRatePercent: null,
        },
      }),
      requestSnapshot: ref({
        requestCount30s: 0,
        failedRequestCount30s: 0,
        activeRequestCount: 0,
        avgLatencyMs30s: null,
        recentRequests: [],
      }),
      backendHealthStatus: ref("mock"),
      backendHealthLatencyMs: ref(null),
      backendVersion: ref(null),
      backendPerformance: ref({ supported: false }),
      togglePaused: vi.fn(),
      clearStats: vi.fn(),
      copySummary: vi.fn(),
    })),
  }
})

function mountTools() {
  return mount(DevDebugTools, {
    global: {
      stubs: {
        DevDebugDialog: {
          name: "DevDebugDialog",
          props: ["open", "tab"],
          emits: ["update:open", "update:tab"],
          template: '<div v-if="open"><slot name="performance" /></div>',
        },
        Button: {
          template: '<button v-bind="$attrs"><slot /></button>',
        },
        ClipboardCopy: true,
        Pause: true,
        Play: true,
        RotateCcw: true,
      },
    },
  })
}

describe("DevDebugTools", () => {
  beforeEach(() => vi.clearAllMocks())

  it("keeps one monitor across debug dialog reopenings", async () => {
    const wrapper = mountTools()
    await wrapper.setProps({ debugOpen: true })
    expect(wrapper.text()).toContain("Dev Performance Monitor")
    expect(wrapper.text()).toContain("FPS: 60")
    expect(wrapper.text()).toContain("Status: mock")
    await wrapper.setProps({ debugOpen: false })
    await wrapper.setProps({ debugOpen: true })
    expect(useDevPerformanceMonitor).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it("preserves performance controls in the dialog", async () => {
    const wrapper = mountTools()
    await wrapper.setProps({ debugOpen: true })
    const monitor = vi.mocked(useDevPerformanceMonitor).mock.results[0]!.value
    for (const [label, action] of [
      ["Pause", monitor.togglePaused],
      ["Clear", monitor.clearStats],
      ["Copy Summary", monitor.copySummary],
    ] as const) {
      await wrapper.findAll("button").find(button => button.text() === label)!.trigger("click")
      expect(action).toHaveBeenCalledTimes(1)
    }
    wrapper.unmount()
  })
})

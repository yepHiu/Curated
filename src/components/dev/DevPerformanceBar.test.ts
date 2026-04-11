import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import DevPerformanceBar from "./DevPerformanceBar.vue"
import {
  DEV_PERFORMANCE_BAR_HIDDEN_STORAGE_KEY,
  setDevPerformanceBarHidden,
} from "@/lib/dev-performance/visibility"

vi.mock("@/composables/use-dev-performance-monitor", async () => {
  const { ref } = await vi.importActual<typeof import("vue")>("vue")

  return {
    useDevPerformanceMonitor: () => ({
      useWebApi: false,
      expanded: ref(false),
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
      toggleExpanded: vi.fn(),
      togglePaused: vi.fn(),
      clearStats: vi.fn(),
      copySummary: vi.fn(),
    }),
  }
})

function mountBar() {
  return mount(DevPerformanceBar, {
    global: {
      stubs: {
        Teleport: true,
        Button: {
          template: '<button v-bind="$attrs"><slot /></button>',
        },
        ChevronDown: true,
        ChevronUp: true,
        ClipboardCopy: true,
        Cpu: true,
        Pause: true,
        Play: true,
        RotateCcw: true,
        X: true,
      },
    },
  })
}

describe("DevPerformanceBar visibility", () => {
  beforeEach(() => {
    localStorage.clear()
    setDevPerformanceBarHidden(false)
  })

  it("hides the full monitor from its own toolbar and persists the local preference", async () => {
    const wrapper = mountBar()

    expect(wrapper.text()).toContain("Perf")

    await wrapper.get('[aria-label="Hide performance monitor"]').trigger("click")

    expect(localStorage.getItem(DEV_PERFORMANCE_BAR_HIDDEN_STORAGE_KEY)).toBe("true")
    expect(wrapper.text()).not.toContain("CPU")
    expect(wrapper.text()).not.toContain("Show Perf")
  })
})

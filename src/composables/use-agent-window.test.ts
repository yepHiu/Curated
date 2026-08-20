import { beforeEach, describe, expect, it } from "vitest"
import {
  AGENT_WINDOW_HEIGHT,
  AGENT_WINDOW_MIN_HEIGHT,
  AGENT_WINDOW_MIN_WIDTH,
  AGENT_WINDOW_WIDTH,
  useAgentWindow,
} from "./use-agent-window"

const STORAGE_KEY = "curated-agent-window-state-v1"

describe("useAgentWindow", () => {
  beforeEach(() => {
    localStorage.clear()
    const { moveTo, resizeTo } = useAgentWindow()
    moveTo(40, 40)
    resizeTo(AGENT_WINDOW_WIDTH, AGENT_WINDOW_HEIGHT)
  })

  it("persists resized dimensions", () => {
    const { size, resizeTo } = useAgentWindow()
    resizeTo(500, 640)
    expect(size.value).toEqual({ width: 500, height: 640 })
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}")).toMatchObject({
      x: 40,
      y: 40,
      width: 500,
      height: 640,
    })
  })

  it("clamps size to the minimum", () => {
    const { size, resizeTo } = useAgentWindow()
    resizeTo(120, 80)
    expect(size.value.width).toBe(AGENT_WINDOW_MIN_WIDTH)
    expect(size.value.height).toBe(AGENT_WINDOW_MIN_HEIGHT)
  })

  it("does not drop size when only moving", () => {
    const { moveTo, resizeTo } = useAgentWindow()
    resizeTo(480, 600)
    moveTo(64, 72)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}")).toMatchObject({
      x: 64,
      y: 72,
      width: 480,
      height: 600,
    })
  })

  it("persists sidebar open state", () => {
    const { sidebarOpen, setSidebarOpen } = useAgentWindow()
    setSidebarOpen(false)
    expect(sidebarOpen.value).toBe(false)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}")).toMatchObject({
      sidebarOpen: false,
    })
    setSidebarOpen(true)
    expect(sidebarOpen.value).toBe(true)
  })
})

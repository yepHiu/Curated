import { beforeEach, describe, expect, it } from "vitest"
import { AGENT_WINDOW_WIDTH, AGENT_WINDOW_MIN_WIDTH, AGENT_WINDOW_MAX_WIDTH, useAgentWindow } from "./use-agent-window"

const STORAGE_KEY = "curated-agent-panel-state-v1"

describe("useAgentWindow", () => {
  beforeEach(() => {
    localStorage.clear()
    const state = useAgentWindow()
    state.closeWindow()
    state.resizeTo(AGENT_WINDOW_WIDTH)
    state.setSidebarOpen(false)
  })

  it("persists panel width without floating coordinates", () => {
    useAgentWindow().resizeTo(500)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}")).toEqual({ width: 500, sidebarOpen: false })
  })

  it("clamps width and rejects non-finite dimensions", () => {
    const state = useAgentWindow()
    state.resizeTo(120)
    expect(state.width.value).toBe(AGENT_WINDOW_MIN_WIDTH)
    state.resizeTo(2000)
    expect(state.width.value).toBe(AGENT_WINDOW_MAX_WIDTH)
    state.resizeTo(NaN)
    expect(state.width.value).toBe(AGENT_WINDOW_WIDTH)
  })

  it("persists the history sidebar preference", () => {
    const state = useAgentWindow()
    state.setSidebarOpen(true)
    expect(state.sidebarOpen.value).toBe(true)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || "{}").sidebarOpen).toBe(true)
  })

  it("toggles the panel open and closed", () => {
    const state = useAgentWindow()
    state.openWindow()
    expect(state.open.value).toBe(true)
    state.toggleWindow()
    expect(state.open.value).toBe(false)
    state.toggleWindow()
    expect(state.open.value).toBe(true)
  })
})

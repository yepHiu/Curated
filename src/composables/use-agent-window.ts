import { computed, ref, watch } from "vue"
import { useMediaQuery } from "@vueuse/core"
import { useExperimentalAgent } from "@/lib/experimental-agent"

/** Agent panel state is independent of the retired floating-window geometry. */
const PANEL_STATE_KEY = "curated-agent-panel-state-v1"
export const AGENT_WINDOW_WIDTH = 400
export const AGENT_WINDOW_MIN_WIDTH = 320
export const AGENT_WINDOW_MAX_WIDTH = 960
export const AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH = 560
export const AGENT_WINDOW_CHAT_WIDE_MIN = 720

function clampWidth(width: number) {
  return Number.isFinite(width)
    ? Math.min(AGENT_WINDOW_MAX_WIDTH, Math.max(AGENT_WINDOW_MIN_WIDTH, Math.round(width)))
    : AGENT_WINDOW_WIDTH
}

function readState(): { width: number; sidebarOpen: boolean } {
  try {
    const parsed = JSON.parse(localStorage.getItem(PANEL_STATE_KEY) || "{}")
    return {
      width: clampWidth(typeof parsed?.width === "number" ? parsed.width : AGENT_WINDOW_WIDTH),
      sidebarOpen: parsed?.sidebarOpen === true,
    }
  } catch {
    return { width: AGENT_WINDOW_WIDTH, sidebarOpen: false }
  }
}

const initial = readState()
const open = ref(false)
const width = ref(initial.width)
const sidebarOpen = ref(initial.sidebarOpen)

function persistCurrent() {
  try {
    localStorage.setItem(PANEL_STATE_KEY, JSON.stringify({ width: width.value, sidebarOpen: sidebarOpen.value }))
  } catch {
    // Storage may be unavailable; the panel remains usable for this session.
  }
}

const { enabled } = useExperimentalAgent()
watch(enabled, (value) => {
  open.value = value
})

export function useAgentWindow() {
  const isMobileViewport = useMediaQuery("(max-width: 1023px)")

  return {
    open: computed(() => open.value),
    width: computed(() => width.value),
    sidebarOpen: computed(() => sidebarOpen.value),
    isMobileViewport,
    openWindow() { open.value = true },
    closeWindow() { open.value = false },
    toggleWindow() { open.value = !open.value },
    resizeTo(nextWidth: number) {
      width.value = clampWidth(nextWidth)
      persistCurrent()
    },
    setSidebarOpen(next: boolean) {
      sidebarOpen.value = next
      persistCurrent()
    },
  }
}

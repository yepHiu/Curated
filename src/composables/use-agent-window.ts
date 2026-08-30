import { computed, ref, watch } from "vue"
import { useMediaQuery } from "@vueuse/core"
import { useExperimentalAgent } from "@/lib/experimental-agent"

/** Agent Window（浮动对话窗口）状态：开关、位置与尺寸记忆、历史侧栏、移动端全屏判定。 */
const WINDOW_STATE_KEY = "curated-agent-window-state-v1"
export const AGENT_WINDOW_WIDTH = 640
export const AGENT_WINDOW_HEIGHT = 620
export const AGENT_WINDOW_MIN_WIDTH = 320
export const AGENT_WINDOW_MIN_HEIGHT = 360
export const AGENT_WINDOW_SIDEBAR_WIDTH = 220
export const AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH = 560
/** Chat column gets extra side padding once the floating window is this wide. */
export const AGENT_WINDOW_CHAT_WIDE_MIN = 720

interface AgentWindowPosition {
  x: number
  y: number
}

interface AgentWindowSize {
  width: number
  height: number
}

interface AgentWindowState extends AgentWindowPosition, AgentWindowSize {
  sidebarOpen: boolean
}

function clampPosition(p: AgentWindowPosition): AgentWindowPosition {
  if (typeof window === "undefined") return p
  const maxX = Math.max(8, window.innerWidth - 160)
  const maxY = Math.max(8, window.innerHeight - 48)
  return {
    x: Math.min(Math.max(8, Math.round(p.x)), maxX),
    y: Math.min(Math.max(8, Math.round(p.y)), maxY),
  }
}

function clampSize(size: AgentWindowSize, origin: AgentWindowPosition): AgentWindowSize {
  const width = Math.round(size.width)
  const height = Math.round(size.height)
  if (typeof window === "undefined") {
    return {
      width: Math.max(AGENT_WINDOW_MIN_WIDTH, width),
      height: Math.max(AGENT_WINDOW_MIN_HEIGHT, height),
    }
  }
  const maxWidth = Math.max(120, window.innerWidth - origin.x - 8)
  const maxHeight = Math.max(160, window.innerHeight - origin.y - 8)
  return {
    width: Math.min(Math.max(AGENT_WINDOW_MIN_WIDTH, width), maxWidth),
    height: Math.min(Math.max(AGENT_WINDOW_MIN_HEIGHT, height), maxHeight),
  }
}

function defaultPosition(): AgentWindowPosition {
  if (typeof window === "undefined") return { x: 64, y: 64 }
  return clampPosition({
    x: window.innerWidth - AGENT_WINDOW_WIDTH - 32,
    y: Math.max(16, Math.min(96, window.innerHeight - AGENT_WINDOW_HEIGHT - 32)),
  })
}

function defaultSize(): AgentWindowSize {
  return { width: AGENT_WINDOW_WIDTH, height: AGENT_WINDOW_HEIGHT }
}

function readState(): AgentWindowState {
  const fallbackPosition = defaultPosition()
  const fallbackSize = defaultSize()
  try {
    const raw = localStorage.getItem(WINDOW_STATE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AgentWindowState>
      if (typeof parsed.x === "number" && typeof parsed.y === "number") {
        const nextPosition = clampPosition({ x: parsed.x, y: parsed.y })
        const nextSize = clampSize(
          {
            width: typeof parsed.width === "number" ? parsed.width : fallbackSize.width,
            height: typeof parsed.height === "number" ? parsed.height : fallbackSize.height,
          },
          nextPosition,
        )
        const sidebarOpen =
          typeof parsed.sidebarOpen === "boolean"
            ? parsed.sidebarOpen
            : nextSize.width >= AGENT_WINDOW_SIDEBAR_INLINE_MIN_WIDTH
        return { ...nextPosition, ...nextSize, sidebarOpen }
      }
    }
  } catch {
    // ignore malformed local storage
  }
  return { ...fallbackPosition, ...fallbackSize, sidebarOpen: true }
}

function persistState(state: AgentWindowState) {
  try {
    localStorage.setItem(WINDOW_STATE_KEY, JSON.stringify(state))
  } catch {
    // ignore storage failures
  }
}

const initial = readState()
const open = ref(false)
const position = ref<AgentWindowPosition>({ x: initial.x, y: initial.y })
const size = ref<AgentWindowSize>({ width: initial.width, height: initial.height })
const sidebarOpen = ref(initial.sidebarOpen)

function persistCurrent() {
  persistState({ ...position.value, ...size.value, sidebarOpen: sidebarOpen.value })
}

const { enabled } = useExperimentalAgent()
watch(enabled, (value) => {
  if (value) open.value = true
})

export function useAgentWindow() {
  const isMobileViewport = useMediaQuery("(max-width: 767px)")

  return {
    open: computed(() => open.value),
    position: computed(() => position.value),
    size: computed(() => size.value),
    sidebarOpen: computed(() => sidebarOpen.value),
    isMobileViewport,
    openWindow() {
      open.value = true
    },
    closeWindow() {
      open.value = false
    },
    toggleWindow() {
      open.value = !open.value
    },
    moveTo(x: number, y: number) {
      position.value = clampPosition({ x, y })
      persistCurrent()
    },
    resizeTo(width: number, height: number) {
      size.value = clampSize({ width, height }, position.value)
      persistCurrent()
    },
    setSidebarOpen(next: boolean) {
      sidebarOpen.value = next
      persistCurrent()
    },
  }
}

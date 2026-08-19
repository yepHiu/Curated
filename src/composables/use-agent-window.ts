import { computed, ref, watch } from "vue"
import { useMediaQuery } from "@vueuse/core"
import { useExperimentalAgent } from "@/lib/experimental-agent"

/** Agent Window（浮动对话窗口）状态：开关、位置记忆与移动端全屏判定。 */
const WINDOW_STATE_KEY = "curated-agent-window-state-v1"
export const AGENT_WINDOW_WIDTH = 420
export const AGENT_WINDOW_HEIGHT = 560

interface AgentWindowState {
  x: number
  y: number
}

function clampPosition(p: AgentWindowState): AgentWindowState {
  if (typeof window === "undefined") return p
  // 至少保留窗口的一部分在视口内，避免拖出后无法找回
  const maxX = Math.max(8, window.innerWidth - 160)
  const maxY = Math.max(8, window.innerHeight - 48)
  return {
    x: Math.min(Math.max(8, Math.round(p.x)), maxX),
    y: Math.min(Math.max(8, Math.round(p.y)), maxY),
  }
}

function defaultPosition(): AgentWindowState {
  if (typeof window === "undefined") return { x: 64, y: 64 }
  return clampPosition({
    x: window.innerWidth - AGENT_WINDOW_WIDTH - 32,
    y: Math.max(16, Math.min(96, window.innerHeight - AGENT_WINDOW_HEIGHT - 32)),
  })
}

function readPosition(): AgentWindowState {
  try {
    const raw = localStorage.getItem(WINDOW_STATE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AgentWindowState>
      if (typeof parsed.x === "number" && typeof parsed.y === "number") {
        return clampPosition({ x: parsed.x, y: parsed.y })
      }
    }
  } catch {
    // ignore malformed local storage
  }
  return defaultPosition()
}

function persistPosition(p: AgentWindowState) {
  try {
    localStorage.setItem(WINDOW_STATE_KEY, JSON.stringify(p))
  } catch {
    // ignore storage failures
  }
}

const open = ref(false)
const position = ref<AgentWindowState>(readPosition())

// 开关从关到开时自动弹出窗口（设置页 toggle → 立即看到 Agent Window）
const { enabled } = useExperimentalAgent()
watch(enabled, (value) => {
  if (value) open.value = true
})

export function useAgentWindow() {
  const isMobileViewport = useMediaQuery("(max-width: 767px)")

  return {
    open: computed(() => open.value),
    position: computed(() => position.value),
    isMobileViewport,
    openWindow() {
      open.value = true
    },
    closeWindow() {
      open.value = false
    },
    moveTo(x: number, y: number) {
      const next = clampPosition({ x, y })
      position.value = next
      persistPosition(next)
    },
  }
}

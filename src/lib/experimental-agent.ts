import { computed, ref } from "vue"

/**
 * 实验性 Agent 功能门控（E1，见 docs/plan/2026-08-19-agent-milestone-plan.md §1）。
 * 每浏览器状态（localStorage）；E4 毕业时迁为后端全局设置键。
 * gating 是纯前端呈现层：后端 /api/ai/* 端点始终注册且受 PIN 保护。
 */
const STORAGE_KEY = "curated-agent-experimental-v1"

function readEnabled(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "true"
  } catch {
    return false
  }
}

function persistEnabled(value: boolean) {
  try {
    localStorage.setItem(STORAGE_KEY, value ? "true" : "false")
  } catch {
    // ignore storage failures
  }
}

const enabled = ref(readEnabled())

export function useExperimentalAgent() {
  return {
    /** Agent UI 总开关；关闭时顶栏入口、就地按钮、Agent Window 一律不渲染 */
    enabled: computed(() => enabled.value),
    setEnabled(value: boolean) {
      enabled.value = value
      persistEnabled(value)
    },
  }
}

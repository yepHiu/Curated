import { computed, ref } from "vue"
import type { AIGovernanceSettings } from "@/services/contracts/ai-governance-service"

/**
 * Shared AI presentation state. Web starts disabled until global settings load.
 * The legacy storage key is only used for Mock compatibility.
 * Backend permissions are enforced independently of this presentation state.
 */
const STORAGE_KEY = "curated-agent-experimental-v1"

function readEnabled(): boolean {
  if (import.meta.env.VITE_USE_WEB_API === "true") return false
  try {
    const governance = localStorage.getItem("curated-ai-governance-mock-v1")
    if (governance) return JSON.parse(governance).enabled === true
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
const readOnly = ref(readMockReadOnly())
function readMockReadOnly(): boolean {
  if (import.meta.env.VITE_USE_WEB_API === "true") return false
  try { return JSON.parse(localStorage.getItem("curated-ai-governance-mock-v1") ?? "{}").readOnly === true }
  catch { return false }
}

export function applyAIGovernance(settings: AIGovernanceSettings) {
  enabled.value = settings.enabled
  readOnly.value = settings.readOnly
}

export function useExperimentalAgent() {
  return {
    /** Agent UI 总开关；关闭时顶栏入口、就地按钮、Agent Window 一律不渲染 */
    enabled: computed(() => enabled.value),
    writeEnabled: computed(() => enabled.value && !readOnly.value),
    setEnabled(value: boolean) {
      enabled.value = value
      persistEnabled(value)
    },
  }
}

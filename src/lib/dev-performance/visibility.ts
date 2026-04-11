import { ref } from "vue"

export const DEV_PERFORMANCE_BAR_HIDDEN_STORAGE_KEY = "curated-dev-performance-bar-hidden-v1"

function loadHiddenPreference(): boolean {
  if (typeof localStorage === "undefined") {
    return false
  }
  return localStorage.getItem(DEV_PERFORMANCE_BAR_HIDDEN_STORAGE_KEY) === "true"
}

export const devPerformanceBarHidden = ref(loadHiddenPreference())

export function setDevPerformanceBarHidden(nextHidden: boolean) {
  devPerformanceBarHidden.value = nextHidden
  if (typeof localStorage === "undefined") {
    return
  }
  try {
    localStorage.setItem(DEV_PERFORMANCE_BAR_HIDDEN_STORAGE_KEY, nextHidden ? "true" : "false")
  } catch {
    // Ignore private-mode/quota failures; the in-memory state still works for this page.
  }
}

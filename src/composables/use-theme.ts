import { computed, onMounted, ref, watch } from "vue"
import {
  applyThemeToDocument,
  getStoredThemePreference,
  setStoredThemePreference,
  type ThemePreference,
  isDarkForPreference,
} from "@/lib/theme-storage"

function readInitialPreference(): ThemePreference {
  if (typeof localStorage === "undefined") return "dark"
  return getStoredThemePreference()
}

const themePreference = ref<ThemePreference>(readInitialPreference())

let watchStarted = false
let systemListenerAttached = false

function attachSystemColorSchemeListener(): void {
  if (systemListenerAttached || typeof window === "undefined") return
  systemListenerAttached = true
  const mq = window.matchMedia("(prefers-color-scheme: dark)")
  mq.addEventListener("change", () => {
    if (themePreference.value === "system") {
      applyThemeToDocument("system")
    }
  })
}

function startWatch(): void {
  if (watchStarted) return
  watchStarted = true
  watch(
    themePreference,
    (p) => {
      setStoredThemePreference(p)
      applyThemeToDocument(p)
    },
    { flush: "sync", immediate: true },
  )
}

/**
 * 全局主题：与 `curated-ui-theme` localStorage、`html.dark` 同步。
 * 多处调用共享同一 `themePreference`；仅在首次调用时注册 watch 与系统配色监听。
 */
export function useTheme() {
  startWatch()
  onMounted(() => {
    themePreference.value = getStoredThemePreference()
    applyThemeToDocument(themePreference.value)
    attachSystemColorSchemeListener()
  })

  const resolvedMode = computed<"light" | "dark">(() =>
    isDarkForPreference(themePreference.value) ? "dark" : "light",
  )

  function setThemePreference(p: ThemePreference) {
    themePreference.value = p
  }

  return {
    themePreference,
    resolvedMode,
    setThemePreference,
  }
}

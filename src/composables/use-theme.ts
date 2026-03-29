import { computed, nextTick, onMounted, ref, watch } from "vue"
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
/** 首屏与同步 watch 完成前不跑主题过渡，避免加载时闪一下 */
let themeViewTransitionAllowed = false

function attachSystemColorSchemeListener(): void {
  if (systemListenerAttached || typeof window === "undefined") return
  systemListenerAttached = true
  const mq = window.matchMedia("(prefers-color-scheme: dark)")
  mq.addEventListener("change", () => {
    if (themePreference.value === "system") {
      applyThemeToDocument("system", {
        viewTransition: themeViewTransitionAllowed,
      })
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
      applyThemeToDocument(p, { viewTransition: themeViewTransitionAllowed })
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
    void nextTick(() => {
      themeViewTransitionAllowed = true
    })
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

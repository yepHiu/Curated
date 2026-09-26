import { ref } from "vue"
import type { DesktopInfo, DesktopUpdateResult } from "../../electron/desktop-contract"

export function useDesktopUpdate() {
  const bridge = window.javLibrary
  const available = !!bridge?.getDesktopInfo
  const info = ref<DesktopInfo | null>(null)
  const result = ref<DesktopUpdateResult | null>(null)
  const loading = ref(false)
  const infoError = ref(false)

  async function load() {
    if (!bridge?.getDesktopInfo) return
    infoError.value = false
    try {
      info.value = await bridge.getDesktopInfo()
      result.value = info.value.development ? { status: "development" } : info.value.distribution === "legacy" ? { status: "bundled" } : null
    } catch {
      infoError.value = true
    }
  }

  async function check() {
    if (loading.value || !bridge?.checkDesktopUpdate) return
    loading.value = true
    try {
      if (!info.value) await load()
      result.value = await bridge.checkDesktopUpdate()
    } catch {
      result.value = { status: "error" }
    } finally {
      loading.value = false
    }
  }

  return { available, info, infoError, result, loading, load, check }
}

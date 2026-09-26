import { computed, onMounted, onUnmounted, ref } from "vue"
import type { DesktopServerConnections } from "../../electron/desktop-contract"

export function useServerConnections() {
  const bridge = window.javLibrary
  const available = Boolean(bridge?.getServerConnections && bridge?.openServerConnections)
  const snapshot = ref<DesktopServerConnections | null>(null)
  const loading = ref(false)
  const failed = ref(false)
  const actionFailed = ref(false)
  const opening = ref(false)
  let disposed = false
  let timer: ReturnType<typeof setInterval> | undefined

  async function refresh() {
    if (!available || !bridge?.getServerConnections || loading.value || disposed) return
    loading.value = true
    try {
      const result = await bridge.getServerConnections()
      if (disposed) return
      snapshot.value = result
      failed.value = false
    } catch {
      if (!disposed) failed.value = true
    } finally {
      if (!disposed) loading.value = false
    }
  }

  async function openManager(serverId?: string) {
    if (opening.value || snapshot.value?.connecting || !bridge?.openServerConnections) return
    opening.value = true
    actionFailed.value = false
    try {
      await bridge.openServerConnections(serverId)
    } catch {
      if (!disposed) actionFailed.value = true
    } finally {
      if (!disposed) {
        opening.value = false
        await refresh()
      }
    }
  }

  onMounted(() => {
    if (!available) return
    void refresh()
    window.addEventListener("focus", refresh)
    timer = setInterval(() => { if (document.visibilityState !== "hidden") void refresh() }, 5000)
  })
  onUnmounted(() => {
    disposed = true
    window.removeEventListener("focus", refresh)
    clearInterval(timer)
  })

  return {
    available, snapshot, loading, failed, actionFailed, opening, refresh, openManager,
    currentServer: computed(() => snapshot.value?.servers.find(server => server.url === snapshot.value?.currentServerUrl)),
  }
}

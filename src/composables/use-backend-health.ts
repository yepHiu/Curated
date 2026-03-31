import { computed, onMounted, onUnmounted, ref } from "vue"
import { api } from "@/api/endpoints"
import type { HealthDTO } from "@/api/types"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const POLL_MS = 30_000
const RECHECK_MIN_SPIN_MS = 500

export type BackendHealthStatus = "mock" | "checking" | "online" | "offline"

export function useBackendHealth() {
  const status = ref<BackendHealthStatus>(USE_WEB ? "checking" : "mock")
  const probing = ref(false)
  const health = ref<HealthDTO | null>(null)
  const versionDisplay = computed(() => {
    const current = health.value
    if (!current?.version) {
      return null
    }
    return current.channel ? `${current.version} (${current.channel})` : current.version
  })

  let timer: ReturnType<typeof setInterval> | undefined

  async function probe(silent: boolean) {
    if (!USE_WEB) {
      return
    }
    const spinStartedAt = Date.now()
    if (!silent) {
      probing.value = true
    }
    try {
      health.value = await api.health()
      status.value = "online"
    } catch {
      health.value = null
      status.value = "offline"
    } finally {
      if (!silent) {
        const elapsed = Date.now() - spinStartedAt
        if (elapsed < RECHECK_MIN_SPIN_MS) {
          await new Promise((resolve) => setTimeout(resolve, RECHECK_MIN_SPIN_MS - elapsed))
        }
        probing.value = false
      }
    }
  }

  function checkNow() {
    void probe(false)
  }

  onMounted(() => {
    if (!USE_WEB) {
      return
    }
    void probe(true)
    timer = setInterval(() => {
      void probe(true)
    }, POLL_MS)
  })

  onUnmounted(() => {
    if (timer !== undefined) {
      clearInterval(timer)
    }
  })

  return {
    useWebApi: USE_WEB,
    status,
    probing,
    health,
    versionDisplay,
    checkNow,
  }
}

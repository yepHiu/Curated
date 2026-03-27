import { onMounted, onUnmounted, ref } from "vue"
import { api } from "@/api/endpoints"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const POLL_MS = 30_000
/** 手动点击「重新检测」时至少保持转动提示的毫秒数，避免请求过快结束看不到动画 */
const RECHECK_MIN_SPIN_MS = 500

export type BackendHealthStatus = "mock" | "checking" | "online" | "offline"

/**
 * 轮询 GET /api/health，用于侧栏等处的后端在线状态。
 * Mock 模式（未启用 VITE_USE_WEB_API）不发起请求，状态恒为 mock。
 */
export function useBackendHealth() {
  const status = ref<BackendHealthStatus>(USE_WEB ? "checking" : "mock")
  const probing = ref(false)

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
      await api.health()
      status.value = "online"
    } catch {
      status.value = "offline"
    } finally {
      if (!silent) {
        const elapsed = Date.now() - spinStartedAt
        if (elapsed < RECHECK_MIN_SPIN_MS) {
          await new Promise((r) => setTimeout(r, RECHECK_MIN_SPIN_MS - elapsed))
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
    checkNow,
  }
}

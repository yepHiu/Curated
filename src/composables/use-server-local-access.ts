import { computed, onMounted, ref } from "vue"
import { devRemoteSimulation } from "@/lib/dev-remote-simulation"
import { api } from "@/api/endpoints"
import { resolveApiBaseUrl } from "@/api/http-client"
import { isLocalServerTarget } from "@/lib/app-update-target"

/** 联合请求级本机信号与实际连接目标；未知或远程连接默认隐藏本机管理选项。 */
export function useServerLocalAccess() {
  const useWeb = import.meta.env.VITE_USE_WEB_API === "true"
  const isServerLocal = ref(!useWeb)
  const isAccessReady = ref(!useWeb)

  onMounted(async () => {
    // Mock 无服务端；真实连接需等待 health 和 Desktop 主进程确认。
    if (!useWeb) return
    try {
      const desktop = !!window.javLibrary
      const [health, info] = await Promise.all([
        api.health(),
        desktop ? window.javLibrary?.getDesktopInfo?.() : null,
      ])
      // 当前 health 的路径管理能力由服务端直接 loopback 判定提供，复用作本机信号。
      // 这里只控制 UI，不为其它 API 赋予权限。
      isServerLocal.value = health.canManageLibraryPaths === true && isLocalServerTarget(
        resolveApiBaseUrl(import.meta.env), window.location.origin, desktop, info ?? null,
      )
    } catch {
      isServerLocal.value = false
    } finally {
      isAccessReady.value = true
    }
  })

  return {
    isAccessReady: computed(() => isAccessReady.value),
    isServerLocal: computed(() => !devRemoteSimulation.value && isServerLocal.value),
  }
}

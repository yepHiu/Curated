import { onMounted, readonly, ref } from "vue"
import { api } from "@/api/endpoints"
import { resolveApiBaseUrl } from "@/api/http-client"
import { isLocalServerTarget } from "@/lib/app-update-target"

/** Directory configuration belongs to Server. Mock data stays editable. */
export function useLibraryPathAccess() {
  const useWeb = import.meta.env.VITE_USE_WEB_API === "true"
  const canManagePaths = ref(!useWeb)

  onMounted(async () => {
    if (!useWeb) return
    try {
      const desktop = !!window.javLibrary
      const [health, info] = await Promise.all([
        api.health(),
        desktop ? window.javLibrary?.getDesktopInfo?.() : null,
      ])
      canManagePaths.value = health.canManageLibraryPaths === true && isLocalServerTarget(
        resolveApiBaseUrl(import.meta.env), window.location.origin, desktop, info ?? null,
      )
    } catch {
      canManagePaths.value = false
    }
  })

  return { canManagePaths: readonly(canManagePaths) }
}

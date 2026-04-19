import { computed, readonly, ref } from "vue"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { AppUpdateStatusDTO } from "@/api/types"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const AUTO_CHECK_DELAY_MS = 12_000

export type AppUpdateUiStatus =
  | "idle"
  | "checking"
  | "unsupported"
  | "up-to-date"
  | "update-available"
  | "error"

const summary = ref<AppUpdateStatusDTO | null>(
  USE_WEB
    ? null
    : {
        supported: false,
        status: "unsupported",
        errorMessage: "Web API disabled",
      },
)
const status = ref<AppUpdateUiStatus>(USE_WEB ? "idle" : "unsupported")
const loading = ref(false)
const loaded = ref(!USE_WEB)
const errorMessage = ref("")

let requestSeq = 0
let autoCheckScheduled = false

async function runRequest(kind: "status" | "check", options?: { silent?: boolean }) {
  if (!USE_WEB) {
    status.value = "unsupported"
    loaded.value = true
    return summary.value
  }

  const requestId = ++requestSeq
  const silent = options?.silent ?? false
  if (!silent) {
    loading.value = true
    status.value = "checking"
  }

  try {
    const next =
      kind === "check" ? await api.checkAppUpdateNow() : await api.getAppUpdateStatus()
    if (requestId !== requestSeq) {
      return summary.value
    }

    summary.value = next
    errorMessage.value = next.errorMessage?.trim() ?? ""
    status.value = next.status
    loaded.value = true
    return next
  } catch (err) {
    if (requestId !== requestSeq) {
      return summary.value
    }

    const message =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message
          ? err.message
          : "Failed to check app update"

    errorMessage.value = message
    status.value = "error"
    loaded.value = true
    summary.value = {
      supported: true,
      status: "error",
      errorMessage: message,
      releaseUrl: summary.value?.releaseUrl,
      installedVersion: summary.value?.installedVersion,
      latestVersion: summary.value?.latestVersion,
      checkedAt: summary.value?.checkedAt,
    }
    return summary.value
  } finally {
    if (!silent && requestId === requestSeq) {
      loading.value = false
    }
  }
}

function ensureLoaded() {
  if (!USE_WEB || loaded.value || loading.value) {
    return
  }
  void runRequest("status")
}

function checkNow() {
  return runRequest("check")
}

function scheduleAutoCheck() {
  if (!USE_WEB || autoCheckScheduled) {
    return
  }
  autoCheckScheduled = true
  window.setTimeout(() => {
    if (loaded.value || loading.value) {
      return
    }
    void runRequest("status", { silent: true })
  }, AUTO_CHECK_DELAY_MS)
}

export function useAppUpdate() {
  scheduleAutoCheck()

  return {
    useWebApi: USE_WEB,
    summary: readonly(summary),
    status: readonly(status),
    loading: readonly(loading),
    loaded: readonly(loaded),
    errorMessage: readonly(errorMessage),
    hasUpdateBadge: computed(() => status.value === "update-available" && summary.value?.hasUpdate === true),
    ensureLoaded,
    checkNow,
  }
}

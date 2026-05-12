import { computed, readonly, ref } from "vue"
import { api } from "@/api/endpoints"
import { HttpClientError } from "@/api/http-client"
import type { AppUpdateInstallBody, AppUpdateStatusDTO } from "@/api/types"
import { i18n } from "@/i18n"
import { useNotificationCenter } from "@/composables/use-notification-center"

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
const downloading = ref(false)
const installing = ref(false)
const loaded = ref(!USE_WEB)
const errorMessage = ref("")

let requestSeq = 0
let autoCheckScheduled = false
const notifiedUpdateVersions = new Set<string>()
const autoDownloadAttemptedVersions = new Set<string>()

function maybeRecordUpdateAvailableNotification(next: AppUpdateStatusDTO) {
  if (next.status !== "update-available" || next.hasUpdate !== true) {
    return
  }
  const version = next.latestVersion?.trim() || next.releaseUrl?.trim() || "unknown"
  if (notifiedUpdateVersions.has(version)) {
    return
  }
  notifiedUpdateVersions.add(version)
  useNotificationCenter().addNotification({
    type: "update",
    severity: "warning",
    title: i18n.global.t("notificationCenter.titles.updateAvailable"),
    message: i18n.global.t("settings.appUpdateToastAvailable", {
      version: next.latestVersion ?? "-",
    }),
    source: { route: "/settings?section=about" },
  })
}

function applySummary(next: AppUpdateStatusDTO) {
  summary.value = next
  errorMessage.value = next.errorMessage?.trim() ?? next.lastInstallError?.trim() ?? ""
  status.value = next.status
  loaded.value = true
}

function errorSummary(message: string, previous: AppUpdateStatusDTO | null): AppUpdateStatusDTO {
  return {
    supported: true,
    status: "error",
    errorMessage: message,
    releaseUrl: previous?.releaseUrl,
    installerDownloadUrl: previous?.installerDownloadUrl,
    installerSha256: previous?.installerSha256,
    installedVersion: previous?.installedVersion,
    latestVersion: previous?.latestVersion,
    checkedAt: previous?.checkedAt,
    publishedAt: previous?.publishedAt,
    releaseName: previous?.releaseName,
    releaseNotesSnippet: previous?.releaseNotesSnippet,
    source: previous?.source,
    artifactStatus: previous?.artifactStatus,
    downloadedVersion: previous?.downloadedVersion,
    downloadedFileName: previous?.downloadedFileName,
    downloadedBytes: previous?.downloadedBytes,
    totalBytes: previous?.totalBytes,
    downloadProgress: previous?.downloadProgress,
    signatureStatus: previous?.signatureStatus,
    installReady: previous?.installReady,
    lastInstallAttemptAt: previous?.lastInstallAttemptAt,
    lastInstallError: previous?.lastInstallError,
  }
}

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

    applySummary(next)
    if (kind === "status") {
      maybeRecordUpdateAvailableNotification(next)
    }
    return next
  } catch (err) {
    if (requestId !== requestSeq) {
      return summary.value
    }

    const previous = summary.value
    const message =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message
          ? err.message
          : "Failed to check app update"

    errorMessage.value = message
    status.value = "error"
    loaded.value = true
    summary.value = errorSummary(message, previous)
    return summary.value
  } finally {
    if (!silent && requestId === requestSeq) {
      loading.value = false
    }
  }
}

async function runMutation(
  request: () => Promise<AppUpdateStatusDTO>,
  busyRef: typeof downloading | typeof installing,
) {
  if (!USE_WEB) {
    status.value = "unsupported"
    loaded.value = true
    return summary.value
  }

  const requestId = ++requestSeq
  busyRef.value = true

  try {
    const next = await request()
    if (requestId !== requestSeq) {
      return summary.value
    }
    applySummary(next)
    return next
  } catch (err) {
    if (requestId !== requestSeq) {
      return summary.value
    }
    const previous = summary.value
    const message =
      err instanceof HttpClientError && err.apiError?.message
        ? err.apiError.message
        : err instanceof Error && err.message
          ? err.message
          : "Failed to update app"

    errorMessage.value = message
    status.value = "error"
    loaded.value = true
    summary.value = errorSummary(message, previous)
    return summary.value
  } finally {
    if (requestId === requestSeq) {
      busyRef.value = false
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

function checkNowSilent() {
  return runRequest("check", { silent: true })
}

function downloadInstaller() {
  return runMutation(() => api.downloadAppUpdateInstaller(), downloading)
}

function installUpdate(mode: AppUpdateInstallBody["mode"] = "interactive") {
  return runMutation(() => api.installAppUpdate({ mode }), installing)
}

function clearDownloadedInstaller() {
  return runMutation(() => api.clearDownloadedAppUpdateInstaller(), downloading)
}

function autoDownloadVersionKey(next: AppUpdateStatusDTO): string {
  return next.latestVersion?.trim() || next.releaseUrl?.trim() || ""
}

function canAutoDownloadInstaller(next: AppUpdateStatusDTO | null): next is AppUpdateStatusDTO {
  if (!next) return false
  if (next.status !== "update-available" || next.hasUpdate !== true) return false
  if (next.installReady === true || next.artifactStatus === "verified") return false
  if (!next.installerDownloadUrl?.trim() || !next.installerSha256?.trim()) return false
  return autoDownloadVersionKey(next) !== ""
}

async function maybeAutoDownloadInstaller(next: AppUpdateStatusDTO | null) {
  if (!canAutoDownloadInstaller(next)) {
    return
  }
  const versionKey = autoDownloadVersionKey(next)
  if (autoDownloadAttemptedVersions.has(versionKey)) {
    return
  }
  autoDownloadAttemptedVersions.add(versionKey)

  try {
    const settings = await api.getSettings()
    if (settings.autoDownloadUpdates !== true) {
      return
    }
    await downloadInstaller()
  } catch {
    // Auto-download is opportunistic. About keeps the manual download/install CTA available.
  }
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
    void runRequest("status", { silent: true }).then((next) => maybeAutoDownloadInstaller(next))
  }, AUTO_CHECK_DELAY_MS)
}

export function useAppUpdate() {
  scheduleAutoCheck()

  return {
    useWebApi: USE_WEB,
    summary: readonly(summary),
    status: readonly(status),
    loading: readonly(loading),
    downloading: readonly(downloading),
    installing: readonly(installing),
    loaded: readonly(loaded),
    errorMessage: readonly(errorMessage),
    hasUpdateBadge: computed(() => status.value === "update-available" && summary.value?.hasUpdate === true),
    ensureLoaded,
    checkNow,
    checkNowSilent,
    downloadInstaller,
    installUpdate,
    clearDownloadedInstaller,
  }
}

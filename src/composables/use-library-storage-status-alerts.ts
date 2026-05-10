import { onMounted } from "vue"
import { useI18n } from "vue-i18n"
import type { LibraryPathStorageStatusDTO } from "@/api/types"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const STORAGE_ALERT_SEEN_KEY = "curated-library-storage-status-alert-seen"
const SETTINGS_LIBRARY_ROUTE = "/settings?section=library"

function abnormal(status: LibraryPathStorageStatusDTO): boolean {
  return status.status !== "online"
}

function statusKey(status: LibraryPathStorageStatusDTO): string {
  return [status.libraryPathId, status.status, status.currentVolumeId ?? "", status.message].join("\u001f")
}

function readSeen(): Set<string> {
  try {
    const raw = sessionStorage.getItem(STORAGE_ALERT_SEEN_KEY)
    if (!raw) return new Set()
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return new Set()
    return new Set(arr.filter((item): item is string => typeof item === "string" && item.length > 0))
  } catch {
    return new Set()
  }
}

function writeSeen(seen: Set<string>) {
  try {
    sessionStorage.setItem(STORAGE_ALERT_SEEN_KEY, JSON.stringify([...seen].slice(-200)))
  } catch {
    // session storage can be unavailable in private mode.
  }
}

export function useLibraryStorageStatusAlerts() {
  const { t } = useI18n()
  const libraryService = useLibraryService()

  async function checkAndNotify() {
    await libraryService.checkLibraryPathStorageStatus()
    const offline = libraryService.libraryPathStorageStatuses.value.filter(abnormal)
    if (offline.length === 0) {
      return
    }

    const seen = readSeen()
    const fresh = offline.filter((status) => !seen.has(statusKey(status)))
    if (fresh.length === 0) {
      return
    }
    for (const status of fresh) {
      seen.add(statusKey(status))
    }
    writeSeen(seen)

    if (fresh.length === 1) {
      const item = fresh[0]
      pushAppToast(
        t("toasts.storagePathOffline", {
          title: item.title || item.path,
          message: item.message,
        }),
        {
          variant: "warning",
          durationMs: 8000,
          notification: {
            type: "storage",
            title: t("notificationCenter.titles.storageOffline"),
            source: {
              route: SETTINGS_LIBRARY_ROUTE,
              libraryPathId: item.libraryPathId,
            },
          },
        },
      )
      return
    }

    pushAppToast(t("toasts.storagePathsAbnormal", { count: fresh.length }), {
      variant: "warning",
      durationMs: 9000,
      notification: {
        type: "storage",
        title: t("notificationCenter.titles.storageOffline"),
        source: { route: SETTINGS_LIBRARY_ROUTE },
      },
    })
  }

  onMounted(() => {
    if (!USE_WEB) {
      return
    }
    void checkAndNotify().catch((err) => {
      console.warn("[storage-status] startup check failed", err)
    })
  })
}

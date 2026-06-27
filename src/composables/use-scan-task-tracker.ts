import { computed, onUnmounted, ref, shallowRef } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import { pushAppToast, taskTerminalToastVariant } from "@/composables/use-app-toast"
import { i18n } from "@/i18n"
import {
  subscribeBackendEvents,
  type BackendEventSubscription,
} from "@/lib/backend-events"
import { useComicLibraryService } from "@/services/comic-library-service"
import { useLibraryService } from "@/services/library-service"

function isFsnotifyLibraryScan(task: TaskDTO): boolean {
  return task.type === "scan.library" && task.metadata?.trigger === "fsnotify"
}

function taskMetaNumber(task: TaskDTO, key: string): number {
  const value = task.metadata?.[key]
  if (typeof value === "number" && Number.isFinite(value)) {
    return value
  }
  if (typeof value === "string") {
    const n = Number(value)
    return Number.isFinite(n) ? n : 0
  }
  return 0
}

const POLL_MS = 500

const libraryService = useLibraryService()
const comicLibraryService = useComicLibraryService()

const activeTask = shallowRef<TaskDTO | null>(null)
const pollError = ref<string | null>(null)

interface ScanTaskTrackerStartOptions {
  hideProgressDock?: boolean
  notifyScanStart?: boolean
  notifyMovieScrape?: boolean
}

let intervalId: ReturnType<typeof setInterval> | null = null
let dismissTimer: ReturnType<typeof setTimeout> | null = null
let trackedTaskId: string | null = null
const trackedTaskOptions = ref<ScanTaskTrackerStartOptions>({})
let consumerCount = 0
let backendEventsSubscription: BackendEventSubscription | null = null

const progressTask = computed(() =>
  trackedTaskOptions.value.hideProgressDock ? null : activeTask.value,
)
const progressPollError = computed(() =>
  trackedTaskOptions.value.hideProgressDock ? null : pollError.value,
)

function clearDismissTimer() {
  if (dismissTimer) {
    clearTimeout(dismissTimer)
    dismissTimer = null
  }
}

function stopPolling() {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

function stopBackendEvents() {
  backendEventsSubscription?.close()
  backendEventsSubscription = null
}

function isTerminalStatus(status: TaskDTO["status"]): boolean {
  return (
    status === "completed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "partial_failed"
  )
}

function importNotificationSource(taskId: string) {
  return {
    taskId,
    route: "/settings?section=library",
  }
}

function libraryScanNotificationSource(taskId: string) {
  return {
    taskId,
    route: "/settings?section=library",
  }
}

function comicTaskNotificationSource(taskId: string) {
  return {
    taskId,
    route: "/settings?section=comics",
  }
}

function movieScrapeNotificationSource(task: TaskDTO) {
  const movieId =
    typeof task.metadata?.movieId === "string" && task.metadata.movieId.trim()
      ? task.metadata.movieId
      : undefined
  return {
    taskId: task.taskId,
    ...(movieId
      ? {
          movieId,
          route: `/detail/${encodeURIComponent(movieId)}`,
        }
      : {}),
  }
}

function movieScrapeToastMessage(task: TaskDTO) {
  const tr = i18n.global.t
  const message = task.errorMessage?.trim() || task.message?.trim() || ""
  return task.status === "completed"
    ? tr("toasts.manualMovieScrapeDone", { message })
    : tr("toasts.manualMovieScrapeFailed", { message })
}

function scheduleTerminalDismiss(taskId: string) {
  clearDismissTimer()
  dismissTimer = setTimeout(() => {
    if (
      trackedTaskId === taskId &&
      activeTask.value?.taskId === taskId &&
      isTerminalStatus(activeTask.value.status)
    ) {
      dismiss()
    }
  }, 5000)
}

function handleTerminalTask(t: TaskDTO, dismissTaskId = t.taskId) {
  stopPolling()
  stopBackendEvents()
  if (t.type === "scan.library") {
    if (!isFsnotifyLibraryScan(t)) {
      const msg = t.message ?? ""
      const tr = i18n.global.t
      pushAppToast(
        t.status === "completed"
          ? tr("toasts.manualLibraryScanDone", { message: msg })
          : tr("toasts.manualLibraryScanFailed", { message: msg }),
        {
          variant: taskTerminalToastVariant(t.status),
          notification: {
            type: "scan",
            title:
              t.status === "completed"
                ? tr("notificationCenter.titles.scanDone")
                : tr("notificationCenter.titles.scanFailed"),
            source: libraryScanNotificationSource(t.taskId),
          },
        },
      )
    }
    void libraryService.reloadMoviesFromApi()
  } else if (t.type === "scan.comics") {
    const msg = t.message ?? ""
    const tr = i18n.global.t
    pushAppToast(
      t.status === "completed"
        ? tr("toasts.manualComicScanDone", { message: msg })
        : tr("toasts.manualComicScanFailed", { message: msg }),
      {
        variant: taskTerminalToastVariant(t.status),
        notification: {
          type: "scan",
          title:
            t.status === "completed"
              ? tr("notificationCenter.titles.scanDone")
              : tr("notificationCenter.titles.scanFailed"),
          source: comicTaskNotificationSource(t.taskId),
        },
      },
    )
    void comicLibraryService.reloadComicsFromApi()
  } else if (t.type === "scrape.movie" && trackedTaskOptions.value.notifyMovieScrape) {
    const tr = i18n.global.t
    pushAppToast(movieScrapeToastMessage(t), {
      variant: taskTerminalToastVariant(t.status),
      notification: {
        type: "scrape",
        title:
          t.status === "completed"
            ? tr("notificationCenter.titles.scrapeDone")
            : tr("notificationCenter.titles.scrapeFailed"),
        source: movieScrapeNotificationSource(t),
      },
    })
  } else if (t.type === "import.movies") {
    const tr = i18n.global.t
    if (t.status === "completed") {
      pushAppToast(
        tr("toasts.movieImportDone", {
          completed: taskMetaNumber(t, "completedFiles"),
        }),
        {
          variant: taskTerminalToastVariant(t.status),
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importDone"),
            source: importNotificationSource(t.taskId),
          },
        },
      )
    } else if (t.status === "partial_failed") {
      pushAppToast(
        tr("toasts.movieImportPartial", {
          completed: taskMetaNumber(t, "completedFiles"),
          failed: taskMetaNumber(t, "failedFiles"),
        }),
        {
          variant: taskTerminalToastVariant(t.status),
          durationMs: 6500,
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importFailed"),
            source: importNotificationSource(t.taskId),
          },
        },
      )
    } else if (t.status === "failed") {
      pushAppToast(
        tr("toasts.movieImportFailed", { message: t.errorMessage ?? t.message ?? "" }),
        {
          variant: taskTerminalToastVariant(t.status),
          durationMs: 6500,
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importFailed"),
            source: importNotificationSource(t.taskId),
          },
        },
      )
    }
    void libraryService.reloadMoviesFromApi()
  } else if (t.type === "import.comics") {
    const tr = i18n.global.t
    if (t.status === "completed") {
      pushAppToast(
        tr("toasts.comicImportDone", {
          completed: taskMetaNumber(t, "completedFiles"),
        }),
        {
          variant: taskTerminalToastVariant(t.status),
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importDone"),
            source: comicTaskNotificationSource(t.taskId),
          },
        },
      )
    } else if (t.status === "partial_failed") {
      pushAppToast(
        tr("toasts.comicImportPartial", {
          completed: taskMetaNumber(t, "completedFiles"),
          failed: taskMetaNumber(t, "failedFiles"),
        }),
        {
          variant: taskTerminalToastVariant(t.status),
          durationMs: 6500,
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importFailed"),
            source: comicTaskNotificationSource(t.taskId),
          },
        },
      )
    } else if (t.status === "failed") {
      pushAppToast(
        tr("toasts.comicImportFailed", { message: t.errorMessage ?? t.message ?? "" }),
        {
          variant: taskTerminalToastVariant(t.status),
          durationMs: 6500,
          notification: {
            type: "system",
            title: tr("notificationCenter.titles.importFailed"),
            source: comicTaskNotificationSource(t.taskId),
          },
        },
      )
    }
    void comicLibraryService.reloadComicsFromApi()
  }
  scheduleTerminalDismiss(dismissTaskId)
}

function applyTaskUpdate(
  t: TaskDTO,
  options: { requireTrackedTaskId?: boolean; dismissTaskId?: string } = {},
) {
  if (!trackedTaskId) return
  if (options.requireTrackedTaskId && t.taskId !== trackedTaskId) return
  if (
    activeTask.value?.taskId === t.taskId &&
    isTerminalStatus(activeTask.value.status) &&
    !isTerminalStatus(t.status)
  ) {
    return
  }
  pollError.value = null
  activeTask.value = t
  if (isTerminalStatus(t.status)) {
    handleTerminalTask(t, options.dismissTaskId)
  }
}

async function poll() {
  if (!trackedTaskId) return
  const taskId = trackedTaskId
  try {
    pollError.value = null
    const t = await api.getTaskStatus(taskId)
    if (trackedTaskId !== taskId) return
    applyTaskUpdate(t, { dismissTaskId: taskId })
  } catch (e) {
    stopPolling()
    stopBackendEvents()
    activeTask.value = null
    pollError.value = e instanceof Error ? e.message : i18n.global.t("scanTask.fetchFailed")
    trackedTaskId = null
    trackedTaskOptions.value = {}
  }
}

function dismiss() {
  clearDismissTimer()
  stopPolling()
  stopBackendEvents()
  trackedTaskId = null
  trackedTaskOptions.value = {}
  activeTask.value = null
  pollError.value = null
}

export function useScanTaskTracker() {
  consumerCount += 1

  onUnmounted(() => {
    consumerCount = Math.max(0, consumerCount - 1)
    if (consumerCount > 0) {
      return
    }
    clearDismissTimer()
    stopPolling()
    stopBackendEvents()
    trackedTaskId = null
    trackedTaskOptions.value = {}
    activeTask.value = null
    pollError.value = null
  })

  function start(taskId: string, options: ScanTaskTrackerStartOptions = {}) {
    clearDismissTimer()
    stopPolling()
    stopBackendEvents()
    trackedTaskId = taskId
    trackedTaskOptions.value = { ...options }
    activeTask.value = null
    pollError.value = null
    if (options.notifyScanStart) {
      const tr = i18n.global.t
      pushAppToast(tr("toasts.manualLibraryScanStarted"), {
        variant: "default",
        durationMs: 2600,
        notification: {
          type: "scan",
          title: tr("notificationCenter.titles.scanStarted"),
          source: libraryScanNotificationSource(taskId),
        },
      })
    }
    backendEventsSubscription = subscribeBackendEvents({
      onTaskUpdated(task) {
        applyTaskUpdate(task, { requireTrackedTaskId: true })
      },
    })
    void poll()
    intervalId = setInterval(() => void poll(), POLL_MS)
  }

  return {
    activeTask,
    progressTask,
    pollError,
    progressPollError,
    start,
    dismiss,
  }
}

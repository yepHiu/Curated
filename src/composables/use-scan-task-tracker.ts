import { computed, onUnmounted, ref, shallowRef } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import {
  dismissAppToast,
  pushAppToast,
  taskTerminalToastVariant,
  type AppToastId,
} from "@/composables/use-app-toast"
import { i18n } from "@/i18n"
import {
  subscribeBackendEvents,
  type BackendEventSubscription,
} from "@/lib/backend-events"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"
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
const photoLibraryService = usePhotoLibraryService()

const activeTask = shallowRef<TaskDTO | null>(null)
const pollError = ref<string | null>(null)

interface ScanTaskTrackerStartOptions {
  hideProgressDock?: boolean
  notifyScanStart?: boolean
  notifyMovieScrape?: boolean
  loadingToastId?: AppToastId
  scrapeCode?: string
}

let intervalId: ReturnType<typeof setInterval> | null = null
let dismissTimer: ReturnType<typeof setTimeout> | null = null
let trackedTaskId: string | null = null
const trackedTaskOptions = ref<ScanTaskTrackerStartOptions>({})
let consumerCount = 0
let backendEventsSubscription: BackendEventSubscription | null = null
let trackedLoadingToastId: AppToastId | null = null

const progressTask = computed(() => {
  if (trackedTaskOptions.value.hideProgressDock) return null
  const task = activeTask.value
  if (task?.type === "scrape.movie") return null
  return task
})
const progressPollError = computed(() => {
  if (trackedTaskOptions.value.hideProgressDock) return null
  if (activeTask.value?.type === "scrape.movie") return null
  return pollError.value
})

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

function photoTaskNotificationSource(taskId: string) {
  return {
    taskId,
    route: "/settings?section=photos",
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

function clearLoadingToast() {
  if (trackedLoadingToastId == null) {
    return
  }
  dismissAppToast(trackedLoadingToastId)
  trackedLoadingToastId = null
}

function attachLoadingToast(id: AppToastId | undefined) {
  if (id == null) {
    clearLoadingToast()
    return
  }
  if (trackedLoadingToastId != null && trackedLoadingToastId !== id) {
    dismissAppToast(trackedLoadingToastId)
  }
  trackedLoadingToastId = id
}

function taskMetaString(task: TaskDTO, key: string): string {
  const value = task.metadata?.[key]
  return typeof value === "string" ? value.trim() : ""
}

function movieScrapeCode(task: TaskDTO): string {
  return taskMetaString(task, "number") || trackedTaskOptions.value.scrapeCode?.trim() || ""
}

function movieScrapeToastMessage(task: TaskDTO) {
  const tr = i18n.global.t
  const code = movieScrapeCode(task)
  if (task.status === "completed") {
    return code
      ? tr("toasts.manualMovieScrapeDone", { code })
      : tr("toasts.manualMovieScrapeDoneGeneric")
  }
  if (task.status === "cancelled") {
    return code
      ? tr("toasts.manualMovieScrapeCancelled", { code })
      : tr("toasts.manualMovieScrapeCancelledGeneric")
  }
  return code
    ? tr("toasts.manualMovieScrapeFailed", { code })
    : tr("toasts.manualMovieScrapeFailedGeneric")
}

function movieScrapeToastOptions(task: TaskDTO) {
  const tr = i18n.global.t
  return {
    variant: taskTerminalToastVariant(task.status),
    ...(trackedLoadingToastId == null ? {} : { id: trackedLoadingToastId }),
    notification: {
      messageId: task.status === "completed" ? ("MSG-0021" as const) : ("MSG-0022" as const),
      type: "scrape" as const,
      title:
        task.status === "completed"
          ? tr("notificationCenter.titles.scrapeDone")
          : tr("notificationCenter.titles.scrapeFailed"),
      source: movieScrapeNotificationSource(task),
    },
  }
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
            messageId: t.status === "completed" ? ("MSG-0020" as const) : ("MSG-0013" as const),
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
  } else if (t.type === "scan.comics" || t.type === "scan.photos") {
    const photo = t.type === "scan.photos"
    const msg = t.message ?? ""
    const tr = i18n.global.t
    pushAppToast(
      t.status === "completed"
        ? tr(photo ? "toasts.manualPhotoScanDone" : "toasts.manualComicScanDone", { message: msg })
        : tr(photo ? "toasts.manualPhotoScanFailed" : "toasts.manualComicScanFailed", { message: msg }),
      {
        variant: taskTerminalToastVariant(t.status),
        notification: {
          messageId: t.status === "completed" ? "MSG-0020" : "MSG-0013",
          type: "scan",
          title:
            t.status === "completed"
              ? tr("notificationCenter.titles.scanDone")
              : tr("notificationCenter.titles.scanFailed"),
          source: photo ? photoTaskNotificationSource(t.taskId) : comicTaskNotificationSource(t.taskId),
        },
      },
    )
    if (photo) void photoLibraryService.reloadPhotosFromApi()
    else void comicLibraryService.reloadComicsFromApi()
  } else if (t.type === "scrape.movie" && trackedTaskOptions.value.notifyMovieScrape) {
    pushAppToast(movieScrapeToastMessage(t), movieScrapeToastOptions(t))
    trackedLoadingToastId = null
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
            messageId: "MSG-0023",
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
            messageId: "MSG-0012",
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
            messageId: "MSG-0011",
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
            messageId: "MSG-0023",
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
            messageId: "MSG-0012",
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
            messageId: "MSG-0011",
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
    const message = e instanceof Error ? e.message : i18n.global.t("scanTask.fetchFailed")
    pollError.value = message
    const loadingId = trackedLoadingToastId
    if (trackedTaskOptions.value.notifyMovieScrape || loadingId != null) {
      pushAppToast(message, {
        variant: "destructive",
        ...(loadingId == null ? {} : { id: loadingId }),
      })
    }
    trackedLoadingToastId = null
    trackedTaskId = null
    trackedTaskOptions.value = {}
  }
}

function dismiss() {
  clearDismissTimer()
  stopPolling()
  stopBackendEvents()
  clearLoadingToast()
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
    clearLoadingToast()
    trackedTaskId = null
    trackedTaskOptions.value = {}
    activeTask.value = null
    pollError.value = null
  })

  function start(taskId: string, options: ScanTaskTrackerStartOptions = {}) {
    clearDismissTimer()
    stopPolling()
    stopBackendEvents()
    attachLoadingToast(options.loadingToastId)
    trackedTaskId = taskId
    trackedTaskOptions.value = { ...options }
    activeTask.value = null
    pollError.value = null
    if (options.notifyScanStart) {
      const tr = i18n.global.t
      pushAppToast(tr("toasts.manualLibraryScanStarted"), {
        variant: "default",
        durationMs: 2600,
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

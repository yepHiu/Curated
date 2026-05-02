import { onUnmounted, ref, shallowRef } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import { pushAppToast, taskTerminalToastVariant } from "@/composables/use-app-toast"
import { i18n } from "@/i18n"
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

const activeTask = shallowRef<TaskDTO | null>(null)
const pollError = ref<string | null>(null)

let intervalId: ReturnType<typeof setInterval> | null = null
let dismissTimer: ReturnType<typeof setTimeout> | null = null
let trackedTaskId: string | null = null
let consumerCount = 0

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

function isTerminalStatus(status: TaskDTO["status"]): boolean {
  return (
    status === "completed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "partial_failed"
  )
}

async function poll() {
  if (!trackedTaskId) return
  const taskId = trackedTaskId
  try {
    pollError.value = null
    const t = await api.getTaskStatus(taskId)
    if (trackedTaskId !== taskId) return
    activeTask.value = t
      if (isTerminalStatus(t.status)) {
        stopPolling()
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
                title: t.status === "completed"
                  ? tr("notificationCenter.titles.scanDone")
                  : tr("notificationCenter.titles.scanFailed"),
                source: { taskId: t.taskId },
              },
            },
          )
        }
        void libraryService.reloadMoviesFromApi()
      } else if (t.type === "import.movies") {
        const tr = i18n.global.t
        if (t.status === "completed") {
          pushAppToast(
            tr("toasts.movieImportDone", {
              completed: taskMetaNumber(t, "completedFiles"),
            }),
            { variant: taskTerminalToastVariant(t.status) },
          )
        } else if (t.status === "partial_failed") {
          pushAppToast(
            tr("toasts.movieImportPartial", {
              completed: taskMetaNumber(t, "completedFiles"),
              failed: taskMetaNumber(t, "failedFiles"),
            }),
            { variant: taskTerminalToastVariant(t.status), durationMs: 6500 },
          )
        } else if (t.status === "failed") {
          pushAppToast(
            tr("toasts.movieImportFailed", { message: t.errorMessage ?? t.message ?? "" }),
            { variant: taskTerminalToastVariant(t.status), durationMs: 6500 },
          )
        }
        void libraryService.reloadMoviesFromApi()
      }
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
  } catch (e) {
    stopPolling()
    activeTask.value = null
    pollError.value = e instanceof Error ? e.message : i18n.global.t("scanTask.fetchFailed")
    trackedTaskId = null
  }
}

function dismiss() {
  clearDismissTimer()
  stopPolling()
  trackedTaskId = null
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
    trackedTaskId = null
    activeTask.value = null
    pollError.value = null
  })

  function start(taskId: string) {
    clearDismissTimer()
    stopPolling()
    trackedTaskId = taskId
    activeTask.value = null
    pollError.value = null
    void poll()
    intervalId = setInterval(() => void poll(), POLL_MS)
  }

  return {
    activeTask,
    pollError,
    start,
    dismiss,
  }
}

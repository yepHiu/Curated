import { ref, shallowRef } from "vue"
import type { TaskDTO } from "@/api/types"
import { api } from "@/api/endpoints"
import { pushAppToast } from "@/composables/use-app-toast"
import { i18n } from "@/i18n"
import { useLibraryService } from "@/services/library-service"

function isFsnotifyLibraryScan(task: TaskDTO): boolean {
  return task.type === "scan.library" && task.metadata?.trigger === "fsnotify"
}

const POLL_MS = 500

const libraryService = useLibraryService()

const activeTask = shallowRef<TaskDTO | null>(null)
const pollError = ref<string | null>(null)

let intervalId: ReturnType<typeof setInterval> | null = null
let dismissTimer: ReturnType<typeof setTimeout> | null = null
let trackedTaskId: string | null = null

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
          if (t.status === "completed") {
            pushAppToast(tr("toasts.manualLibraryScanDone", { message: msg }), { variant: "success" })
          } else {
            pushAppToast(tr("toasts.manualLibraryScanFailed", { message: msg }), { variant: "destructive" })
          }
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
    pollError.value = e instanceof Error ? e.message : "无法获取扫描任务状态"
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

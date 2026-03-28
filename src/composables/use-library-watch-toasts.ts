import { onBeforeUnmount, onMounted } from "vue"
import { useI18n } from "vue-i18n"
import { api } from "@/api/endpoints"
import type { TaskDTO } from "@/api/types"
import { pushAppToast, taskTerminalToastVariant } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const POLL_MS = 2000

/** 同标签页刷新后仍会去拉 recent tasks；持久化已弹过的 taskId，避免每次 F5 重复 toast */
const SEEN_TOAST_STORAGE_KEY = "curated-library-watch-toast-seen-ids"
const MAX_PERSISTED_SEEN_IDS = 400

function readSeenToastIds(): Set<string> {
  try {
    const raw = sessionStorage.getItem(SEEN_TOAST_STORAGE_KEY)
    if (!raw) return new Set()
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return new Set()
    return new Set(arr.filter((x): x is string => typeof x === "string" && x.length > 0))
  } catch {
    return new Set()
  }
}

function persistSeenToastIds(seen: Set<string>) {
  try {
    let ids = [...seen]
    if (ids.length > MAX_PERSISTED_SEEN_IDS) {
      ids = ids.slice(-MAX_PERSISTED_SEEN_IDS)
      seen.clear()
      for (const id of ids) {
        seen.add(id)
      }
    }
    sessionStorage.setItem(SEEN_TOAST_STORAGE_KEY, JSON.stringify(ids))
  } catch {
    /* quota / private mode */
  }
}

function isTerminalStatus(s: TaskDTO["status"]): boolean {
  return (
    s === "completed" ||
    s === "failed" ||
    s === "partial_failed" ||
    s === "cancelled"
  )
}

function isFsnotifyScan(task: TaskDTO): boolean {
  const tr = task.metadata?.trigger
  return task.type === "scan.library" && tr === "fsnotify"
}

function parentScanId(task: TaskDTO): string {
  const p = task.metadata?.parentScanTaskId
  return typeof p === "string" ? p : ""
}

/** Polls GET /api/tasks/recent and shows toasts for fsnotify-driven scan + linked scrape completions. */
export function useLibraryWatchToasts() {
  const { t } = useI18n()
  const libraryService = useLibraryService()
  let timer: ReturnType<typeof setInterval> | null = null
  const seenToastIds = readSeenToastIds()
  const fsnotifyScanParents = new Set<string>()

  function markToastSeen(taskId: string) {
    seenToastIds.add(taskId)
    persistSeenToastIds(seenToastIds)
  }

  async function poll() {
    try {
      const { tasks } = await api.getRecentTasks(40)

      let needsMovieReload = false
      for (const task of tasks) {
        if (!isTerminalStatus(task.status) || !isFsnotifyScan(task)) {
          continue
        }
        fsnotifyScanParents.add(task.taskId)
        if (seenToastIds.has(task.taskId)) {
          continue
        }
        markToastSeen(task.taskId)
        needsMovieReload = true
        const msg = task.message ?? ""
        pushAppToast(t("toasts.libraryWatchScanDone", { message: msg }), {
          variant: taskTerminalToastVariant(task.status),
        })
      }

      for (const task of tasks) {
        if (!isTerminalStatus(task.status) || task.type !== "scrape.movie") {
          continue
        }
        const parent = parentScanId(task)
        if (!parent || !fsnotifyScanParents.has(parent)) {
          continue
        }
        if (seenToastIds.has(task.taskId)) {
          continue
        }
        markToastSeen(task.taskId)
        needsMovieReload = true
        const msg = task.message ?? ""
        pushAppToast(t("toasts.libraryWatchScrapeDone", { message: msg }), {
          variant: taskTerminalToastVariant(task.status),
        })
      }

      if (needsMovieReload) {
        void libraryService.reloadMoviesFromApi()
      }

      if (fsnotifyScanParents.size > 300) {
        fsnotifyScanParents.clear()
      }
    } catch {
      // Offline / transient API errors: skip silently
    }
  }

  onMounted(() => {
    if (!USE_WEB) {
      return
    }
    void poll()
    timer = setInterval(() => void poll(), POLL_MS)
  })

  onBeforeUnmount(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })
}

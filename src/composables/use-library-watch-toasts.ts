import { onBeforeUnmount, onMounted } from "vue"
import { useI18n } from "vue-i18n"
import { api } from "@/api/endpoints"
import type { TaskDTO } from "@/api/types"
import { pushAppToast, taskTerminalToastVariant } from "@/composables/use-app-toast"
import { bumpMovieImageVersion } from "@/lib/image-version"
import { useLibraryService } from "@/services/library-service"

const USE_WEB = import.meta.env.VITE_USE_WEB_API === "true"
const POLL_MS = 2000
const REDUNDANT_NO_CHANGE_SCAN_WINDOW_MS = 15_000

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

function movieId(task: TaskDTO): string {
  const mid = task.metadata?.movieId
  return typeof mid === "string" ? mid : ""
}

function taskMetaString(task: TaskDTO, key: string): string {
  const value = task.metadata?.[key]
  return typeof value === "string" ? value : ""
}

function taskFinishedMs(task: TaskDTO): number {
  const raw = task.finishedAt?.trim()
  if (!raw) return 0
  const ms = Date.parse(raw)
  return Number.isFinite(ms) ? ms : 0
}

function scanPathsKey(task: TaskDTO): string {
  const raw = task.metadata?.paths
  if (!Array.isArray(raw)) return ""
  const paths = raw.filter((x): x is string => typeof x === "string" && x.trim().length > 0)
  if (paths.length === 0) return ""
  return [...paths].sort().join("\u0000")
}

function taskHasMetadata(task: TaskDTO, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(task.metadata ?? {}, key)
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

function fsnotifyScanChanged(task: TaskDTO): boolean {
  return taskMetaNumber(task, "scanImported") + taskMetaNumber(task, "scanUpdated") > 0
}

function fsnotifyScanNoChange(task: TaskDTO): boolean {
  const hasScanCounters =
    taskHasMetadata(task, "scanTotal") ||
    taskHasMetadata(task, "scanImported") ||
    taskHasMetadata(task, "scanUpdated") ||
    taskHasMetadata(task, "scanSkipped")
  return hasScanCounters && !fsnotifyScanChanged(task)
}

/** Polls GET /api/tasks/recent and shows toasts for fsnotify-driven scan + linked scrape completions. */
/** 仅处理本会话开始后结束的任务，避免首次拉 recent 时对历史 asset.download 全员 bump */
let libraryWatchAppMountedAtMs = 0

function taskFinishedAfterSessionStart(task: TaskDTO): boolean {
  const raw = task.finishedAt?.trim()
  if (!raw) return false
  const ms = Date.parse(raw)
  return Number.isFinite(ms) && ms >= libraryWatchAppMountedAtMs
}

export function useLibraryWatchToasts() {
  const { t } = useI18n()
  const libraryService = useLibraryService()
  let timer: ReturnType<typeof setInterval> | null = null
  const seenToastIds = readSeenToastIds()
  const fsnotifyScanParents = new Set<string>()
  const bumpedAssetDownloadTaskIds = new Set<string>()
  const changedScanFinishedAtByPaths = new Map<string, number>()

  function markToastSeen(taskId: string) {
    seenToastIds.add(taskId)
    persistSeenToastIds(seenToastIds)
  }

  function libraryWatchScanToastMessage(task: TaskDTO): string {
    const hasScanCounters =
      taskHasMetadata(task, "scanTotal") ||
      taskHasMetadata(task, "scanImported") ||
      taskHasMetadata(task, "scanUpdated") ||
      taskHasMetadata(task, "scanSkipped")
    if (!hasScanCounters) {
      return t("toasts.libraryWatchScanDone", { message: task.message ?? "" })
    }

    const discovered = taskMetaNumber(task, "scanTotal")
    const imported = taskMetaNumber(task, "scanImported")
    const updated = taskMetaNumber(task, "scanUpdated")
    const skipped = taskMetaNumber(task, "scanSkipped")
    if (imported + updated === 0) {
      return t("toasts.libraryWatchScanDoneNoChanges", { discovered, skipped })
    }
    return t("toasts.libraryWatchScanDoneWithChanges", {
      discovered,
      imported,
      updated,
      skipped,
    })
  }

  function rememberChangedFsnotifyScan(task: TaskDTO) {
    if (!fsnotifyScanChanged(task)) return
    const key = scanPathsKey(task)
    const finishedMs = taskFinishedMs(task)
    if (!key || finishedMs <= 0) return
    const previous = changedScanFinishedAtByPaths.get(key) ?? 0
    if (finishedMs > previous) {
      changedScanFinishedAtByPaths.set(key, finishedMs)
    }
  }

  function shouldSuppressRedundantNoChangeScan(task: TaskDTO): boolean {
    if (!fsnotifyScanNoChange(task)) return false
    const key = scanPathsKey(task)
    const finishedMs = taskFinishedMs(task)
    if (!key || finishedMs <= 0) return false
    const changedFinishedMs = changedScanFinishedAtByPaths.get(key) ?? 0
    return (
      changedFinishedMs > 0 &&
      finishedMs >= changedFinishedMs &&
      finishedMs - changedFinishedMs <= REDUNDANT_NO_CHANGE_SCAN_WINDOW_MS
    )
  }

  async function poll() {
    try {
      const { tasks } = await api.getRecentTasks(40)
      for (const task of tasks) {
        if (isTerminalStatus(task.status) && isFsnotifyScan(task)) {
          rememberChangedFsnotifyScan(task)
        }
      }

      let needsMovieReload = false
      for (const task of tasks) {
        if (!isTerminalStatus(task.status) || !isFsnotifyScan(task)) {
          continue
        }
        fsnotifyScanParents.add(task.taskId)
        if (seenToastIds.has(task.taskId)) {
          continue
        }
        if (shouldSuppressRedundantNoChangeScan(task)) {
          markToastSeen(task.taskId)
          continue
        }
        markToastSeen(task.taskId)
        needsMovieReload = true
        pushAppToast(libraryWatchScanToastMessage(task), {
          variant: taskTerminalToastVariant(task.status),
          notification: {
            type: "scan",
            title: t("notificationCenter.titles.scanDone"),
            source: { taskId: task.taskId },
          },
        })
      }

      const isFsnotifyLinkedScrape = (task: TaskDTO) => {
        const parent = parentScanId(task)
        return (
          taskMetaString(task, "parentScanTrigger") === "fsnotify" ||
          taskMetaString(task, "trigger") === "fsnotify" ||
          (!!parent && fsnotifyScanParents.has(parent))
        )
      }

      for (const task of tasks) {
        if (!isTerminalStatus(task.status) || task.type !== "scrape.movie") {
          continue
        }
        if (seenToastIds.has(task.taskId)) {
          continue
        }
        markToastSeen(task.taskId)
        needsMovieReload = true
        const mid = movieId(task)
        if (mid && task.status === "completed") {
          bumpMovieImageVersion(mid)
        }
        if (!isFsnotifyLinkedScrape(task)) {
          continue
        }
        const msg = task.message ?? ""
        pushAppToast(t("toasts.libraryWatchScrapeDone", { message: msg }), {
          variant: taskTerminalToastVariant(task.status),
          notification: {
            type: "scrape",
            title: t("notificationCenter.titles.scrapeDone"),
            source: { taskId: task.taskId },
          },
        })
      }

      if (needsMovieReload) {
        void libraryService.reloadMoviesFromApi()
      }

      for (const task of tasks) {
        if (
          !isTerminalStatus(task.status) ||
          task.type !== "asset.download" ||
          task.status !== "completed"
        ) {
          continue
        }
        if (bumpedAssetDownloadTaskIds.has(task.taskId)) continue
        if (!taskFinishedAfterSessionStart(task)) continue
        const mid = movieId(task)
        if (!mid) continue
        bumpedAssetDownloadTaskIds.add(task.taskId)
        bumpMovieImageVersion(mid)
      }

      if (fsnotifyScanParents.size > 300) {
        fsnotifyScanParents.clear()
      }
      if (changedScanFinishedAtByPaths.size > 300) {
        changedScanFinishedAtByPaths.clear()
      }
    } catch {
      // Offline / transient API errors: skip silently
    }
  }

  onMounted(() => {
    if (!USE_WEB) {
      return
    }
    libraryWatchAppMountedAtMs = Date.now() - 15_000
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

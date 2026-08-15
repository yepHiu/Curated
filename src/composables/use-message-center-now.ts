import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { TaskDTO } from "@/api/types"
import { useScanTaskTracker } from "@/composables/use-scan-task-tracker"

export interface MessageCenterNowItem {
  id: string
  title: string
  message: string
}

function isActiveTask(task: TaskDTO | null | undefined): task is TaskDTO {
  if (!task) return false
  return (
    task.status !== "completed" &&
    task.status !== "failed" &&
    task.status !== "cancelled" &&
    task.status !== "partial_failed"
  )
}

function metaNumber(task: TaskDTO, key: string): number | null {
  const value = task.metadata?.[key]
  if (typeof value === "number" && Number.isFinite(value)) {
    return value
  }
  if (typeof value === "string") {
    const n = Number(value)
    return Number.isFinite(n) ? n : null
  }
  return null
}

export function useMessageCenterNow() {
  const { t } = useI18n()
  const { activeTask, progressTask } = useScanTaskTracker()

  const nowItems = computed((): MessageCenterNowItem[] => {
    const items: MessageCenterNowItem[] = []
    const dockTask = progressTask.value
    if (isActiveTask(dockTask)) {
      if (dockTask.type === "import.movies") {
        const completed = metaNumber(dockTask, "completedFiles")
        const total = metaNumber(dockTask, "totalFiles")
        items.push({
          id: "MSG-0002",
          title: t("notificationCenter.nowImportingTitle"),
          message:
            completed != null && total != null
              ? t("notificationCenter.nowImportingMessage", { completed, total })
              : (dockTask.message ?? t("notificationCenter.nowImportingTitle")),
        })
      } else {
        items.push({
          id: "MSG-0001",
          title: t("notificationCenter.nowScanningTitle"),
          message: dockTask.message?.trim() || t("notificationCenter.nowScanningMessage"),
        })
      }
    }

    const scrapeTask = activeTask.value
    if (isActiveTask(scrapeTask) && scrapeTask.type === "scrape.movie") {
      items.push({
        id: "MSG-0003",
        title: t("notificationCenter.nowScrapingTitle"),
        message: scrapeTask.message?.trim() || t("notificationCenter.nowScrapingMessage"),
      })
    }
    return items
  })

  return { nowItems }
}

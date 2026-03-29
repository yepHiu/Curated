import { watch } from "vue"
import type { TaskDTO } from "@/api/types"

export function isTerminalTaskStatus(status: TaskDTO["status"]): boolean {
  return (
    status === "completed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "partial_failed"
  )
}

/** 在已调用 `scanTaskTracker.start(taskId)` 后，等待轮询中的 `activeTask` 对该 id 进入终端态。 */
export function waitForTrackedTaskTerminal(
  getActiveTask: () => TaskDTO | null,
  taskId: string,
): Promise<TaskDTO> {
  return new Promise((resolve) => {
    const stop = watch(
      getActiveTask,
      (task) => {
        if (task?.taskId === taskId && isTerminalTaskStatus(task.status)) {
          stop()
          resolve(task)
        }
      },
      { immediate: true, flush: "sync" },
    )
  })
}

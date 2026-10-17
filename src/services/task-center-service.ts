import { httpClient } from "@/api/http-client"
import { useAIService } from "./ai-service"
import { useSubtitleService } from "./subtitle-service"
import type { TaskCenterService, TaskPage, TaskSummary, CenterTask } from "./contracts/task-center-service"
import { projectOrganization, projectSubtitle, summarizeTasks, taskActive, taskAttention, taskHistory } from "@/lib/task-center"

const action: TaskCenterService["action"] = async (task, operation) => {
  if (!task.canManage) throw new Error("TASK_READ_ONLY")
  if (task.kind === "subtitle") {
    if (operation === "undo") throw new Error("TASK_ACTION_UNAVAILABLE")
    const next = await useSubtitleService().action(task.id, operation)
    if (operation === "restart") return { taskKey: `subtitle:${next.id}` }
  } else {
    if (operation === "restart") throw new Error("TASK_ACTION_UNAVAILABLE")
    const service = useAIService()
    if (operation === "undo") return service.undoTagOrganization(task.id)
    if (operation === "cancel") await service.cancelTagOrganization(task.id)
    else await service.retryTagOrganization(task.id)
  }
}
const web: TaskCenterService = {
  list: (query = {}) => httpClient.get<TaskPage>("/task-center/tasks", { ...query }),
  summary: () => httpClient.get<TaskSummary>("/task-center/summary"),
  get: key => httpClient.get<CenterTask>(`/task-center/tasks/${encodeURIComponent(key)}`),
  action,
}
async function mockTasks() {
  const [subtitles, organizations] = await Promise.all([useSubtitleService().jobs(), useAIService().listTagOrganizations()])
  return [...subtitles.map(projectSubtitle), ...organizations.map(projectOrganization)].sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.key.localeCompare(a.key))
}
const mock: TaskCenterService = {
  async list(query = {}) {
    const all = await mockTasks()
    const filtered = all.filter(t => (!query.kind || query.kind === "all" || t.kind === query.kind) && (!query.state || query.state === "all" || query.state === "active" && taskActive(t) || query.state === "attention" && taskAttention(t) || query.state === "history" && taskHistory(t)))
    const start = query.cursor ? filtered.findIndex(t => t.key === query.cursor) + 1 : 0
    const items = filtered.slice(start, start + (query.limit ?? 30))
    return { items, nextCursor: start + items.length < filtered.length ? items.at(-1)!.key : "", summary: summarizeTasks(all) }
  },
  async summary() { return summarizeTasks(await mockTasks()) },
  async get(key) { const task = (await mockTasks()).find(t => t.key === key); if (!task) throw new Error("TASK_NOT_FOUND"); return task },
  action,
}
export const useTaskCenterService = () => import.meta.env.VITE_USE_WEB_API === "true" ? web : mock

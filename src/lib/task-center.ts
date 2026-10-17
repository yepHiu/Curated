import type { BookOrganizationJob } from "@/services/contracts/book-organization-service"
import type { CenterTask, TaskState, TaskSummary } from "@/services/contracts/task-center-service"
import type { SubtitleJob } from "@/services/contracts/subtitle-service"
import type { TagOrganizationJob } from "@/services/contracts/topic-service"

export const emptyTaskSummary = (): TaskSummary => ({ total: 0, running: 0, queued: 0, waiting: 0, attention: 0, history: 0 })
export const taskActive = (task: CenterTask) => ["queued", "running", "waiting"].includes(task.state)
export const taskAttention = (task: CenterTask) => !task.subtitle?.supersededBy && ["blocked", "partial", "failed", "needs_confirmation"].includes(task.state)
export const taskHistory = (task: CenterTask) => !!task.subtitle?.supersededBy || ["succeeded", "partial", "failed", "cancelled"].includes(task.state)
export const taskPercent = (task: CenterTask) => task.total > 0 ? Math.max(0, Math.min(100, task.completed / task.total * 100)) : undefined
export const taskRoute = (key: string) => `/tasks/${encodeURIComponent(key)}`
export function taskState(state: string, stage: string): TaskState {
  if (state === "running" && stage === "waiting_quota") return "waiting"
  if (state === "completed") return "succeeded"
  if (state === "partial_failed") return "partial"
  return ["queued", "running", "blocked", "failed", "cancelled", "needs_confirmation"].includes(state) ? state as TaskState : "unknown"
}
export function projectSubtitle(j: SubtitleJob): CenterTask {
  return { key: `subtitle:${j.id}`, id: j.id, kind: "subtitle", state: taskState(j.state, j.stage), stage: j.stage, completed: j.completed, total: j.total, unit: j.stage === "translating" ? "cues" : "segments", createdAt: j.createdAt, updatedAt: j.updatedAt, canManage: true, subtitle: j }
}
export function projectOrganization(j: TagOrganizationJob): CenterTask {
  return { key: `organize:${j.id}`, id: j.id, kind: "organize", state: taskState(j.status, j.stage), stage: j.stage, completed: j.stage === "vocabulary" ? j.vocabularyProcessed ?? 0 : j.stage === "translating_metadata" ? j.translationProcessed ?? 0 : j.processed, total: j.total, unit: "movies", createdAt: j.createdAt, updatedAt: j.updatedAt, canManage: true, organization: j }
}
export function summarizeTasks(tasks: CenterTask[]): TaskSummary {
  return { total: tasks.length, running: tasks.filter(t => t.state === "running").length, queued: tasks.filter(t => t.state === "queued").length, waiting: tasks.filter(t => t.state === "waiting").length, attention: tasks.filter(taskAttention).length, history: tasks.filter(taskHistory).length }
}

export function projectBookOrganization(j: BookOrganizationJob): CenterTask {
  return { key: `book-organize:${j.id}`, id: j.id, kind: "organize", state: taskState(j.status, j.stage), stage: j.stage, completed: j.processed, total: j.total, unit: "books", createdAt: j.createdAt, updatedAt: j.updatedAt, canManage: true, bookOrganization: j }
}

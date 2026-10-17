import type { BookOrganizationJob } from "./book-organization-service"
import type { SubtitleJob } from "./subtitle-service"
import type { TagOrganizationJob } from "./topic-service"

export type TaskKind = "subtitle" | "organize"
export type TaskState = "queued" | "running" | "waiting" | "blocked" | "succeeded" | "partial" | "failed" | "cancelled" | "unknown" | "needs_confirmation"
export type TaskFilter = "all" | "active" | "attention" | "history"
export interface CenterTask {
  key: string; kind: TaskKind; id: string; state: TaskState; stage: string
  completed: number; total: number; unit: "segments" | "cues" | "movies" | "books"
  createdAt: string; updatedAt: string; canManage: boolean
  subtitle?: SubtitleJob; organization?: TagOrganizationJob
  bookOrganization?: BookOrganizationJob
  audioTrack?: number; textModel?: string
}
export interface TaskSummary { total: number; running: number; queued: number; waiting: number; attention: number; history: number }
export interface TaskQuery { kind?: TaskKind | "all"; state?: TaskFilter; cursor?: string; limit?: number }
export interface TaskPage { items: CenterTask[]; nextCursor: string; summary: TaskSummary }
export interface TaskCenterService {
  list(query?: TaskQuery): Promise<TaskPage>
  summary(): Promise<TaskSummary>
  get(key: string): Promise<CenterTask>
  action(task: CenterTask, action: "cancel" | "retry" | "restart" | "undo"): Promise<void | { restored: number; conflicts: number }>
}

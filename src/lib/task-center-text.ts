import { i18n } from "@/i18n"
import type { CenterTask } from "@/services/contracts/task-center-service"
export function taskTitle(task: CenterTask) { return i18n.global.t(task.bookOrganization ? `bookAI.taskTitles.${task.bookOrganization.kind}` : task.kind === "subtitle" ? "subtitles.title" : "topics.organize") }
export function taskSubject(task: CenterTask) { return task.subtitle?.fileName || i18n.global.t(task.bookOrganization ? "bookAI.count" : "tasks.movieCount", { count: task.total }) }
export function taskStage(task: CenterTask) {
  if (task.bookOrganization && task.stage === "recognizing") return i18n.global.t("bookAI.recognizing")
  const key = task.kind === "subtitle" ? `subtitles.stages.${task.stage}` : `tasks.stages.${task.stage}`
  return i18n.global.te(key) ? i18n.global.t(key) : i18n.global.t(`tasks.states.${task.state}`)
}
export function taskDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString(i18n.global.locale.value) }

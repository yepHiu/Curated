import { computed, ref } from "vue"
import { useAIService } from "@/services/ai-service"
import { useLibraryService } from "@/services/library-service"
import type { TagOrganizationJob, TagOrganizationStats, TagOrganizationScope } from "@/services/contracts/topic-service"
import { HttpClientError } from "@/api/http-client"
import { pushAppToast } from "@/composables/use-app-toast"
import { useNotificationCenter } from "@/composables/use-notification-center"
import { i18n } from "@/i18n"

const jobs = ref<TagOrganizationJob[]>([])
const error = ref("")
const busy = ref(false)
const connected = ref(true)
const dialogOpen = ref(false)
const selection = ref<{ movieIds: string[]; title?: string } | null>(null)
const revision = ref(0)
const quiet = ref(false)
const stats = ref<TagOrganizationStats | null>(null)
const statsLoading = ref(false)
const statsError = ref(false)
const notices = new Set<string>()
let timer: ReturnType<typeof setInterval> | undefined
let loading: Promise<void> | undefined
let statsRequest: Promise<void> | undefined
let initialized = false
let generation = 0

/** 运行状态与结果状态保持分离。 */
export function isOrganizationActive(job: TagOrganizationJob) { return job.status === "queued" || job.status === "running" }

/** 显示实际阶段；词汇准备不再伪装成零片分类进度。 */
export function organizationProgressText(job: TagOrganizationJob): string {
  const t = i18n.global.t
  if (job.stage === "vocabulary") return t("topics.vocabularyProgress", { done: job.vocabularyProcessed ?? 0, total: job.total })
  if (job.stage === "waiting_quota") return t("topics.waitingQuota", { done: job.processed, total: job.total })
  return t("topics.progress", { done: job.processed, total: job.total })
}

/** Preparation and classification each have their own measured progress. */
export function organizationProgressValue(job: TagOrganizationJob): number {
  const done = job.stage === "vocabulary" ? job.vocabularyProcessed ?? 0 : job.processed
  return job.total > 0 ? Math.max(0, Math.min(100, done * 100 / job.total)) : 0
}

/** Menus capture the current selection; later grid changes cannot expand the task. */
function openSelected(movieIds: readonly string[], title?: string) {
  selection.value = { movieIds: [...new Set(movieIds)], title }
  error.value = ""
  dialogOpen.value = true
}

function openAll() {
  selection.value = null
  error.value = ""
  dialogOpen.value = true
}

/** Coverage is fetched on demand; it does not depend on the last 50 task rows. */
async function refreshStats() {
  if (statsRequest) return statsRequest
  const currentGeneration = generation
  statsLoading.value = true
  statsError.value = false
  statsRequest = (async () => {
    try {
      const next = await useAIService().getTagOrganizationStats()
      if (currentGeneration === generation) stats.value = next
    } catch {
      if (currentGeneration === generation) { statsError.value = true; stats.value = null }
    } finally {
      if (currentGeneration === generation) { statsLoading.value = false; statsRequest = undefined }
    }
  })()
  return statsRequest
}

/** 稳定错误码映射为操作提示，不展示模型原文或数据库内部信息。 */
export function organizationErrorText(code: string): string {
  const key = `topics.errors.${code}`
  return i18n.global.te(key) ? i18n.global.t(key) : i18n.global.t("topics.needsAttention")
}

/** 全屏时只保存结果，不弹提示。 */
function notifyResult(job: TagOrganizationJob) {
  const noticeKey = `${job.id}:${job.status}:${job.revision}`
  if (notices.has(noticeKey)) return
  notices.add(noticeKey)
  const t = i18n.global.t
  const text = job.status === "completed" ? t("topics.completed", { count: job.succeeded }) : t("topics.needsAttention")
  useNotificationCenter().addNotification({ messageId: job.status === "completed" ? "MSG-0048" : "MSG-0049", type: "system", severity: job.status === "completed" ? "success" : "warning", title: t("topics.organize"), message: text, source: { taskId: job.id, route: "/" }, group: `tag-organization-${job.id}` })
  if (!document.fullscreenElement && !quiet.value && !dialogOpen.value) pushAppToast(text, { variant: job.status === "completed" ? "success" : "warning" })
}

/** 单飞读取持久任务；首次快照不重放 toast，迟到请求不污染新生命周期。 */
async function refreshJobs() {
  if (loading) return loading
  const currentGeneration = generation
  loading = (async () => {
    // 快照可补足错过的 SSE。
    try {
      const next = await useAIService().listTagOrganizations()
      if (currentGeneration !== generation) return
      if (initialized) for (const job of next) {
        const previous = jobs.value.find((item) => { /* 用任务 ID 对账。 */ return item.id === job.id })
        if (previous && isOrganizationActive(previous) && !isOrganizationActive(job)) { if (connected.value && job.status !== "cancelled") notifyResult(job); revision.value++; if (dialogOpen.value) void refreshStats(); void useLibraryService().reloadMoviesFromApi().catch(() => { /* 已有库错误状态负责展示。 */ }) }
      }
      jobs.value = next; connected.value = true; initialized = true
    } catch { if (currentGeneration === generation) connected.value = false }
    finally { if (currentGeneration === generation) loading = undefined }
  })()
  return loading
}

/** 复用已有后端事件连接投递的任务信号。 */
function onTaskEvent() { void refreshJobs() }

/** AppShell 持有唯一任务观察器，离开首页仍可感知整理。 */
export function startTagOrganizationTracking() {
  if (timer) return
  void refreshJobs()
  timer = setInterval(() => { /* 断线时兜底读快照，不启动整理。 */ void refreshJobs(); if (dialogOpen.value) void refreshStats() }, 10_000)
  window.addEventListener("curated:tag-organization-updated", onTaskEvent)
}

/** 锁定或离开壳层后撤销观察，不取消服务端任务。 */
export function stopTagOrganizationTracking() {
  if (timer) clearInterval(timer)
  timer = undefined; generation++; loading = undefined; initialized = false; jobs.value = []; dialogOpen.value = false; selection.value = null
  statsRequest = undefined; stats.value = null; statsLoading.value = false; statsError.value = false
  window.removeEventListener("curated:tag-organization-updated", onTaskEvent)
}

/** 所有显式操作共用忙碌和错误状态。 */
async function operate(action: () => Promise<unknown>) {
  if (busy.value) return
  busy.value = true; error.value = ""
  try { await action(); if (loading) await loading; await refreshJobs(); revision.value++; if (statsRequest) await statsRequest; await refreshStats(); await useLibraryService().reloadMoviesFromApi() }
  catch (e) {
    const code = e instanceof HttpClientError ? e.apiError?.code : e instanceof Error ? e.message : ""
    error.value = code && i18n.global.te(`topics.errors.${code}`) ? organizationErrorText(code) : e instanceof Error ? e.message : String(e)
  }
  finally { busy.value = false }
}

/** 首页与全局状态组件共享同一实例。 */
export function useTagOrganization() {
  return {
    jobs, error, busy, connected, dialogOpen, revision, quiet, selection, openAll, openSelected, stats, statsLoading, statsError, refreshStats,
    active: computed(() => { /* 优先展示活动任务。 */ return jobs.value.find(isOrganizationActive) }),
    refresh: refreshJobs,
    start: (scope: Exclude<TagOrganizationScope, "selected"> = "unorganized") => {
      if (jobs.value.some(isOrganizationActive)) return
      const ids = selection.value ? [...selection.value.movieIds] : undefined
      if (ids && (ids.length === 0 || ids.length > 600)) { error.value = i18n.global.t("topics.selectionLimit"); return }
      if (!ids && (!stats.value || statsError.value || statsLoading.value)) return
      const count = stats.value && (scope === "all" ? stats.value.total : stats.value[scope])
      if (!ids && !count) { error.value = organizationErrorText("AI_ORGANIZATION_NO_MOVIES"); return }
      return operate(() => useAIService().startTagOrganization(ids ? "selected" : scope, ids))
    },
    cancel: (id: string) => operate(() => { /* 取消保留已写入结果。 */ return useAIService().cancelTagOrganization(id) }),
    retry: (id: string) => operate(() => { /* 只重试后端失败项。 */ return useAIService().retryTagOrganization(id) }),
    undo: (id: string) => operate(async () => { /* 撤销反馈显式报告冲突。 */ const result = await useAIService().undoTagOrganization(id); error.value = i18n.global.t("topics.undoResult", result) }),
  }
}

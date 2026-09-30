import { computed, ref } from "vue"
import { useAIService } from "@/services/ai-service"
import { useLibraryService } from "@/services/library-service"
import type { TagOrganizationJob } from "@/services/contracts/topic-service"
import { pushAppToast } from "@/composables/use-app-toast"
import { useNotificationCenter } from "@/composables/use-notification-center"
import { i18n } from "@/i18n"

const jobs = ref<TagOrganizationJob[]>([])
const error = ref("")
const busy = ref(false)
const connected = ref(true)
const dialogOpen = ref(false)
const revision = ref(0)
const quiet = ref(false)
const notices = new Set<string>()
let timer: ReturnType<typeof setInterval> | undefined
let loading: Promise<void> | undefined
let initialized = false
let generation = 0

/** 运行状态与结果状态保持分离。 */
export function isOrganizationActive(job: TagOrganizationJob) { return job.status === "queued" || job.status === "running" }

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
        if (previous && isOrganizationActive(previous) && !isOrganizationActive(job) && job.status !== "cancelled") { if (connected.value) notifyResult(job); revision.value++; void useLibraryService().reloadMoviesFromApi().catch(() => { /* 已有库错误状态负责展示。 */ }) }
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
  timer = setInterval(() => { /* 断线时兜底读快照，不启动整理。 */ void refreshJobs() }, 10_000)
  window.addEventListener("curated:tag-organization-updated", onTaskEvent)
}

/** 锁定或离开壳层后撤销观察，不取消服务端任务。 */
export function stopTagOrganizationTracking() {
  if (timer) clearInterval(timer)
  timer = undefined; generation++; loading = undefined; initialized = false; jobs.value = []; dialogOpen.value = false
  window.removeEventListener("curated:tag-organization-updated", onTaskEvent)
}

/** 所有显式操作共用忙碌和错误状态。 */
async function operate(action: () => Promise<unknown>) {
  if (busy.value) return
  busy.value = true; error.value = ""
  try { await action(); if (loading) await loading; await refreshJobs(); revision.value++; await useLibraryService().reloadMoviesFromApi() }
  catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}

/** 首页与全局状态组件共享同一实例。 */
export function useTagOrganization() {
  return {
    jobs, error, busy, connected, dialogOpen, revision, quiet,
    active: computed(() => { /* 优先展示活动任务。 */ return jobs.value.find(isOrganizationActive) }),
    refresh: refreshJobs,
    start: () => operate(() => { /* 明确点击才发起全库整理。 */ return useAIService().startTagOrganization("all") }),
    cancel: (id: string) => operate(() => { /* 取消保留已写入结果。 */ return useAIService().cancelTagOrganization(id) }),
    retry: (id: string) => operate(() => { /* 只重试后端失败项。 */ return useAIService().retryTagOrganization(id) }),
    undo: (id: string) => operate(async () => { /* 撤销反馈显式报告冲突。 */ const result = await useAIService().undoTagOrganization(id); error.value = i18n.global.t("topics.undoResult", result) }),
  }
}

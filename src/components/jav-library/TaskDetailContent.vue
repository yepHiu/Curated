<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { useI18n } from "vue-i18n"
import { Captions, Download, FileText, FolderOpen, Sparkles } from "lucide-vue-next"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import TaskStageNode, { type TaskStageNodeState } from "@/components/jav-library/TaskStageNode.vue"
import TaskStatusBadge from "@/components/jav-library/TaskStatusBadge.vue"
import MediaEmptyState from "@/components/jav-library/MediaEmptyState.vue"
import { HttpClientError } from "@/api/http-client"
import { useTaskCenterService } from "@/services/task-center-service"
import { useSubtitleService } from "@/services/subtitle-service"
import { useAIService } from "@/services/ai-service"
import { useLibraryService } from "@/services/library-service"
import { useTaskCenter } from "@/composables/use-task-center"
import { useTagOrganization, organizationErrorText } from "@/composables/use-tag-organization"
import { useSubtitleJobs } from "@/composables/use-subtitle-jobs"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { taskActive, taskAttention, taskPercent } from "@/lib/task-center"
import { taskDate, taskStage, taskSubject, taskTitle } from "@/lib/task-center-text"
import type { CenterTask } from "@/services/contracts/task-center-service"
import type { SubtitleAsset } from "@/services/contracts/subtitle-service"
import type { TagOrganizationItem } from "@/services/contracts/topic-service"

const props = defineProps<{ taskKey: string }>()
const { t, te } = useI18n(), service = useTaskCenterService(), center = useTaskCenter()
const { writeEnabled } = useExperimentalAgent()
const task = ref<CenterTask>(), loading = ref(true), unavailable = ref(false), disconnected = ref(false)
const error = ref(""), busy = ref(false), feedback = ref("")
const confirm = ref<"cancel" | "retry" | "restart" | "undo" | "">("")
const preview = ref(""), previewName = ref(""), previewOpen = ref(false)
const items = ref<TagOrganizationItem[]>([]), itemsLoading = ref(false), itemsError = ref(false), itemsMore = ref(false)
const issuesOnly = ref(false)
const visibleItems = computed(() => issuesOnly.value ? items.value.filter(item => ["failed", "conflict"].includes(item.status)) : items.value)
const key = computed(() => props.taskKey)
const canWrite = computed(() => task.value?.canManage && !disconnected.value && (task.value.kind === "subtitle" || writeEnabled.value))
const canRetry = computed(() => {
  const current = task.value
  if (!current || current.subtitle?.supersededBy || !["blocked", "partial", "failed", "cancelled"].includes(current.state)) return false
  if (current.subtitle?.expiresAt && Date.parse(current.subtitle.expiresAt) <= Date.now()) return false
  return true
})
const canRestart = computed(() => task.value?.kind === "subtitle" && !task.value.subtitle?.supersededBy && ["failed", "blocked", "cancelled"].includes(task.value.state))
const steps = computed(() => {
  const current = task.value
  if (!current) return []
  // Extraction/recognition are interleaved per segment: never claim all extraction finished.
  const labels = current.kind === "subtitle" ? ["preparing", "recognizing", "translating", "saving"] : ["queued", "vocabulary", "classifying", "done"]
  const stage = current.kind === "subtitle" && ["extracting", "uploading"].includes(current.stage) ? "recognizing" : current.stage === "waiting_quota" || current.stage === "classify" ? "classifying" : current.stage === "finished" || current.stage === "done" ? labels.at(-1)! : current.stage
  const index = labels.indexOf(stage)
  return labels.map((name, i) => {
    let state: TaskStageNodeState = "pending"
    let text = "tasks.stepPending"
    if (current.state === "succeeded" || index >= 0 && i < index) { state = "success"; text = "tasks.stepDone" }
    else if (i === index) {
      if (["failed", "blocked", "partial"].includes(current.state)) { state = "error"; text = `tasks.states.${current.state}` }
      else if (["waiting", "cancelled"].includes(current.state)) { state = "paused"; text = `tasks.states.${current.state}` }
      else if (current.state === "running") { state = disconnected.value ? "paused" : "running"; text = disconnected.value ? "tasks.disconnected" : "tasks.stepCurrent" }
    }
    return { name, state, text }
  })
})
let generation = 0, itemGeneration = 0, request = 0, alive = true
async function load(quiet = false) {
  const seq = generation, req = ++request, id = key.value
  if (!quiet) loading.value = true
  try {
    const next = await service.get(id)
    if (!alive || seq !== generation || req !== request) return
    task.value = next; disconnected.value = false; unavailable.value = false
  } catch (e) {
    if (!alive || seq !== generation || req !== request) return
    unavailable.value = e instanceof HttpClientError && e.status === 404 || e instanceof Error && e.message === "TASK_NOT_FOUND"
    disconnected.value = !unavailable.value
    if (unavailable.value) task.value = undefined
  } finally { if (alive && seq === generation && req === request) loading.value = false }
}
async function loadItems(more = false) {
  const current = task.value
  if (!current?.organization || itemsLoading.value) return
  const seq = ++itemGeneration, scope = generation
  itemsLoading.value = true; itemsError.value = false
  const offset = more ? items.value.length : 0
  try {
    const page = await useAIService().getTagOrganizationItems(current.id, offset)
    if (scope !== generation || seq !== itemGeneration) return
    items.value = more ? [...items.value, ...page] : page; itemsMore.value = page.length === 25
  } catch { if (scope === generation && seq === itemGeneration) itemsError.value = true }
  finally { if (scope === generation && seq === itemGeneration) itemsLoading.value = false }
}
watch(key, async () => {
  generation++; itemGeneration++; task.value = undefined; items.value = []; itemsLoading.value = false; itemsMore.value = false; itemsError.value = false
  error.value = ""; feedback.value = ""; confirm.value = ""; previewOpen.value = false; busy.value = false; issuesOnly.value = false
  const scope = generation
  await load()
  if (scope === generation) void loadItems()
}, { immediate: true })
watch(center.revision, () => { if (!loading.value && !busy.value) void load(true) })
watch(() => task.value?.organization?.revision, (value, old) => { if (old !== undefined && value !== old && items.value.length <= 25) void loadItems() })
onBeforeUnmount(() => { alive = false; generation++; itemGeneration++ })

async function perform() {
  const current = task.value, operation = confirm.value, seq = generation
  if (!current || !operation || !canWrite.value || busy.value) return
  busy.value = true; error.value = ""; feedback.value = ""
  try {
    const result = await service.action(current, operation)
    if (!alive || seq !== generation) return
    confirm.value = ""
    if (result) feedback.value = t("topics.undoResult", result)
    await load(true)
    if (!alive || seq !== generation) return
    if (current.organization) {
      await useTagOrganization().refresh()
      await useLibraryService().reloadMoviesFromApi()
      if (!alive || seq !== generation) return
      await loadItems()
    } else await useSubtitleJobs().refresh()
    await center.refresh()
  } catch (e) { if (alive && seq === generation) error.value = e instanceof Error ? e.message : t("tasks.actionFailed") }
  finally { if (alive && seq === generation) busy.value = false }
}
async function assetAction(asset: SubtitleAsset, action: "preview" | "download" | "reveal") {
  const seq = generation
  busy.value = true; error.value = ""
  try {
    if (action === "reveal") { await useSubtitleService().reveal(asset.id); return }
    const content = await useSubtitleService().content(asset.id)
    if (!alive || seq !== generation) return
    if (action === "preview") { preview.value = content; previewName.value = asset.name; previewOpen.value = true }
    else { const url = URL.createObjectURL(new Blob([content], { type: "text/plain;charset=utf-8" })); const link = document.createElement("a"); link.href = url; link.download = asset.name; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000) }
  } catch (e) { if (alive && seq === generation) error.value = e instanceof Error ? e.message : t("tasks.actionFailed") }
  finally { if (alive && seq === generation) busy.value = false }
}
</script>
<template>
  <div class="min-w-0" data-task-detail>
    <div class="flex min-w-0 flex-col gap-5">
      <Skeleton v-if="loading" class="h-80 w-full rounded-2xl" />
      <MediaEmptyState v-else-if="unavailable" :title="t('tasks.unavailable')" :description="t('tasks.unavailableDescription')" />
      <Alert v-if="disconnected"><AlertDescription class="flex flex-wrap items-center justify-between gap-3"><span>{{ task ? t('tasks.stale', { time: taskDate(task.updatedAt) }) : t('tasks.loadFailed') }}</span><Button variant="outline" size="sm" @click="load()">{{ t('tasks.reload') }}</Button></AlertDescription></Alert>
      <template v-if="task">
        <div class="flex flex-wrap items-center justify-between gap-4"><div class="flex min-w-0 items-center gap-3"><span class="flex size-12 shrink-0 items-center justify-center rounded-xl bg-secondary text-primary"><Captions v-if="task.kind === 'subtitle'" class="size-6" /><Sparkles v-else class="size-6" /></span><div class="min-w-0"><h2 class="text-xl font-semibold tracking-tight sm:text-2xl">{{ taskTitle(task) }}</h2><p class="mt-1 break-words text-xs text-muted-foreground">{{ taskSubject(task) }}</p></div></div><div class="flex flex-wrap gap-2"><Button v-if="taskActive(task)" variant="outline" class="rounded-full" :disabled="!canWrite || busy" @click="confirm = 'cancel'">{{ t('tasks.cancel') }}</Button><Button v-if="canRetry" class="rounded-full" :disabled="!canWrite || busy" @click="confirm = 'retry'">{{ t('tasks.retry') }}</Button><Button v-if="canRestart" variant="outline" class="rounded-full" :disabled="!canWrite || busy" @click="confirm = 'restart'">{{ t('tasks.restart') }}</Button><Button v-if="task.organization && !taskActive(task) && task.organization.succeeded > 0" variant="outline" class="rounded-full" :disabled="!canWrite || busy" @click="confirm = 'undo'">{{ t('tasks.undo') }}</Button></div></div>
        <Alert v-if="!task.canManage || task.kind === 'organize' && !writeEnabled"><AlertDescription>{{ t(!task.canManage ? 'tasks.readOnly' : 'tasks.aiReadOnly') }}</AlertDescription></Alert>
        <Alert v-if="task.subtitle?.supersededBy"><AlertDescription class="flex flex-wrap items-center justify-between gap-3"><span>{{ t('tasks.superseded') }}</span><Button as-child variant="link"><RouterLink :to="`/tasks/${encodeURIComponent(`subtitle:${task.subtitle.supersededBy}`)}`">{{ t('tasks.viewReplacement') }}</RouterLink></Button></AlertDescription></Alert>
        <Alert v-if="error" variant="destructive"><AlertDescription>{{ error }}</AlertDescription></Alert>
        <Alert v-if="feedback"><AlertDescription>{{ feedback }}</AlertDescription></Alert>
        <Alert v-if="taskAttention(task) || task.subtitle?.supersededBy"><AlertTitle>{{ t(task.state === 'partial' ? 'tasks.partialNotice' : 'tasks.problemNotice') }}</AlertTitle><AlertDescription><p>{{ task.organization?.error ? organizationErrorText(task.organization.error) : task.kind === 'subtitle' ? t(task.stage === 'saving' ? 'tasks.saveProblem' : 'tasks.subtitleProblem') : t('tasks.inspectItems') }}</p><pre v-if="task.subtitle?.providerError" class="mt-2 max-h-48 overflow-y-auto whitespace-pre-wrap break-all font-mono text-xs">{{ task.subtitle.providerError }}</pre></AlertDescription></Alert>
        <div class="grid items-start gap-5 lg:grid-cols-[minmax(0,1fr)_17rem]">
          <div class="flex min-w-0 flex-col gap-5">
            <Card><CardHeader><div class="flex flex-wrap items-start justify-between gap-3"><div class="flex flex-col gap-2"><span class="text-xs text-muted-foreground">{{ t('tasks.currentStage') }}</span><CardTitle>{{ taskStage(task) }}</CardTitle><TaskStatusBadge :state="task.state" /></div><div v-if="taskActive(task) && task.total > 0" class="text-2xl tabular-nums">{{ task.completed }} <span class="text-sm text-muted-foreground">/ {{ task.total }} {{ t(`tasks.units.${task.unit}`) }}</span></div></div></CardHeader><CardContent class="flex flex-col gap-5">
              <div v-if="taskActive(task) && task.total > 0"><Progress :model-value="taskPercent(task)" class="h-1.5" :aria-label="taskStage(task)" /><p class="mt-2 text-xs text-muted-foreground">{{ t('tasks.phaseProgress', { percent: Math.round(taskPercent(task) ?? 0) }) }}</p></div>
              <p v-else-if="taskActive(task)" class="text-sm text-muted-foreground">{{ t(task.state === 'queued' ? 'tasks.queuedDescription' : task.state === 'waiting' ? 'tasks.waitingDescription' : 'tasks.unknownProgress') }}</p>
              <ol class="flex flex-col gap-5"><li v-for="step in steps" :key="`${task.key}:${step.name}`" class="flex items-start gap-3"><TaskStageNode :state="step.state" /><div><p class="text-sm font-medium">{{ t(`tasks.stages.${step.name}`) }}</p><p class="mt-1 text-xs text-muted-foreground">{{ t(step.text) }}</p></div></li></ol>
            </CardContent></Card>
            <Card v-if="task.subtitle"><CardHeader><CardTitle>{{ t('tasks.results') }}</CardTitle></CardHeader><CardContent><p v-if="!task.subtitle.assets.length" class="text-sm text-muted-foreground">{{ t('tasks.noAssets') }}</p><ul v-else class="divide-y divide-border"><li v-for="asset in task.subtitle.assets" :key="asset.id" class="flex flex-wrap items-center gap-3 py-3"><FileText class="size-5 shrink-0 text-muted-foreground" /><div class="min-w-0 flex-1"><p class="break-all text-sm">{{ asset.name }}</p><p class="text-xs text-muted-foreground">{{ asset.language }} · {{ asset.format }}<span v-if="asset.missing"> · {{ t('subtitles.missing') }}</span></p></div><div class="flex flex-wrap gap-1"><Button variant="ghost" size="sm" :disabled="asset.missing || busy || disconnected" @click="assetAction(asset, 'preview')">{{ t('tasks.preview') }}</Button><Button variant="ghost" size="icon" :aria-label="t('subtitles.download')" :disabled="asset.missing || busy || disconnected" @click="assetAction(asset, 'download')"><Download /></Button><Button v-if="task.canManage" variant="ghost" size="icon" :aria-label="t('subtitles.reveal')" :disabled="asset.missing || busy || disconnected" @click="assetAction(asset, 'reveal')"><FolderOpen /></Button></div></li></ul></CardContent></Card>
            <Card v-if="task.organization"><CardHeader><CardTitle>{{ t('tasks.items') }}</CardTitle></CardHeader><CardContent class="flex flex-col gap-4">
              <div class="grid grid-cols-3 gap-3"><div v-for="stat in [{ value: task.organization.succeeded, label: 'applied' }, { value: task.organization.unresolved, label: 'unmatched' }, { value: task.organization.failed, label: 'problems' }]" :key="stat.label"><p class="text-2xl tabular-nums">{{ stat.value }}</p><p class="text-xs text-muted-foreground">{{ t(`tasks.${stat.label}`) }}</p></div></div>
              <Separator /><div class="flex flex-wrap items-center justify-between gap-2"><Button variant="outline" size="sm" :aria-pressed="issuesOnly" @click="issuesOnly = !issuesOnly">{{ t(issuesOnly ? 'tasks.showAllItems' : 'tasks.showIssues') }}</Button><Button variant="ghost" size="sm" :disabled="itemsLoading" @click="loadItems()">{{ t('tasks.reload') }}</Button></div>
              <Alert v-if="itemsError" variant="destructive"><AlertDescription>{{ t('tasks.itemsFailed') }}</AlertDescription></Alert>
              <p v-if="!visibleItems.length && !itemsLoading" class="text-sm text-muted-foreground">{{ t('tasks.noLoadedItems') }}</p>
              <ul class="divide-y divide-border"><li v-for="item in visibleItems" :key="item.movieId" class="flex flex-col gap-2 py-4"><div class="flex flex-wrap items-center justify-between gap-2"><RouterLink :to="{ name: 'detail', params: { id: item.movieId } }" class="text-sm font-medium hover:underline">{{ item.title || item.movieId }}</RouterLink><Badge :variant="['failed','conflict'].includes(item.status) ? 'warning' : 'secondary'">{{ t(te(`tasks.itemStates.${item.status}`) ? `tasks.itemStates.${item.status}` : 'tasks.states.unknown') }}</Badge></div><p v-if="item.reason" class="text-xs text-muted-foreground">{{ organizationErrorText(item.reason) }}</p><details v-if="item.evidence.length"><summary class="cursor-pointer text-xs text-muted-foreground">{{ t('tasks.evidence') }}</summary><blockquote v-for="(evidence, index) in item.evidence" :key="index" class="mt-2 border-l-2 border-border pl-3 text-xs"><span class="font-medium">{{ evidence.topic }} · {{ evidence.field }}</span><p class="mt-1 whitespace-pre-wrap break-words text-muted-foreground">{{ evidence.quote }}</p></blockquote></details></li></ul>
              <Button v-if="itemsMore" variant="outline" :disabled="itemsLoading" @click="loadItems(true)">{{ t(itemsLoading ? 'tasks.loading' : 'tasks.loadMore') }}</Button><p v-if="itemsLoading" class="text-xs text-muted-foreground" role="status">{{ t('tasks.loading') }}</p>
            </CardContent></Card>
          </div>
          <Card><CardHeader><CardTitle>{{ t('tasks.info') }}</CardTitle></CardHeader><CardContent class="flex flex-col gap-5"><dl class="flex flex-col gap-4 text-sm"><div><dt class="text-xs text-muted-foreground">{{ t('tasks.subject') }}</dt><dd class="mt-1 break-words">{{ taskSubject(task) }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('tasks.created') }}</dt><dd class="mt-1">{{ taskDate(task.createdAt) }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('tasks.updated') }}</dt><dd class="mt-1">{{ taskDate(task.updatedAt) }}</dd></div><div><dt class="text-xs text-muted-foreground">{{ t('tasks.id') }}</dt><dd class="mt-1 break-all">{{ task.key }}</dd></div></dl><Separator /><details v-if="task.subtitle"><summary class="cursor-pointer text-xs text-muted-foreground">{{ t('tasks.parameters') }}</summary><dl class="mt-3 flex flex-col gap-3 text-xs"><div><dt class="text-muted-foreground">{{ t('subtitles.recognitionModel') }}</dt><dd class="mt-1 break-all">{{ task.subtitle.recognitionModel || 'qwen-audio-3.1-asr-flash-filetrans' }}</dd></div><div><dt class="text-muted-foreground">{{ t('subtitles.translationModel') }}</dt><dd class="mt-1 break-all">{{ task.textModel || task.subtitle.translationModel }}</dd></div><div v-if="task.audioTrack !== undefined"><dt class="text-muted-foreground">{{ t('subtitles.audioTrack') }}</dt><dd>{{ task.audioTrack + 1 }}</dd></div></dl></details><p class="text-xs text-muted-foreground">{{ t('tasks.timeNote') }}</p><Button v-if="task.subtitle" as-child variant="outline" class="rounded-full"><RouterLink :to="{ name: 'detail', params: { id: task.subtitle.movieId }, query: { fileId: task.subtitle.fileId } }">{{ t('tasks.movie') }}</RouterLink></Button></CardContent></Card>
        </div>
      </template>
    </div>
    <Dialog :open="!!confirm" @update:open="!$event && (confirm = '')"><DialogContent><DialogHeader><DialogTitle>{{ t(`tasks.${confirm || 'cancel'}`) }}</DialogTitle><DialogDescription>{{ t(`tasks.confirm.${confirm || 'cancel'}`) }}</DialogDescription></DialogHeader><Alert v-if="error" variant="destructive"><AlertDescription>{{ error }}</AlertDescription></Alert><DialogFooter><Button variant="outline" :disabled="busy" @click="confirm = ''">{{ t('tasks.close') }}</Button><Button :disabled="busy || !canWrite" @click="perform">{{ t(busy ? 'tasks.loading' : 'tasks.confirmAction') }}</Button></DialogFooter></DialogContent></Dialog>
    <Dialog v-model:open="previewOpen"><DialogContent class="max-h-[85dvh] overflow-y-auto sm:max-w-2xl"><DialogHeader><DialogTitle>{{ previewName }}</DialogTitle><DialogDescription>{{ t('tasks.preview') }}</DialogDescription></DialogHeader><pre class="whitespace-pre-wrap break-words text-sm">{{ preview }}</pre></DialogContent></Dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { ChevronDown, Download, FolderOpen, Loader2 } from "lucide-vue-next"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import MoviePartSelect from "./MoviePartSelect.vue"
import SubtitleModelSelect from "./SubtitleModelSelect.vue"
import { useSubtitleJobs } from "@/composables/use-subtitle-jobs"
import { useSubtitleService } from "@/services/subtitle-service"
import type { Movie } from "@/domain/movie/types"
import type { SubtitleAsset, SubtitleJob, SubtitleModel, SubtitleSettings } from "@/services/contracts/subtitle-service"
const props = defineProps<{ open: boolean; movie: Movie; initialFileId?: string }>()
const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { t, locale } = useI18n()
const expandedJob = ref("")
const service = useSubtitleService()
const tracking = useSubtitleJobs()
const cfg = ref<SubtitleSettings>()
const fileId = ref<string>()
const model = ref<SubtitleModel>("configured-ai")
const tracks = ref<Awaited<ReturnType<typeof service.audioTracks>>>([])
const audioTrack = ref("0")
const tracksLoading = ref(false)
const tracksError = ref("")
let trackRevision = 0
const jobs = ref<SubtitleJob[]>([])
const busy = ref(false)
const loading = ref(false)
const error = ref("")
const requestId = ref(crypto.randomUUID())
let generation = 0
let timer: ReturnType<typeof setTimeout> | undefined
const job = computed(() => jobs.value.find(j => !fileId.value || j.fileId === fileId.value))
const active = computed(() => job.value && ["queued", "running"].includes(job.value.state))
watch(job, current => { if (current && ["queued", "running"].includes(current.state)) model.value = current.translationModel })
const history = computed(() => jobs.value.filter(j => (!fileId.value || j.fileId === fileId.value) && j.assets.length))
const assets = computed(() => jobs.value.find(j => (!fileId.value || j.fileId === fileId.value) && j.state === "completed")?.assets ?? [])
const ready = computed(() => cfg.value?.canManage && cfg.value.enabled && cfg.value.apiKeyConfigured && (model.value !== "configured-ai" || cfg.value.configuredModel))
async function refresh(seq: number) {
  try { const result = await service.jobs(props.movie.id); if (seq === generation) { jobs.value = result; error.value = "" } }
  catch (e) { if (seq === generation) error.value = e instanceof Error ? e.message : String(e) }
  finally { if (seq === generation && props.open) { loading.value = false; timer = setTimeout(() => void refresh(seq), 2000) } }
}
watch(() => [props.open, props.movie.id] as const, async ([open]) => {
  generation++; clearTimeout(timer); jobs.value = []; error.value = ""
  if (!open) return
  const seq = generation; loading.value = true; fileId.value = props.initialFileId ?? props.movie.files?.[0]?.id
  try { const settings = await service.settings(); if (seq !== generation) return; cfg.value = settings; model.value = settings.translationModel }
  catch (e) { if (seq === generation) error.value = String(e) }
  if (seq === generation) await refresh(seq)
}, { immediate: true })
watch(fileId, async () => {
  requestId.value = crypto.randomUUID(); tracks.value = []; audioTrack.value = "0"
  const seq = ++trackRevision; tracksLoading.value = true; tracksError.value = ""
  try { const result = await service.audioTracks(props.movie.id, fileId.value); if (seq === trackRevision) { tracks.value = result; if (!result.length) tracksError.value = t("subtitles.noAudio"); audioTrack.value = String(result.find(track => ['ja', 'jpn'].includes(track.language))?.index ?? 0) } }
  catch (e) { if (seq === trackRevision) tracksError.value = e instanceof Error ? e.message : String(e) }
  finally { if (seq === trackRevision) tracksLoading.value = false }
}, { immediate: true })
onBeforeUnmount(() => { generation++; trackRevision++; clearTimeout(timer) })
async function run(action: "start" | "cancel" | "retry") {
  busy.value = true; error.value = ""; const seq = generation
  try {
    const result = action === "start" ? await service.create(props.movie.id, { requestId: requestId.value, fileId: fileId.value, translationModel: model.value, audioTrack: Number(audioTrack.value) }) : await service.action(job.value!.id, action)
    tracking.remember(result)
    if (seq === generation) { jobs.value = [result, ...jobs.value.filter(j => j.id !== result.id)]; if (action === "start") requestId.value = crypto.randomUUID() }
  } catch (e) { if (seq === generation) error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
async function fileAction(asset: SubtitleAsset, reveal: boolean) {
  try {
    if (reveal) { await service.reveal(asset.id); return }
    const content = await service.content(asset.id); const url = URL.createObjectURL(new Blob([content], { type: "text/plain;charset=utf-8" }))
    const link = document.createElement("a"); link.href = url; link.download = asset.name; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (e) { error.value = e instanceof Error ? e.message : String(e) }
}
</script>
<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-h-[min(90dvh,48rem)] overflow-y-auto sm:max-w-xl">
      <DialogHeader><DialogTitle>{{ t('subtitles.title') }}</DialogTitle><DialogDescription>{{ movie.code }} · {{ t('subtitles.dialogDescription') }}</DialogDescription></DialogHeader>
      <Alert v-if="error || job?.error || tracksError" variant="destructive"><AlertDescription>{{ error || job?.error || tracksError }}</AlertDescription></Alert>
      <FieldGroup class="gap-3">
        <Field v-if="(movie.files?.length ?? 0) > 1"><FieldLabel for="subtitle-job-file">{{ t('subtitles.videoFile') }}</FieldLabel><MoviePartSelect id="subtitle-job-file" class="sm:w-full" v-model="fileId" :files="movie.files!" /></Field>
        <Field v-if="tracks.length > 1"><FieldLabel for="subtitle-job-audio">{{ t('subtitles.audioTrack') }}</FieldLabel><Select v-model="audioTrack" :disabled="active || busy"><SelectTrigger id="subtitle-job-audio" class="w-full min-w-0 min-h-11 rounded-full px-4 sm:min-h-9" :aria-label="t('subtitles.audioTrack')"><SelectValue /></SelectTrigger><SelectContent><SelectGroup><SelectItem v-for="track in tracks" :key="track.index" :value="String(track.index)">{{ track.index + 1 }} · {{ track.title || track.language || track.codec }} · {{ track.channels }} ch</SelectItem></SelectGroup></SelectContent></Select></Field>
        <Field><FieldLabel for="subtitle-job-model">{{ t('subtitles.translationModel') }}</FieldLabel><SubtitleModelSelect pill id="subtitle-job-model" v-model="model" :configured-model="cfg?.configuredModel" :disabled="active || busy" /></Field>
      </FieldGroup>
      <div v-if="job" class="flex min-w-0 flex-col gap-3 rounded-xl border border-border/50 bg-muted/5 p-4" role="status" aria-live="polite">
        <div class="flex items-center justify-between gap-2"><Badge variant="secondary">{{ t(`subtitles.states.${job.state}`) }}</Badge><span class="text-sm text-muted-foreground">{{ t(`subtitles.stages.${job.stage}`) }}</span></div>
        <Progress v-if="active && job.total > 0" :model-value="job.completed / job.total * 100" />
        <p v-if="active" class="flex items-center gap-2 text-sm text-muted-foreground"><Loader2 class="size-4 shrink-0 motion-safe:animate-spin" aria-hidden="true" />{{ job.total ? `${job.completed} / ${job.total}` : t('subtitles.waiting') }} · {{ job.translationModel }}</p>
        <p v-if="job.expiresAt" class="text-xs text-muted-foreground">{{ t('subtitles.retryHint') }}</p>
      </div>
      <section v-if="history.length" class="flex min-w-0 flex-col gap-2" :aria-label="t('subtitles.outputs')">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <h3 class="font-medium">{{ t('subtitles.outputs') }}</h3>
          <span>{{ t('subtitles.historyCount', { count: history.length }) }}</span>
        </div>
        <ol class="flex max-h-60 min-w-0 flex-col overflow-y-auto overscroll-contain rounded-lg border border-border/60" data-subtitle-history>
          <li v-for="record in history" :key="record.id" class="border-b border-border/60 last:border-0">
            <button type="button" class="flex min-h-11 w-full min-w-0 items-center gap-3 rounded-md px-3 py-2 text-left text-xs hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60" :aria-expanded="expandedJob === record.id" @click="expandedJob = expandedJob === record.id ? '' : record.id">
              <time :datetime="record.createdAt" class="shrink-0 tabular-nums text-muted-foreground">{{ new Date(record.createdAt).toLocaleString(locale, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }}</time>
              <span class="min-w-0 flex-1 truncate" :title="record.fileName">{{ record.translationModel }}</span>
              <Badge variant="secondary">{{ t(`subtitles.states.${record.state}`) }}</Badge>
              <ChevronDown class="size-3 shrink-0" :class="{ 'rotate-180': expandedJob === record.id }" aria-hidden="true" />
            </button>
            <ul v-if="expandedJob === record.id" class="flex flex-col px-3 pb-2">
              <li v-for="asset in record.assets" :key="asset.id" class="flex min-w-0 items-center gap-2 py-1">
                <span class="min-w-0 flex-1 break-all text-xs">{{ asset.name }}<span v-if="asset.missing"> · {{ t('subtitles.missing') }}</span></span>
                <Button size="icon" variant="ghost" class="size-11 shrink-0 rounded-full sm:size-9" :disabled="asset.missing" :aria-label="t('subtitles.download')" @click="fileAction(asset, false)"><Download /></Button>
                <Button v-if="cfg?.canManage" size="icon" variant="ghost" class="size-11 shrink-0 rounded-full sm:size-9" :disabled="asset.missing" :aria-label="t('subtitles.reveal')" @click="fileAction(asset, true)"><FolderOpen /></Button>
              </li>
            </ul>
          </li>
        </ol>
      </section>
      <p class="text-xs text-muted-foreground">{{ active ? t('subtitles.backgroundHint') : t('subtitles.cloudNotice') }}</p>
      <p v-if="cfg && !ready" class="text-sm text-muted-foreground">{{ cfg.canManage ? t('subtitles.configureFirst') : t('subtitles.localOnly') }}</p>
      <DialogFooter class="flex-row flex-wrap justify-end gap-2">
        <Button v-if="active && cfg?.canManage" class="min-h-11 rounded-full sm:min-h-9" variant="destructive" :disabled="busy" @click="run('cancel')">{{ t('subtitles.cancel') }}</Button>
        <Button v-if="job && ['failed','blocked','cancelled'].includes(job.state) && cfg?.canManage" class="min-h-11 rounded-full sm:min-h-9" variant="outline" :disabled="busy" @click="run('retry')">{{ t('subtitles.retry') }}</Button>
        <Button v-if="!active" class="min-h-11 rounded-full sm:min-h-9" :disabled="!ready || busy || loading || tracksLoading || !tracks.length" @click="run('start')">{{ assets.length ? t('subtitles.regenerate') : t('subtitles.start') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

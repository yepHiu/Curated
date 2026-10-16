<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { ArrowLeft, ChevronRight } from "lucide-vue-next"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import MoviePartSelect from "./MoviePartSelect.vue"
import SubtitleModelSelect from "./SubtitleModelSelect.vue"
import TaskDetailDialog from "./TaskDetailDialog.vue"
import TaskStatusBadge from "./TaskStatusBadge.vue"
import { isSubtitleActive, useSubtitleJobs } from "@/composables/use-subtitle-jobs"
import { useSubtitleService } from "@/services/subtitle-service"
import { taskState } from "@/lib/task-center"
import type { Movie } from "@/domain/movie/types"
import type { SubtitleJob, SubtitleModel, SubtitleSettings } from "@/services/contracts/subtitle-service"

const props = defineProps<{ open: boolean; movie: Movie; initialFileId?: string }>()
const emit = defineEmits<{ "update:open": [value: boolean] }>()
const { t, locale } = useI18n()
const service = useSubtitleService()
const tracking = useSubtitleJobs()
const cfg = ref<SubtitleSettings>()
const fileId = ref<string>()
const model = ref<SubtitleModel>("configured-ai")
const tracks = ref<Awaited<ReturnType<typeof service.audioTracks>>>([])
const audioTrack = ref("0")
const tracksLoading = ref(false)
const tracksError = ref("")
const jobs = ref<SubtitleJob[]>([])
const busy = ref(false)
const loading = ref(false)
const error = ref("")
const requestId = ref(crypto.randomUUID())
const selectedTaskKey = ref("")
const setupEl = ref<HTMLElement>()
let generation = 0, trackRevision = 0

const history = computed(() => jobs.value.filter(job => !fileId.value || job.fileId === fileId.value))
const activeJob = computed(() => history.value.find(isSubtitleActive))
const hasAssets = computed(() => history.value.some(job => job.assets.length))
const ready = computed(() => cfg.value?.canManage && cfg.value.enabled && cfg.value.apiKeyConfigured && (model.value !== "configured-ai" || cfg.value.configuredModel))
watch(activeJob, current => { model.value = current?.translationModel ?? cfg.value?.translationModel ?? "configured-ai" })

watch(tracking.jobs, next => {
  if (props.open) jobs.value = next.filter(item => item.movieId === props.movie.id)
})
watch(() => [props.open, props.movie.id] as const, async ([open]) => {
  const seq = ++generation
  jobs.value = []; cfg.value = undefined; error.value = ""; selectedTaskKey.value = ""; loading.value = open
  if (!open) return
  fileId.value = props.initialFileId ?? props.movie.files?.[0]?.id
  try {
    const [settings] = await Promise.allSettled([service.settings(), tracking.refresh()])
    if (seq !== generation) return
    if (settings.status === "fulfilled") {
      cfg.value = settings.value; model.value = activeJob.value?.translationModel ?? settings.value.translationModel
    } else error.value = settings.reason instanceof Error ? settings.reason.message : String(settings.reason)
    jobs.value = tracking.jobs.value.filter(item => item.movieId === props.movie.id)
    if (activeJob.value) selectedTaskKey.value = `subtitle:${activeJob.value.id}`
  } catch (e) {
    if (seq === generation) error.value = e instanceof Error ? e.message : String(e)
  } finally { if (seq === generation) loading.value = false }
}, { immediate: true })

watch([() => props.open, () => props.movie.id, fileId], async ([open]) => {
  const seq = ++trackRevision
  requestId.value = crypto.randomUUID(); tracks.value = []; audioTrack.value = "0"; tracksError.value = ""
  tracksLoading.value = open
  if (!open) return
  try {
    const result = await service.audioTracks(props.movie.id, fileId.value)
    if (seq !== trackRevision) return
    tracks.value = result
    if (!result.length) tracksError.value = t("subtitles.noAudio")
    audioTrack.value = String(result.find(track => ['ja', 'jpn'].includes(track.language))?.index ?? result[0]?.index ?? 0)
  } catch (e) { if (seq === trackRevision) tracksError.value = e instanceof Error ? e.message : String(e) }
  finally { if (seq === trackRevision) tracksLoading.value = false }
}, { immediate: true })
onBeforeUnmount(() => { generation++; trackRevision++ })

async function showSetup() {
  selectedTaskKey.value = ""
  await nextTick()
  setupEl.value?.focus()
}
async function start() {
  if (!ready.value || busy.value || loading.value || tracksLoading.value || !tracks.value.length || activeJob.value || !tracking.connected.value) return
  busy.value = true; error.value = ""
  const seq = generation
  try {
    const result = await service.create(props.movie.id, { requestId: requestId.value, fileId: fileId.value, translationModel: model.value, audioTrack: Number(audioTrack.value) })
    // Execution belongs to Server; closing the dialog never cancels a submitted job.
    tracking.remember(result)
    window.dispatchEvent(new Event("curated:task-center-updated"))
    if (seq === generation) {
      jobs.value = [result, ...jobs.value.filter(job => job.id !== result.id)]
      requestId.value = crypto.randomUUID()
      selectedTaskKey.value = `subtitle:${result.id}`
    }
  } catch (e) { if (seq === generation) error.value = e instanceof Error ? e.message : String(e) }
  finally { busy.value = false }
}
</script>

<template>
  <TaskDetailDialog
    :open="open"
    :task-key="selectedTaskKey"
    :title="selectedTaskKey ? t('tasks.detailTitle') : t('subtitles.title')"
    :description="selectedTaskKey ? undefined : `${movie.code} · ${t('subtitles.dialogDescription')}`"
    @update:open="emit('update:open', $event)"
  >
    <template #actions>
      <Button v-if="selectedTaskKey" variant="ghost" size="sm" class="mt-2 self-start rounded-full" @click="showSetup">
        <ArrowLeft data-icon="inline-start" />{{ t('subtitles.setupAndHistory') }}
      </Button>
    </template>
    <div ref="setupEl" class="flex min-w-0 flex-col gap-5 outline-none" tabindex="-1" data-subtitle-setup>
      <Alert v-if="error || tracksError" variant="destructive"><AlertDescription>{{ error || tracksError }}</AlertDescription></Alert>
      <Alert v-if="!tracking.connected.value"><AlertDescription>{{ t('tasks.loadFailed') }}</AlertDescription></Alert>
      <FieldGroup class="gap-3">
        <Field v-if="(movie.files?.length ?? 0) > 1">
          <FieldLabel for="subtitle-job-file">{{ t('subtitles.videoFile') }}</FieldLabel>
          <MoviePartSelect id="subtitle-job-file" class="sm:w-full" v-model="fileId" :files="movie.files!" :disabled="busy || loading" />
        </Field>
        <Field v-if="tracks.length > 1">
          <FieldLabel for="subtitle-job-audio">{{ t('subtitles.audioTrack') }}</FieldLabel>
          <Select v-model="audioTrack" :disabled="!!activeJob || busy || loading">
            <SelectTrigger id="subtitle-job-audio" class="w-full min-w-0 min-h-11 rounded-full px-4 sm:min-h-9" :aria-label="t('subtitles.audioTrack')"><SelectValue /></SelectTrigger>
            <SelectContent><SelectGroup><SelectItem v-for="track in tracks" :key="track.index" :value="String(track.index)">{{ track.index + 1 }} · {{ track.title || track.language || track.codec }} · {{ track.channels }} ch</SelectItem></SelectGroup></SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel for="subtitle-job-model">{{ t('subtitles.translationModel') }}</FieldLabel>
          <SubtitleModelSelect pill id="subtitle-job-model" v-model="model" :configured-model="cfg?.configuredModel" :disabled="!!activeJob || busy || loading" />
        </Field>
      </FieldGroup>
      <section v-if="history.length" class="flex min-w-0 flex-col gap-2" :aria-label="t('subtitles.taskHistory')">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <h3 class="font-medium">{{ t('subtitles.taskHistory') }}</h3>
          <span>{{ t('subtitles.historyCount', { count: history.length }) }}</span>
        </div>
        <ol class="flex max-h-60 min-w-0 flex-col overflow-y-auto overscroll-contain rounded-lg border border-border/60" data-subtitle-history>
          <li v-for="record in history" :key="record.id" class="border-b border-border/60 last:border-0">
            <button
              type="button" aria-haspopup="dialog"
              class="flex min-h-11 w-full min-w-0 items-center gap-3 px-3 py-3 text-left text-xs hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
              @click="selectedTaskKey = `subtitle:${record.id}`"
            >
              <time :datetime="record.createdAt" class="shrink-0 tabular-nums text-muted-foreground">{{ new Date(record.createdAt).toLocaleString(locale, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }) }}</time>
              <span class="min-w-0 flex-1 truncate" :title="record.fileName">{{ record.translationModel }}</span>
              <TaskStatusBadge :state="taskState(record.state, record.stage)" />
              <ChevronRight class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
            </button>
          </li>
        </ol>
      </section>
      <p class="text-xs text-muted-foreground">{{ activeJob ? t('subtitles.backgroundHint') : t('subtitles.cloudNotice') }}</p>
      <p v-if="cfg && !ready" class="text-sm text-muted-foreground">{{ cfg.canManage ? t('subtitles.configureFirst') : t('subtitles.localOnly') }}</p>
      <div class="flex flex-wrap justify-end gap-2">
        <Button v-if="activeJob" class="min-h-11 rounded-full sm:min-h-9" @click="selectedTaskKey = `subtitle:${activeJob.id}`">{{ t('tasks.viewTask') }}</Button>
        <Button v-else class="min-h-11 rounded-full sm:min-h-9" :disabled="!ready || busy || loading || tracksLoading || !tracks.length || !tracking.connected.value" @click="start">{{ hasAssets ? t('subtitles.regenerate') : t('subtitles.start') }}</Button>
      </div>
    </div>
  </TaskDetailDialog>
</template>

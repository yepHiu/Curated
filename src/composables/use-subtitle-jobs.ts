import { computed, ref } from "vue"
import { useSubtitleService } from "@/services/subtitle-service"
import type { SubtitleJob } from "@/services/contracts/subtitle-service"
import type { Movie } from "@/domain/movie/types"
import { useNotificationCenter } from "./use-notification-center"
import { i18n } from "@/i18n"

const jobs = ref<SubtitleJob[]>([])
const connected = ref(true)
const error = ref("")
const dialogOpen = ref(false)
const movie = ref<Movie>()
const fileId = ref<string>()
let timer: ReturnType<typeof setTimeout> | undefined
let generation = 0
let opening = 0
let pending: Promise<void> | undefined
let initialized = false
export const isSubtitleActive = (job: SubtitleJob) => ["queued", "running"].includes(job.state)
export const subtitleProgress = (job: SubtitleJob) => job.total > 0 ? Math.max(0, Math.min(100, job.completed / job.total * 100)) : 0

async function refresh() {
  if (pending) return pending
  const seq = generation
  pending = (async () => {
    try {
      const next = await useSubtitleService().jobs()
      if (seq === generation) {
        if (initialized && connected.value) for (const job of next) {
          const previous = jobs.value.find(item => item.id === job.id)
          if (previous && isSubtitleActive(previous) && ["completed", "failed", "blocked"].includes(job.state)) {
            const success = job.state === "completed"
            useNotificationCenter().addNotification({ messageId: success ? "MSG-0050" : "MSG-0051", type: "system", severity: success ? "success" : "warning", title: i18n.global.t("subtitles.title"), message: `${job.fileName} · ${i18n.global.t(`subtitles.states.${job.state}`)}`, source: { taskId: job.id, movieId: job.movieId, route: `/tasks/${encodeURIComponent(`subtitle:${job.id}`)}` }, group: `subtitle-${job.id}` })
          }
        }
        jobs.value = next; connected.value = true; initialized = true
      }
    } catch { if (seq === generation) connected.value = false }
    finally { if (seq === generation) pending = undefined }
  })()
  return pending
}
async function poll(seq: number) {
  await refresh()
  if (seq === generation) timer = setTimeout(() => void poll(seq), jobs.value.some(isSubtitleActive) ? 2000 : 10000)
}
export function startSubtitleTracking() {
  stopSubtitleTracking()
  void poll(generation)
}
export function stopSubtitleTracking() {
  generation++; opening++; clearTimeout(timer); pending = undefined; initialized = false; connected.value = true
  jobs.value = []; movie.value = undefined; dialogOpen.value = false; error.value = ""
}
async function openJob(job: SubtitleJob) {
  const seq = ++opening
  error.value = ""
  try {
    const { useLibraryService } = await import("@/services/library-service")
    const detail = await useLibraryService().loadMovieDetail(job.movieId)
    if (seq !== opening) return
    if (!detail) throw new Error("subtitles.movieUnavailable")
    movie.value = detail; fileId.value = job.fileId; dialogOpen.value = true
  } catch (e) { if (seq === opening) error.value = e instanceof Error ? e.message : String(e) }
}
export function useSubtitleJobs() {
  return { jobs, connected, error, dialogOpen, movie, fileId, refresh, openJob,
    active: computed(() => jobs.value.filter(isSubtitleActive)),
    current: computed(() => jobs.value.find(isSubtitleActive) ?? jobs.value[0]),
    remember(job: SubtitleJob) { jobs.value = [job, ...jobs.value.filter(item => item.id !== job.id)] },
  }
}

import { ref } from "vue"
import { useTaskCenterService } from "@/services/task-center-service"
import { emptyTaskSummary } from "@/lib/task-center"
import { useTagOrganization, stopTagOrganizationTracking } from "./use-tag-organization"
import { useSubtitleJobs, stopSubtitleTracking } from "./use-subtitle-jobs"

const summary = ref(emptyTaskSummary())
const connected = ref(true), loaded = ref(false), updatedAt = ref("")
const revision = ref(0)
let generation = 0, pollVersion = 0, timer: ReturnType<typeof setTimeout> | undefined, pending: Promise<void> | undefined
async function refresh() {
  if (pending) return pending
  const seq = generation
  pending = (async () => {
    try {
      const [next] = await Promise.all([useTaskCenterService().summary(), useTagOrganization().refresh(), useSubtitleJobs().refresh()])
      if (seq !== generation) return
      summary.value = next; connected.value = true; loaded.value = true; updatedAt.value = new Date().toISOString(); revision.value++
    } catch { if (seq === generation) connected.value = false }
    finally { if (seq === generation) pending = undefined }
  })()
  return pending
}
async function poll(seq: number) {
  const version = ++pollVersion
  await refresh()
  if (seq !== generation || version !== pollVersion) return
  const busy = summary.value.running + summary.value.queued + summary.value.waiting > 0
  timer = setTimeout(() => void poll(seq), document.hidden ? 30000 : busy ? 2000 : 10000)
}
function wake() { if (!document.hidden) { clearTimeout(timer); void poll(generation) } }
export function startTaskCenterTracking() {
  stopTaskCenterTracking(); void poll(generation)
  document.addEventListener("visibilitychange", wake)
  window.addEventListener("curated:task-center-updated", wake)
}
export function stopTaskCenterTracking() {
  generation++; pollVersion++; clearTimeout(timer); pending = undefined
  stopTagOrganizationTracking(); stopSubtitleTracking()
  summary.value = emptyTaskSummary(); loaded.value = false; connected.value = true; updatedAt.value = ""
  document.removeEventListener("visibilitychange", wake)
  window.removeEventListener("curated:task-center-updated", wake)
}
export function useTaskCenter() { return { summary, connected, loaded, updatedAt, revision, refresh } }

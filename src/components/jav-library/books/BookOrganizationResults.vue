<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { HttpClientError } from "@/api/http-client"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import TaskProcessingItem from "@/components/jav-library/TaskProcessingItem.vue"
import TaskItemsPagination from "@/components/jav-library/TaskItemsPagination.vue"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import BookMetadataFacts from "./BookMetadataFacts.vue"
import { useBookOrganizationService } from "@/services/book-organization-service"
import { useComicLibraryService } from "@/services/comic-library-service"
import { usePhotoLibraryService } from "@/services/photo-library-service"
import type { BookOrganizationJob, BookOrganizationItem } from "@/services/contracts/book-organization-service"

const props = defineProps<{ job: BookOrganizationJob; canWrite: boolean }>()
const emit = defineEmits<{ updated: [] }>()
const { t } = useI18n(), service = useBookOrganizationService()
const items = ref<BookOrganizationItem[]>([]), selected = ref<Record<string, number>>({})
const busy = ref(false), loading = ref(false), error = ref(""), feedback = ref(""), offset = ref(0)
const ready = computed(() => items.value.filter(item => item.status === "ready"))
const chosen = computed(() => ready.value.filter(item => selected.value[item.bookId] === item.revision))
const mockMode = import.meta.env.VITE_USE_WEB_API !== "true"
const quotaSeconds = ref(0)
let cancelQuotaWait: (() => void) | undefined
function stopQuotaWait() { cancelQuotaWait?.() }
function waitForQuota(): Promise<boolean> {
  return new Promise(resolve => {
    quotaSeconds.value = 60
    const finish = (resume: boolean) => {
      clearInterval(timer); quotaSeconds.value = 0; cancelQuotaWait = undefined; resolve(resume)
    }
    const timer = setInterval(() => { if (--quotaSeconds.value <= 0) finish(true) }, 1000)
    cancelQuotaWait = () => finish(false)
  })
}
function failureMessage(reason: string) {
  const keys: Record<string, string> = {
    AI_WRITE_CONFLICT: "sourceConflict",
    BOOK_METADATA_INVALID_RESULT: "invalidResult",
    BOOK_METADATA_PROVIDER_FAILED: "providerFailed",
    BOOK_METADATA_TRUNCATED: "truncatedResult",
  }
  return t(`bookAI.${keys[reason] ?? "recognitionFailed"}`)
}
let generation = 0, request = 0, alive = true, requestedOffset = 0
async function load(nextOffset = requestedOffset) {
  const seq = generation, req = ++request, id = props.job.id
  requestedOffset = nextOffset
  loading.value = true
  try {
    const next = await service.items(id, nextOffset)
    if (!alive || seq !== generation || req !== request) return
    if (offset.value !== nextOffset) selected.value = {}
    offset.value = nextOffset
    items.value = next
    selected.value = Object.fromEntries(next.filter(i => i.status === "ready" && selected.value[i.bookId] === i.revision).map(i => [i.bookId, i.revision]))
  } catch { if (alive && seq === generation && req === request) { requestedOffset = offset.value; error.value = t("tasks.itemsFailed") } }
  finally { if (alive && seq === generation && req === request) loading.value = false }
}
watch(() => props.job.id, () => { generation++; stopQuotaWait(); busy.value = false; requestedOffset = 0; offset.value = 0; selected.value = {}; items.value = []; error.value = ""; feedback.value = ""; void load() }, { immediate: true })
watch(() => [props.job.updatedAt, props.job.processed, props.job.pendingReview, props.job.status], () => { if (!busy.value) void load() })
watch(() => props.canWrite, value => { if (!value) stopQuotaWait() })
onBeforeUnmount(() => { alive = false; generation++; stopQuotaWait() })
function select(item: BookOrganizationItem, checked: boolean | "indeterminate") {
  if (checked === true) selected.value[item.bookId] = item.revision
  else delete selected.value[item.bookId]
}
function page(nextOffset: number) { error.value = ""; void load(nextOffset) }
async function review(decision: "apply" | "discard") {
  if (busy.value || loading.value || !props.canWrite || !chosen.value.length) return
  const batch = [...chosen.value], id = props.job.id, kind = props.job.kind, seq = generation
  busy.value = true; error.value = ""; feedback.value = ""
  let succeeded = 0, failed = 0
  let paused = false, conflict = false
  const completed = new Set<string>()
  const active = () => alive && seq === generation && props.canWrite
  for (const item of batch) {
    let quotaRetries = 0
    while (active()) {
      try {
        await service.review(id, { bookId: item.bookId, revision: item.revision, decision })
        succeeded++; completed.add(item.bookId); break
      } catch (err) {
        const code = err instanceof HttpClientError ? err.apiError?.code : undefined
        if (code === "AI_RATE_LIMITED" && decision === "apply" && active()) {
          // Keep the confirmed revision, without regenerating metadata or bypassing the write budget.
          if (quotaRetries++ < 3 && await waitForQuota()) continue
          paused = true; break
        }
        conflict ||= code === "AI_WRITE_CONFLICT"
        failed++; break
      }
    }
    if (paused || !active()) break
  }
  window.dispatchEvent(new Event("curated:task-center-updated"))
  if (alive && seq === generation) {
    selected.value = Object.fromEntries(Object.entries(selected.value).filter(([bookId]) => !completed.has(bookId)))
    feedback.value = paused ? t("bookAI.reviewPaused", { succeeded }) : t("bookAI.reviewResult", { succeeded, failed })
    if (failed) error.value = t(conflict ? "bookAI.sourceConflict" : "bookAI.reviewFailed")
    await load(); emit("updated")
    busy.value = false
  }
  // Refresh shared library state after the durable review, independently of navigation.
  try {
    if (kind === "comic") await useComicLibraryService().reloadComicsFromApi()
    else await usePhotoLibraryService().reloadPhotosFromApi()
  } catch { /* The task result is authoritative; the library can be refreshed separately. */ }
}
</script>

<template>
  <Card data-book-organization-results><CardHeader><CardTitle>{{ t('bookAI.results') }}</CardTitle></CardHeader><CardContent class="flex flex-col gap-4">
    <p class="text-sm text-muted-foreground">{{ t('bookAI.reviewSummary', { pending: job.pendingReview, applied: job.applied, failed: job.failed }) }}</p>
    <Alert v-if="mockMode"><AlertDescription>{{ t('bookAI.mock') }}</AlertDescription></Alert>
    <p class="text-sm text-muted-foreground">{{ t('bookAI.reviewHint') }}</p>
    <div class="flex flex-wrap gap-2">
      <Button variant="outline" size="sm" :disabled="busy || loading || !ready.length || !canWrite" @click="selected = Object.fromEntries(ready.map(i => [i.bookId, i.revision]))">{{ t('bookAI.selectPage') }}</Button>
      <Button size="sm" :disabled="busy || loading || !chosen.length || !canWrite" @click="review('apply')">{{ t('bookAI.apply', { count: chosen.length }) }}</Button>
      <Button variant="outline" size="sm" :disabled="busy || loading || !chosen.length || !canWrite" @click="review('discard')">{{ t('bookAI.discard', { count: chosen.length }) }}</Button>
      <Button variant="ghost" size="sm" :disabled="busy || loading" @click="error = ''; load()">{{ t('tasks.reload') }}</Button>
    </div>
    <Alert v-if="error" variant="destructive"><AlertDescription>{{ error }}</AlertDescription></Alert>
    <p v-if="feedback" class="text-sm" role="status">{{ feedback }}</p>
    <div v-if="quotaSeconds" class="flex flex-wrap items-center gap-2">
      <p class="text-sm text-muted-foreground" role="status">{{ t('bookAI.waitingQuota', { seconds: quotaSeconds }) }}</p>
      <Button variant="outline" size="sm" @click="stopQuotaWait">{{ t('bookAI.stopWaiting') }}</Button>
    </div>
    <p v-else-if="busy || loading" class="text-sm text-muted-foreground" role="status">{{ t('tasks.loading') }}</p>
    <ul class="divide-y divide-border" :aria-busy="loading"><li v-for="item in items" :key="item.bookId" class="flex min-w-0 items-start gap-1 py-0.5">
      <label class="flex size-11 shrink-0 cursor-pointer items-center justify-center">
        <Checkbox :model-value="selected[item.bookId] === item.revision" :disabled="item.status !== 'ready' || busy || loading || !canWrite" :aria-label="t('bookAI.select', { title: item.title })" @update:model-value="select(item, $event)" />
      </label>
      <TaskProcessingItem :title="item.title || item.bookId" :status="t(`bookAI.status.${item.status}`)" :warning="['failed', 'conflict'].includes(item.status)">
        <template v-if="item.metadata.cleanTitle">
          <p class="text-sm text-muted-foreground">{{ t('bookAI.before') }} · {{ item.before.title }}</p>
          <p class="text-sm font-medium">{{ t('bookAI.after') }} · {{ item.metadata.translatedTitle }}</p>
          <details v-if="item.before.metadata?.cleanTitle"><summary class="cursor-pointer text-sm text-muted-foreground">{{ t('bookAI.saved') }}</summary><BookMetadataFacts :metadata="item.before.metadata" evidence /></details>
          <BookMetadataFacts :metadata="item.metadata" evidence />
        </template>
        <p v-if="item.reason" class="text-sm text-muted-foreground">{{ failureMessage(item.reason) }}</p>
      </TaskProcessingItem>
    </li></ul>
    <TaskItemsPagination :offset="offset" :total="job.total" :disabled="busy || loading" @change="page" />
  </CardContent></Card>
</template>

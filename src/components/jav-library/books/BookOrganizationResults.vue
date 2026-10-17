<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
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
function failureMessage(reason: string) {
  const keys: Record<string, string> = {
    AI_WRITE_CONFLICT: "sourceConflict",
    BOOK_METADATA_INVALID_RESULT: "invalidResult",
    BOOK_METADATA_PROVIDER_FAILED: "providerFailed",
    BOOK_METADATA_TRUNCATED: "truncatedResult",
  }
  return t(`bookAI.${keys[reason] ?? "recognitionFailed"}`)
}
let generation = 0, request = 0, alive = true
async function load() {
  const seq = generation, req = ++request, id = props.job.id
  loading.value = true
  try {
    const next = await service.items(id, offset.value)
    if (!alive || seq !== generation || req !== request) return
    items.value = next
    selected.value = Object.fromEntries(next.filter(i => i.status === "ready" && selected.value[i.bookId] === i.revision).map(i => [i.bookId, i.revision]))
  } catch { if (alive && seq === generation && req === request) error.value = t("tasks.itemsFailed") }
  finally { if (alive && seq === generation && req === request) loading.value = false }
}
watch(() => props.job.id, () => { generation++; offset.value = 0; selected.value = {}; items.value = []; error.value = ""; feedback.value = ""; void load() }, { immediate: true })
watch(() => [props.job.updatedAt, props.job.processed, props.job.pendingReview, props.job.status], () => { if (!busy.value) void load() })
onBeforeUnmount(() => { alive = false; generation++ })
function select(item: BookOrganizationItem, checked: boolean | "indeterminate") {
  if (checked === true) selected.value[item.bookId] = item.revision
  else delete selected.value[item.bookId]
}
function page(delta: number) { offset.value += delta * 25; selected.value = {}; error.value = ""; void load() }
async function review(decision: "apply" | "discard") {
  if (busy.value || loading.value || !props.canWrite || !chosen.value.length) return
  const batch = [...chosen.value], id = props.job.id, kind = props.job.kind, seq = generation
  busy.value = true; error.value = ""; feedback.value = ""
  let succeeded = 0, failed = 0
  for (const item of batch) {
    if (!alive || seq !== generation || !props.canWrite) break
    try { await service.review(id, { bookId: item.bookId, revision: item.revision, decision }); succeeded++ }
    catch { failed++ }
  }
  window.dispatchEvent(new Event("curated:task-center-updated"))
  if (alive && seq === generation) {
    selected.value = {}; feedback.value = t("bookAI.reviewResult", { succeeded, failed })
    if (failed) error.value = t("bookAI.reviewFailed")
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
    <p v-if="busy || loading" class="text-sm text-muted-foreground" role="status">{{ t('tasks.loading') }}</p>
    <ul class="divide-y divide-border"><li v-for="item in items" :key="item.bookId" class="flex flex-col gap-3 py-4">
      <div class="flex items-start gap-3">
        <Checkbox :model-value="selected[item.bookId] === item.revision" :disabled="item.status !== 'ready' || busy || loading || !canWrite" :aria-label="t('bookAI.select', { title: item.title })" @update:model-value="select(item, $event)" />
        <span class="min-w-0 flex-1 break-words text-sm font-medium">{{ item.title }}</span><Badge :variant="['failed', 'conflict'].includes(item.status) ? 'warning' : 'secondary'">{{ t(`bookAI.status.${item.status}`) }}</Badge>
      </div>
      <template v-if="item.metadata.cleanTitle">
        <p class="break-words text-sm text-muted-foreground">{{ t('bookAI.before') }} · {{ item.before.title }}</p>
        <p class="break-words text-sm font-medium">{{ t('bookAI.after') }} · {{ item.metadata.translatedTitle }}</p>
        <details v-if="item.before.metadata?.cleanTitle"><summary class="cursor-pointer text-sm text-muted-foreground">{{ t('bookAI.saved') }}</summary><BookMetadataFacts :metadata="item.before.metadata" evidence /></details>
        <BookMetadataFacts :metadata="item.metadata" evidence />
      </template>
      <p v-if="item.reason" class="text-sm text-muted-foreground">{{ failureMessage(item.reason) }}</p>
    </li></ul>
    <div v-if="job.total > 25" class="flex items-center justify-between gap-2"><Button variant="outline" :disabled="offset === 0 || busy || loading" @click="page(-1)">{{ t('bookAI.previous') }}</Button><span class="text-sm text-muted-foreground">{{ offset / 25 + 1 }} / {{ Math.ceil(job.total / 25) }}</span><Button variant="outline" :disabled="offset + 25 >= job.total || busy || loading" @click="page(1)">{{ t('bookAI.next') }}</Button></div>
  </CardContent></Card>
</template>

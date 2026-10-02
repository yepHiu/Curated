<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { useI18n } from "vue-i18n"
import { ChevronDown, MoreHorizontal } from "lucide-vue-next"
import { useLibraryService } from "@/services/library-service"
import { useAIService } from "@/services/ai-service"
import type { TagOrganizationItem, LibraryTopic } from "@/services/contracts/topic-service"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { useTagOrganization, isOrganizationActive, organizationProgressText, organizationProgressValue, organizationErrorText } from "@/composables/use-tag-organization"

const { t, locale } = useI18n()
const state = useTagOrganization()
const { writeEnabled } = useExperimentalAgent()
const hiddenTopics = ref<LibraryTopic[]>([])
const scopeLabel = computed(() => {
  const selected = state.selection.value
  if (!selected) return t("topics.allMovies")
  return selected.movieIds.length === 1 && selected.title ? selected.title : t("topics.selectedMovies", { count: selected.movieIds.length })
})
const startLabel = computed(() => !state.selection.value ? t("topics.startUnorganized", { count: state.stats.value?.unorganized ?? 0 }) : t("topics.startSelected", { count: state.selection.value.movieIds.length }))
const coverageUnavailable = computed(() => !state.stats.value || state.statsLoading.value || state.statsError.value)
const invalidSelection = computed(() => state.selection.value !== null && (state.selection.value.movieIds.length === 0 || state.selection.value.movieIds.length > 600))
const expandedJob = ref("")
const selectedJob = ref("")
const items = ref<TagOrganizationItem[]>([])
const resultLoading = ref(false)
const resultError = ref(false)
const hasMore = ref(false)

watch(state.dialogOpen, async (open) => {
  if (!open) return
  void state.refresh()
  void state.refreshStats()
  try { hiddenTopics.value = (await useLibraryService().listTopics()).filter((topic) => topic.hidden) }
  catch { hiddenTopics.value = [] }
}, { immediate: true })

function formatStartedAt(value: string) {
  return new Intl.DateTimeFormat(locale.value, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hour12: false }).format(new Date(value))
}

async function restoreTopic(id: string) {
  try {
    await useLibraryService().setTopicHidden(id, false)
    hiddenTopics.value = hiddenTopics.value.filter((topic) => topic.id !== id)
    state.revision.value++
  } catch { state.error.value = t("topics.loadFailed") }
}

/** Read saved evidence only on demand; expanding the list never calls a model. */
async function showResults(id: string, more = false) {
  if (resultLoading.value) return
  expandedJob.value = id
  if (!more) { selectedJob.value = id; items.value = [] }
  resultLoading.value = true; resultError.value = false
  try {
    const page = await useAIService().getTagOrganizationItems(id, items.value.length)
    items.value.push(...page); hasMore.value = page.length === 25
  } catch { resultError.value = true }
  finally { resultLoading.value = false }
}
</script>

<template>
  <Dialog v-model:open="state.dialogOpen.value">
    <DialogContent class="max-h-[85dvh] overflow-y-auto sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>{{ t("topics.organize") }}</DialogTitle>
        <DialogDescription>{{ t("topics.explanation") }}</DialogDescription>
      </DialogHeader>
      <p v-if="!state.connected.value" role="status" class="text-sm text-muted-foreground">{{ t("topics.disconnected") }}</p>
      <p v-if="state.error.value" role="alert" class="text-sm text-destructive">{{ state.error.value }}</p>
      <section class="flex flex-col gap-2" :aria-label="t('topics.coverageTitle')" data-organization-coverage>
        <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <h3 class="font-medium">{{ t("topics.coverageTitle") }}</h3>
          <span v-if="state.stats.value">{{ t("topics.movieCount", { count: state.stats.value.total }) }}</span>
        </div>
        <p v-if="state.statsError.value" role="alert" class="text-sm text-destructive">
          {{ t("topics.coverageFailed") }}
          <Button variant="ghost" size="sm" class="min-h-11 lg:min-h-8" @click="state.refreshStats">{{ t("topics.retryLoad") }}</Button>
        </p>
        <dl v-else-if="state.stats.value" class="grid grid-cols-3 gap-3 rounded-lg border border-border/60 p-3">
          <div v-for="kind in (['organized', 'unorganized', 'outdated'] as const)" :key="kind" class="flex min-w-0 flex-col gap-1">
            <dt class="text-xs text-muted-foreground">{{ t(`topics.coverage.${kind}`) }}</dt>
            <dd class="text-lg font-medium tabular-nums" :data-organization-count="kind">{{ state.stats.value[kind] }}</dd>
          </div>
        </dl>
        <p v-else role="status" class="text-sm text-muted-foreground">{{ t("topics.loading") }}</p>
        <p v-if="state.stats.value?.outdated" class="text-xs text-muted-foreground">{{ t("topics.outdatedHint") }}</p>
        <p v-if="state.stats.value?.unresolved" class="text-xs text-muted-foreground">{{ t("topics.analyzedWithoutMatch", { count: state.stats.value.unresolved }) }}</p>
      </section>
      <div v-if="state.active.value" class="flex min-w-0 flex-col gap-2 rounded-lg bg-muted/40 p-3" data-organization-active>
        <div class="flex min-w-0 items-center justify-between gap-3 text-sm">
          <span class="min-w-0 truncate" role="status">{{ organizationProgressText(state.active.value) }}</span>
          <Button variant="ghost" size="sm" class="min-h-11 shrink-0 lg:min-h-8" :disabled="state.busy.value" @click="state.cancel(state.active.value.id)">{{ t("topics.cancel") }}</Button>
        </div>
        <Progress :model-value="organizationProgressValue(state.active.value)" class="h-1" :aria-label="organizationProgressText(state.active.value)" />
      </div>
      <div v-else class="flex min-w-0 flex-wrap items-center justify-between gap-3" data-organization-scope>
        <p class="w-full min-w-0 truncate text-sm font-medium sm:w-auto sm:flex-1" :title="scopeLabel">{{ scopeLabel }}</p>
        <div class="flex w-full flex-wrap items-center gap-2 sm:w-auto">
          <Button class="min-h-11 shrink-0 lg:min-h-9" :disabled="state.busy.value || invalidSelection || !writeEnabled || (!state.selection.value && (coverageUnavailable || !state.stats.value?.unorganized))" data-organize-unorganized @click="state.start()">{{ startLabel }}</Button>
          <template v-if="!state.selection.value">
            <Button v-if="state.stats.value?.outdated" variant="outline" class="min-h-11 lg:min-h-9" :disabled="state.busy.value || !writeEnabled || coverageUnavailable" data-organize-outdated @click="state.start('outdated')">{{ t("topics.startOutdated", { count: state.stats.value.outdated }) }}</Button>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="icon" class="size-11 lg:size-9" :aria-label="t('topics.scopeActions')" :disabled="state.busy.value || !writeEnabled || coverageUnavailable || !state.stats.value?.total"><MoreHorizontal /></Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end"><DropdownMenuGroup>
                <DropdownMenuItem :disabled="state.busy.value || !writeEnabled || coverageUnavailable || !state.stats.value?.total" @select="state.start('all')">{{ t("topics.reorganizeAll") }}</DropdownMenuItem>
              </DropdownMenuGroup></DropdownMenuContent>
            </DropdownMenu>
          </template>
        </div>
        <p v-if="!state.selection.value && state.stats.value && state.stats.value.total > 0 && !state.stats.value.unorganized && !state.stats.value.outdated" role="status" class="w-full text-xs text-muted-foreground">{{ t("topics.allOrganized") }}</p>
        <p v-if="invalidSelection" role="alert" class="w-full text-xs text-destructive">{{ t("topics.selectionLimit") }}</p>
        <p v-if="!writeEnabled" class="w-full text-xs text-muted-foreground">{{ t("topics.errors.AI_PERMISSION_REQUIRED") }}</p>
      </div>
      <section class="flex min-w-0 flex-col gap-2" :aria-label="t('topics.history')">
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <h3 class="font-medium">{{ t("topics.history") }}</h3>
          <span>{{ t("topics.historyCount", { count: state.jobs.value.length }) }}</span>
        </div>
        <p v-if="!state.jobs.value.length" class="py-3 text-center text-sm text-muted-foreground">{{ t("topics.noHistory") }}</p>
        <ol v-else class="flex max-h-60 min-w-0 flex-col overflow-y-auto overscroll-contain rounded-lg border border-border/60" data-organization-history>
          <li v-for="job in state.jobs.value" :key="job.id" class="border-b border-border/60 last:border-0" data-organization-history-row>
            <div class="flex min-h-11 min-w-0 items-center gap-1 px-2 lg:min-h-10">
              <button type="button" class="grid min-h-11 min-w-0 flex-1 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2 gap-y-1 rounded-md px-1 py-2 text-left text-xs hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 sm:flex sm:gap-3 lg:min-h-9" :aria-expanded="expandedJob === job.id" @click="expandedJob = expandedJob === job.id ? '' : job.id">
                <time :datetime="job.createdAt" :title="new Date(job.createdAt).toLocaleString(locale)" class="col-span-2 shrink-0 tabular-nums text-muted-foreground">{{ formatStartedAt(job.createdAt) }}</time>
                <span class="min-w-0 flex-1 truncate tabular-nums">{{ t("topics.movieCount", { count: job.total }) }}</span>
                <span class="flex shrink-0 items-center gap-2">
                  <Badge variant="secondary" class="shrink-0 text-[10px]">{{ t(`topics.status.${job.status}`) }}</Badge>
                  <ChevronDown class="size-3 shrink-0 text-muted-foreground" :class="{ 'rotate-180': expandedJob === job.id }" aria-hidden="true" />
                </span>
              </button>
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon" class="size-11 shrink-0 lg:size-8" :aria-label="t('topics.taskActions')"><MoreHorizontal /></Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuGroup>
                    <DropdownMenuItem :disabled="resultLoading" @select="showResults(job.id)">{{ t("topics.details") }}</DropdownMenuItem>
                    <DropdownMenuItem v-if="isOrganizationActive(job)" :disabled="state.busy.value" @select="state.cancel(job.id)">{{ t("topics.cancel") }}</DropdownMenuItem>
                    <template v-else>
                      <DropdownMenuItem v-if="['failed', 'partial_failed', 'blocked', 'cancelled'].includes(job.status)" :disabled="state.busy.value || Boolean(state.active.value) || !writeEnabled" @select="state.retry(job.id)">{{ t("topics.retry") }}</DropdownMenuItem>
                      <DropdownMenuItem :disabled="state.busy.value || !writeEnabled" @select="state.undo(job.id)">{{ t("topics.undo") }}</DropdownMenuItem>
                    </template>
                  </DropdownMenuGroup>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
            <div v-if="expandedJob === job.id" class="flex flex-col gap-2 px-3 pb-3 text-xs" data-organization-history-detail>
              <p class="text-muted-foreground">{{ t("topics.result", { success: job.succeeded, unresolved: job.unresolved, failed: job.failed }) }}</p>
              <p v-if="job.error" class="text-destructive">{{ organizationErrorText(job.error) }}</p>
              <Button variant="ghost" size="sm" class="min-h-11 self-start lg:min-h-8" :disabled="resultLoading" @click="showResults(job.id)">{{ t("topics.details") }}</Button>
              <div v-if="selectedJob === job.id" class="flex flex-col gap-3">
                <p v-if="resultError" role="alert" class="text-destructive">{{ t("topics.loadFailed") }}</p>
                <article v-for="item in items" :key="item.movieId" class="flex flex-col gap-1 border-b border-border/60 pb-2 last:border-0">
                  <RouterLink :to="{ name: 'detail', params: { id: item.movieId } }" class="break-words font-medium hover:underline" @click="state.dialogOpen.value = false">{{ item.title }}</RouterLink>
                  <p class="text-muted-foreground">{{ t(`topics.itemStatus.${item.status}`) }}</p>
                  <p v-if="item.reason && ['failed', 'conflict'].includes(item.status)" class="text-destructive">{{ organizationErrorText(item.reason) }}</p>
                  <p v-for="evidence in item.evidence" :key="evidence.topic" class="break-words text-muted-foreground">{{ evidence.topic }} · {{ t(`topics.evidenceField.${evidence.field}`) }}：{{ evidence.quote }}</p>
                </article>
                <p v-if="resultLoading" role="status" class="text-muted-foreground">{{ t("topics.loading") }}</p>
                <Button v-if="hasMore" variant="ghost" size="sm" class="min-h-11 self-start lg:min-h-8" :disabled="resultLoading" @click="showResults(job.id, true)">{{ t("topics.more") }}</Button>
              </div>
            </div>
          </li>
        </ol>
      </section>
      <div v-if="hiddenTopics.length" class="flex flex-wrap gap-2">
        <Button v-for="topic in hiddenTopics" :key="topic.id" variant="outline" size="sm" @click="restoreTopic(topic.id)">{{ t("topics.restore", { name: topic.name }) }}</Button>
      </div>
    </DialogContent>
  </Dialog>
</template>

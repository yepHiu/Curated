<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { RouterLink } from "vue-router"
import { useLibraryService } from "@/services/library-service"
import { useAIService } from "@/services/ai-service"
import type { TagOrganizationItem, LibraryTopic } from "@/services/contracts/topic-service"
import { useI18n } from "vue-i18n"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { useTagOrganization, isOrganizationActive, organizationProgressText, organizationErrorText } from "@/composables/use-tag-organization"

const { t } = useI18n()
const state = useTagOrganization()
const { writeEnabled } = useExperimentalAgent()
const scopeLabel = computed(() => {
  const selected = state.selection.value
  if (!selected) return t("topics.allMovies")
  return selected.movieIds.length === 1 && selected.title ? selected.title : t("topics.selectedMovies", { count: selected.movieIds.length })
})
const startLabel = computed(() => !state.selection.value ? t("topics.start") : t("topics.startSelected", { count: state.selection.value.movieIds.length }))
const invalidSelection = computed(() => state.selection.value !== null && (state.selection.value.movieIds.length === 0 || state.selection.value.movieIds.length > 600))
const hiddenTopics = ref<LibraryTopic[]>([])
watch(state.dialogOpen, async (open) => {
  // 进入整理面板时读取隐藏题材，不触发整理。
  if (!open) return
  try { hiddenTopics.value = (await useLibraryService().listTopics()).filter((topic) => { /* 只列待恢复条目。 */ return topic.hidden }) }
  catch { hiddenTopics.value = [] }
})
/** 恢复主题可见性，不改影片标签。 */
async function restoreTopic(id: string) {
  try { await useLibraryService().setTopicHidden(id, false); hiddenTopics.value = hiddenTopics.value.filter((topic) => { /* 移除已恢复项。 */ return topic.id !== id }); state.revision.value++ }
  catch { state.error.value = t("topics.loadFailed") }
}
const selectedJob = ref("")
const items = ref<TagOrganizationItem[]>([])
const resultLoading = ref(false)
const resultError = ref(false)
const hasMore = ref(false)

/** 按页打开已验证的来源摘录，不请求模型重新分析。 */
async function showResults(id: string, more = false) {
  if (resultLoading.value) return
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
      <div v-if="hiddenTopics.length" class="flex flex-wrap gap-2">
        <Button v-for="topic in hiddenTopics" :key="topic.id" variant="outline" size="sm" @click="restoreTopic(topic.id)">{{ t("topics.restore", { name: topic.name }) }}</Button>
      </div>
      <div v-if="!state.active.value" class="flex min-w-0 flex-wrap items-center justify-between gap-3" data-organization-scope>
        <p class="min-w-0 flex-1 truncate text-sm font-medium" :title="scopeLabel">{{ scopeLabel }}</p>
        <Button class="min-h-11 shrink-0 lg:min-h-9" :disabled="state.busy.value || invalidSelection || !writeEnabled" @click="state.start">{{ startLabel }}</Button>
        <p v-if="invalidSelection" role="alert" class="w-full text-xs text-destructive">{{ t("topics.selectionLimit") }}</p>
        <p v-if="!writeEnabled" class="w-full text-xs text-muted-foreground">{{ t("topics.errors.AI_PERMISSION_REQUIRED") }}</p>
      </div>
      <div v-for="job in state.jobs.value" :key="job.id" class="space-y-3 border-t border-border py-4">
        <div class="flex flex-wrap justify-between gap-2 text-sm">
          <span>{{ t(`topics.status.${job.status}`) }}<template v-if="!isOrganizationActive(job)"> · {{ job.processed }}/{{ job.total }}</template></span>
          <span class="text-muted-foreground">{{ new Date(job.createdAt).toLocaleString() }}</span>
        </div>
        <p v-if="isOrganizationActive(job)" role="status" class="text-sm">{{ organizationProgressText(job) }}</p>
        <p v-if="job.stage === 'vocabulary' && isOrganizationActive(job)" class="text-xs text-muted-foreground">{{ t("topics.vocabularyHint") }}</p>
        <p class="text-xs text-muted-foreground">{{ t("topics.trigger") }}: {{ t(`topics.reason.${job.triggerReason}`) }} · {{ t("topics.result", { success: job.succeeded, unresolved: job.unresolved, failed: job.failed }) }}</p>
        <p v-if="job.error" class="text-xs text-destructive">{{ organizationErrorText(job.error) }}</p>
        <div class="flex flex-wrap gap-2">
          <Button variant="ghost" size="sm" :disabled="resultLoading" @click="showResults(job.id)">{{ t("topics.details") }}</Button>
          <Button v-if="isOrganizationActive(job)" variant="outline" size="sm" :disabled="state.busy.value" @click="state.cancel(job.id)">{{ t("topics.cancel") }}</Button>
          <template v-else>
            <Button v-if="['failed', 'partial_failed', 'blocked', 'cancelled'].includes(job.status)" variant="outline" size="sm" :disabled="state.busy.value" @click="state.retry(job.id)">{{ t("topics.retry") }}</Button>
            <Button variant="ghost" size="sm" :disabled="state.busy.value" @click="state.undo(job.id)">{{ t("topics.undo") }}</Button>
          </template>
        </div>
        <div v-if="selectedJob === job.id" class="space-y-3 rounded-lg bg-muted/40 p-3">
          <p v-if="resultError" role="alert" class="text-sm text-destructive">{{ t("topics.loadFailed") }}</p>
          <article v-for="item in items" :key="item.movieId" class="space-y-1 border-b border-border pb-2 text-sm last:border-0">
            <RouterLink :to="{ name: 'detail', params: { id: item.movieId } }" class="font-medium hover:underline" @click="state.dialogOpen.value = false">{{ item.title }}</RouterLink>
            <p class="text-xs text-muted-foreground">{{ t(`topics.itemStatus.${item.status}`) }}</p>
            <p v-if="item.reason && ['failed', 'conflict'].includes(item.status)" class="text-xs text-destructive">{{ organizationErrorText(item.reason) }}</p>
            <p v-for="evidence in item.evidence" :key="evidence.topic" class="break-words text-xs text-muted-foreground">{{ evidence.topic }} · {{ t(`topics.evidenceField.${evidence.field}`) }}：{{ evidence.quote }}</p>
          </article>
          <p v-if="resultLoading" role="status" class="text-sm text-muted-foreground">{{ t("topics.loading") }}</p>
          <Button v-if="hasMore" variant="ghost" size="sm" :disabled="resultLoading" @click="showResults(job.id, true)">{{ t("topics.more") }}</Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

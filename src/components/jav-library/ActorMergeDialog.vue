<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { GitMerge, Loader2, RefreshCw } from "lucide-vue-next"
import type { ActorListItemDTO, ActorMergePreviewDTO, ActorMergeProfileSelection, ActorProfileDTO } from "@/api/types"
import { HttpClientError } from "@/api/http-client"
import ActorMergePicker from "@/components/jav-library/ActorMergePicker.vue"
import ActorMergeProfileFields from "@/components/jav-library/ActorMergeProfileFields.vue"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{ open: boolean; sourceName: string }>()
const emit = defineEmits<{ "update:open": [value: boolean]; merged: [targetName: string] }>()
const { t } = useI18n()
const libraryService = useLibraryService()
const selectedActor = ref<ActorListItemDTO | null>(null)
const originalProfile = ref<ActorProfileDTO | null>(null)
const keepOriginal = ref(false)
const preview = ref<ActorMergePreviewDTO | null>(null)
const profileDecisions = ref<Record<string, ActorMergeProfileSelection>>({})
const previewing = ref(false)
const applying = ref(false)
const errorMessage = ref("")
let revision = 0
let session = 0

const modelOpen = computed({
  get: () => props.open,
  set: (value: boolean) => { if (!applying.value) emit("update:open", value) },
})
const sourceName = computed(() => keepOriginal.value ? selectedActor.value?.name ?? "" : props.sourceName)
const targetName = computed(() => keepOriginal.value ? props.sourceName : selectedActor.value?.name ?? "")
const canPreview = computed(() => Boolean(selectedActor.value) && !previewing.value && !applying.value)
const unresolvedConflicts = computed(() => (preview.value?.requiredDecisions ?? []).filter((field) => !profileDecisions.value[field]))
const canApply = computed(() => Boolean(preview.value?.canApply) && unresolvedConflicts.value.length === 0 && !previewing.value && !applying.value)
const actorCards = computed(() => [
  { name: props.sourceName, avatarUrl: originalProfile.value?.avatarUrl, original: true, count: preview.value ? (keepOriginal.value ? preview.value.movies.targetCount : preview.value.movies.sourceCount) : undefined },
  { name: selectedActor.value?.name ?? "", avatarUrl: selectedActor.value?.avatarUrl, original: false, count: preview.value ? (keepOriginal.value ? preview.value.movies.sourceCount : preview.value.movies.targetCount) : selectedActor.value?.movieCount },
])
const avatars = computed(() => ({
  source: actorCards.value[keepOriginal.value ? 1 : 0]?.avatarUrl,
  target: actorCards.value[keepOriginal.value ? 0 : 1]?.avatarUrl,
}))
const resultAliases = computed(() => [...new Set([...(preview.value?.target.aliases ?? []), ...(preview.value?.aliasesToMove ?? [])])].filter((name) => name !== preview.value?.target.name))

function invalidatePreview() {
  ++revision
  preview.value = null
  profileDecisions.value = {}
  previewing.value = false
  errorMessage.value = ""
}
watch(keepOriginal, invalidatePreview, { flush: "sync" })
watch(() => [props.open, props.sourceName] as const, async ([open]) => {
  const currentSession = ++session
  invalidatePreview()
  selectedActor.value = null
  keepOriginal.value = false
  originalProfile.value = null
  if (!open) return
  try {
    const profile = await libraryService.getActorProfile(props.sourceName)
    if (currentSession === session) originalProfile.value = profile
  } catch {
    // Name and preview remain usable when the optional avatar cannot load.
  }
}, { immediate: true, flush: "sync" })
onBeforeUnmount(() => { ++revision; ++session })

function selectActor(actor: ActorListItemDTO) {
  invalidatePreview()
  selectedActor.value = actor
  keepOriginal.value = false
}
function changeActor() {
  invalidatePreview()
  selectedActor.value = null
  keepOriginal.value = false
}
function mergeError(error: unknown, fallback: string) {
  const code = error instanceof HttpClientError ? error.apiError?.code : undefined
  const messages: Record<string, string> = {
    ACTOR_MERGE_STALE_PREVIEW: "stalePreview",
    ACTOR_MERGE_NOT_FOUND: "actorMissing",
    ACTOR_MERGE_SELF: "sameActor",
    ACTOR_MERGE_SOURCE_IS_ALIAS: "sourceMerged",
    ACTOR_MERGE_CONFLICT: "identityConflict",
    ACTOR_MERGE_LINK_LIMIT: "linkLimit",
  }
  return t(`actors.merge.${code && messages[code] ? messages[code] : fallback}`)
}
async function loadPreview() {
  if (!canPreview.value) return
  invalidatePreview()
  const current = revision
  previewing.value = true
  try {
    const next = await libraryService.previewActorMerge({ sourceName: sourceName.value, targetName: targetName.value })
    if (current !== revision || !props.open) return
    preview.value = next
    profileDecisions.value = Object.fromEntries(next.profileFields.filter((field) => !field.conflict).map((field) => [field.field, field.defaultSelection]))
  } catch (error) {
    if (current === revision) errorMessage.value = mergeError(error, "previewFailed")
  } finally {
    if (current === revision) previewing.value = false
  }
}
async function applyMerge() {
  const current = preview.value
  if (!current || !canApply.value) return
  const currentSession = session
  applying.value = true
  errorMessage.value = ""
  try {
    const audit = await libraryService.applyActorMerge({ sourceName: current.source.name, targetName: current.target.name, previewToken: current.previewToken, confirm: true, profileDecisions: { ...profileDecisions.value } })
    pushAppToast(t(audit.refreshFailed ? "actors.merge.refreshFailed" : "actors.merge.success", { source: audit.sourceName, target: audit.targetName }), { variant: audit.refreshFailed ? "warning" : "success" })
    if (currentSession === session) {
      // Clear the executable preview before notifying parents or changing routes.
      invalidatePreview()
      emit("merged", audit.targetName)
      emit("update:open", false)
    }
  } catch (error) {
    if (currentSession === session) {
      invalidatePreview()
      errorMessage.value = mergeError(error, "applyFailed")
    }
  } finally {
    applying.value = false
  }
}
function setProfileDecision(fields: string[], selection: ActorMergeProfileSelection) {
  profileDecisions.value = { ...profileDecisions.value, ...Object.fromEntries(fields.map((field) => [field, selection])) }
}
</script>

<template>
  <Dialog v-model:open="modelOpen">
    <DialogContent class="max-h-[min(88dvh,52rem)] overflow-y-auto sm:max-w-2xl" :show-close-button="!applying" @escape-key-down="applying && $event.preventDefault()" @interact-outside="applying && $event.preventDefault()">
      <DialogHeader>
        <DialogTitle>{{ t("actors.merge.title") }}</DialogTitle>
        <DialogDescription>{{ t("actors.merge.description", { source: props.sourceName }) }}</DialogDescription>
      </DialogHeader>
      <div class="flex min-w-0 flex-col gap-4">
        <ActorMergePicker v-if="open && !selectedActor" :source-name="props.sourceName" @select="selectActor" />
        <template v-if="selectedActor">
          <fieldset class="min-w-0" :disabled="applying">
            <legend class="mb-3 text-sm font-medium">{{ t("actors.merge.keepActor") }}</legend>
            <div class="grid min-w-0 gap-3 sm:grid-cols-2">
              <label v-for="actor in actorCards" :key="actor.name" class="flex min-h-20 min-w-0 cursor-pointer items-center gap-3 rounded-xl border border-border/70 p-3 has-[:checked]:border-primary has-[:checked]:bg-primary/10">
                <input v-model="keepOriginal" type="radio" name="actor-merge-keep" :value="actor.original" class="shrink-0 accent-primary" />
                <Avatar class="size-12 shrink-0 rounded-lg">
                  <AvatarImage v-if="actor.avatarUrl" :src="actor.avatarUrl" :alt="actor.name" class="object-cover" />
                  <AvatarFallback class="rounded-lg">{{ actor.name.slice(0, 2) }}</AvatarFallback>
                </Avatar>
                <span class="flex min-w-0 flex-col gap-1">
                  <span class="break-words text-sm font-medium">{{ actor.name }}</span>
                  <span v-if="actor.count !== undefined" class="text-xs text-muted-foreground">{{ t("actors.movieCount", { n: actor.count }) }}</span>
                  <span v-if="keepOriginal === actor.original" class="text-xs text-primary">{{ t("actors.merge.retained") }}</span>
                </span>
              </label>
            </div>
          </fieldset>
          <div class="flex flex-wrap justify-end gap-2">
            <Button variant="ghost" class="min-h-11 sm:min-h-9" :disabled="applying" @click="changeActor">{{ t("actors.merge.changeActor") }}</Button>
            <Button variant="outline" class="min-h-11 sm:min-h-9" :disabled="!canPreview" @click="loadPreview">
              <Loader2 v-if="previewing" data-icon="inline-start" class="motion-safe:animate-spin" aria-hidden="true" />
              <RefreshCw v-else data-icon="inline-start" aria-hidden="true" />
              {{ t("actors.merge.previewAction") }}
            </Button>
          </div>
        </template>
        <p v-if="errorMessage" role="alert" class="text-sm text-destructive">{{ errorMessage }}</p>
        <template v-if="preview">
          <section class="flex min-w-0 flex-col gap-3 rounded-xl border border-border/70 bg-muted/25 p-4" data-actor-merge-result>
            <h3 class="break-words text-sm font-semibold">{{ t("actors.merge.resultName", { name: preview.target.name }) }}</h3>
            <p class="text-sm">{{ t("actors.merge.movieSummary", { source: preview.movies.sourceCount, target: preview.movies.targetCount, duplicates: preview.movies.duplicateCount, result: preview.movies.resultCount }) }}</p>
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <span class="text-xs text-muted-foreground">{{ t("actors.merge.aliases") }}</span>
              <Badge v-for="alias in resultAliases" :key="alias" variant="outline" class="max-w-full whitespace-normal break-words">{{ alias }}</Badge>
            </div>
            <div v-if="preview.userTags.result.length" class="flex min-w-0 flex-wrap items-center gap-2">
              <span class="text-xs text-muted-foreground">{{ t("actors.merge.userTags") }}</span>
              <Badge v-for="tag in preview.userTags.result" :key="tag" variant="secondary" class="max-w-full whitespace-normal break-words">{{ tag }}</Badge>
            </div>
            <details class="min-w-0 text-sm">
              <summary class="cursor-pointer rounded-sm text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring">{{ t("actors.merge.otherChanges") }}</summary>
              <ul class="mt-3 flex min-w-0 flex-col gap-2 text-muted-foreground">
                <li>{{ t("actors.merge.externalLinks") }}: {{ preview.externalLinks.result.length }}</li>
                <li v-for="link in preview.externalLinks.result" :key="link" class="break-all">{{ link }}</li>
                <li>{{ t("actors.merge.feedback") }}: {{ preview.recommendationFeedback.resultCount }}</li>
                <li>{{ t("actors.merge.frames") }}: {{ preview.curatedFramesAffected }}</li>
              </ul>
            </details>
          </section>
          <section v-if="preview.blockingReasons.length" role="alert" class="rounded-xl border border-destructive/40 bg-destructive/10 p-4 text-sm text-destructive">
            <h3 class="font-semibold">{{ t("actors.merge.blockedTitle") }}</h3>
            <ul class="mt-2 list-disc space-y-2 pl-5">
              <li v-for="reason in preview.blockingReasons" :key="reason.code + reason.message">{{ t(reason.code === 'ACTOR_MERGE_LINK_LIMIT' ? 'actors.merge.linkLimit' : 'actors.merge.identityConflict') }}</li>
            </ul>
          </section>
          <ActorMergeProfileFields :preview="preview" :decisions="profileDecisions" :avatars="avatars" :disabled="applying" @select="setProfileDecision" />
          <p v-if="unresolvedConflicts.length" role="status" class="text-sm text-muted-foreground">{{ t("actors.merge.conflictsTitle") }}</p>
          <p class="break-words text-sm text-muted-foreground">{{ t("actors.merge.confirmWarning", { source: preview.source.name, target: preview.target.name }) }}</p>
        </template>
      </div>
      <DialogFooter>
        <Button variant="outline" class="min-h-11 sm:min-h-9" :disabled="applying" @click="emit('update:open', false)">{{ t("common.cancel") }}</Button>
        <Button v-if="selectedActor" variant="destructive" class="min-h-11 sm:min-h-9" :disabled="!canApply" @click="applyMerge">
          <Loader2 v-if="applying" data-icon="inline-start" class="motion-safe:animate-spin" aria-hidden="true" />
          <GitMerge v-else data-icon="inline-start" aria-hidden="true" />
          {{ applying ? t("actors.merge.applying") : t("actors.merge.confirmAction") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

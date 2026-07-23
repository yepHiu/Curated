<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { GitMerge, Loader2, RefreshCw } from "lucide-vue-next"
import type {
  ActorMergePreviewDTO,
  ActorMergeProfileSelection,
} from "@/api/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { pushAppToast } from "@/composables/use-app-toast"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  open: boolean
  sourceName: string
}>()

const emit = defineEmits<{
  "update:open": [value: boolean]
  merged: [targetName: string]
}>()

const { t } = useI18n()
const libraryService = useLibraryService()
const targetName = ref("")
const preview = ref<ActorMergePreviewDTO | null>(null)
const profileDecisions = ref<Record<string, ActorMergeProfileSelection>>({})
const previewing = ref(false)
const applying = ref(false)
const errorMessage = ref("")

const modelOpen = computed({
  get: () => props.open,
  set: (value: boolean) => emit("update:open", value),
})

const unresolvedConflicts = computed(() =>
  (preview.value?.requiredDecisions ?? []).filter(
    (field) => !profileDecisions.value[field],
  ),
)

const canPreview = computed(
  () =>
    targetName.value.trim() !== "" &&
    targetName.value.trim() !== props.sourceName.trim() &&
    !previewing.value &&
    !applying.value,
)

const canApply = computed(
  () =>
    Boolean(preview.value?.canApply) &&
    unresolvedConflicts.value.length === 0 &&
    !previewing.value &&
    !applying.value,
)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    targetName.value = ""
    preview.value = null
    profileDecisions.value = {}
    errorMessage.value = ""
  },
)

watch(targetName, () => {
  if (!preview.value) return
  if (targetName.value.trim() === preview.value.target.name) return
  preview.value = null
  profileDecisions.value = {}
  errorMessage.value = ""
})

function initializeProfileDecisions(next: ActorMergePreviewDTO) {
  const decisions: Record<string, ActorMergeProfileSelection> = {}
  for (const field of next.profileFields) {
    if (!field.conflict) decisions[field.field] = field.defaultSelection
  }
  profileDecisions.value = decisions
}

async function loadPreview() {
  if (!canPreview.value) return
  previewing.value = true
  errorMessage.value = ""
  try {
    const next = await libraryService.previewActorMerge({
      sourceName: props.sourceName.trim(),
      targetName: targetName.value.trim(),
    })
    preview.value = next
    targetName.value = next.target.name
    initializeProfileDecisions(next)
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : t("actors.merge.previewFailed")
  } finally {
    previewing.value = false
  }
}

async function applyMerge() {
  const current = preview.value
  if (!current || !canApply.value) return
  applying.value = true
  errorMessage.value = ""
  try {
    const audit = await libraryService.applyActorMerge({
      sourceName: current.source.name,
      targetName: current.target.name,
      previewToken: current.previewToken,
      confirm: true,
      profileDecisions: { ...profileDecisions.value },
    })
    pushAppToast(
      t("actors.merge.success", {
        source: audit.sourceName,
        target: audit.targetName,
      }),
      { variant: "success" },
    )
    emit("merged", audit.targetName)
    emit("update:open", false)
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : t("actors.merge.applyFailed")
  } finally {
    applying.value = false
  }
}

function setProfileDecision(field: string, selection: ActorMergeProfileSelection) {
  profileDecisions.value = {
    ...profileDecisions.value,
    [field]: selection,
  }
}
</script>

<template>
  <Dialog v-model:open="modelOpen">
    <DialogContent class="max-h-[min(88dvh,52rem)] overflow-y-auto sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ t("actors.merge.title") }}</DialogTitle>
        <DialogDescription>
          {{ t("actors.merge.description", { source: sourceName }) }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-2">
          <label for="actor-merge-target" class="text-sm font-medium text-foreground">
            {{ t("actors.merge.targetLabel") }}
          </label>
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Input
              id="actor-merge-target"
              v-model="targetName"
              :placeholder="t('actors.merge.targetPlaceholder')"
              :disabled="previewing || applying"
              autocomplete="off"
              @keydown.enter.prevent="loadPreview"
            />
            <Button
              type="button"
              variant="outline"
              class="min-h-11 shrink-0 sm:min-h-9"
              :disabled="!canPreview"
              @click="loadPreview"
            >
              <Loader2
                v-if="previewing"
                data-icon="inline-start"
                class="motion-safe:animate-spin"
                aria-hidden="true"
              />
              <RefreshCw v-else data-icon="inline-start" aria-hidden="true" />
              {{ t("actors.merge.previewAction") }}
            </Button>
          </div>
          <p class="text-xs text-muted-foreground">
            {{ t("actors.merge.targetHint") }}
          </p>
        </div>

        <p v-if="errorMessage" role="alert" class="text-sm text-destructive">
          {{ errorMessage }}
        </p>

        <template v-if="preview">
          <section class="flex flex-col gap-3 rounded-xl border border-border/70 bg-muted/25 p-4">
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="outline">{{ preview.source.name }}</Badge>
              <span aria-hidden="true" class="text-muted-foreground">→</span>
              <Badge>{{ preview.target.name }}</Badge>
            </div>
            <dl class="grid gap-3 text-sm sm:grid-cols-2">
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.movies") }}</dt>
                <dd class="font-medium text-foreground">
                  {{ preview.movies.resultCount }}
                  <span v-if="preview.movies.duplicateCount > 0" class="text-muted-foreground">
                    ({{ t("actors.merge.duplicates", { count: preview.movies.duplicateCount }) }})
                  </span>
                </dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.aliases") }}</dt>
                <dd class="font-medium text-foreground">{{ preview.aliasesToMove.length }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.userTags") }}</dt>
                <dd class="font-medium text-foreground">{{ preview.userTags.result.length }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.externalLinks") }}</dt>
                <dd class="font-medium text-foreground">{{ preview.externalLinks.result.length }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.feedback") }}</dt>
                <dd class="font-medium text-foreground">
                  {{ preview.recommendationFeedback.resultCount }}
                </dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-muted-foreground">{{ t("actors.merge.frames") }}</dt>
                <dd class="font-medium text-foreground">{{ preview.curatedFramesAffected }}</dd>
              </div>
            </dl>
          </section>

          <section
            v-if="preview.blockingReasons.length > 0"
            class="flex flex-col gap-2 rounded-xl border border-destructive/40 bg-destructive/10 p-4"
            role="alert"
          >
            <h3 class="text-sm font-semibold text-destructive">
              {{ t("actors.merge.blockedTitle") }}
            </h3>
            <ul class="flex list-disc flex-col gap-1 pl-5 text-sm text-destructive">
              <li v-for="reason in preview.blockingReasons" :key="`${reason.code}-${reason.message}`">
                {{ reason.message }}
              </li>
            </ul>
          </section>

          <section v-if="preview.requiredDecisions.length > 0" class="flex flex-col gap-3">
            <div class="flex flex-col gap-1">
              <h3 class="text-sm font-semibold text-foreground">
                {{ t("actors.merge.conflictsTitle") }}
              </h3>
              <p class="text-xs text-muted-foreground">
                {{ t("actors.merge.conflictsDescription") }}
              </p>
            </div>
            <fieldset
              v-for="field in preview.profileFields.filter((item) => item.conflict)"
              :key="field.field"
              class="flex flex-col gap-2 rounded-xl border border-border/70 p-3"
            >
              <legend class="px-1 text-sm font-medium text-foreground">
                {{ t(`actors.merge.profileField.${field.field}`) }}
              </legend>
              <label class="flex min-h-11 cursor-pointer items-start gap-2 rounded-lg border border-border/60 p-2 text-sm">
                <input
                  type="radio"
                  :name="`actor-merge-${field.field}`"
                  value="target"
                  :checked="profileDecisions[field.field] === 'target'"
                  @change="setProfileDecision(field.field, 'target')"
                />
                <span class="min-w-0">
                  <span class="font-medium text-foreground">{{ t("actors.merge.keepTarget") }}</span>
                  <span class="block break-words text-muted-foreground">{{ field.targetValue }}</span>
                </span>
              </label>
              <label class="flex min-h-11 cursor-pointer items-start gap-2 rounded-lg border border-border/60 p-2 text-sm">
                <input
                  type="radio"
                  :name="`actor-merge-${field.field}`"
                  value="source"
                  :checked="profileDecisions[field.field] === 'source'"
                  @change="setProfileDecision(field.field, 'source')"
                />
                <span class="min-w-0">
                  <span class="font-medium text-foreground">{{ t("actors.merge.keepSource") }}</span>
                  <span class="block break-words text-muted-foreground">{{ field.sourceValue }}</span>
                </span>
              </label>
            </fieldset>
          </section>

          <p class="text-sm text-muted-foreground">
            {{ t("actors.merge.confirmWarning", { source: preview.source.name, target: preview.target.name }) }}
          </p>
        </template>
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          class="min-h-11 sm:min-h-9"
          :disabled="applying"
          @click="emit('update:open', false)"
        >
          {{ t("common.cancel") }}
        </Button>
        <Button
          type="button"
          variant="destructive"
          class="min-h-11 sm:min-h-9"
          :disabled="!canApply"
          @click="applyMerge"
        >
          <Loader2
            v-if="applying"
            data-icon="inline-start"
            class="motion-safe:animate-spin"
            aria-hidden="true"
          />
          <GitMerge v-else data-icon="inline-start" aria-hidden="true" />
          {{ applying ? t("actors.merge.applying") : t("actors.merge.confirmAction") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

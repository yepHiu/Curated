<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Loader2, Sparkles } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import type { AIActionPreviewDTO, PatchMovieBody } from "@/api/types"
import type { Movie } from "@/domain/movie/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  movie: Movie
  patchMovieDisplay: (body: PatchMovieBody, done: (err?: unknown) => void) => void
}>()

const open = defineModel<boolean>("open", { required: true })

const { t, locale } = useI18n()
const aiService = useAIService()
const { run: runAIAction, pending: aiActionPending, cancel: cancelAIAction } = useAIActionRequest(aiService)
const libraryService = useLibraryService()
const { enabled: agentEnabled } = useExperimentalAgent()

const movieEditSaving = ref(false)
const movieEditError = ref("")
const editDraftTitle = ref("")
const editDraftStudio = ref("")
const editDraftSummary = ref("")
const editDraftRelease = ref("")
const editDraftRuntime = ref("")
const aiBusyField = ref<"title" | "summary" | null>(null)
const preview = ref<AIActionPreviewDTO | null>(null)
const previewOpen = ref(false)
const aiBusy = computed(() => aiBusyField.value !== null)

const releaseDateInputRx = /^\d{4}-\d{2}-\d{2}$/
const showAgentActions = computed(() => agentEnabled.value)

function syncDraftsFromMovie() {
  movieEditError.value = ""
  editDraftTitle.value = props.movie.title
  editDraftStudio.value = props.movie.studio
  editDraftSummary.value = props.movie.summary
  editDraftRelease.value = props.movie.releaseDate?.trim() ?? ""
  editDraftRuntime.value =
    props.movie.runtimeMinutes > 0 ? String(props.movie.runtimeMinutes) : ""
}

watch(
  () => props.movie.id,
  () => {
    open.value = false
    movieEditError.value = ""
  },
)

watch(open, (isOpen) => {
  if (isOpen) {
    syncDraftsFromMovie()
  }
}, { immediate: true })

function buildMovieDisplayPatchBody(): PatchMovieBody {
  const rt = editDraftRuntime.value.trim()
  return {
    userTitle: editDraftTitle.value.trim() === "" ? null : editDraftTitle.value.trim(),
    userStudio: editDraftStudio.value.trim() === "" ? null : editDraftStudio.value.trim(),
    userSummary: editDraftSummary.value.trim() === "" ? null : editDraftSummary.value.trim(),
    userReleaseDate:
      editDraftRelease.value.trim() === "" ? null : editDraftRelease.value.trim(),
    userRuntimeMinutes: rt === "" ? null : Number.parseInt(editDraftRuntime.value, 10),
  }
}

function submitMovieEditDialog() {
  movieEditError.value = ""
  const rd = editDraftRelease.value.trim()
  if (rd !== "" && !releaseDateInputRx.test(rd)) {
    movieEditError.value = t("detailPanel.movieEditInvalidRelease")
    return
  }
  const rt = editDraftRuntime.value.trim()
  if (rt !== "") {
    const n = Number.parseInt(rt, 10)
    if (Number.isNaN(n) || n < 0 || n > 99999) {
      movieEditError.value = t("detailPanel.movieEditInvalidRuntime")
      return
    }
  }
  const body = buildMovieDisplayPatchBody()
  movieEditSaving.value = true
  props.patchMovieDisplay(body, (err?: unknown) => {
    movieEditSaving.value = false
    if (err) {
      movieEditError.value =
        err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieEditSaveFailed")
      return
    }
    open.value = false
  })
}

type DisplayActionName = "translate_title" | "translate_summary"

function fieldForDisplayAction(name: string): "title" | "summary" {
  return name === "translate_title" ? "title" : "summary"
}

watch([() => props.movie.id, open], cancelAIAction)
watch(agentEnabled, cancelAIAction)

async function runDisplayAction(name: DisplayActionName) {
  if (!showAgentActions.value || aiBusy.value) {
    return
  }
  const field = fieldForDisplayAction(name)
  const body = field === "title" ? editDraftTitle.value.trim() : editDraftSummary.value.trim()
  if (!body) {
    return
  }
  aiBusyField.value = field
  movieEditError.value = ""
  try {
    const dto = await runAIAction(name, {
      movieId: props.movie.id,
      body,
      locale: locale.value,
    })
    if (dto.noop) {
      movieEditError.value =
        name === "translate_summary" ? t("detailPanel.movieAiTranslateSummaryNoop") : t("detailPanel.movieAiTranslateNoop")
      return
    }
    preview.value = dto
    previewOpen.value = true
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE") {
      movieEditError.value = t("detailPanel.movieAiUnconfigured")
    } else {
      movieEditError.value = err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieAiError")
    }
  } finally {
    aiBusyField.value = null
  }
}

async function applyDisplayPreview() {
  const current = preview.value
  if (!current?.confirmToken || !current.sessionId) {
    previewOpen.value = false
    return
  }
  aiBusyField.value = fieldForDisplayAction(current.action)
  movieEditError.value = ""
  try {
    await aiService.confirmTool({
      sessionId: current.sessionId,
      name: current.name,
      arguments: current.arguments ?? { movieId: props.movie.id },
      confirmToken: current.confirmToken,
    })
    await libraryService.loadMovieDetail(props.movie.id)
    if (current.action === "translate_title") {
      editDraftTitle.value = current.proposedText ?? editDraftTitle.value
    } else if (current.action === "translate_summary") {
      editDraftSummary.value = current.proposedText ?? editDraftSummary.value
    }
    previewOpen.value = false
    preview.value = null
  } catch (err) {
    movieEditError.value = err instanceof Error && err.message.trim() ? err.message : t("detailPanel.movieAiApplyError")
  } finally {
    aiBusyField.value = null
  }
}

function discardDisplayPreview() {
  previewOpen.value = false
  preview.value = null
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-h-[min(90vh,40rem)] overflow-y-auto rounded-3xl border-border/70 sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("detailPanel.editMovieTitle") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("detailPanel.editMovieDesc") }}
        </DialogDescription>
      </DialogHeader>
      <div class="flex flex-col gap-4 py-2">
        <p
          v-if="movieEditError"
          class="rounded-xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ movieEditError }}
        </p>
        <div class="grid gap-2">
          <span class="text-sm font-medium text-muted-foreground">
            {{ t("detailPanel.readOnlyCode") }}
          </span>
          <p class="rounded-xl border border-border/60 bg-muted/30 px-3 py-2 text-sm">
            {{ movie.code }}
          </p>
        </div>
        <div class="grid gap-2">
          <span class="text-sm font-medium text-muted-foreground">
            {{ t("detailPanel.readOnlyLocation") }}
          </span>
          <p
            class="max-h-24 overflow-y-auto rounded-xl border border-border/60 bg-muted/30 px-3 py-2 font-mono text-xs break-all"
          >
            {{ movie.location }}
          </p>
        </div>
        <div class="grid gap-2">
          <label class="text-sm font-medium" for="movie-edit-title">{{
            t("detailPanel.fieldTitle")
          }}</label>
          <div
            class="rounded-xl border border-border/60 bg-muted/40 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
            data-movie-edit-title-field
          >
            <Input
              id="movie-edit-title"
              v-model="editDraftTitle"
              class="rounded-xl border-0 bg-transparent shadow-none focus-visible:border-0 focus-visible:ring-0"
              autocomplete="off"
            />
            <div v-if="showAgentActions" class="flex items-center justify-end px-1 pb-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="min-h-11 rounded-lg text-muted-foreground hover:text-foreground md:h-8 md:min-h-8"
                :disabled="aiBusy || !editDraftTitle.trim()"
                data-movie-edit-ai-translate
                @click="runDisplayAction('translate_title')"
              >
                <Loader2 v-if="aiBusyField === 'title'" class="size-4 motion-safe:animate-spin" />
                <Sparkles v-else class="size-4" />
                {{ t("detailPanel.movieAiTranslateTitle") }}
              </Button>
          <Button v-if="aiActionPending" type="button" variant="ghost" size="sm" data-ai-action-cancel @click="cancelAIAction">{{ t("common.cancel") }}</Button>
            </div>
          </div>
        </div>
        <div class="grid gap-2">
          <label class="text-sm font-medium" for="movie-edit-studio">{{
            t("detailPanel.fieldStudio")
          }}</label>
          <Input
            id="movie-edit-studio"
            v-model="editDraftStudio"
            class="rounded-xl text-sm"
            autocomplete="off"
          />
        </div>
        <div class="grid gap-2">
          <label class="text-sm font-medium" for="movie-edit-summary">{{
            t("detailPanel.fieldSummary")
          }}</label>
          <div
            class="rounded-xl border border-border/60 bg-muted/40 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
            data-movie-edit-summary-field
          >
            <textarea
              id="movie-edit-summary"
              v-model="editDraftSummary"
              rows="5"
              class="min-h-[120px] w-full resize-y border-0 bg-transparent px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground"
            />
            <div v-if="showAgentActions" class="flex items-center justify-end px-1 pb-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="min-h-11 rounded-lg text-muted-foreground hover:text-foreground md:h-8 md:min-h-8"
                :disabled="aiBusy || !editDraftSummary.trim()"
                data-movie-edit-ai-translate-summary
                @click="runDisplayAction('translate_summary')"
              >
                <Loader2 v-if="aiBusyField === 'summary'" class="size-4 motion-safe:animate-spin" />
                <Sparkles v-else class="size-4" />
                {{ t("detailPanel.movieAiTranslateSummary") }}
              </Button>
            </div>
          </div>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 sm:gap-3">
          <div class="grid gap-2">
            <label class="text-sm font-medium" for="movie-edit-release">{{
              t("detailPanel.fieldReleaseDate")
            }}</label>
            <Input
              id="movie-edit-release"
              v-model="editDraftRelease"
              class="rounded-xl text-sm"
              placeholder="YYYY-MM-DD"
              autocomplete="off"
            />
          </div>
          <div class="grid gap-2">
            <label class="text-sm font-medium" for="movie-edit-runtime">{{
              t("detailPanel.fieldRuntimeMinutes")
            }}</label>
            <Input
              id="movie-edit-runtime"
              v-model="editDraftRuntime"
              class="rounded-xl text-sm"
              inputmode="numeric"
              autocomplete="off"
            />
          </div>
        </div>
      </div>
      <DialogFooter class="gap-3">
        <DialogClose as-child>
          <Button type="button" variant="outline" class="rounded-2xl" :disabled="movieEditSaving">
            {{ t("common.cancel") }}
          </Button>
        </DialogClose>
        <Button
          type="button"
          class="rounded-2xl"
          :disabled="movieEditSaving"
          @click="submitMovieEditDialog"
        >
          {{ movieEditSaving ? t("detailPanel.movieEditSaving") : t("detailPanel.saveMovieEdit") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
  <Dialog :open="previewOpen" @update:open="previewOpen = $event">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("detailPanel.movieAiPreviewTitle") }}</DialogTitle>
        <DialogDescription>{{ t("detailPanel.movieAiPreviewDesc") }}</DialogDescription>
      </DialogHeader>
      <div class="grid gap-3 text-sm">
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPanel.movieAiBefore") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.originalText || "—" }}</p>
        </div>
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPanel.movieAiAfter") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.proposedText || "—" }}</p>
        </div>
      </div>
      <DialogFooter class="gap-2">
        <Button type="button" variant="outline" data-movie-edit-ai-discard @click="discardDisplayPreview">
          {{ t("detailPanel.movieAiDiscard") }}
        </Button>
        <Button type="button" :disabled="aiBusy" data-movie-edit-ai-apply @click="applyDisplayPreview">
          {{ t("detailPanel.movieAiApply") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

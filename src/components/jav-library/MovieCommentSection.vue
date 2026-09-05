<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { Loader2, Sparkles } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { HttpClientError } from "@/api/http-client"
import { MAX_MOVIE_COMMENT_RUNES, type AIActionPreviewDTO, type MovieCommentDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useLibraryService } from "@/services/library-service"

const props = withDefaults(
  defineProps<{
    movieId: string
    /** 回收站中仅展示，不可保存 */
    readonly?: boolean
  }>(),
  { readonly: false },
)

const { t, locale } = useI18n()
const libraryService = useLibraryService()
const aiService = useAIService()
const { run: runAIAction, pending: aiActionPending, cancel: cancelAIAction } = useAIActionRequest(aiService)
const { enabled: agentEnabled } = useExperimentalAgent()

const draft = ref("")
const updatedAt = ref("")
const loading = ref(false)
const saving = ref(false)
const loadError = ref("")
const saveError = ref("")
const lastSavedBody = ref("")
const hydrating = ref(false)
const commentSavedFlash = ref(false)
let commentSavedFlashTimer: ReturnType<typeof setTimeout> | null = null
let commentSavePromise: Promise<void> | null = null
let saveQueued = false

const aiBusy = ref(false)
const aiError = ref("")
const previewOpen = ref(false)
const preview = ref<AIActionPreviewDTO | null>(null)

function countRunes(s: string): number {
  return [...s].length
}

const runeCount = computed(() => countRunes(draft.value))
const normalizedDraft = computed(() => draft.value.trim())
const isTooLong = computed(() => runeCount.value > MAX_MOVIE_COMMENT_RUNES)
const hasUnsavedChanges = computed(() => normalizedDraft.value !== lastSavedBody.value)

const updatedLabel = computed(() => {
  const iso = updatedAt.value.trim()
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(d)
})

function formatClientErr(err: unknown, fallback: string) {
  if (err instanceof HttpClientError) {
    return err.apiError?.message?.trim() || err.message || fallback
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message
  }
  return fallback
}

async function load() {
  const id = props.movieId.trim()
  if (!id) {
    return
  }
  loading.value = true
  hydrating.value = true
  loadError.value = ""
  saveError.value = ""
  try {
    const dto: MovieCommentDTO = await libraryService.getMovieComment(id)
    draft.value = dto.body
    lastSavedBody.value = dto.body
    updatedAt.value = dto.updatedAt
  } catch (err) {
    loadError.value = formatClientErr(err, t("detailPage.commentLoadError"))
  } finally {
    hydrating.value = false
    loading.value = false
  }
}

function flashCommentSaved() {
  commentSavedFlash.value = true
  if (commentSavedFlashTimer) clearTimeout(commentSavedFlashTimer)
  commentSavedFlashTimer = setTimeout(() => {
    commentSavedFlash.value = false
    commentSavedFlashTimer = null
  }, 2200)
}

async function performSaveComment(movieId = props.movieId.trim()) {
  if (props.readonly) {
    return
  }
  const id = movieId.trim()
  if (!id) {
    return
  }
  if (isTooLong.value) {
    saveError.value = t("detailPage.commentTooLong", { n: MAX_MOVIE_COMMENT_RUNES })
    return
  }
  const bodyToSave = normalizedDraft.value
  if (bodyToSave === lastSavedBody.value) {
    return
  }
  saving.value = true
  saveError.value = ""
  try {
    const dto: MovieCommentDTO = await libraryService.putMovieComment(id, {
      body: bodyToSave,
    })
    lastSavedBody.value = dto.body
    updatedAt.value = dto.updatedAt
    if (normalizedDraft.value === bodyToSave) {
      draft.value = dto.body
    }
    flashCommentSaved()
  } catch (err) {
    saveError.value = formatClientErr(err, t("detailPage.commentSaveError"))
  } finally {
    saving.value = false
  }
}

async function saveCommentNow(movieId = props.movieId.trim()) {
  if (commentSavePromise) {
    saveQueued = true
    return commentSavePromise
  }

  commentSavePromise = (async () => {
    do {
      saveQueued = false
      await performSaveComment(movieId)
    } while (saveQueued && hasUnsavedChanges.value)
  })()

  try {
    await commentSavePromise
  } finally {
    commentSavePromise = null
  }
}

const showAgentActions = computed(() => agentEnabled.value && !props.readonly)

watch(() => props.movieId, cancelAIAction)
watch(agentEnabled, cancelAIAction)

async function polishComment() {
  if (!showAgentActions.value || aiBusy.value) {
    return
  }
  const id = props.movieId.trim()
  const body = normalizedDraft.value
  if (!id || !body) {
    aiError.value = t("detailPage.commentAiEmpty")
    return
  }
  if (isTooLong.value) {
    aiError.value = t("detailPage.commentTooLong", { n: MAX_MOVIE_COMMENT_RUNES })
    return
  }
  aiBusy.value = true
  aiError.value = ""
  try {
    const dto = await runAIAction("polish_comment", {
      movieId: id,
      body,
    })
    if (dto.noop) {
      aiError.value = t("detailPage.commentAiNoop")
      return
    }
    preview.value = dto
    previewOpen.value = true
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE") {
      aiError.value = t("detailPage.commentAiUnconfigured")
    } else {
      aiError.value = formatClientErr(err, t("detailPage.commentAiError"))
    }
  } finally {
    aiBusy.value = false
  }
}

async function applyCommentPreview() {
  const current = preview.value
  const movieId = props.movieId
  if (!current?.confirmToken || !current.sessionId) {
    previewOpen.value = false
    return
  }
  aiBusy.value = true
  aiError.value = ""
  try {
    const applied = await aiService.confirmTool({
      sessionId: current.sessionId,
      name: current.name,
      arguments: current.arguments ?? { movieId, body: current.proposedText },
      confirmToken: current.confirmToken,
    })
    const data = applied.replayed
      ? await libraryService.getMovieComment(movieId)
      : applied.data as { body?: string; updatedAt?: string } | undefined
    if (props.movieId !== movieId || preview.value !== current) return
    const body = data?.body ?? current.proposedText ?? ""
    draft.value = body
    lastSavedBody.value = body
    if (data?.updatedAt) {
      updatedAt.value = data.updatedAt
    }
    previewOpen.value = false
    preview.value = null
    flashCommentSaved()
  } catch (err) {
    aiError.value = formatClientErr(err, t("detailPage.commentAiApplyError"))
  } finally {
    aiBusy.value = false
  }
}

function discardCommentPreview() {
  previewOpen.value = false
  preview.value = null
}

watchDebounced(
  draft,
  async () => {
    if (hydrating.value || loading.value || props.readonly) {
      return
    }
    if (isTooLong.value) {
      saveError.value = t("detailPage.commentTooLong", { n: MAX_MOVIE_COMMENT_RUNES })
      return
    }
    if (!hasUnsavedChanges.value) {
      return
    }
    saveError.value = ""
    await saveCommentNow()
  },
  { debounce: 800, maxWait: 5000 },
)

watch(
  () => props.movieId,
  async (nextMovieId, previousMovieId) => {
    if (
      previousMovieId &&
      previousMovieId.trim() &&
      previousMovieId.trim() !== nextMovieId.trim() &&
      !props.readonly &&
      !isTooLong.value &&
      hasUnsavedChanges.value
    ) {
      await saveCommentNow(previousMovieId)
    }
    await load()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (!props.readonly && !isTooLong.value && hasUnsavedChanges.value) {
    void saveCommentNow()
  }
  if (commentSavedFlashTimer) clearTimeout(commentSavedFlashTimer)
})
</script>

<template>
  <Card class="gap-4 rounded-3xl border-border/70 bg-card/85">
    <CardHeader>
      <CardTitle>{{ t("detailPage.commentTitle") }}</CardTitle>
      <CardDescription v-if="props.readonly" class="text-pretty">
        {{ t("detailPage.commentReadonlyTrashHint") }}
      </CardDescription>
    </CardHeader>
    <CardContent class="flex flex-col gap-2">
      <p
        v-if="loading"
        class="text-sm text-muted-foreground"
      >
        {{ t("detailPage.commentLoading") }}
      </p>
      <p
        v-else-if="loadError"
        class="text-sm text-destructive"
      >
        {{ loadError }}
      </p>
      <template v-else>
        <label class="sr-only" for="movie-comment-body">{{ t("detailPage.commentTitle") }}</label>
        <div
          class="rounded-xl border border-border/60 bg-muted/40 text-foreground shadow-sm outline-none transition-[color,box-shadow] focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
          data-comment-field
        >
          <textarea
            id="movie-comment-body"
            v-model="draft"
            rows="6"
            :disabled="props.readonly"
            :readonly="props.readonly"
            :placeholder="t('detailPage.commentPlaceholder')"
            class="min-h-[140px] w-full resize-y border-0 bg-transparent px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
          />
          <div
            v-if="showAgentActions"
            class="flex items-center justify-end px-1 pb-1"
            data-comment-ai-actions
          >
            <Button
              type="button"
              variant="ghost"
              size="sm"
              class="min-h-11 rounded-lg text-muted-foreground hover:text-foreground md:h-8 md:min-h-8"
              :disabled="aiBusy || !normalizedDraft"
              data-comment-ai-polish
              @click="polishComment"
            >
              <Loader2 v-if="aiBusy" class="size-4 motion-safe:animate-spin" />
              <Sparkles v-else class="size-4" />
              {{ t("detailPage.commentAiPolish") }}
            </Button>
          <Button v-if="aiActionPending" type="button" variant="ghost" size="sm" data-ai-action-cancel @click="cancelAIAction">{{ t("common.cancel") }}</Button>
          </div>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            {{ t("detailPage.commentRuneCount", { n: runeCount, max: MAX_MOVIE_COMMENT_RUNES }) }}
          </span>
          <span v-if="saving">
            {{ t("detailPage.commentSaving") }}
          </span>
          <span v-else-if="hasUnsavedChanges && !saveError">
            {{ t("detailPage.commentUnsaved") }}
          </span>
          <span v-else-if="commentSavedFlash">
            {{ t("detailPage.commentAutoSaved") }}
          </span>
          <span v-else-if="updatedLabel">
            {{ t("detailPage.commentUpdatedAt", { time: updatedLabel }) }}
          </span>
        </div>
        <p
          v-if="saveError"
          class="text-sm text-destructive"
        >
          {{ saveError }}
        </p>
        <p v-if="aiError" class="text-sm text-destructive">{{ aiError }}</p>
      </template>
    </CardContent>
  </Card>
  <Dialog :open="previewOpen" @update:open="previewOpen = $event">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("detailPage.commentAiPreviewTitle") }}</DialogTitle>
        <DialogDescription>{{ t("detailPage.commentAiPreviewDesc") }}</DialogDescription>
      </DialogHeader>
      <div class="grid gap-3 text-sm">
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPage.commentAiBefore") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.originalText || "—" }}</p>
        </div>
        <div>
          <p class="mb-1 text-xs text-muted-foreground">{{ t("detailPage.commentAiAfter") }}</p>
          <p class="whitespace-pre-wrap rounded-lg bg-muted/50 p-2">{{ preview?.proposedText || "—" }}</p>
        </div>
      </div>
      <DialogFooter class="gap-2">
        <Button type="button" variant="outline" data-comment-ai-discard @click="discardCommentPreview">
          {{ t("detailPage.commentAiDiscard") }}
        </Button>
        <Button type="button" :disabled="aiBusy" data-comment-ai-apply @click="applyCommentPreview">
          {{ t("detailPage.commentAiApply") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { watchDebounced } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { Loader2, MessageSquare } from "lucide-vue-next"
import { HttpClientError } from "@/api/http-client"
import { MAX_MOVIE_COMMENT_RUNES, type MovieCommentDTO } from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
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
        <textarea
          id="movie-comment-body"
          v-model="draft"
          rows="6"
          :disabled="props.readonly"
          :readonly="props.readonly"
          :placeholder="t('detailPage.commentPlaceholder')"
          class="text-foreground placeholder:text-muted-foreground flex min-h-[140px] w-full rounded-xl border border-border/60 bg-muted/40 px-3 py-2 text-sm shadow-sm transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
        />
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
        <div v-if="!props.readonly" class="flex justify-end">
          <Button
            type="button"
            class="rounded-full"
            :disabled="saving || !hasUnsavedChanges || isTooLong"
            data-comment-save
            @click="saveCommentNow()"
          >
            <Loader2
              v-if="saving"
              class="size-4 shrink-0 animate-spin"
              aria-hidden="true"
            />
            <MessageSquare
              v-else
              class="size-4 shrink-0"
              data-icon="inline-start"
              aria-hidden="true"
            />
            {{ saving ? t("detailPage.commentSaving") : t("detailPage.commentSave") }}
          </Button>
        </div>
      </template>
    </CardContent>
  </Card>
</template>

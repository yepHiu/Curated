<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { api } from "@/api/endpoints"
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
import { getLocalMovieComment, putLocalMovieComment } from "@/lib/movie-comment-local-storage"

const useWeb = import.meta.env.VITE_USE_WEB_API === "true"

const props = withDefaults(
  defineProps<{
    movieId: string
    /** 回收站中仅展示，不可保存 */
    readonly?: boolean
  }>(),
  { readonly: false },
)

const { t, locale } = useI18n()

const draft = ref("")
const updatedAt = ref("")
const loading = ref(false)
const saving = ref(false)
const loadError = ref("")
const saveError = ref("")

function countRunes(s: string): number {
  return [...s].length
}

const runeCount = computed(() => countRunes(draft.value))

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
  loadError.value = ""
  saveError.value = ""
  try {
    let dto: MovieCommentDTO
    if (useWeb) {
      dto = await api.getMovieComment(id)
    } else {
      dto = getLocalMovieComment(id)
    }
    draft.value = dto.body
    updatedAt.value = dto.updatedAt
  } catch (err) {
    loadError.value = formatClientErr(err, t("detailPage.commentLoadError"))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (props.readonly) {
    return
  }
  const id = props.movieId.trim()
  if (!id) {
    return
  }
  if (countRunes(draft.value) > MAX_MOVIE_COMMENT_RUNES) {
    saveError.value = t("detailPage.commentTooLong", { n: MAX_MOVIE_COMMENT_RUNES })
    return
  }
  const bodyTrimmed = draft.value.trim()
  saving.value = true
  saveError.value = ""
  try {
    let dto: MovieCommentDTO
    if (useWeb) {
      dto = await api.putMovieComment(id, { body: bodyTrimmed })
    } else {
      dto = putLocalMovieComment(id, bodyTrimmed)
    }
    draft.value = dto.body
    updatedAt.value = dto.updatedAt
  } catch (err) {
    saveError.value = formatClientErr(err, t("detailPage.commentSaveError"))
  } finally {
    saving.value = false
  }
}

watch(
  () => props.movieId,
  () => {
    void load()
  },
  { immediate: true },
)
</script>

<template>
  <Card class="rounded-3xl border-border/70 bg-card/85">
    <CardHeader>
      <CardTitle>{{ t("detailPage.commentTitle") }}</CardTitle>
      <CardDescription class="text-pretty">
        {{
          props.readonly ? t("detailPage.commentReadonlyTrashHint") : t("detailPage.commentHint")
        }}
      </CardDescription>
    </CardHeader>
    <CardContent class="flex flex-col gap-3">
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
          :disabled="saving || props.readonly"
          :readonly="props.readonly"
          :placeholder="t('detailPage.commentPlaceholder')"
          class="text-foreground placeholder:text-muted-foreground flex min-h-[140px] w-full rounded-xl border border-border/60 bg-muted/40 px-3 py-2 text-sm shadow-sm transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
        />
        <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            {{ t("detailPage.commentRuneCount", { n: runeCount, max: MAX_MOVIE_COMMENT_RUNES }) }}
          </span>
          <span v-if="updatedLabel">
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
            class="rounded-xl"
            :disabled="saving"
            @click="save()"
          >
            {{ saving ? t("detailPage.commentSaving") : t("detailPage.commentSave") }}
          </Button>
        </div>
      </template>
    </CardContent>
  </Card>
</template>

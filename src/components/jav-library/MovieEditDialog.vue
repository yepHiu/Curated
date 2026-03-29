<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { PatchMovieBody } from "@/api/types"
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

const props = defineProps<{
  movie: Movie
  patchMovieDisplay: (body: PatchMovieBody, done: (err?: unknown) => void) => void
}>()

const open = defineModel<boolean>("open", { required: true })

const { t } = useI18n()

const movieEditSaving = ref(false)
const movieEditError = ref("")
const editDraftTitle = ref("")
const editDraftStudio = ref("")
const editDraftSummary = ref("")
const editDraftRelease = ref("")
const editDraftRuntime = ref("")

const releaseDateInputRx = /^\d{4}-\d{2}-\d{2}$/

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
})

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
          <Input
            id="movie-edit-title"
            v-model="editDraftTitle"
            class="rounded-xl text-sm"
            autocomplete="off"
          />
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
          <textarea
            id="movie-edit-summary"
            v-model="editDraftSummary"
            rows="5"
            class="text-foreground placeholder:text-muted-foreground flex min-h-[120px] w-full rounded-xl border border-border/60 bg-muted/40 px-3 py-2 text-sm shadow-sm transition-[color,box-shadow] outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
          />
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
</template>

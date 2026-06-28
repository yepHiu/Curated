<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, Save } from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
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

const props = defineProps<{
  comic: ComicBook
  patchComic: (patch: ComicPatch, done: (err?: unknown) => void) => void
}>()

const open = defineModel<boolean>("open", { required: true })

const { t } = useI18n()

const saving = ref(false)
const errorText = ref("")
const titleDraft = ref("")
const tagsDraft = ref("")
const ratingDraft = ref("")
const favoriteDraft = ref(false)

const tagSuggestions = ["author:", "series:", "volume:", "circle:"]

function syncDraftsFromComic() {
  errorText.value = ""
  titleDraft.value = props.comic.title
  tagsDraft.value = props.comic.tags.join(", ")
  ratingDraft.value = props.comic.rating == null ? "" : String(props.comic.rating)
  favoriteDraft.value = props.comic.isFavorite
}

watch(
  () => props.comic.id,
  () => {
    open.value = false
    errorText.value = ""
  },
)

watch(
  open,
  (isOpen) => {
    if (isOpen) {
      syncDraftsFromComic()
    }
  },
  { immediate: true },
)

const parsedTags = computed(() =>
  tagsDraft.value
    .split(/[,，\n]/)
    .map((tag) => tag.trim())
    .filter(Boolean),
)

function parsedRating(): number | null {
  const raw = String(ratingDraft.value ?? "").trim()
  if (!raw) return null
  const n = Number(raw)
  if (!Number.isFinite(n)) return props.comic.rating ?? null
  return Math.min(5, Math.max(0, n))
}

function applySuggestion(prefix: string) {
  const current = tagsDraft.value.trim()
  tagsDraft.value = current ? `${current}, ${prefix}` : prefix
}

function submitComicEditDialog() {
  errorText.value = ""
  saving.value = true
  try {
    props.patchComic(
      {
        title: titleDraft.value.trim() || props.comic.title,
        tags: parsedTags.value,
        rating: parsedRating(),
        favorite: favoriteDraft.value,
      },
      (err?: unknown) => {
        saving.value = false
        if (err) {
          errorText.value =
            err instanceof Error && err.message.trim()
              ? err.message
              : t("comics.editSaveFailed")
          return
        }
        open.value = false
      },
    )
  } catch (err) {
    saving.value = false
    errorText.value =
      err instanceof Error && err.message.trim() ? err.message : t("comics.editSaveFailed")
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="max-h-[min(90vh,40rem)] overflow-y-auto rounded-3xl border-border/70 sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("comics.editComicTitle") }}</DialogTitle>
        <DialogDescription class="text-pretty">
          {{ t("comics.editComicDesc") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-col gap-4 py-2">
        <p
          v-if="errorText"
          class="rounded-xl border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorText }}
        </p>

        <div class="grid gap-2">
          <span class="text-sm font-medium text-muted-foreground">
            {{ t("comics.sourceFile") }}
          </span>
          <p class="rounded-xl border border-border/60 bg-muted/30 px-3 py-2 text-sm break-all">
            {{ comic.sourceFileName }}
          </p>
        </div>

        <div class="grid gap-2">
          <span class="text-sm font-medium text-muted-foreground">
            {{ t("comics.sourceLocation") }}
          </span>
          <p
            class="max-h-24 overflow-y-auto rounded-xl border border-border/60 bg-muted/30 px-3 py-2 font-mono text-xs break-all"
          >
            {{ comic.location }}
          </p>
        </div>

        <div class="grid gap-2">
          <label class="text-sm font-medium" for="comic-edit-title">
            {{ t("comics.detailTitleLabel") }}
          </label>
          <Input
            id="comic-edit-title"
            v-model="titleDraft"
            data-comic-title-input
            class="rounded-xl text-sm"
            autocomplete="off"
          />
        </div>

        <div class="grid gap-2">
          <label class="text-sm font-medium" for="comic-edit-tags">
            {{ t("comics.detailTagsLabel") }}
          </label>
          <Input
            id="comic-edit-tags"
            v-model="tagsDraft"
            data-comic-tags-input
            class="rounded-xl text-sm"
            autocomplete="off"
          />
          <div class="flex flex-wrap gap-2">
            <Badge
              v-for="prefix in tagSuggestions"
              :key="prefix"
              as-child
              variant="secondary"
              class="rounded-full px-2.5 py-1 text-xs font-normal"
            >
              <button type="button" @click="applySuggestion(prefix)">
                {{ prefix }}
              </button>
            </Badge>
          </div>
        </div>

        <div class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end sm:gap-3">
          <div class="grid gap-2">
            <label class="text-sm font-medium" for="comic-edit-rating">
              {{ t("comics.detailRatingLabel") }}
            </label>
            <Input
              id="comic-edit-rating"
              v-model="ratingDraft"
              data-comic-rating-input
              class="rounded-xl text-sm"
              inputmode="decimal"
              type="number"
              min="0"
              max="5"
              step="0.5"
            />
          </div>

          <Button
            type="button"
            variant="outline"
            class="justify-start rounded-xl"
            data-comic-favorite-toggle
            :aria-pressed="favoriteDraft"
            @click="favoriteDraft = !favoriteDraft"
          >
            <Heart
              data-icon="inline-start"
              :class="favoriteDraft ? 'fill-primary text-primary' : 'text-muted-foreground'"
            />
            {{ favoriteDraft ? t("comics.favoriteOn") : t("comics.favoriteOff") }}
          </Button>
        </div>
      </div>

      <DialogFooter class="gap-3">
        <Button
          type="button"
          variant="outline"
          class="rounded-2xl"
          data-comic-edit-cancel
          :disabled="saving"
          @click="open = false"
        >
          {{ t("common.cancel") }}
        </Button>
        <Button
          type="button"
          class="rounded-2xl"
          data-comic-save
          :disabled="saving"
          @click="submitComicEditDialog"
        >
          <Save data-icon="inline-start" />
          {{ saving ? t("comics.editSaving") : t("comics.saveDetail") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

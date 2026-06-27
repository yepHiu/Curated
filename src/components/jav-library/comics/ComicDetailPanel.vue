<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, Save } from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  comic: ComicBook
  busy?: boolean
}>()

const emit = defineEmits<{
  patch: [patch: ComicPatch]
}>()

const { t } = useI18n()

const titleDraft = ref(props.comic.title)
const tagsDraft = ref(props.comic.tags.join(", "))
const ratingDraft = ref(props.comic.rating == null ? "" : String(props.comic.rating))
const favoriteDraft = ref(props.comic.isFavorite)

const tagSuggestions = ["作者:", "系列:", "卷:", "社团:"]

watch(
  () => props.comic,
  (comic) => {
    titleDraft.value = comic.title
    tagsDraft.value = comic.tags.join(", ")
    ratingDraft.value = comic.rating == null ? "" : String(comic.rating)
    favoriteDraft.value = comic.isFavorite
  },
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

function emitPatch() {
  emit("patch", {
    title: titleDraft.value.trim() || props.comic.title,
    tags: parsedTags.value,
    rating: parsedRating(),
    favorite: favoriteDraft.value,
  })
}
</script>

<template>
  <section class="flex min-w-0 flex-col gap-5">
    <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
      <div class="flex min-w-0 flex-col gap-3">
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          {{ t("comics.detailTitleLabel") }}
          <input
            v-model="titleDraft"
            data-comic-title-input
            class="h-10 rounded-xl border border-input bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          >
        </label>

        <label class="flex flex-col gap-1.5 text-sm font-medium">
          {{ t("comics.detailTagsLabel") }}
          <input
            v-model="tagsDraft"
            data-comic-tags-input
            class="h-10 rounded-xl border border-input bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          >
        </label>

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

      <div class="flex min-w-0 flex-col gap-3">
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          {{ t("comics.detailRatingLabel") }}
          <input
            v-model="ratingDraft"
            data-comic-rating-input
            class="h-10 rounded-xl border border-input bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
            inputmode="decimal"
            type="number"
            min="0"
            max="5"
            step="0.5"
          >
        </label>

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

        <Button
          type="button"
          class="rounded-xl"
          data-comic-save
          :disabled="props.busy"
          @click="emitPatch"
        >
          <Save data-icon="inline-start" />
          {{ t("comics.saveDetail") }}
        </Button>
      </div>
    </div>
  </section>
</template>

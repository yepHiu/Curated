<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, Heart, Save, Star } from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"

const props = defineProps<{
  comic: ComicBook
  busy?: boolean
}>()

const emit = defineEmits<{
  patch: [patch: ComicPatch]
  startReading: [pageIndex: number]
}>()

const { t } = useI18n()

const titleDraft = ref(props.comic.title)
const tagsDraft = ref(props.comic.tags.join(", "))
const ratingDraft = ref(props.comic.rating == null ? "" : String(props.comic.rating))
const favoriteDraft = ref(props.comic.isFavorite)

const tagSuggestions = ["作者:", "系列:", "卷:", "社团:"]

const coverSrc = computed(() => props.comic.coverUrl ?? props.comic.pages?.[0]?.thumbUrl ?? "")
const ratingLabel = computed(() =>
  props.comic.rating == null ? t("comics.noRating") : String(props.comic.rating),
)
const progressLabel = computed(() => {
  if (props.comic.pageCount <= 0) return "0 / 0"
  const page = Math.min(props.comic.pageCount, Math.max(1, props.comic.currentPageIndex + 1))
  return `${page} / ${props.comic.pageCount}`
})

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
  <Card
    data-comic-detail-panel
    class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10"
  >
    <CardContent
      data-comic-detail-content
      class="grid w-full min-w-0 gap-6 overflow-x-hidden p-5 sm:p-6 lg:grid-cols-[minmax(0,30rem)_minmax(0,1fr)] xl:grid-cols-[minmax(0,34rem)_minmax(0,1fr)]"
    >
      <div class="w-full min-w-0 max-w-full overflow-hidden lg:mx-auto lg:max-w-[min(100%,30rem)] xl:max-w-[min(100%,34rem)]">
        <div class="relative isolate w-full overflow-hidden rounded-[1.5rem] border border-border/60 bg-muted/40 aspect-[358/537]">
          <img
            v-if="coverSrc"
            data-comic-detail-cover
            :src="coverSrc"
            :alt="comic.title"
            class="absolute inset-0 z-0 h-full w-full object-cover"
            loading="eager"
            fetchpriority="high"
          >
          <div
            v-else
            class="absolute inset-0 z-0 flex items-center justify-center bg-muted text-muted-foreground"
            aria-hidden="true"
          >
            <BookOpen class="size-12" />
          </div>
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/55 via-transparent to-black/25"
            aria-hidden="true"
          />
          <div class="pointer-events-none absolute inset-x-0 top-0 z-[2] flex justify-start p-4">
            <Badge
              variant="outline"
              class="pointer-events-auto max-w-full truncate rounded-full border-border/40 bg-background/90 shadow-sm backdrop-blur-sm"
            >
              {{ comic.sourceFileName }}
            </Badge>
          </div>
        </div>

        <div
          data-comic-detail-rating-card
          class="mt-3 rounded-2xl border border-border/70 bg-background/50 p-3"
        >
          <p class="text-xs text-muted-foreground">{{ t("comics.detailRatingLabel") }}</p>
          <p class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-semibold">
            <Star class="size-4 shrink-0 text-primary" aria-hidden="true" />
            <span>{{ ratingLabel }}</span>
            <span class="text-xs font-normal text-muted-foreground">
              {{ t("comics.pageCount", { count: comic.pageCount }) }} · {{ progressLabel }}
            </span>
          </p>
        </div>
      </div>

      <div class="flex min-w-0 max-w-full flex-col gap-5">
        <div class="flex min-w-0 max-w-full flex-col gap-2">
          <CardTitle class="break-words text-2xl sm:text-3xl">
            {{ comic.title }}
          </CardTitle>
          <CardDescription class="break-words text-sm text-muted-foreground sm:text-base">
            {{ comic.sourceFileName }}
          </CardDescription>
        </div>

        <div class="grid min-w-0 gap-4 lg:grid-cols-[minmax(0,1fr)_14rem]">
          <div class="flex min-w-0 flex-col gap-3">
            <label class="flex flex-col gap-1.5 text-sm font-medium">
              {{ t("comics.detailTitleLabel") }}
              <Input
                v-model="titleDraft"
                data-comic-title-input
                class="h-10 rounded-xl"
              />
            </label>

            <label class="flex flex-col gap-1.5 text-sm font-medium">
              {{ t("comics.detailTagsLabel") }}
              <Input
                v-model="tagsDraft"
                data-comic-tags-input
                class="h-10 rounded-xl"
              />
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
              <Input
                v-model="ratingDraft"
                data-comic-rating-input
                class="h-10 rounded-xl"
                inputmode="decimal"
                type="number"
                min="0"
                max="5"
                step="0.5"
              />
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

            <Button
              type="button"
              variant="secondary"
              class="rounded-xl"
              data-comic-start-reading
              @click="emit('startReading', comic.currentPageIndex)"
            >
              <BookOpen data-icon="inline-start" />
              {{ t("comics.startReading") }}
            </Button>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

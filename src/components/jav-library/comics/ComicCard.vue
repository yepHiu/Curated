<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, Heart, Star } from "lucide-vue-next"
import type { ComicBook } from "@/domain/comic/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  comic: ComicBook
  selected?: boolean
}>()

const emit = defineEmits<{
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
}>()

const { t } = useI18n()

const coverSrc = computed(() => props.comic.coverUrl ?? props.comic.pages?.[0]?.thumbUrl ?? "")
const progressLabel = computed(() => {
  if (props.comic.pageCount <= 0) {
    return "0 / 0"
  }
  const page = Math.min(props.comic.pageCount, Math.max(1, props.comic.currentPageIndex + 1))
  return `${page} / ${props.comic.pageCount}`
})
const progressPercent = computed(() => {
  if (props.comic.pageCount <= 0) return 0
  return Math.min(100, Math.max(0, ((props.comic.currentPageIndex + 1) / props.comic.pageCount) * 100))
})
const ratingLabel = computed(() =>
  props.comic.rating == null ? t("comics.noRating") : String(props.comic.rating),
)
const visibleTags = computed(() => props.comic.tags.slice(0, 3))
const hiddenTagCount = computed(() => Math.max(0, props.comic.tags.length - visibleTags.value.length))

function toggleFavorite() {
  emit("toggleFavorite", {
    comicId: props.comic.id,
    nextValue: !props.comic.isFavorite,
  })
}
</script>

<template>
  <article
    data-comic-card
    :data-comic-card-id="comic.id"
    class="group flex min-w-0 flex-col overflow-hidden rounded-xl border border-border/70 bg-card/80 shadow-sm shadow-black/5 transition-[border-color,box-shadow] hover:border-primary/30 hover:shadow-md motion-reduce:transition-none"
    :class="props.selected ? 'border-primary/55 ring-2 ring-primary/20' : ''"
  >
    <button
      type="button"
      class="flex min-w-0 flex-1 flex-col text-left outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
      @click="emit('openDetails', comic.id)"
    >
      <div class="relative aspect-[2/3] overflow-hidden bg-muted/40">
        <img
          v-if="coverSrc"
          :src="coverSrc"
          :alt="comic.title"
          class="h-full w-full object-cover"
          loading="lazy"
        >
        <div
          v-else
          class="flex h-full w-full items-center justify-center bg-muted text-muted-foreground"
          aria-hidden="true"
        >
          <BookOpen class="size-10" />
        </div>
        <div class="absolute inset-x-0 bottom-0 h-1 bg-black/35" aria-hidden="true">
          <div class="h-full bg-primary" :style="{ width: `${progressPercent}%` }" />
        </div>
      </div>

      <div class="flex min-h-[8.5rem] min-w-0 flex-col gap-2 p-3">
        <div class="flex min-w-0 items-start justify-between gap-2">
          <h3 class="line-clamp-2 min-w-0 text-sm font-medium leading-snug">
            {{ comic.title }}
          </h3>
          <span class="inline-flex shrink-0 items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs tabular-nums text-muted-foreground">
            <Star class="size-3 fill-current" aria-hidden="true" />
            {{ ratingLabel }}
          </span>
        </div>

        <div class="flex flex-wrap gap-1.5">
          <Badge
            v-for="tag in visibleTags"
            :key="tag"
            variant="secondary"
            class="max-w-full truncate rounded-full px-2 py-0.5 text-[11px] font-normal"
          >
            {{ tag }}
          </Badge>
          <Badge
            v-if="hiddenTagCount > 0"
            variant="outline"
            class="rounded-full px-2 py-0.5 text-[11px] font-normal text-muted-foreground"
          >
            +{{ hiddenTagCount }}
          </Badge>
        </div>

        <div class="mt-auto flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>{{ t("comics.pageCount", { count: comic.pageCount }) }}</span>
          <span class="tabular-nums">{{ progressLabel }}</span>
        </div>
      </div>
    </button>

    <div class="flex items-center justify-between gap-2 border-t border-border/60 px-3 py-2">
      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="h-8 rounded-lg px-2 text-xs"
        @click="emit('openReader', comic.id, comic.currentPageIndex)"
      >
        <BookOpen data-icon="inline-start" />
        {{ t("comics.startReading") }}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="size-8 rounded-lg"
        :data-comic-favorite="String(comic.isFavorite)"
        :aria-pressed="comic.isFavorite"
        :aria-label="t('comics.favorite')"
        @click="toggleFavorite"
      >
        <Heart
          class="size-4"
          :class="comic.isFavorite ? 'fill-primary text-primary' : 'text-muted-foreground'"
          aria-hidden="true"
        />
      </Button>
    </div>
  </article>
</template>

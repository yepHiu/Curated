<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { BookOpen, Heart, Star } from "lucide-vue-next"
import type { ComicBook } from "@/domain/comic/types"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { Toggle } from "@/components/ui/toggle"

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
  <Card
    data-comic-card
    :data-comic-card-id="comic.id"
    class="group gap-0 overflow-hidden rounded-[1.2rem] border-border/70 bg-card/80 py-0 shadow-md shadow-black/5 transition-[box-shadow,border-color] duration-150 hover:border-primary/25 hover:shadow-lg motion-reduce:transition-none"
    :class="props.selected ? 'border-primary/55 shadow-lg shadow-primary/10 ring-2 ring-primary/25' : ''"
  >
    <button
      type="button"
      class="flex w-full flex-col text-left focus-visible:outline-none"
      @click="emit('openDetails', comic.id)"
    >
      <div class="p-[var(--movie-card-padding)] pb-0">
        <div
          data-comic-poster
          class="relative flex w-full items-start overflow-hidden rounded-[0.95rem] border border-border/60 bg-muted/40 aspect-[358/537]"
        >
          <img
            v-if="coverSrc"
            :src="coverSrc"
            :alt="comic.title"
            class="absolute inset-0 z-[1] h-full w-full object-cover"
            loading="lazy"
          >
          <div
            v-else
            class="absolute inset-0 z-[1] flex items-center justify-center bg-muted text-muted-foreground"
            aria-hidden="true"
          >
            <BookOpen class="size-10" />
          </div>

          <div
            v-if="coverSrc"
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/50 via-transparent to-black/25"
            aria-hidden="true"
          />

          <Badge
            class="relative z-[2] m-[var(--movie-card-padding)] h-5 max-w-[calc(100%-1.25rem)] truncate rounded-full border border-border/40 bg-background/85 px-1.5 text-[10px] text-foreground shadow-sm backdrop-blur-sm"
          >
            {{ comic.sourceFileName }}
          </Badge>

          <Toggle
            :pressed="props.comic.isFavorite"
            variant="outline"
            size="sm"
            class="absolute right-2.5 bottom-2.5 z-[2] rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
            :data-comic-favorite="String(comic.isFavorite)"
            :aria-label="t('comics.favorite')"
            @update:pressed="toggleFavorite"
            @click.stop
          >
            <Heart />
          </Toggle>

          <div class="absolute right-0 bottom-0 left-0 z-[2] h-1 bg-black/50" aria-hidden="true">
            <div class="h-full bg-primary transition-[width] duration-300 motion-reduce:transition-none" :style="{ width: `${progressPercent}%` }" />
          </div>
        </div>
      </div>

      <CardContent
        data-comic-card-body
        class="flex min-h-[var(--movie-card-body-min-height)] flex-col justify-between gap-[var(--movie-card-body-gap)] p-[var(--movie-card-padding)]"
      >
        <div class="flex min-h-0 min-w-0 flex-col justify-start gap-0.5">
          <CardTitle class="truncate text-[13px]">{{ comic.title }}</CardTitle>
          <CardDescription class="truncate text-[11px]">
            {{ t("comics.pageCount", { count: comic.pageCount }) }} · {{ progressLabel }}
          </CardDescription>
        </div>

        <div class="flex min-h-6 items-center gap-1 overflow-hidden">
          <Badge
            v-if="comic.rating != null"
            variant="outline"
            class="shrink-0 rounded-full border-primary/40 px-1.5 text-[10px] leading-tight text-primary"
          >
            <Star class="size-3 fill-current" aria-hidden="true" />
            {{ ratingLabel }}
          </Badge>
          <Badge
            v-for="tag in visibleTags"
            :key="tag"
            variant="secondary"
            class="max-w-[4.75rem] truncate rounded-full border border-border/60 bg-secondary/70 px-1.5 text-[10px] leading-tight"
          >
            {{ tag }}
          </Badge>
          <Badge
            v-if="hiddenTagCount > 0"
            variant="outline"
            class="shrink-0 rounded-full border-muted-foreground/35 px-1.5 text-[10px] leading-tight text-muted-foreground"
          >
            +{{ hiddenTagCount }}
          </Badge>
        </div>
      </CardContent>
    </button>
  </Card>
</template>

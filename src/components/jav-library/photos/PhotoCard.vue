<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Images, Star } from "lucide-vue-next"
import type { PhotoBook } from "@/domain/photo/types"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"

const props = defineProps<{
  photo: PhotoBook
}>()

const emit = defineEmits<{
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const { t } = useI18n()

const coverSrc = computed(() => props.photo.coverUrl ?? props.photo.pages?.[0]?.thumbUrl ?? "")
const progressLabel = computed(() => {
  if (props.photo.pageCount <= 0) {
    return "0 / 0"
  }
  const page = Math.min(props.photo.pageCount, Math.max(1, props.photo.currentPageIndex + 1))
  return `${page} / ${props.photo.pageCount}`
})
const progressPercent = computed(() => {
  if (props.photo.pageCount <= 0 || props.photo.currentPageIndex === 0) return 0
  return Math.min(100, Math.max(0, ((props.photo.currentPageIndex + 1) / props.photo.pageCount) * 100))
})
const ratingLabel = computed(() =>
  props.photo.rating == null ? t("photos.noRating") : String(props.photo.rating),
)
const visibleTags = computed(() => props.photo.tags.slice(0, 3))
const hiddenTagCount = computed(() => Math.max(0, props.photo.tags.length - visibleTags.value.length))

function openDetails() {
  emit("openDetails", props.photo.id)
}
</script>

<template>
  <Card
    data-photo-card
    :data-photo-card-id="photo.id"
    class="group gap-0 overflow-hidden rounded-[1.2rem] border-border/70 bg-card/80 py-0 shadow-md shadow-black/5 transition-[box-shadow,border-color] duration-150 hover:border-primary/25 hover:shadow-lg motion-reduce:transition-none"
  >
    <button
      type="button"
      data-photo-card-open
      :aria-label="photo.title"
      class="flex w-full flex-col text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
      @click="openDetails"
      @dblclick.stop="emit('openViewer', photo.id, photo.currentPageIndex)"
    >
      <div class="p-[var(--movie-card-padding)] pb-0">
        <div
          data-photo-poster
          class="relative flex w-full items-start overflow-hidden rounded-[0.95rem] border border-border/60 bg-muted/40 aspect-[358/537]"
        >
          <img
            v-if="coverSrc"
            :src="coverSrc"
            :alt="photo.title"
            class="absolute inset-0 z-[1] h-full w-full object-cover"
            decoding="async"
            loading="lazy"
          >
          <div
            v-else
            class="absolute inset-0 z-[1] flex items-center justify-center bg-muted text-muted-foreground"
            aria-hidden="true"
          >
            <Images class="size-10" />
          </div>

          <div
            v-if="coverSrc"
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/50 via-transparent to-black/25"
            aria-hidden="true"
          />

          <div class="absolute right-0 bottom-0 left-0 z-[2] h-1 bg-black/50" aria-hidden="true">
            <div class="h-full bg-primary transition-[width] duration-300 motion-reduce:transition-none" :style="{ width: `${progressPercent}%` }" />
          </div>
        </div>
      </div>

      <CardContent
        data-photo-card-body
        class="flex min-h-[var(--movie-card-body-min-height)] flex-col justify-between gap-[var(--movie-card-body-gap)] p-[var(--movie-card-padding)]"
      >
        <div class="flex min-h-0 min-w-0 flex-col justify-start gap-0.5">
          <CardTitle class="line-clamp-2 text-[13px] leading-snug">{{ photo.title }}</CardTitle>
          <CardDescription class="truncate text-[11px]">
            {{ t("photos.pageCount", { count: photo.pageCount }) }} <template v-if="photo.currentPageIndex > 0">· {{ progressLabel }}</template>
          </CardDescription>
        </div>

        <div class="flex min-h-6 items-center gap-1 overflow-hidden">
          <Badge
            v-if="photo.rating != null"
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

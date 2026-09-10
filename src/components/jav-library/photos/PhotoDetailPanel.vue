<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  Images,
  Star,
} from "lucide-vue-next"
import type { PhotoBook } from "@/domain/photo/types"
import DetailTagAddControl from "../DetailTagAddControl.vue"
import BookDetailFacts from "@/components/jav-library/books/BookDetailFacts.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardTitle,
} from "@/components/ui/card"


const props = defineProps<{
  photo: PhotoBook
  busy?: boolean
}>()

const emit = defineEmits<{
  startBrowsing: [pageIndex: number]
  browseByTag: [payload: { tag: string }]
  addTag: [tag: string, done: (error?: unknown) => void]
}>()

const { t } = useI18n()

function addTag(tag: string, done: (error?: unknown) => void) {
  emit("addTag", tag, done)
}

const coverSrc = computed(() => props.photo.coverUrl ?? props.photo.pages?.[0]?.thumbUrl ?? "")
const ratingLabel = computed(() =>
  props.photo.rating == null ? t("photos.noRating") : String(props.photo.rating),
)

function browseByTag(tag: string) {
  const value = tag.trim()
  if (!value) return
  emit("browseByTag", { tag: value })
}

</script>

<template>
  <Card
    data-photo-detail-panel
    class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 py-0 shadow-xl shadow-black/10"
  >
    <CardContent
      data-photo-detail-content
      class="relative grid w-full min-w-0 items-start justify-items-start gap-6 overflow-x-hidden p-5 sm:p-6 lg:justify-start sm:grid-cols-[fit-content(14rem)_minmax(0,1fr)] lg:grid-cols-[fit-content(18rem)_minmax(0,1fr)] xl:grid-cols-[fit-content(20rem)_minmax(0,1fr)]"
    >
      <div
        data-photo-detail-media-column
        class="w-fit min-w-0 max-w-full overflow-hidden"
      >
        <div
          data-photo-detail-cover-frame
          class="overflow-hidden rounded-[1.5rem] border border-border/60 bg-muted/40"
          :class="coverSrc
            ? 'relative isolate flex w-fit max-h-[min(56vh,24rem)] max-w-full'
            : 'relative isolate flex aspect-[358/537] w-56 max-w-full lg:w-72 xl:w-80'"
        >
          <button type="button" class="absolute inset-0 z-[2] rounded-[1.5rem] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" :aria-label="t('photos.startBrowsing')" :disabled="busy || photo.pageCount === 0" @click="emit('startBrowsing', 0)" />
          <img
            v-if="coverSrc"
            data-photo-detail-cover
            :src="coverSrc"
            :alt="photo.title"
            class="relative z-0 block h-auto max-h-[min(56vh,24rem)] w-auto max-w-full object-contain"
            decoding="async"
            loading="eager"
            fetchpriority="high"
          >
          <div
            v-else
            class="absolute inset-0 z-0 flex items-center justify-center bg-muted text-muted-foreground"
            aria-hidden="true"
          >
            <Images class="size-12" />
          </div>
          <div
            v-if="coverSrc"
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/50 via-transparent to-black/20"
            aria-hidden="true"
          />
        </div>
      </div>

      <div
        data-photo-detail-info-column
        class="flex min-w-0 max-w-full flex-col justify-start gap-4"
      >
        <div class="min-w-0 max-w-full">
          <CardTitle data-photo-detail-title class="break-words pr-12 text-2xl leading-snug sm:pr-14 sm:text-3xl">
            {{ photo.title }}
          </CardTitle>
        </div>

        <div v-if="photo.rating != null" data-photo-detail-rating class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium">{{ t("photos.detailRatingLabel") }}</span>
          <Badge
            variant="outline"
            class="inline-flex h-7 items-center gap-1 rounded-full border-primary/40 px-2 text-xs text-primary"
          >
            <Star class="size-3.5 fill-current" aria-hidden="true" />
            {{ ratingLabel }}
          </Badge>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            class="min-h-11 rounded-full lg:min-h-9"
            data-photo-start-browsing
            :disabled="busy || photo.pageCount === 0"
            @click="emit('startBrowsing', photo.currentPageIndex)"
          >
            <BookOpen data-icon="inline-start" aria-hidden="true" />
            {{ photo.currentPageIndex > 0 ? t('bookBrowser.continueAt', { page: Math.min(photo.pageCount, photo.currentPageIndex + 1) }) : t("photos.startBrowsing") }}
          </Button>
        </div>

        <BookDetailFacts :page-count="photo.pageCount" :file-name="photo.sourceFileName" :location="photo.location" :added-at="photo.addedAt" />

        <div data-photo-detail-tags class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("photos.detailTagsLabel") }}</p>
          <div class="flex flex-wrap items-center gap-2">
            <Badge
              v-for="tag in photo.tags"
              :key="tag"
              variant="secondary"
              as-child
              class="h-[29px] max-h-[29px] min-h-[29px] rounded-full border border-border/60 bg-secondary/70 px-2"
            >
              <button
                type="button"
                class="flex h-full max-w-[12rem] cursor-pointer items-center truncate rounded-[inherit] px-1 text-left text-xs font-medium transition hover:bg-secondary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                :aria-label="t('detailPanel.ariaSearchInLibrary', { tag })"
                @click="browseByTag(tag)"
              >
                {{ tag }}
              </button>
            </Badge>
            <DetailTagAddControl
              :key="photo.id"
              :tags="photo.tags"
              :disabled="busy"
              :save-error-message="t('photos.detailSaveError')"
              @add="addTag"
            />
          </div>
        </div>


      </div>

    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  Images,
  Plus,
  Star,
} from "lucide-vue-next"
import type { PhotoBook } from "@/domain/photo/types"
import BookDetailFacts from "@/components/jav-library/books/BookDetailFacts.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
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

const tagDraft = ref("")
const tagInputOpen = ref(false)
const tagSaving = ref(false)
const tagError = ref("")
const tagInput = ref<InstanceType<typeof Input> | null>(null)

async function openTagInput() {
  tagInputOpen.value = true
  await nextTick()
  tagInput.value?.$el?.focus()
}

function cancelTagInput() {
  if (tagSaving.value) return
  tagInputOpen.value = false
  tagDraft.value = ""
  tagError.value = ""
}

function addTag() {
  if (tagSaving.value || props.busy) return
  const tag = tagDraft.value.trim()
  tagError.value = ""
  if (!tag) return
  if ([...tag].length > 64) {
    tagError.value = t("curated.tagMaxRunes", { n: 64 })
    return
  }
  if (props.photo.tags.includes(tag)) {
    cancelTagInput()
    return
  }
  if (props.photo.tags.length >= 64) {
    tagError.value = t("curated.tagMaxCount", { n: 64 })
    return
  }
  tagSaving.value = true
  emit("addTag", tag, (error?: unknown) => {
    tagSaving.value = false
    if (error) {
      tagError.value = error instanceof Error && error.message ? error.message : t("photos.detailSaveError")
      return
    }
    cancelTagInput()
  })
}

function onTagKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  if (event.key === "Enter") { event.preventDefault(); addTag() }
  if (event.key === "Escape") { event.preventDefault(); cancelTagInput() }
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
      class="relative grid w-full min-w-0 items-start justify-items-start gap-6 overflow-x-hidden p-5 sm:p-6 lg:justify-start sm:grid-cols-[minmax(0,14rem)_minmax(0,1fr)] lg:grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)] xl:grid-cols-[minmax(13rem,20rem)_minmax(0,1fr)]"
    >
      <div
        data-photo-detail-media-column
        class="w-full min-w-0 max-w-full overflow-hidden lg:max-w-[min(100%,18rem)] xl:max-w-[min(100%,20rem)]"
      >
        <div
          data-photo-detail-cover-frame
          class="overflow-hidden rounded-[1.5rem] border border-border/60 bg-muted/40"
          :class="coverSrc
            ? 'relative isolate flex w-fit max-h-[min(56vh,24rem)] max-w-full'
            : 'relative isolate flex aspect-[358/537] w-full'"
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
            class="min-h-11 rounded-full px-8"
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
            <span v-if="!photo.tags.length" class="text-sm text-muted-foreground">
              {{ t("photos.noTags") }}
            </span>
            <Button
              v-if="!tagInputOpen"
              type="button"
              variant="secondary"
              class="min-h-11 rounded-full px-3 sm:min-h-8"
              data-photo-add-tag
              :disabled="busy"
              @click="openTagInput"
            >
              <Plus data-icon="inline-start" aria-hidden="true" />
              {{ t("common.add") }}
            </Button>
            <div v-else class="flex max-w-full flex-wrap items-center gap-2" :aria-busy="tagSaving">
              <Input
                ref="tagInput"
                v-model="tagDraft"
                data-photo-new-tag-input
                class="w-44 max-w-full rounded-xl"
                :disabled="tagSaving"
                :placeholder="t('detailPanel.newTagPlaceholder')"
                :aria-label="t('detailPanel.newTagPlaceholder')"
                :aria-invalid="Boolean(tagError)"
                autocomplete="off"
                @keydown="onTagKeydown"
              />
              <Button type="button" variant="secondary" class="min-h-11 rounded-full sm:min-h-8" data-photo-save-tag :disabled="tagSaving || busy || !tagDraft.trim()" @click="addTag">
                {{ tagSaving ? t('common.saving') : t('common.add') }}
              </Button>
              <Button type="button" variant="ghost" class="min-h-11 rounded-full sm:min-h-8" :disabled="tagSaving" @click="cancelTagInput">
                {{ t('common.cancel') }}
              </Button>
            </div>
          </div>
          <p v-if="tagError" role="alert" class="text-sm text-destructive">{{ tagError }}</p>
        </div>


      </div>

    </CardContent>
  </Card>
</template>

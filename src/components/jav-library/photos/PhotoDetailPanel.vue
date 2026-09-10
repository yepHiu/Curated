<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import {
  Eye,
  FolderOpen,
  Images,
  MoreVertical,
  Pencil,
  Star,
  Trash2,
} from "lucide-vue-next"
import type { PhotoBook } from "@/domain/photo/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const props = defineProps<{
  photo: PhotoBook
  busy?: boolean
}>()

const emit = defineEmits<{
  startBrowsing: [pageIndex: number]
  editPhoto: [photoId: string]
  deletePhoto: [photoId: string]
  revealSource: [photoId: string]
  browseByTag: [payload: { tag: string }]
}>()

const { t } = useI18n()

const coverSrc = computed(() => props.photo.coverUrl ?? props.photo.pages?.[0]?.thumbUrl ?? "")
const canRevealSource = computed(() => Boolean(props.photo.location.trim()))
const ratingLabel = computed(() =>
  props.photo.rating == null ? t("photos.noRating") : String(props.photo.rating),
)

function browseByTag(tag: string) {
  const value = tag.trim()
  if (!value) return
  emit("browseByTag", { tag: value })
}

function revealSource() {
  if (!canRevealSource.value) return
  emit("revealSource", props.photo.id)
}
</script>

<template>
  <Card
    data-photo-detail-panel
    class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10"
  >
    <CardContent
      data-photo-detail-content
      class="relative grid w-full min-w-0 items-start justify-items-start gap-5 overflow-x-hidden p-5 sm:p-6 lg:justify-start lg:grid-cols-[minmax(12rem,18rem)_minmax(0,1fr)] xl:grid-cols-[minmax(13rem,20rem)_minmax(0,1fr)]"
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
          <img
            v-if="coverSrc"
            data-photo-detail-cover
            :src="coverSrc"
            :alt="photo.title"
            class="relative z-0 block h-auto max-h-[min(56vh,24rem)] w-auto max-w-full object-contain"
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
          <CardTitle data-photo-detail-title class="break-words pr-12 text-xl sm:pr-14 sm:text-2xl">
            {{ photo.title }}
          </CardTitle>
        </div>

        <div data-photo-detail-rating class="flex flex-wrap items-center gap-2">
          <span class="text-sm font-medium">{{ t("photos.detailRatingLabel") }}</span>
          <Badge
            variant="outline"
            class="inline-flex h-7 items-center gap-1 rounded-full border-primary/40 px-2 text-xs text-primary"
          >
            <Star class="size-3.5 fill-current" aria-hidden="true" />
            {{ ratingLabel }}
          </Badge>
        </div>

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
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            class="rounded-full px-8"
            data-photo-start-browsing
            :disabled="busy"
            @click="emit('startBrowsing', photo.currentPageIndex)"
          >
            <Eye data-icon="inline-start" />
            {{ t("photos.startBrowsing") }}
          </Button>
        </div>
      </div>

      <div
        data-photo-more-actions-zone
        class="absolute right-4 top-4 sm:right-6 sm:top-6"
      >
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="shrink-0 rounded-xl"
              data-photo-more-actions
              :aria-label="t('photos.moreActions')"
            >
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="min-w-[11rem]">
            <DropdownMenuGroup>
              <DropdownMenuItem data-photo-edit-action @click="emit('editPhoto', photo.id)">
                <Pencil class="size-4 shrink-0" aria-hidden="true" />
                {{ t("photos.editPhoto") }}
              </DropdownMenuItem>
              <DropdownMenuItem
                data-photo-reveal-source
                :disabled="!canRevealSource"
                :title="!canRevealSource ? t('photos.revealPhotoNoPath') : undefined"
                @click="revealSource"
              >
                <FolderOpen class="size-4 shrink-0" aria-hidden="true" />
                {{ t("photos.revealPhotoSource") }}
              </DropdownMenuItem>
              <DropdownMenuItem
                data-photo-delete-action
                variant="destructive"
                @click="emit('deletePhoto', photo.id)"
              >
                <Trash2 class="size-4 shrink-0" aria-hidden="true" />
                {{ t("photos.deletePhoto") }}
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  FolderOpen,
  Heart,
  MoreVertical,
  Pencil,
  Star,
  Trash2,
} from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import ComicDeleteConfirmDialog from "./ComicDeleteConfirmDialog.vue"
import ComicEditDialog from "./ComicEditDialog.vue"

const props = defineProps<{
  comic: ComicBook
  busy?: boolean
}>()

const emit = defineEmits<{
  patch: [patch: ComicPatch, done: (err?: unknown) => void]
  startReading: [pageIndex: number]
  deleteComic: [comicId: string]
  revealSource: [comicId: string]
}>()

const { t } = useI18n()

const editOpen = ref(false)
const deleteConfirmOpen = ref(false)

const coverSrc = computed(() => props.comic.coverUrl ?? props.comic.pages?.[0]?.thumbUrl ?? "")
const ratingLabel = computed(() =>
  props.comic.rating == null ? t("comics.noRating") : String(props.comic.rating),
)
const progressLabel = computed(() => {
  if (props.comic.pageCount <= 0) return "0 / 0"
  const page = Math.min(props.comic.pageCount, Math.max(1, props.comic.currentPageIndex + 1))
  return `${page} / ${props.comic.pageCount}`
})
const canRevealSource = computed(() => Boolean(props.comic.location.trim()))

watch(
  () => props.comic.id,
  () => {
    editOpen.value = false
    deleteConfirmOpen.value = false
  },
)

function patchComicFromEdit(patch: ComicPatch, done: (err?: unknown) => void) {
  emit("patch", patch, done)
}

function revealSource() {
  if (!canRevealSource.value) return
  emit("revealSource", props.comic.id)
}

function confirmDeleteComic() {
  emit("deleteComic", props.comic.id)
}
</script>

<template>
  <Card
    data-comic-detail-panel
    class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10"
  >
    <CardContent
      data-comic-detail-content
      class="grid w-full min-w-0 gap-6 overflow-x-hidden p-5 sm:p-6 lg:grid-cols-[minmax(0,24rem)_minmax(0,1fr)] xl:grid-cols-[minmax(0,28rem)_minmax(0,1fr)]"
    >
      <div
        data-comic-detail-media-column
        class="w-full min-w-0 max-w-full overflow-hidden lg:mx-auto lg:max-w-[min(100%,24rem)] xl:max-w-[min(100%,28rem)]"
      >
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
          <p class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-medium">
            <Heart
              class="size-4 shrink-0"
              :class="comic.isFavorite ? 'fill-primary text-primary' : 'text-muted-foreground'"
              aria-hidden="true"
            />
            <span>{{ comic.isFavorite ? t("comics.favoriteOn") : t("comics.favoriteOff") }}</span>
          </p>
        </div>
      </div>

      <div class="flex min-w-0 max-w-full flex-col gap-5">
        <div class="flex min-w-0 max-w-full flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
          <div class="min-w-0 max-w-full flex-1">
            <CardTitle class="break-words text-2xl sm:text-3xl">
              {{ comic.title }}
            </CardTitle>
            <CardDescription class="break-words text-sm text-muted-foreground sm:text-base">
              {{ comic.sourceFileName }}
            </CardDescription>
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="shrink-0 rounded-xl"
                data-comic-more-actions
                :aria-label="t('comics.moreActions')"
              >
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[11rem]">
              <DropdownMenuGroup>
                <DropdownMenuItem data-comic-edit-action @click="editOpen = true">
                  <Pencil class="size-4 shrink-0" aria-hidden="true" />
                  {{ t("comics.editComic") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  data-comic-reveal-source
                  :disabled="!canRevealSource"
                  :title="!canRevealSource ? t('comics.revealComicNoPath') : undefined"
                  @click="revealSource"
                >
                  <FolderOpen class="size-4 shrink-0" aria-hidden="true" />
                  {{ t("comics.revealComicSource") }}
                </DropdownMenuItem>
                <DropdownMenuItem
                  data-comic-delete-action
                  variant="destructive"
                  @click="deleteConfirmOpen = true"
                >
                  <Trash2 class="size-4 shrink-0" aria-hidden="true" />
                  {{ t("comics.deleteComic") }}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <ComicEditDialog
            v-model:open="editOpen"
            :comic="comic"
            :patch-comic="patchComicFromEdit"
          />

          <ComicDeleteConfirmDialog
            v-model:open="deleteConfirmOpen"
            @confirm="confirmDeleteComic"
          />
        </div>

        <div data-comic-detail-tags class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("comics.detailTagsLabel") }}</p>
          <p v-if="comic.tags.length === 0" class="text-sm text-muted-foreground">
            {{ t("comics.noTags") }}
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <Badge
              v-for="tag in comic.tags"
              :key="tag"
              variant="secondary"
              class="rounded-full border border-border/60 bg-secondary/70 px-3 py-1 text-xs font-medium"
            >
              {{ tag }}
            </Badge>
          </div>
        </div>

        <div class="grid min-w-0 gap-3 rounded-2xl border border-border/70 bg-background/50 p-4 text-sm">
          <div class="min-w-0">
            <p class="text-xs text-muted-foreground">{{ t("comics.sourceFile") }}</p>
            <p class="mt-1 break-all font-medium">{{ comic.sourceFileName }}</p>
          </div>
          <div class="min-w-0">
            <p class="text-xs text-muted-foreground">{{ t("comics.sourceLocation") }}</p>
            <p class="mt-1 break-all font-mono text-xs text-muted-foreground">
              {{ comic.location || "—" }}
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            class="rounded-full px-8"
            data-comic-start-reading
            @click="emit('startReading', comic.currentPageIndex)"
          >
            <BookOpen data-icon="inline-start" />
            {{ t("comics.startReading") }}
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

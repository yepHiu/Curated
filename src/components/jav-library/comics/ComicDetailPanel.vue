<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  FolderOpen,
  Info,
  MoreVertical,
  Pencil,
  Trash2,
  X,
} from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import DetailTagAddControl from "../DetailTagAddControl.vue"
import BookDetailFacts from "@/components/jav-library/books/BookDetailFacts.vue"
import BookMediaInfoDialog from "@/components/jav-library/books/BookMediaInfoDialog.vue"
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
import ComicDeleteConfirmDialog from "./ComicDeleteConfirmDialog.vue"
import ComicEditDialog from "./ComicEditDialog.vue"
import BookRatingCard from "@/components/jav-library/books/BookRatingCard.vue"

const props = defineProps<{
  comic: ComicBook
  busy?: boolean
}>()

const emit = defineEmits<{
  patch: [patch: ComicPatch, done: (err?: unknown) => void]
  startReading: [pageIndex: number]
  deleteComic: [comicId: string]
  revealSource: [comicId: string]
  browseByTag: [payload: { tag: string }]
  reload: []
}>()

const { t } = useI18n()

const editOpen = ref(false)
const mediaInfoOpen = ref(false)
const deleteConfirmOpen = ref(false)
const tagError = ref("")

const coverSrc = computed(() => props.comic.coverUrl ?? props.comic.pages?.[0]?.thumbUrl ?? "")
const canRevealSource = computed(() => Boolean(props.comic.location.trim()))

watch(
  () => props.comic.id,
  () => {
    editOpen.value = false
    mediaInfoOpen.value = false
    deleteConfirmOpen.value = false
    tagError.value = ""
  },
)

/** 把漫画编辑弹窗的补丁交给详情页写入。 */
function patchComicFromEdit(patch: ComicPatch, done: (err?: unknown) => void) {
  emit("patch", patch, done)
}

/** 把媒体信息弹窗里的展示标题交给详情页写入。 */
function saveMediaTitle(title: string, done: (err?: unknown) => void) {
  emit("patch", { title }, done)
}

/** 把标签补丁交给详情页写入。 */
function patchComicTags(tags: string[]) {
  tagError.value = ""
  emit("patch", { tags }, (err?: unknown) => {
    // 标签保存失败时在详情页就地提示。
    if (!err) return
    tagError.value =
      err instanceof Error && err.message.trim()
        ? err.message
        : t("comics.detailSaveError")
  })
}

function addTag(tag: string, done: (error?: unknown) => void) {
  tagError.value = ""
  emit("patch", { tags: [...props.comic.tags, tag] }, done)
}

function removeTag(tag: string) {
  patchComicTags(props.comic.tags.filter((item) => item !== tag))
}

function browseByTagLabel(tag: string) {
  const value = tag.trim()
  if (!value) return
  emit("browseByTag", { tag: value })
}

function revealSource() {
  if (!canRevealSource.value) return
  emit("revealSource", props.comic.id)
}

function confirmDeleteComic() {
  emit("deleteComic", props.comic.id)
}

/** 把详情评分卡的选择写入当前漫画。 */
function commitRating(value: number | null) {
  emit("patch", { rating: value }, () => {})
}
</script>

<template>
  <Card
    data-comic-detail-panel
    class="min-w-0 w-full rounded-3xl border-border/70 bg-card/85 py-0 shadow-xl shadow-black/10"
  >
    <CardContent
      data-comic-detail-content
      class="relative grid w-full min-w-0 items-start justify-items-start gap-6 overflow-x-hidden p-5 sm:p-6 lg:justify-start sm:grid-cols-[fit-content(14rem)_minmax(0,1fr)] lg:grid-cols-[fit-content(18rem)_minmax(0,1fr)] xl:grid-cols-[fit-content(20rem)_minmax(0,1fr)]"
    >
      <div
        data-comic-detail-media-column
        class="w-fit min-w-0 max-w-full overflow-hidden"
      >
        <div
          data-comic-detail-cover-frame
          class="overflow-hidden rounded-[1.5rem] border border-border/60 bg-muted/40"
          :class="coverSrc
            ? 'relative isolate flex w-fit max-h-[min(56vh,24rem)] max-w-full'
            : 'relative isolate flex aspect-[358/537] w-56 max-w-full lg:w-72 xl:w-80'"
        >
          <button type="button" class="absolute inset-0 z-[2] rounded-[1.5rem] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" :aria-label="t('comics.startReading')" :disabled="busy || comic.pageCount === 0" @click="emit('startReading', 0)" />
          <img
            v-if="coverSrc"
            data-comic-detail-cover
            :src="coverSrc"
            :alt="comic.title"
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
            <BookOpen class="size-12" />
          </div>
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/55 via-transparent to-black/25"
            aria-hidden="true"
          />
        </div>
      </div>

      <div
        data-comic-detail-info-column
        class="flex min-w-0 max-w-full flex-col justify-start gap-4"
      >
        <div class="min-w-0 max-w-full">
          <CardTitle data-comic-detail-title class="break-words pr-12 text-2xl leading-snug sm:pr-14 sm:text-3xl">
            {{ comic.title }}
          </CardTitle>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <Button
            type="button"
            class="min-h-11 rounded-full lg:min-h-9"
            data-comic-start-reading
            :disabled="busy || comic.pageCount === 0"
            @click="emit('startReading', comic.currentPageIndex)"
          >
            <BookOpen data-icon="inline-start" />
            {{ comic.currentPageIndex > 0 ? t('bookBrowser.continueAt', { page: Math.min(comic.pageCount, comic.currentPageIndex + 1) }) : t("comics.startReading") }}
          </Button>
        </div>

        <BookDetailFacts :page-count="comic.pageCount" :added-at="comic.addedAt" />

        <div data-comic-detail-tags class="flex flex-col gap-3">
          <p class="text-sm font-medium">{{ t("comics.detailTagsLabel") }}</p>
          <div class="flex flex-wrap items-center gap-2">
            <Badge
              v-for="tag in comic.tags"
              :key="tag"
              variant="secondary"
              as-child
              class="h-[29px] max-h-[29px] min-h-[29px] rounded-full border border-border/60 bg-secondary/70 py-0 pl-2 pr-1"
            >
              <span class="inline-flex h-full max-w-full items-center gap-0.5 rounded-[inherit] py-0 pl-1">
                <button
                  type="button"
                  class="flex h-full min-w-0 max-w-[12rem] cursor-pointer items-center truncate rounded-md px-1.5 text-left text-xs font-medium transition hover:bg-secondary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                  :aria-label="t('detailPanel.ariaSearchInLibrary', { tag })"
                  @click="browseByTagLabel(tag)"
                >
                  {{ tag }}
                </button>
                <button
                  type="button"
                  class="inline-flex size-[1.375rem] shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-destructive/15 hover:text-destructive"
                  :data-comic-remove-tag="tag"
                  :aria-label="t('detailPanel.ariaRemoveMyTag', { tag })"
                  @click.stop="removeTag(tag)"
                >
                  <X class="size-3" />
                </button>
              </span>
            </Badge>

            <DetailTagAddControl
              :key="comic.id"
              :tags="comic.tags"
              :disabled="busy"
              :save-error-message="t('comics.detailSaveError')"
              @add="addTag"
            />
          </div>
          <p v-if="tagError" class="text-sm text-destructive">{{ tagError }}</p>
        </div>

        <BookRatingCard
          data-comic-detail-rating-card
          :rating="comic.rating ?? null"
          :disabled="busy"
          @commit="commitRating"
        />
      </div>

      <div
        data-comic-more-actions-zone
        class="absolute right-4 top-4 sm:right-6 sm:top-6"
      >
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              class="shrink-0 rounded-xl" :disabled="busy"
              data-comic-more-actions
              :aria-label="t('comics.moreActions')"
            >
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="min-w-[11rem]">
            <DropdownMenuGroup>
              <DropdownMenuItem data-comic-media-info @click="mediaInfoOpen = true">
                <Info aria-hidden="true" />
                {{ t("bookBrowser.mediaInfo") }}
              </DropdownMenuItem>
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
      </div>

      <BookMediaInfoDialog
        v-model:open="mediaInfoOpen"
        kind="comics"
        :entity-id="comic.id"
        :book="comic"
        :busy="busy"
        @save-title="saveMediaTitle"
        @applied="emit('reload')"
      />

      <ComicEditDialog
        v-model:open="editOpen"
        :comic="comic"
        :patch-comic="patchComicFromEdit"
      />

      <ComicDeleteConfirmDialog
        v-model:open="deleteConfirmOpen"
        @confirm="confirmDeleteComic"
      />
    </CardContent>
  </Card>
</template>

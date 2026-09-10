<script setup lang="ts">
import { onClickOutside } from "@vueuse/core"
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  BookOpen,
  FolderOpen,
  MoreVertical,
  Pencil,
  Plus,
  Trash2,
  X,
} from "lucide-vue-next"
import type { ComicBook, ComicPatch } from "@/domain/comic/types"
import BookDetailFacts from "@/components/jav-library/books/BookDetailFacts.vue"
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
}>()

const { t } = useI18n()

const editOpen = ref(false)
const deleteConfirmOpen = ref(false)
const tagDraft = ref("")
const tagError = ref("")
const tagInputOpen = ref(false)
const tagInputRef = ref<HTMLInputElement | null>(null)
const tagInlineZoneRef = ref<HTMLElement | null>(null)

const coverSrc = computed(() => props.comic.coverUrl ?? props.comic.pages?.[0]?.thumbUrl ?? "")
const canRevealSource = computed(() => Boolean(props.comic.location.trim()))
const maxComicTags = 64
const maxComicTagRunes = 64

watch(
  () => props.comic.id,
  () => {
    editOpen.value = false
    deleteConfirmOpen.value = false
    tagDraft.value = ""
    tagError.value = ""
    tagInputOpen.value = false
  },
)

function patchComicFromEdit(patch: ComicPatch, done: (err?: unknown) => void) {
  emit("patch", patch, done)
}

function patchComicTags(tags: string[]) {
  tagError.value = ""
  emit("patch", { tags }, (err?: unknown) => {
    if (!err) return
    tagError.value =
      err instanceof Error && err.message.trim()
        ? err.message
        : t("comics.detailSaveError")
  })
}

function cancelTagInput() {
  tagInputOpen.value = false
  tagDraft.value = ""
  tagError.value = ""
}

async function onTagAddButtonClick() {
  tagError.value = ""
  if (!tagInputOpen.value) {
    tagInputOpen.value = true
    await nextTick()
    tagInputRef.value?.focus()
    return
  }
  addTag()
}

function addTagWithValue(raw: string) {
  tagError.value = ""
  const tagText = raw.trim()
  if (!tagText) return
  if ([...tagText].length > maxComicTagRunes) {
    tagError.value = t("curated.tagMaxRunes", { n: maxComicTagRunes })
    return
  }
  if (props.comic.tags.includes(tagText)) {
    tagDraft.value = ""
    return
  }
  if (props.comic.tags.length >= maxComicTags) {
    tagError.value = t("curated.tagMaxCount", { n: maxComicTags })
    return
  }
  patchComicTags([...props.comic.tags, tagText])
  tagDraft.value = ""
}

function addTag() {
  addTagWithValue(tagDraft.value)
}

function onTagInputKeydown(event: KeyboardEvent) {
  if (event.key === "Enter") {
    event.preventDefault()
    addTag()
  } else if (event.key === "Escape") {
    event.preventDefault()
    cancelTagInput()
  }
}

onClickOutside(tagInlineZoneRef, () => {
  if (!tagInputOpen.value) return
  cancelTagInput()
})

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

        <BookDetailFacts :page-count="comic.pageCount" :file-name="comic.sourceFileName" :location="comic.location" :added-at="comic.addedAt" />

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

            <div
              ref="tagInlineZoneRef"
              class="flex max-w-full flex-wrap items-center gap-2"
            >
              <Button
                type="button"
                variant="secondary"
                class="h-[29px] shrink-0 rounded-2xl px-3 py-0 text-xs leading-none"
                data-comic-add-tag
                :disabled="props.busy"
                @click="onTagAddButtonClick"
              >
                <Plus class="size-3.5 shrink-0" data-icon="inline-start" />
                {{ t("common.add") }}
              </Button>
              <div
                v-if="tagInputOpen"
                class="relative max-w-full min-w-[min(100%,12rem)]"
              >
                <div
                  class="flex h-9 w-full items-center gap-0.5 rounded-2xl border border-border/80 bg-background/80 pl-3 pr-0.5 shadow-sm"
                >
                  <input
                    ref="tagInputRef"
                    v-model="tagDraft"
                    data-comic-new-tag-input
                    type="text"
                    maxlength="64"
                    autocomplete="off"
                    :placeholder="t('detailPanel.newTagPlaceholder')"
                    class="placeholder:text-muted-foreground h-8 min-w-0 flex-1 border-0 bg-transparent px-0 text-sm shadow-none outline-none focus-visible:ring-0"
                    @keydown="onTagInputKeydown"
                  >
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    class="size-8 shrink-0 rounded-xl text-muted-foreground hover:bg-muted hover:text-foreground"
                    :aria-label="t('detailPanel.ariaCancelTagInput')"
                    @click="cancelTagInput"
                  >
                    <X class="size-4" />
                  </Button>
                </div>
              </div>
            </div>
          </div>
          <p v-if="tagError" class="text-sm text-destructive">{{ tagError }}</p>
        </div>


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
              class="min-h-11 min-w-11 shrink-0 rounded-full" :disabled="busy"
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
      </div>

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

<script setup lang="ts">
import { computed } from "vue"
import { ChevronLeft, ChevronRight } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import BookReaderSettingsMenu from "@/components/jav-library/books/BookReaderSettingsMenu.vue"

type ReaderMode = "page" | "scroll"
type ReaderFit = "contain" | "width"
type ReaderDirection = "ltr" | "rtl"

const props = withDefaults(
  defineProps<{
    kind: "comics" | "photos"
    pageIndex: number
    pageCount: number
    mode: ReaderMode
    fit: ReaderFit
    direction: ReaderDirection
    showStitch?: boolean
    stitched?: boolean
  }>(),
  {
    showStitch: false,
    stitched: false,
  },
)

const emit = defineEmits<{
  previous: []
  next: []
  "update:mode": [value: ReaderMode]
  "update:fit": [value: ReaderFit]
  "update:direction": [value: ReaderDirection]
  stitchPrevious: []
  stitchNext: []
  clearStitch: []
}>()

const { t } = useI18n()

const previousLabel = computed(() =>
  props.kind === "photos" ? t("photos.viewerPrevious") : t("comics.readerPrevious"),
)
const nextLabel = computed(() =>
  props.kind === "photos" ? t("photos.viewerNext") : t("comics.readerNext"),
)
const progressPercent = computed(() => {
  if (props.pageCount <= 0) return 0
  return Math.min(100, ((props.pageIndex + 1) / props.pageCount) * 100)
})
const progressNow = computed(() => props.pageIndex + 1)

/** 把设置菜单的模式变更转发给阅读器。 */
function updateMode(value: ReaderMode) {
  emit("update:mode", value)
}

/** 把设置菜单的适配变更转发给阅读器。 */
function updateFit(value: ReaderFit) {
  emit("update:fit", value)
}

/** 把设置菜单的方向变更转发给阅读器。 */
function updateDirection(value: ReaderDirection) {
  emit("update:direction", value)
}
</script>

<template>
  <div
    data-book-reader-chrome
    :data-reader-chrome="kind === 'comics' ? '' : undefined"
    :data-photo-viewer-chrome="kind === 'photos' ? '' : undefined"
    class="pointer-events-auto flex w-full max-w-md flex-col gap-2 rounded-xl border border-border/70 bg-background/90 p-2 shadow-lg shadow-black/10 backdrop-blur"
  >
    <div
      data-reader-progress
      class="h-0.5 w-full overflow-hidden rounded-full bg-muted"
      role="progressbar"
      :aria-label="t('bookBrowser.readerProgress')"
      :aria-valuemin="1"
      :aria-valuemax="Math.max(1, pageCount)"
      :aria-valuenow="progressNow"
    >
      <div class="h-full bg-primary transition-[width] duration-200" :style="{ width: `${progressPercent}%` }" />
    </div>
    <div class="flex flex-wrap items-center justify-center gap-2">
      <Button
        data-reader-previous
        :data-photo-viewer-previous="kind === 'photos' ? '' : undefined"
        type="button"
        variant="ghost"
        size="icon"
        class="rounded-lg"
        :aria-label="previousLabel"
        :title="previousLabel"
        @click="emit('previous')"
      >
        <ChevronLeft />
      </Button>
      <span class="px-2 text-sm tabular-nums text-muted-foreground">
        {{ progressNow }} / {{ pageCount }}
      </span>
      <Button
        data-reader-next
        :data-photo-viewer-next="kind === 'photos' ? '' : undefined"
        type="button"
        variant="ghost"
        size="icon"
        class="rounded-lg"
        :aria-label="nextLabel"
        :title="nextLabel"
        @click="emit('next')"
      >
        <ChevronRight />
      </Button>
      <span class="mx-1 h-6 w-px bg-border" aria-hidden="true" />
      <BookReaderSettingsMenu
        :kind="kind"
        :mode="mode"
        :fit="fit"
        :direction="direction"
        :show-stitch="showStitch"
        :stitched="stitched"
        @update:mode="updateMode"
        @update:fit="updateFit"
        @update:direction="updateDirection"
        @stitch-previous="emit('stitchPrevious')"
        @stitch-next="emit('stitchNext')"
        @clear-stitch="emit('clearStitch')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import type { ComicBook, ComicReaderSettings } from "@/domain/comic/types"
import type { ComicReadingPreferencesDTO, ComicReadingProgressDTO } from "@/api/types"
import {
  clampComicPageIndex,
  clearTemporaryStitch,
  getTemporaryStitch,
  resolveComicReaderPreferences,
  resolveReaderKeyStep,
  resolveStitchPairDisplayOrder,
  setTemporaryStitch,
  type TemporaryStitch,
} from "@/lib/comic-reader-controls"
import ComicReaderChrome from "@/components/jav-library/comics/ComicReaderChrome.vue"
import ComicReaderSettingsMenu from "@/components/jav-library/comics/ComicReaderSettingsMenu.vue"

const PROGRESS_SAVE_DELAY_MS = 300

const props = withDefaults(
  defineProps<{
    comic: ComicBook
    readerDefaults: ComicReaderSettings
    initialPageIndex?: number
    loadPreferences?: (comicId: string) => Promise<ComicReadingPreferencesDTO | null | undefined>
    saveProgress?: (
      comicId: string,
      pageIndex: number,
      completed: boolean,
    ) => Promise<ComicReadingProgressDTO | unknown>
  }>(),
  {
    initialPageIndex: 0,
    loadPreferences: undefined,
    saveProgress: undefined,
  },
)

const pageIndex = ref(clampComicPageIndex(props.initialPageIndex, props.comic.pageCount))
const preferences = ref<ComicReaderSettings>(resolveComicReaderPreferences(props.readerDefaults))
const stitch = ref<TemporaryStitch | undefined>(getTemporaryStitch(props.comic.id))
let progressTimer: ReturnType<typeof setTimeout> | undefined

const pages = computed(() => props.comic.pages ?? [])
const visiblePageIndexes = computed(() => {
  if (!stitch.value) {
    return [pageIndex.value]
  }
  return resolveStitchPairDisplayOrder(
    stitch.value.anchorPageIndex,
    stitch.value.adjacentPageIndex,
    preferences.value.direction,
  )
})

const readerSurfaceClass = computed(() => {
  const fit = preferences.value.fit === "width" ? "w-full max-w-5xl" : "max-h-full max-w-full"
  return preferences.value.mode === "scroll"
    ? `${fit} mx-auto`
    : `${fit} mx-auto`
})

function clearProgressTimer() {
  if (progressTimer !== undefined) {
    clearTimeout(progressTimer)
    progressTimer = undefined
  }
}

function scheduleProgressSave() {
  if (!props.saveProgress) return
  clearProgressTimer()
  progressTimer = setTimeout(() => {
    const completed = props.comic.pageCount > 0 && pageIndex.value >= props.comic.pageCount - 1
    void props.saveProgress?.(props.comic.id, pageIndex.value, completed)
    progressTimer = undefined
  }, PROGRESS_SAVE_DELAY_MS)
}

function clearStitch() {
  stitch.value = undefined
  clearTemporaryStitch(props.comic.id)
}

function moveBy(step: number) {
  if (step === 0) return
  clearStitch()
  const next = clampComicPageIndex(pageIndex.value + step, props.comic.pageCount)
  if (next === pageIndex.value) return
  pageIndex.value = next
}

function stitchWith(offset: -1 | 1) {
  const adjacent = clampComicPageIndex(pageIndex.value + offset, props.comic.pageCount)
  if (adjacent === pageIndex.value) return
  const next = {
    anchorPageIndex: pageIndex.value,
    adjacentPageIndex: adjacent,
  }
  stitch.value = next
  setTemporaryStitch(props.comic.id, next)
}

function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.isComposing) {
    return
  }
  const target = event.target
  if (target instanceof HTMLElement) {
    const tag = target.tagName
    if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable) {
      return
    }
  }
  const step = resolveReaderKeyStep(event.key, preferences.value.direction)
  if (step === 0) return
  event.preventDefault()
  moveBy(step)
}

watch(pageIndex, () => {
  scheduleProgressSave()
})

onMounted(async () => {
  window.addEventListener("keydown", onKeydown)
  if (props.loadPreferences) {
    const loaded = await props.loadPreferences(props.comic.id)
    preferences.value = resolveComicReaderPreferences(props.readerDefaults, loaded)
  }
})

onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown)
  clearProgressTimer()
  clearStitch()
})
</script>

<template>
  <div class="relative flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-background text-foreground">
    <div class="sr-only" aria-live="polite">
      <span data-reader-page-index>{{ pageIndex }}</span>
      <span data-reader-direction>{{ preferences.direction }}</span>
      <span data-reader-mode>{{ preferences.mode }}</span>
      <span data-reader-fit>{{ preferences.fit }}</span>
    </div>

    <div
      class="min-h-0 flex-1 overflow-auto"
      :class="preferences.mode === 'scroll' ? 'px-3 py-6' : 'flex items-center justify-center px-3 py-6'"
    >
      <div
        class="flex gap-3"
        :class="preferences.mode === 'scroll' ? 'flex-col' : 'items-center justify-center'"
      >
        <figure
          v-for="idx in visiblePageIndexes"
          :key="idx"
          data-reader-visible-page
          class="flex min-w-0 flex-col items-center gap-2"
        >
          <img
            v-if="pages[idx]?.imageUrl || pages[idx]?.thumbUrl"
            :src="pages[idx]?.imageUrl || pages[idx]?.thumbUrl"
            :alt="`${props.comic.title} ${idx + 1}`"
            class="block rounded-lg bg-muted object-contain shadow-xl shadow-black/10"
            :class="readerSurfaceClass"
          >
          <div
            v-else
            class="flex aspect-[2/3] w-64 items-center justify-center rounded-lg bg-muted text-muted-foreground"
          >
            {{ idx + 1 }}
          </div>
          <figcaption class="text-xs tabular-nums text-muted-foreground">
            {{ idx + 1 }}
          </figcaption>
        </figure>
      </div>
    </div>

    <div class="pointer-events-none absolute inset-x-0 bottom-4 flex flex-col items-center gap-2 px-3">
      <ComicReaderSettingsMenu :preferences="preferences" />
      <ComicReaderChrome
        :page-index="pageIndex"
        :page-count="props.comic.pageCount"
        :stitched="Boolean(stitch)"
        @previous="moveBy(-1)"
        @next="moveBy(1)"
        @stitch-previous="stitchWith(-1)"
        @stitch-next="stitchWith(1)"
        @clear-stitch="clearStitch"
      />
    </div>
  </div>
</template>

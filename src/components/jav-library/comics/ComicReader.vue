<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { RouterLink, type RouteLocationRaw } from "vue-router"
import type { ComicBook, ComicReaderSettings } from "@/domain/comic/types"
import type {
  ComicReadingPreferencesDTO,
  ComicReadingProgressDTO,
  PutComicReadingPreferencesBody,
} from "@/api/types"
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
import { collectAdjacentBookPageUrls, prefetchBookPageUrls } from "@/lib/book-page-prefetch"
import BookReaderChrome from "@/components/jav-library/books/BookReaderChrome.vue"
import { Button } from "@/components/ui/button"

const PROGRESS_SAVE_DELAY_MS = 300

const props = withDefaults(
  defineProps<{
    comic: ComicBook
    readerDefaults: ComicReaderSettings
    initialPageIndex?: number
    loadPreferences?: (comicId: string) => Promise<ComicReadingPreferencesDTO | null | undefined>
    savePreferences?: (
      comicId: string,
      prefs: PutComicReadingPreferencesBody,
    ) => Promise<ComicReadingPreferencesDTO | unknown>
    saveProgress?: (
      comicId: string,
      pageIndex: number,
      completed: boolean,
    ) => Promise<ComicReadingProgressDTO | unknown>
    backTo?: RouteLocationRaw | string
    backLabel?: string
  }>(),
  {
    initialPageIndex: 0,
    loadPreferences: undefined,
    savePreferences: undefined,
    saveProgress: undefined,
    backTo: undefined,
    backLabel: "",
  },
)

const { t } = useI18n()
const pageIndex = ref(clampComicPageIndex(props.initialPageIndex, props.comic.pageCount))
const preferences = ref<ComicReaderSettings>(resolveComicReaderPreferences(props.readerDefaults))
const stitch = ref<TemporaryStitch | undefined>(getTemporaryStitch(props.comic.id))
const chromeVisible = ref(true)
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
  if (preferences.value.mode === "page") {
    return "h-full w-full max-h-full max-w-full"
  }
  const fit =
    preferences.value.fit === "width"
      ? "h-auto w-full max-w-5xl"
      : "h-auto max-w-full"
  return `${fit} mx-auto`
})

const readerScrollportClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "overflow-auto px-3 py-6"
    : "flex h-full w-full items-center justify-center overflow-hidden p-0",
)

const readerPageTrackClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex-col"
    : "h-full max-h-full min-h-0 w-full max-w-full items-center justify-center",
)

const readerPageTrackGapClass = computed(() => (stitch.value ? "gap-0" : "gap-3"))

const readerFigureClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex flex-col"
    : "grid h-full max-h-full min-h-0 w-full flex-1 place-items-center",
)

const readerImageFrameClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex min-w-0 items-center justify-center"
    : "flex h-full max-h-full min-h-0 w-full max-w-full items-center justify-center",
)

const readerImageClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "block rounded-lg bg-muted object-contain shadow-xl shadow-black/10"
    : "block object-contain",
)

/** 拼页时把两页图像贴向中间接缝。 */
function readerImagePositionClass(displayIndex: number) {
  if (!stitch.value || preferences.value.mode !== "page") {
    return "object-center"
  }
  return displayIndex === 0 ? "object-right" : "object-left"
}

const readerChromeOverlayClass = computed(() =>
  chromeVisible.value
    ? "translate-y-0 opacity-100"
    : "pointer-events-none translate-y-3 opacity-0",
)

const leftTurnStep = computed(() => (preferences.value.direction === "rtl" ? 1 : -1))
const rightTurnStep = computed(() => -leftTurnStep.value)
const leftTurnLabel = computed(() =>
  leftTurnStep.value < 0 ? t("comics.readerPrevious") : t("comics.readerNext"),
)
const rightTurnLabel = computed(() =>
  rightTurnStep.value < 0 ? t("comics.readerPrevious") : t("comics.readerNext"),
)
const showPageTurnZones = computed(() => preferences.value.mode === "page")

/** 取消尚未写出的进度保存定时器。 */
function clearProgressTimer() {
  if (progressTimer !== undefined) {
    clearTimeout(progressTimer)
    progressTimer = undefined
  }
}

/** 把当前页进度节流写入资料库。 */
function scheduleProgressSave() {
  if (!props.saveProgress) return
  clearProgressTimer()
  progressTimer = setTimeout(() => {
    const completed = props.comic.pageCount > 0 && pageIndex.value >= props.comic.pageCount - 1
    void props.saveProgress?.(props.comic.id, pageIndex.value, completed)
    progressTimer = undefined
  }, PROGRESS_SAVE_DELAY_MS)
}

/** 清除本次会话的临时拼页。 */
function clearStitch() {
  stitch.value = undefined
  clearTemporaryStitch(props.comic.id)
}

/** 按步长翻页并取消拼页。 */
function moveBy(step: number) {
  if (step === 0) return
  clearStitch()
  const next = clampComicPageIndex(pageIndex.value + step, props.comic.pageCount)
  if (next === pageIndex.value) return
  pageIndex.value = next
}

/** 把当前页与相邻页临时拼在一起，仅当前会话有效。 */
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

/** 把阅读偏好写回当前漫画并在切模式时拆掉拼页。 */
function applyPreferences(next: ComicReaderSettings) {
  if (next.mode !== preferences.value.mode) {
    clearStitch()
  }
  preferences.value = next
  void props.savePreferences?.(props.comic.id, next)
}

/** 从设置菜单写入阅读模式。 */
function updateMode(mode: ComicReaderSettings["mode"]) {
  applyPreferences({ ...preferences.value, mode })
}

/** 从设置菜单写入适配方式。 */
function updateFit(fit: ComicReaderSettings["fit"]) {
  applyPreferences({ ...preferences.value, fit })
}

/** 从设置菜单写入阅读方向。 */
function updateDirection(direction: ComicReaderSettings["direction"]) {
  applyPreferences({ ...preferences.value, direction })
}

/** 点击画面中央时显示或隐藏阅读控件。 */
function toggleChrome() {
  chromeVisible.value = !chromeVisible.value
}

/** 方向键与空格翻页；输入框内不拦截。 */
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

let cancelPrefetch: (() => void) | undefined

/** 预取当前可见页 ±1 的原图，stitch 时以可见页为中心。 */
function syncAdjacentPagePrefetch() {
  cancelPrefetch?.()
  cancelPrefetch = prefetchBookPageUrls(
    collectAdjacentBookPageUrls(pages.value, visiblePageIndexes.value),
  )
}

watch([pages, visiblePageIndexes], syncAdjacentPagePrefetch, { immediate: true })

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
  cancelPrefetch?.()
})
</script>

<template>
  <div
    data-reader-root
    class="relative flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-background text-foreground"
  >
    <div class="sr-only" aria-live="polite">
      <span data-reader-page-index>{{ pageIndex }}</span>
      <span data-reader-direction>{{ preferences.direction }}</span>
      <span data-reader-mode>{{ preferences.mode }}</span>
      <span data-reader-fit>{{ preferences.fit }}</span>
    </div>

    <div
      data-reader-surface
      class="relative h-full min-h-0 min-w-0 flex-1"
      @click="toggleChrome"
    >
      <div
        data-reader-scrollport
        class="min-h-0 flex-1"
        :class="readerScrollportClass"
      >
        <div
          data-reader-page-track
          class="flex"
          :class="[readerPageTrackClass, readerPageTrackGapClass]"
        >
          <figure
            v-for="(idx, displayIndex) in visiblePageIndexes"
            :key="idx"
            data-reader-visible-page
            class="min-w-0 items-center gap-2"
            :class="readerFigureClass"
          >
            <span
              data-reader-image-frame
              :class="readerImageFrameClass"
            >
              <img
                v-if="pages[idx]?.imageUrl || pages[idx]?.thumbUrl"
                data-reader-page-image
                :src="pages[idx]?.imageUrl || pages[idx]?.thumbUrl"
                :alt="`${props.comic.title} ${idx + 1}`"
                :class="[readerImageClass, readerSurfaceClass, readerImagePositionClass(displayIndex)]"
              >
              <div
                v-else
                class="flex aspect-[2/3] w-64 items-center justify-center rounded-lg bg-muted text-muted-foreground"
              >
                {{ idx + 1 }}
              </div>
            </span>
          </figure>
        </div>
      </div>

      <button
        v-if="showPageTurnZones"
        type="button"
        data-reader-turn-previous
        class="absolute inset-y-0 left-0 z-10 w-[18%] cursor-pointer bg-transparent"
        :aria-label="leftTurnLabel"
        @click.stop="moveBy(leftTurnStep)"
      />
      <button
        v-if="showPageTurnZones"
        type="button"
        data-reader-turn-next
        class="absolute inset-y-0 right-0 z-10 w-[18%] cursor-pointer bg-transparent"
        :aria-label="rightTurnLabel"
        @click.stop="moveBy(rightTurnStep)"
      />

      <div
        data-reader-hud-top
        :data-reader-chrome-visible="chromeVisible ? 'true' : 'false'"
        class="pointer-events-none absolute inset-x-0 top-0 z-20 flex items-center gap-2 px-3 py-3 transition duration-200 ease-out"
        :class="chromeVisible ? 'translate-y-0 opacity-100' : 'pointer-events-none -translate-y-3 opacity-0'"
        @click.stop
      >
        <Button
          v-if="backTo"
          as-child
          variant="secondary"
          class="pointer-events-auto min-h-11 shrink-0 rounded-full sm:min-h-8"
        >
          <RouterLink :to="backTo">
            {{ backLabel || t("shell.backPrevious") }}
          </RouterLink>
        </Button>
        <h1 data-reader-title class="min-w-0 truncate text-sm font-medium text-foreground/90">
          {{ comic.title }}
        </h1>
      </div>

      <div
        data-reader-chrome-overlay
        :data-reader-chrome-visible="chromeVisible ? 'true' : 'false'"
        class="pointer-events-none absolute inset-x-0 bottom-4 z-20 flex flex-col items-center gap-2 px-3 transition duration-200 ease-out"
        :class="readerChromeOverlayClass"
      >
        <div class="pointer-events-auto" @click.stop>
          <BookReaderChrome
            kind="comics"
            :page-index="pageIndex"
            :page-count="props.comic.pageCount"
            :mode="preferences.mode"
            :fit="preferences.fit"
            :direction="preferences.direction"
            :show-stitch="true"
            :stitched="Boolean(stitch)"
            @previous="moveBy(-1)"
            @next="moveBy(1)"
            @update:mode="updateMode"
            @update:fit="updateFit"
            @update:direction="updateDirection"
            @stitch-previous="stitchWith(-1)"
            @stitch-next="stitchWith(1)"
            @clear-stitch="clearStitch"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { RouterLink, type RouteLocationRaw } from "vue-router"
import type { PhotoBook, PhotoViewerSettings } from "@/domain/photo/types"
import { collectAdjacentBookPageUrls, prefetchBookPageUrls } from "@/lib/book-page-prefetch"
import BookReaderChrome from "@/components/jav-library/books/BookReaderChrome.vue"
import { Button } from "@/components/ui/button"

const props = withDefaults(
  defineProps<{
    photo: PhotoBook
    viewerDefaults: PhotoViewerSettings
    initialPageIndex?: number
    savePreferences?: (prefs: PhotoViewerSettings) => Promise<unknown>
    backTo?: RouteLocationRaw | string
    backLabel?: string
  }>(),
  {
    initialPageIndex: 0,
    savePreferences: undefined,
    backTo: undefined,
    backLabel: "",
  },
)

const { t } = useI18n()

/** 把页码限制在写真集的合法范围内。 */
function clampPageIndex(raw: number, pageCount: number) {
  if (pageCount <= 0) return 0
  if (!Number.isFinite(raw)) return 0
  return Math.min(pageCount - 1, Math.max(0, Math.floor(raw)))
}

/** 规范化查看器模式、适配和方向，忽略未知值。 */
function normalizeViewerSettings(settings: PhotoViewerSettings): PhotoViewerSettings {
  return {
    mode: settings.mode === "scroll" ? "scroll" : "page",
    fit: settings.fit === "width" ? "width" : "contain",
    direction: settings.direction === "rtl" ? "rtl" : "ltr",
  }
}

const pageIndex = ref(clampPageIndex(props.initialPageIndex, props.photo.pageCount))
const preferences = ref<PhotoViewerSettings>(normalizeViewerSettings(props.viewerDefaults))
const chromeVisible = ref(true)

const pages = computed(() => props.photo.pages ?? [])
const currentPage = computed(() => pages.value[pageIndex.value])
const currentPageSrc = computed(() => currentPage.value?.imageUrl || currentPage.value?.thumbUrl || "")
const leftTurnStep = computed(() => (preferences.value.direction === "rtl" ? 1 : -1))
const rightTurnStep = computed(() => -leftTurnStep.value)
const leftTurnLabel = computed(() =>
  leftTurnStep.value < 0 ? t("photos.viewerPrevious") : t("photos.viewerNext"),
)
const rightTurnLabel = computed(() =>
  rightTurnStep.value < 0 ? t("photos.viewerPrevious") : t("photos.viewerNext"),
)
const showPageTurnZones = computed(() => preferences.value.mode === "page")

const viewerSurfaceClass = computed(() => {
  if (preferences.value.mode === "page") {
    return "h-full w-full max-h-full max-w-full"
  }
  const fit =
    preferences.value.fit === "width"
      ? "h-auto w-full max-w-5xl"
      : "h-auto max-w-full"
  return `${fit} mx-auto`
})

const viewerScrollportClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "overflow-auto px-3 py-6"
    : "flex h-full w-full items-center justify-center overflow-hidden p-0",
)

const viewerPageTrackClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex-col gap-4"
    : "h-full max-h-full min-h-0 w-full max-w-full items-center justify-center gap-3",
)

const viewerFigureClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex flex-col"
    : "grid h-full max-h-full min-h-0 w-full flex-1 place-items-center",
)

const viewerImageFrameClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "flex min-w-0 items-center justify-center"
    : "flex h-full max-h-full min-h-0 w-full max-w-full items-center justify-center",
)

const viewerImageClass = computed(() =>
  preferences.value.mode === "scroll"
    ? "block rounded-lg bg-muted object-contain shadow-xl shadow-black/10"
    : "block object-contain",
)

const chromeOverlayClass = computed(() =>
  chromeVisible.value
    ? "translate-y-0 opacity-100"
    : "pointer-events-none translate-y-3 opacity-0",
)

/** 翻到相邻图片。 */
function moveBy(step: number) {
  if (step === 0) return
  const next = clampPageIndex(pageIndex.value + step, props.photo.pageCount)
  if (next === pageIndex.value) return
  pageIndex.value = next
}

/** 把查看偏好写回调用方，至少覆盖本次会话的全局默认。 */
function applyPreferences(next: PhotoViewerSettings) {
  preferences.value = next
  void props.savePreferences?.(next)
}

/** 从设置菜单写入浏览模式。 */
function updateMode(mode: PhotoViewerSettings["mode"]) {
  applyPreferences({ ...preferences.value, mode })
}

/** 从设置菜单写入适配方式。 */
function updateFit(fit: PhotoViewerSettings["fit"]) {
  applyPreferences({ ...preferences.value, fit })
}

/** 从设置菜单写入浏览方向。 */
function updateDirection(direction: PhotoViewerSettings["direction"]) {
  applyPreferences({ ...preferences.value, direction })
}

/** 点击画面中央时显示或隐藏浏览控件。 */
function toggleChrome() {
  chromeVisible.value = !chromeVisible.value
}

/** 把键盘按键映射成翻页步长。 */
function keyStep(key: string) {
  if (key === " " || key === "Spacebar" || key === "PageDown") return 1
  if (key === "PageUp") return -1
  if (key === "ArrowRight") return preferences.value.direction === "rtl" ? -1 : 1
  if (key === "ArrowLeft") return preferences.value.direction === "rtl" ? 1 : -1
  return 0
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
  const step = keyStep(event.key)
  if (step === 0) return
  event.preventDefault()
  moveBy(step)
}

onMounted(() => {
  window.addEventListener("keydown", onKeydown)
})

let cancelPrefetch: (() => void) | undefined

/** 预取当前页 ±1 的原图，离开查看器时取消。 */
function syncAdjacentPagePrefetch() {
  cancelPrefetch?.()
  cancelPrefetch = prefetchBookPageUrls(
    collectAdjacentBookPageUrls(pages.value, [pageIndex.value]),
  )
}

watch([pages, pageIndex], syncAdjacentPagePrefetch, { immediate: true })

onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown)
  cancelPrefetch?.()
})
</script>

<template>
  <div
    data-photo-viewer-root
    class="relative flex h-full min-h-0 min-w-0 flex-col overflow-hidden bg-background text-foreground"
  >
    <div class="sr-only" aria-live="polite">
      <span data-photo-viewer-page-index>{{ pageIndex }}</span>
      <span data-photo-viewer-direction>{{ preferences.direction }}</span>
      <span data-photo-viewer-mode>{{ preferences.mode }}</span>
      <span data-photo-viewer-fit>{{ preferences.fit }}</span>
    </div>

    <div
      data-photo-viewer-surface
      class="relative h-full min-h-0 min-w-0 flex-1"
      @click="toggleChrome"
    >
      <div
        data-photo-viewer-scrollport
        class="min-h-0 flex-1"
        :class="viewerScrollportClass"
      >
        <div
          data-photo-viewer-page-track
          class="flex"
          :class="viewerPageTrackClass"
        >
          <figure
            data-photo-viewer-visible-page
            class="min-w-0 items-center gap-2"
            :class="viewerFigureClass"
          >
            <span
              data-photo-viewer-image-frame
              :class="viewerImageFrameClass"
            >
              <img
                v-if="currentPageSrc"
                data-photo-viewer-page-image
                :src="currentPageSrc"
                :alt="`${photo.title} ${pageIndex + 1}`"
                :class="[viewerImageClass, viewerSurfaceClass]"
              >
              <div
                v-else
                class="flex aspect-[2/3] w-64 items-center justify-center rounded-lg bg-muted text-muted-foreground"
              >
                {{ pageIndex + 1 }}
              </div>
            </span>
          </figure>
        </div>
      </div>

      <button
        v-if="showPageTurnZones"
        type="button"
        data-photo-viewer-turn-previous
        class="absolute inset-y-0 left-0 z-10 w-[18%] cursor-pointer bg-transparent"
        :aria-label="leftTurnLabel"
        @click.stop="moveBy(leftTurnStep)"
      />
      <button
        v-if="showPageTurnZones"
        type="button"
        data-photo-viewer-turn-next
        class="absolute inset-y-0 right-0 z-10 w-[18%] cursor-pointer bg-transparent"
        :aria-label="rightTurnLabel"
        @click.stop="moveBy(rightTurnStep)"
      />

      <div
        data-photo-viewer-hud-top
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
        <h1 data-photo-viewer-title class="min-w-0 truncate text-sm font-medium text-foreground/90">
          {{ photo.title }}
        </h1>
      </div>

      <div
        data-photo-viewer-chrome-overlay
        :data-photo-viewer-chrome-visible="chromeVisible ? 'true' : 'false'"
        class="pointer-events-none absolute inset-x-0 bottom-4 z-20 flex flex-col items-center gap-2 px-3 transition duration-200 ease-out"
        :class="chromeOverlayClass"
      >
        <div class="pointer-events-auto" @click.stop>
          <span class="sr-only">{{ t("photos.viewerBrowse") }}</span>
          <BookReaderChrome
            kind="photos"
            :page-index="pageIndex"
            :page-count="photo.pageCount"
            :mode="preferences.mode"
            :fit="preferences.fit"
            :direction="preferences.direction"
            @previous="moveBy(-1)"
            @next="moveBy(1)"
            @update:mode="updateMode"
            @update:fit="updateFit"
            @update:direction="updateDirection"
          />
        </div>
      </div>
    </div>
  </div>
</template>

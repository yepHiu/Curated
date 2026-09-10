<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue"
import { ChevronLeft, ChevronRight, Eye, ScrollText } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import type { PhotoBook, PhotoViewerSettings } from "@/domain/photo/types"
import { Button } from "@/components/ui/button"

const props = withDefaults(
  defineProps<{
    photo: PhotoBook
    viewerDefaults: PhotoViewerSettings
    initialPageIndex?: number
  }>(),
  {
    initialPageIndex: 0,
  },
)

const { t } = useI18n()

function clampPageIndex(raw: number, pageCount: number) {
  if (pageCount <= 0) return 0
  if (!Number.isFinite(raw)) return 0
  return Math.min(pageCount - 1, Math.max(0, Math.floor(raw)))
}

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

const nextModeLabel = computed(() =>
  preferences.value.mode === "page"
    ? t("settings.photoViewerModeScroll")
    : t("settings.photoViewerModePage"),
)

function moveBy(step: number) {
  if (step === 0) return
  const next = clampPageIndex(pageIndex.value + step, props.photo.pageCount)
  if (next === pageIndex.value) return
  pageIndex.value = next
}

function toggleViewerMode() {
  preferences.value = {
    ...preferences.value,
    mode: preferences.value.mode === "page" ? "scroll" : "page",
  }
}

function toggleChrome() {
  chromeVisible.value = !chromeVisible.value
}

function keyStep(key: string) {
  if (key === " " || key === "Spacebar" || key === "PageDown") return 1
  if (key === "PageUp") return -1
  if (key === "ArrowRight") return preferences.value.direction === "rtl" ? -1 : 1
  if (key === "ArrowLeft") return preferences.value.direction === "rtl" ? 1 : -1
  return 0
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
  const step = keyStep(event.key)
  if (step === 0) return
  event.preventDefault()
  moveBy(step)
}

onMounted(() => {
  window.addEventListener("keydown", onKeydown)
})

onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown)
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

      <div
        data-photo-viewer-chrome-overlay
        :data-photo-viewer-chrome-visible="chromeVisible ? 'true' : 'false'"
        class="pointer-events-none absolute inset-x-0 bottom-4 z-20 flex flex-col items-center gap-2 px-3 transition duration-200 ease-out"
        :class="chromeOverlayClass"
      >
        <div class="pointer-events-auto" @click.stop>
          <div
            data-photo-viewer-chrome
            class="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl border border-border/70 bg-background/90 p-2 shadow-lg shadow-black/10 backdrop-blur"
          >
            <span class="sr-only">{{ t("photos.viewerBrowse") }}</span>
            <Button
              data-photo-viewer-previous
              type="button"
              variant="ghost"
              size="icon"
              class="rounded-lg"
              :aria-label="t('photos.viewerPrevious')"
              :title="t('photos.viewerPrevious')"
              @click="moveBy(-1)"
            >
              <ChevronLeft />
            </Button>
            <span class="px-2 text-sm tabular-nums text-muted-foreground">
              {{ pageIndex + 1 }} / {{ photo.pageCount }}
            </span>
            <Button
              data-photo-viewer-next
              type="button"
              variant="ghost"
              size="icon"
              class="rounded-lg"
              :aria-label="t('photos.viewerNext')"
              :title="t('photos.viewerNext')"
              @click="moveBy(1)"
            >
              <ChevronRight />
            </Button>
            <span class="mx-1 h-6 w-px bg-border" aria-hidden="true" />
            <Button
              data-photo-viewer-mode-toggle
              type="button"
              variant="ghost"
              size="icon"
              class="rounded-lg"
              :aria-label="`${t('photos.viewerMode')}: ${nextModeLabel}`"
              :aria-pressed="preferences.mode === 'scroll'"
              :title="`${t('photos.viewerMode')}: ${nextModeLabel}`"
              @click="toggleViewerMode"
            >
              <ScrollText v-if="preferences.mode === 'page'" />
              <Eye v-else />
            </Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

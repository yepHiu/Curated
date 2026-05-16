<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useEventListener } from "@vueuse/core"
import emblaCarouselVue from "embla-carousel-vue"
import { ChevronLeft, ChevronRight, Download, X } from "lucide-vue-next"
import { DialogDescription, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  images: readonly string[]
  initialIndex?: number
  movieCode?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const selectedIndex = ref(0)

const [mainViewportRef, mainApiRef] = emblaCarouselVue({
  loop: false,
  align: "center",
})

const [thumbViewportRef, thumbApiRef] = emblaCarouselVue({
  loop: false,
  align: "start",
  containScroll: "keepSnaps",
  dragFree: true,
})

function getMainApi() {
  return mainApiRef.value
}

function getThumbApi() {
  return thumbApiRef.value
}

function applySelectedFromMain() {
  const api = getMainApi()
  if (!api) return
  selectedIndex.value = api.selectedScrollSnap()
  const thumb = getThumbApi()
  thumb?.scrollTo(selectedIndex.value)
}

function scrollMainTo(i: number) {
  const api = getMainApi()
  const n = props.images.length
  if (!api || n === 0) return
  const idx = Math.max(0, Math.min(n - 1, i))
  api.scrollTo(idx)
}

function scrollMainPrev() {
  getMainApi()?.scrollPrev()
}

function scrollMainNext() {
  getMainApi()?.scrollNext()
}

let detachMainSelect: (() => void) | undefined

function attachMainSelectListener() {
  detachMainSelect?.()
  const api = getMainApi()
  if (!api) return
  const onSelect = () => applySelectedFromMain()
  api.on("select", onSelect)
  api.on("reInit", onSelect)
  detachMainSelect = () => {
    api.off("select", onSelect)
    api.off("reInit", onSelect)
  }
  onSelect()
}

watch(
  mainApiRef,
  (api) => {
    if (!api) return
    attachMainSelectListener()
  },
  { immediate: true },
)

const total = computed(() => props.images.length)
const canPrev = computed(() => selectedIndex.value > 0)
const canNext = computed(() => selectedIndex.value < props.images.length - 1)
const currentImageSrc = computed(() => props.images[selectedIndex.value]?.trim() ?? "")
const downloadFileName = computed(() =>
  buildPreviewDownloadFileName(currentImageSrc.value, props.movieCode, selectedIndex.value),
)
const loadedMainImageSrcs = ref<Set<string>>(new Set())
const previewTitle = computed(() =>
  t("preview.title", { code: props.movieCode?.trim() ?? "" }).trim(),
)

function sanitizeDownloadStem(value: string | undefined): string {
  const cleaned =
    value
      ?.trim()
      .replace(/[<>:"/\\|?*\u0000-\u001f]+/g, "_")
      .replace(/\s+/g, "_")
      .replace(/_+/g, "_")
      .replace(/^_+|_+$/g, "") ?? ""
  return cleaned || "preview"
}

function imageExtensionFromUrl(src: string): string {
  try {
    const pathname = new URL(src, window.location.href).pathname.toLowerCase()
    const ext = pathname.match(/\.([a-z0-9]+)$/)?.[1]
    if (ext === "jpeg") return "jpg"
    if (ext && ["jpg", "png", "webp", "gif", "avif"].includes(ext)) return ext
  } catch {
    // Fall back to jpg for opaque or malformed image URLs.
  }
  return "jpg"
}

function buildPreviewDownloadFileName(
  src: string,
  movieCode: string | undefined,
  index: number,
): string {
  const stem = sanitizeDownloadStem(movieCode)
  const sequence = String(index + 1).padStart(2, "0")
  return `${stem}-preview-${sequence}.${imageExtensionFromUrl(src)}`
}

function downloadCurrentImage() {
  const src = currentImageSrc.value
  if (!src) return

  const anchor = document.createElement("a")
  anchor.href = src
  anchor.download = downloadFileName.value
  anchor.target = "_blank"
  anchor.rel = "noopener"
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

function previewImageLabel(i: number): string {
  return t("preview.imageOf", { i })
}

function previewImageAlt(i: number): string {
  const label = previewImageLabel(i)
  return props.movieCode ? `${props.movieCode} ${label}` : label
}

function reinitCarousels() {
  getMainApi()?.reInit()
  getThumbApi()?.reInit()
}

function isSelectedOrNeighbor(index: number): boolean {
  return Math.abs(index - selectedIndex.value) <= 1
}

function mainImageLoading(index: number): "eager" | "lazy" {
  return isSelectedOrNeighbor(index) ? "eager" : "lazy"
}

function mainImageFetchPriority(index: number): "high" | "low" {
  return index === selectedIndex.value ? "high" : "low"
}

function isMainImageLoaded(src: string): boolean {
  return loadedMainImageSrcs.value.has(src)
}

function markMainImageLoaded(src: string) {
  loadedMainImageSrcs.value = new Set([...loadedMainImageSrcs.value, src])
}

watch(
  () => props.images,
  () => {
    loadedMainImageSrcs.value = new Set()
  },
)

watch(
  () => [props.initialIndex, props.images] as const,
  async ([initialIndex, images]) => {
    if (images.length === 0) return
    const start = Math.max(0, Math.min(images.length - 1, initialIndex ?? 0))
    selectedIndex.value = start
    await nextTick()
    reinitCarousels()
    requestAnimationFrame(() => {
      reinitCarousels()
      scrollMainTo(start)
      requestAnimationFrame(() => {
        applySelectedFromMain()
      })
    })
  },
  { immediate: true },
)

watch(
  () => props.images.length,
  async () => {
    if (props.images.length === 0) return
    await nextTick()
    reinitCarousels()
    requestAnimationFrame(() => applySelectedFromMain())
  },
)

useEventListener(
  "keydown",
  (e: KeyboardEvent) => {
    if (props.images.length === 0) return
    if (e.key === "ArrowLeft") {
      e.preventDefault()
      scrollMainPrev()
    } else if (e.key === "ArrowRight") {
      e.preventDefault()
      scrollMainNext()
    }
  },
  { capture: true },
)

defineExpose({
  mainViewportRef,
  thumbViewportRef,
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col overflow-hidden select-none">
    <DialogTitle class="sr-only">
      {{ previewTitle }} {{ selectedIndex + 1 }} / {{ total }}
    </DialogTitle>
    <DialogDescription class="sr-only">
      {{ t("preview.instructions") }}
    </DialogDescription>

    <div
      class="grid h-9 shrink-0 grid-cols-[4.5rem_1fr_4.5rem] items-center rounded-t-2xl bg-zinc-950/95 px-0.5 sm:h-10 sm:grid-cols-[5rem_1fr_5rem]"
    >
      <span aria-hidden="true" class="h-9 w-full sm:h-10" />
      <p class="min-w-0 truncate text-center text-xs text-zinc-400 sm:text-sm">
        {{ selectedIndex + 1 }} / {{ total }}
        <span v-if="movieCode" class="ml-2 text-zinc-500">{{ movieCode }}</span>
      </p>
      <div class="flex justify-self-end">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="group size-9 rounded-lg text-zinc-400 outline-none ring-offset-0 hover:bg-transparent hover:text-white focus-visible:border-transparent focus-visible:ring-0 focus-visible:ring-transparent focus-visible:ring-offset-0 focus-visible:shadow-[inset_0_0_0_0.5px_rgba(255,255,255,0.22)] dark:hover:bg-transparent disabled:opacity-40 sm:size-10"
          :disabled="!currentImageSrc"
          :aria-label="t('preview.download')"
          @click="downloadCurrentImage"
        >
          <span
            class="inline-flex size-7 items-center justify-center rounded-md transition-colors group-hover:bg-white/10 sm:size-8"
          >
            <Download class="size-4" />
          </span>
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          class="group size-9 rounded-lg text-zinc-400 outline-none ring-offset-0 hover:bg-transparent hover:text-white focus-visible:border-transparent focus-visible:ring-0 focus-visible:ring-transparent focus-visible:ring-offset-0 focus-visible:shadow-[inset_0_0_0_0.5px_rgba(255,255,255,0.22)] dark:hover:bg-transparent sm:size-10"
          :aria-label="t('preview.close')"
          @click="emit('close')"
        >
          <span
            class="inline-flex size-7 items-center justify-center rounded-md transition-colors group-hover:bg-white/10 sm:size-8"
          >
            <X class="size-4" />
          </span>
        </Button>
      </div>
    </div>

    <div
      class="flex min-h-0 flex-1 flex-row items-stretch gap-1.5 px-1.5 pt-0 pb-0 sm:gap-2 sm:px-2"
    >
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="size-10 shrink-0 self-center rounded-xl border-0 bg-transparent text-zinc-200 shadow-none hover:bg-white/10 hover:text-white focus-visible:ring-offset-0 dark:hover:bg-white/10 disabled:opacity-25 sm:size-11"
        :disabled="!canPrev"
        :aria-label="t('preview.previous')"
        @click="scrollMainPrev"
      >
        <ChevronLeft class="size-5 sm:size-6" />
      </Button>

      <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden bg-zinc-950 py-1">
        <div ref="mainViewportRef" class="min-h-0 min-w-0 w-full flex-1 overflow-hidden">
          <div class="flex h-full min-h-0 w-full">
            <div
              v-for="(src, i) in images"
              :key="`main-${i}-${src}`"
              class="relative h-full min-h-0 min-w-0 shrink-0 grow-0 basis-full"
            >
              <div
                class="relative flex h-full min-h-[200px] w-full min-w-0 items-center justify-center overflow-hidden px-0.5"
              >
                <img
                  v-if="src"
                  :src="src"
                  :alt="previewImageAlt(i + 1)"
                  :data-loaded="isMainImageLoaded(src) ? 'true' : 'false'"
                  :loading="mainImageLoading(i)"
                  :fetchpriority="mainImageFetchPriority(i)"
                  class="relative z-10 max-h-full max-w-full object-contain transition-opacity duration-200 motion-reduce:transition-none"
                  :class="isMainImageLoaded(src) ? 'opacity-100' : 'opacity-0'"
                  decoding="async"
                  draggable="false"
                  @load="markMainImageLoaded(src)"
                />
                <div
                  v-if="src && !isMainImageLoaded(src)"
                  class="absolute inset-0 z-0 animate-pulse rounded-xl bg-zinc-900"
                  aria-hidden="true"
                />
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="images.length > 1"
          ref="thumbViewportRef"
          class="max-h-[5.5rem] shrink-0 overflow-hidden px-0.5 pb-1"
        >
          <div class="-ml-2 flex h-[5rem] items-stretch">
            <div
              v-for="(src, i) in images"
              :key="`thumb-${i}-${src}`"
              class="flex shrink-0 grow-0 pl-2"
            >
              <button
                type="button"
                class="relative flex h-full max-w-[7.5rem] items-center justify-center overflow-hidden rounded-lg border-2 bg-zinc-900 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/40 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
                :class="
                  selectedIndex === i
                    ? 'border-primary ring-1 ring-primary/40'
                    : 'border-zinc-700 opacity-80 hover:border-zinc-500 hover:opacity-100'
                "
                :aria-label="previewImageLabel(i + 1)"
                :aria-current="selectedIndex === i ? 'true' : undefined"
                @click="scrollMainTo(i)"
              >
                <img
                  v-if="src"
                  :src="src"
                  alt=""
                  loading="lazy"
                  fetchpriority="low"
                  class="max-h-full w-auto max-w-[7.5rem] object-contain"
                  decoding="async"
                  draggable="false"
                />
              </button>
            </div>
          </div>
        </div>
      </div>

      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="size-10 shrink-0 self-center rounded-xl border-0 bg-transparent text-zinc-200 shadow-none hover:bg-white/10 hover:text-white focus-visible:ring-offset-0 dark:hover:bg-white/10 disabled:opacity-25 sm:size-11"
        :disabled="!canNext"
        :aria-label="t('preview.next')"
        @click="scrollMainNext"
      >
        <ChevronRight class="size-5 sm:size-6" />
      </Button>
    </div>
  </div>
</template>

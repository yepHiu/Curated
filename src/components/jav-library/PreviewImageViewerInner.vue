<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useEventListener } from "@vueuse/core"
import emblaCarouselVue from "embla-carousel-vue"
import { ChevronLeft, ChevronRight, X } from "lucide-vue-next"
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

function reinitCarousels() {
  getMainApi()?.reInit()
  getThumbApi()?.reInit()
}

watch(
  () => [props.initialIndex, props.images] as const,
  async ([initialIndex, images]) => {
    if (images.length === 0) return
    const start = Math.max(0, Math.min(images.length - 1, initialIndex ?? 0))
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
      {{ movieCode ? `${movieCode} 预览图` : "预览图" }} {{ selectedIndex + 1 }} / {{ total }}
    </DialogTitle>
    <DialogDescription class="sr-only">
      使用左右方向键或两侧按钮切换图片，Esc 关闭；点击下方缩略图跳转。
    </DialogDescription>

    <div
      class="grid h-9 shrink-0 grid-cols-[2.25rem_1fr_2.25rem] items-center rounded-t-2xl bg-zinc-950/95 px-0.5 sm:h-10 sm:grid-cols-[2.5rem_1fr_2.5rem]"
    >
      <span aria-hidden="true" class="size-9 sm:size-10" />
      <p class="min-w-0 truncate text-center text-xs text-zinc-400 sm:text-sm">
        {{ selectedIndex + 1 }} / {{ total }}
        <span v-if="movieCode" class="ml-2 text-zinc-500">{{ movieCode }}</span>
      </p>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="group size-9 justify-self-end rounded-lg text-zinc-400 outline-none ring-offset-0 hover:bg-transparent hover:text-white focus-visible:border-transparent focus-visible:ring-0 focus-visible:ring-transparent focus-visible:ring-offset-0 focus-visible:shadow-[inset_0_0_0_0.5px_rgba(255,255,255,0.22)] dark:hover:bg-transparent sm:size-10"
        aria-label="关闭"
        @click="emit('close')"
      >
        <span
          class="inline-flex size-7 items-center justify-center rounded-md transition-colors group-hover:bg-white/10 sm:size-8"
        >
          <X class="size-4" />
        </span>
      </Button>
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
        aria-label="上一张"
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
                class="flex h-full min-h-[200px] w-full min-w-0 items-center justify-center px-0.5"
              >
                <img
                  v-if="src"
                  :src="src"
                  :alt="
                    movieCode ? `${movieCode} 预览 ${i + 1}` : `预览 ${i + 1}`
                  "
                  class="max-h-full max-w-full object-contain"
                  decoding="async"
                  draggable="false"
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
                :aria-label="`第 ${i + 1} 张`"
                :aria-current="selectedIndex === i ? 'true' : undefined"
                @click="scrollMainTo(i)"
              >
                <img
                  v-if="src"
                  :src="src"
                  alt=""
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
        aria-label="下一张"
        @click="scrollMainNext"
      >
        <ChevronRight class="size-5 sm:size-6" />
      </Button>
    </div>
  </div>
</template>

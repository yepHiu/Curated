<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useEventListener } from "@vueuse/core"
import emblaCarouselVue from "embla-carousel-vue"
import { ChevronLeft, ChevronRight, X } from "lucide-vue-next"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const props = defineProps<{
  open: boolean
  images: readonly string[]
  /** 打开查看器时选中的索引 */
  initialIndex?: number
  /** 用于无障碍与标题 */
  movieCode?: string
}>()

const emit = defineEmits<{
  "update:open": [value: boolean]
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

function setOpen(v: boolean) {
  emit("update:open", v)
}

function reinitCarousels() {
  getMainApi()?.reInit()
  getThumbApi()?.reInit()
}

/**
 * Dialog 打开时 portal 内尺寸常晚于首帧就绪；Embla 若在宽为 0 时 init 会导致无法 scroll。
 * 在下一帧 + rAF 后 reInit 再定位，左右键/按钮/拖拽才能生效。
 */
watch(
  () => [props.open, props.initialIndex, props.images] as const,
  async ([isOpen, initialIndex, images]) => {
    if (!isOpen || images.length === 0) return
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
)

watch(
  () => props.images.length,
  async () => {
    if (!props.open || props.images.length === 0) return
    await nextTick()
    reinitCarousels()
    requestAnimationFrame(() => applySelectedFromMain())
  },
)

useEventListener(
  "keydown",
  (e: KeyboardEvent) => {
    if (!props.open || props.images.length === 0) return
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
  <Dialog :open="open" @update:open="setOpen">
    <DialogContent
      :show-close-button="false"
      minimal-motion
      :class="
        cn(
          '!flex fixed top-[50%] left-[50%] z-50 h-[min(90dvh,940px)] max-h-[min(90dvh,940px)] w-[min(94vw,1600px)] max-w-[min(94vw,1600px)] translate-x-[-50%] translate-y-[-50%] flex-col gap-0 overflow-hidden rounded-2xl border border-zinc-800 bg-zinc-950 p-0 shadow-2xl ring-1 ring-zinc-800/60 sm:max-w-[min(94vw,1600px)]',
        )
      "
    >
      <DialogTitle class="sr-only">
        {{ movieCode ? `${movieCode} 预览图` : "预览图" }} {{ selectedIndex + 1 }} / {{ total }}
      </DialogTitle>
      <DialogDescription class="sr-only">
        使用左右方向键或两侧按钮切换图片，Esc 关闭；点击下方缩略图跳转。
      </DialogDescription>

      <div class="flex h-full min-h-0 flex-1 flex-col overflow-hidden select-none">
        <div
          class="grid h-9 shrink-0 grid-cols-[2.25rem_1fr_2.25rem] items-center rounded-t-2xl border-b border-zinc-800/80 bg-zinc-950/95 px-0.5 sm:h-10 sm:grid-cols-[2.5rem_1fr_2.5rem]"
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
            class="size-9 justify-self-end rounded-lg text-zinc-400 hover:bg-white/10 hover:text-white focus-visible:border-transparent focus-visible:ring-2 focus-visible:ring-white/45 focus-visible:ring-offset-0 dark:hover:bg-white/10 sm:size-10"
            aria-label="关闭"
            @click="setOpen(false)"
          >
            <X class="size-4" />
          </Button>
        </div>

        <div
          class="flex min-h-0 flex-1 flex-row items-stretch gap-1.5 px-1.5 pt-0 pb-0 sm:gap-2 sm:px-2"
        >
          <Button
            type="button"
            variant="secondary"
            size="icon"
            class="h-auto min-h-[2.5rem] w-10 shrink-0 self-center rounded-xl border-zinc-600 bg-zinc-900/90 text-white shadow-sm focus-visible:ring-offset-0 disabled:opacity-25 sm:w-11"
            :disabled="!canPrev"
            aria-label="上一张"
            @click="scrollMainPrev"
          >
            <ChevronLeft class="size-5 sm:size-6" />
          </Button>

          <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden bg-zinc-950 py-1">
            <!-- 主图：Embla，与 shadcn-vue Carousel Thumbnails 主视口一致 -->
            <div
              ref="mainViewportRef"
              class="min-h-0 min-w-0 w-full flex-1 overflow-hidden"
            >
              <!-- 每页宽度 = 视口 100%：父级须 min-w-0；勿用 touch-pan-y 以免吃掉横向拖拽 -->
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
                        movieCode
                          ? `${movieCode} 预览 ${i + 1}`
                          : `预览 ${i + 1}`
                      "
                      class="max-h-full max-w-full object-contain"
                      decoding="async"
                      draggable="false"
                    />
                  </div>
                </div>
              </div>
            </div>

            <!-- 缩略图条：独立 Embla，点击同步主轮播（Thumbnails 模式） -->
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
            variant="secondary"
            size="icon"
            class="h-auto min-h-[2.5rem] w-10 shrink-0 self-center rounded-xl border-zinc-600 bg-zinc-900/90 text-white shadow-sm focus-visible:ring-offset-0 disabled:opacity-25 sm:w-11"
            :disabled="!canNext"
            aria-label="下一张"
            @click="scrollMainNext"
          >
            <ChevronRight class="size-5 sm:size-6" />
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

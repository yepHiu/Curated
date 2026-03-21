<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useEventListener } from "@vueuse/core"
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

const currentIndex = ref(0)

watch(
  () => [props.open, props.initialIndex, props.images.length] as const,
  ([isOpen]) => {
    if (!isOpen || props.images.length === 0) return
    const start = props.initialIndex ?? 0
    currentIndex.value = Math.max(0, Math.min(props.images.length - 1, start))
  },
)

const currentSrc = computed(() => props.images[currentIndex.value] ?? "")
const total = computed(() => props.images.length)
const canPrev = computed(() => currentIndex.value > 0)
const canNext = computed(() => currentIndex.value < props.images.length - 1)

function setOpen(v: boolean) {
  emit("update:open", v)
}

function prev() {
  if (canPrev.value) currentIndex.value -= 1
}

function next() {
  if (canNext.value) currentIndex.value += 1
}

useEventListener("keydown", (e: KeyboardEvent) => {
  if (!props.open || props.images.length === 0) return
  if (e.key === "ArrowLeft") {
    e.preventDefault()
    prev()
  } else if (e.key === "ArrowRight") {
    e.preventDefault()
    next()
  }
})
</script>

<template>
  <Dialog :open="open" @update:open="setOpen">
    <DialogContent
      :show-close-button="false"
      minimal-motion
      :class="
        cn(
          // 大屏但留边；主图区 flex-1，左右为导航栏，避免按钮叠在图上
          '!flex fixed top-[50%] left-[50%] z-50 h-[min(90dvh,940px)] max-h-[min(90dvh,940px)] w-[min(94vw,1600px)] max-w-[min(94vw,1600px)] translate-x-[-50%] translate-y-[-50%] flex-col gap-0 overflow-hidden rounded-2xl border-zinc-800 bg-zinc-950 p-0 shadow-2xl sm:max-w-[min(94vw,1600px)]',
        )
      "
    >
      <DialogTitle class="sr-only">
        {{ movieCode ? `${movieCode} 预览图` : "预览图" }} {{ currentIndex + 1 }} / {{ total }}
      </DialogTitle>
      <DialogDescription class="sr-only">
        使用左右方向键或两侧按钮切换图片，Esc 关闭。
      </DialogDescription>

      <div class="flex h-full min-h-0 flex-1 flex-col overflow-hidden select-none">
        <div
          class="grid h-9 shrink-0 grid-cols-[2.25rem_1fr_2.25rem] items-center border-b border-zinc-800/80 bg-zinc-950/95 px-0.5 sm:h-10 sm:grid-cols-[2.5rem_1fr_2.5rem]"
        >
          <span aria-hidden="true" class="size-9 sm:size-10" />
          <p class="min-w-0 truncate text-center text-xs text-zinc-400 sm:text-sm">
            {{ currentIndex + 1 }} / {{ total }}
            <span v-if="movieCode" class="ml-2 text-zinc-500">{{ movieCode }}</span>
          </p>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            class="size-9 justify-self-end text-zinc-400 hover:bg-white/10 hover:text-white sm:size-10"
            aria-label="关闭"
            @click="setOpen(false)"
          >
            <X class="size-4" />
          </Button>
        </div>

        <div
          class="flex min-h-0 flex-1 flex-row items-stretch gap-1 px-1 pb-1 sm:gap-1.5 sm:px-1.5 sm:pb-1.5"
        >
          <Button
            type="button"
            variant="secondary"
            size="icon"
            class="h-auto min-h-[2.5rem] w-10 shrink-0 self-center rounded-xl border-zinc-600 bg-zinc-900/90 text-white shadow-sm disabled:opacity-25 sm:w-11"
            :disabled="!canPrev"
            aria-label="上一张"
            @click="prev"
          >
            <ChevronLeft class="size-5 sm:size-6" />
          </Button>

          <div
            class="flex min-h-0 min-w-0 flex-1 items-center justify-center overflow-hidden bg-zinc-950"
          >
            <img
              v-if="currentSrc"
              :src="currentSrc"
              :alt="movieCode ? `${movieCode} 预览 ${currentIndex + 1}` : `预览 ${currentIndex + 1}`"
              class="h-full w-full object-contain"
              decoding="async"
              draggable="false"
            />
          </div>

          <Button
            type="button"
            variant="secondary"
            size="icon"
            class="h-auto min-h-[2.5rem] w-10 shrink-0 self-center rounded-xl border-zinc-600 bg-zinc-900/90 text-white shadow-sm disabled:opacity-25 sm:w-11"
            :disabled="!canNext"
            aria-label="下一张"
            @click="next"
          >
            <ChevronRight class="size-5 sm:size-6" />
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

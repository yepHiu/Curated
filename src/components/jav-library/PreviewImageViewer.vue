<script setup lang="ts">
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import PreviewImageViewerInner from "@/components/jav-library/PreviewImageViewerInner.vue"
import { cn } from "@/lib/utils"

defineProps<{
  open: boolean
  images: readonly string[]
  initialIndex?: number
  movieCode?: string
}>()

const emit = defineEmits<{
  "update:open": [value: boolean]
}>()

function setOpen(v: boolean) {
  emit("update:open", v)
}
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
      <PreviewImageViewerInner
        :images="images"
        :initial-index="initialIndex"
        :movie-code="movieCode"
        @close="setOpen(false)"
      />
    </DialogContent>
  </Dialog>
</template>

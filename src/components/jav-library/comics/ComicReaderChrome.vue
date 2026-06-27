<script setup lang="ts">
import { ChevronLeft, ChevronRight, Columns2, PanelTopClose, PanelTopOpen } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  pageIndex: number
  pageCount: number
  stitched?: boolean
}>()

const emit = defineEmits<{
  previous: []
  next: []
  stitchPrevious: []
  stitchNext: []
  clearStitch: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl border border-border/70 bg-background/90 p-2 shadow-lg shadow-black/10 backdrop-blur">
    <Button type="button" variant="ghost" size="icon" class="rounded-lg" :aria-label="t('comics.readerPrevious')" @click="emit('previous')">
      <ChevronLeft />
    </Button>
    <span class="px-2 text-sm tabular-nums text-muted-foreground">
      {{ props.pageIndex + 1 }} / {{ props.pageCount }}
    </span>
    <Button type="button" variant="ghost" size="icon" class="rounded-lg" :aria-label="t('comics.readerNext')" @click="emit('next')">
      <ChevronRight />
    </Button>
    <span class="mx-1 h-6 w-px bg-border" aria-hidden="true" />
    <Button type="button" variant="ghost" size="icon" class="rounded-lg" :aria-label="t('comics.readerStitchPrevious')" @click="emit('stitchPrevious')">
      <PanelTopOpen />
    </Button>
    <Button type="button" variant="ghost" size="icon" class="rounded-lg" :aria-label="t('comics.readerStitchNext')" @click="emit('stitchNext')">
      <Columns2 />
    </Button>
    <Button
      v-if="props.stitched"
      type="button"
      variant="ghost"
      size="icon"
      class="rounded-lg"
      :aria-label="t('comics.readerClearStitch')"
      @click="emit('clearStitch')"
    >
      <PanelTopClose />
    </Button>
  </div>
</template>

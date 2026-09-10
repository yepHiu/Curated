<script setup lang="ts">
import { computed } from "vue"
import { BookOpen, ChevronLeft, ChevronRight, ScrollText } from "lucide-vue-next"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import type { ComicReaderMode } from "@/domain/comic/types"

const props = defineProps<{
  pageIndex: number
  pageCount: number
  mode: ComicReaderMode
  stitched?: boolean
}>()

const emit = defineEmits<{
  previous: []
  next: []
  toggleMode: []
  stitchPrevious: []
  stitchNext: []
  clearStitch: []
}>()

const { t } = useI18n()

const nextModeLabel = computed(() =>
  props.mode === "page" ? t("settings.comicReaderModeScroll") : t("settings.comicReaderModePage"),
)
const modeToggleLabel = computed(() => `${t("comics.readerMode")}: ${nextModeLabel.value}`)
</script>

<template>
  <div
    data-reader-chrome
    class="pointer-events-auto flex flex-wrap items-center justify-center gap-2 rounded-xl border border-border/70 bg-background/90 p-2 shadow-lg shadow-black/10 backdrop-blur"
  >
    <Button
      data-reader-previous
      type="button"
      variant="ghost"
      size="icon"
      class="rounded-lg"
      :aria-label="t('comics.readerPrevious')"
      :title="t('comics.readerPrevious')"
      @click="emit('previous')"
    >
      <ChevronLeft />
    </Button>
    <span class="px-2 text-sm tabular-nums text-muted-foreground">
      {{ props.pageIndex + 1 }} / {{ props.pageCount }}
    </span>
    <Button
      data-reader-next
      type="button"
      variant="ghost"
      size="icon"
      class="rounded-lg"
      :aria-label="t('comics.readerNext')"
      :title="t('comics.readerNext')"
      @click="emit('next')"
    >
      <ChevronRight />
    </Button>
    <span class="mx-1 h-6 w-px bg-border" aria-hidden="true" />
    <Button
      data-reader-mode-toggle
      type="button"
      variant="ghost"
      size="icon"
      class="rounded-lg"
      :aria-label="modeToggleLabel"
      :aria-pressed="props.mode === 'scroll'"
      :title="modeToggleLabel"
      @click="emit('toggleMode')"
    >
      <ScrollText v-if="props.mode === 'page'" />
      <BookOpen v-else />
    </Button>
    <span class="mx-1 h-6 w-px bg-border" aria-hidden="true" />
    <Button
      data-reader-stitch-previous
      type="button"
      variant="ghost"
      size="sm"
      class="rounded-lg px-2.5 text-xs"
      :aria-label="t('comics.readerStitchPrevious')"
      :title="t('comics.readerStitchPrevious')"
      @click="emit('stitchPrevious')"
    >
      <ChevronLeft class="size-3.5" />
      <span>{{ t("comics.readerStitchPreviousShort") }}</span>
    </Button>
    <Button
      data-reader-stitch-next
      type="button"
      variant="ghost"
      size="sm"
      class="rounded-lg px-2.5 text-xs"
      :aria-label="t('comics.readerStitchNext')"
      :title="t('comics.readerStitchNext')"
      @click="emit('stitchNext')"
    >
      <span>{{ t("comics.readerStitchNextShort") }}</span>
      <ChevronRight class="size-3.5" />
    </Button>
    <Button
      v-if="props.stitched"
      data-reader-clear-stitch
      type="button"
      variant="ghost"
      size="sm"
      class="rounded-lg px-2.5 text-xs"
      :aria-label="t('comics.readerClearStitch')"
      :title="t('comics.readerClearStitch')"
      @click="emit('clearStitch')"
    >
      <span>{{ t("comics.readerClearStitchShort") }}</span>
    </Button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"

defineProps<{
  row: CuratedFrameDbRow
  imageUrl: string
  positionLabel: string
  batchMode: boolean
  selected: boolean
  nearDuplicate: boolean
  sectionActor?: string | null
}>()

const emit = defineEmits<{
  toggleSelection: [id: string, sectionActor?: string]
  open: [row: CuratedFrameDbRow, sectionActor?: string]
  contextmenu: [event: MouseEvent, row: CuratedFrameDbRow, sectionActor?: string]
}>()

const { t } = useI18n()
</script>

<template>
  <div
    class="group relative min-w-0 overflow-hidden rounded-2xl border bg-card/90 shadow-md transition hover:border-primary/40 hover:shadow-lg"
    :class="nearDuplicate ? 'border-amber-400/70' : 'border-border/70'"
  >
    <label
      v-if="batchMode"
      class="absolute top-2 left-2 z-10 flex cursor-pointer items-center justify-center rounded-md p-1.5 text-primary transition-colors hover:bg-foreground/12 focus-within:outline-none focus-within:ring-2 focus-within:ring-ring dark:hover:bg-black/50"
      :title="t('curated.exportToggleAria')"
      @click.stop
    >
      <input
        type="checkbox"
        class="size-4 cursor-pointer rounded accent-primary"
        :checked="selected"
        :aria-label="t('curated.exportToggleAria')"
        @change="emit('toggleSelection', row.id, sectionActor || undefined)"
      />
    </label>
    <span
      v-if="nearDuplicate"
      class="absolute top-2 right-2 z-10 rounded-full bg-amber-500/90 px-2 py-0.5 text-[11px] font-medium text-white shadow-sm"
    >
      {{ t("curated.duplicateReviewBadge") }}
    </span>
    <button
      type="button"
      class="block w-full text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      @contextmenu.prevent="emit('contextmenu', $event, row, sectionActor || undefined)"
      @click="emit('open', row, sectionActor || undefined)"
    >
      <div class="relative aspect-video w-full bg-black/80">
        <img
          :src="imageUrl"
          :alt="row.code"
          class="h-full w-full object-contain"
          loading="lazy"
        />
      </div>
      <div class="space-y-1 p-3">
        <p class="line-clamp-2 text-sm font-medium">{{ row.title }}</p>
        <p class="text-xs text-muted-foreground">
          {{ row.code }} · {{ positionLabel }}
        </p>
      </div>
    </button>
  </div>
</template>

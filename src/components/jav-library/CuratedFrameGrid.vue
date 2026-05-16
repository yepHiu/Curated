<script setup lang="ts">
import CuratedFrameCard from "@/components/jav-library/CuratedFrameCard.vue"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"

export type CuratedFrameGridItem = {
  row: CuratedFrameDbRow
  url: string
}

defineProps<{
  items: readonly CuratedFrameGridItem[]
  batchMode: boolean
  selectedIds: readonly string[]
  nearDuplicateIds: readonly string[]
  sectionActor?: string | null
}>()

const emit = defineEmits<{
  toggleSelection: [id: string, sectionActor?: string]
  open: [item: CuratedFrameGridItem, sectionActor?: string]
  contextmenu: [event: MouseEvent, item: CuratedFrameGridItem, sectionActor?: string]
}>()

function formatClock(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "00:00"
  const s = Math.floor(seconds % 60)
  const m = Math.floor(seconds / 60) % 60
  const h = Math.floor(seconds / 3600)
  const pad = (n: number) => String(n).padStart(2, "0")
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}
</script>

<template>
  <div
    class="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6"
  >
    <CuratedFrameCard
      v-for="item in items"
      :key="item.row.id"
      :row="item.row"
      :image-url="item.url"
      :position-label="formatClock(item.row.positionSec)"
      :batch-mode="batchMode"
      :selected="selectedIds.includes(item.row.id)"
      :near-duplicate="nearDuplicateIds.includes(item.row.id)"
      :section-actor="sectionActor"
      @toggle-selection="emit('toggleSelection', $event, sectionActor || undefined)"
      @contextmenu="(event) => emit('contextmenu', event, item, sectionActor || undefined)"
      @open="emit('open', item, sectionActor || undefined)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useWindowSize, useElementSize } from '@vueuse/core'
import CuratedFrameVirtualRow from './CuratedFrameVirtualRow.vue'
import CuratedFrameCard from "@/components/jav-library/CuratedFrameCard.vue"
import type { CuratedFrameDbRow } from "@/lib/curated-frames/db"

export type CuratedFrameGridItem = {
  row: CuratedFrameDbRow
  url: string
}

const props = defineProps<{
  items: readonly CuratedFrameGridItem[]
  batchMode: boolean
  selectedIds: readonly string[]
  nearDuplicateIds: readonly string[]
  sectionActor?: string | null
}>()
const host = ref<HTMLElement | null>(null)
const { width: viewportWidth } = useWindowSize()
const { width } = useElementSize(host)
const columns = computed(() => viewportWidth.value >= 1536 ? 6 : viewportWidth.value >= 1280 ? 5 : viewportWidth.value >= 1024 ? 4 : viewportWidth.value >= 768 ? 3 : 2)
const rows = computed(() => {
  const out = []
  for (let i = 0; i < props.items.length; i += columns.value) out.push(props.items.slice(i, i + columns.value))
  return out
})
const rowHeight = computed(() => Math.max(1, (width.value - (columns.value - 1) * 16) / columns.value) * 9 / 16 + 44)
const selectedSet = computed(() => new Set(props.selectedIds))
const duplicateSet = computed(() => new Set(props.nearDuplicateIds))

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
    ref="host"
    class="flex flex-col gap-4"
  >
    <CuratedFrameVirtualRow v-for="(row, index) in rows" :key="row[0]?.row.id ?? index" :height="rowHeight" :columns="columns">
    <CuratedFrameCard
      v-for="item in row"
      :key="item.row.id"
      :row="item.row"
      :image-url="item.url"
      :position-label="formatClock(item.row.positionSec)"
      :batch-mode="batchMode"
      :selected="selectedSet.has(item.row.id)"
      :near-duplicate="duplicateSet.has(item.row.id)"
      :section-actor="sectionActor"
      @toggle-selection="emit('toggleSelection', $event, sectionActor || undefined)"
      @contextmenu="(event) => emit('contextmenu', event, item, sectionActor || undefined)"
      @open="emit('open', item, sectionActor || undefined)"
    />
    </CuratedFrameVirtualRow>
  </div>
</template>

<script setup lang="ts">
import CuratedFrameGrid, {
  type CuratedFrameGridItem,
} from "@/components/jav-library/CuratedFrameGrid.vue"

defineProps<{
  items: readonly CuratedFrameGridItem[]
  batchMode: boolean
  selectedIds: readonly string[]
  nearDuplicateIds: readonly string[]
}>()

const emit = defineEmits<{
  toggleSelection: [id: string, sectionActor?: string]
  open: [item: CuratedFrameGridItem, sectionActor?: string]
  contextmenu: [event: MouseEvent, item: CuratedFrameGridItem, sectionActor?: string]
}>()

function forwardToggleSelection(id: string, sectionActor?: string) {
  emit("toggleSelection", id, sectionActor)
}

function forwardOpen(item: CuratedFrameGridItem, sectionActor?: string) {
  emit("open", item, sectionActor)
}

function forwardContextMenu(
  event: MouseEvent,
  item: CuratedFrameGridItem,
  sectionActor?: string,
) {
  emit("contextmenu", event, item, sectionActor)
}
</script>

<template>
  <CuratedFrameGrid
    :items="items"
    :batch-mode="batchMode"
    :selected-ids="selectedIds"
    :near-duplicate-ids="nearDuplicateIds"
    @toggle-selection="forwardToggleSelection"
    @contextmenu="forwardContextMenu"
    @open="forwardOpen"
  />
</template>

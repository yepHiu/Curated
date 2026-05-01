<script setup lang="ts">
import CuratedFrameGrid, {
  type CuratedFrameGridItem,
} from "@/components/jav-library/CuratedFrameGrid.vue"

export type CuratedFrameActorGroup = readonly [
  actor: string,
  items: readonly CuratedFrameGridItem[],
]

const props = withDefaults(defineProps<{
  actorGroups: readonly CuratedFrameActorGroup[]
  batchMode: boolean
  selectedIds: readonly string[]
  nearDuplicateIds: readonly string[]
  selectGroupAriaLabel: string
  batchExportMax?: number
}>(), {
  batchExportMax: 20,
})

const emit = defineEmits<{
  groupSelectionChange: [
    actor: string,
    items: readonly CuratedFrameGridItem[],
    checked: boolean,
  ]
  toggleSelection: [id: string, sectionActor?: string]
  open: [item: CuratedFrameGridItem, sectionActor?: string]
  contextmenu: [event: MouseEvent, item: CuratedFrameGridItem, sectionActor?: string]
}>()

function groupIsFullySelected(items: readonly CuratedFrameGridItem[]): boolean {
  const target = [...new Set(items.map((item) => item.row.id))].slice(0, props.batchExportMax)
  if (target.length === 0 || props.selectedIds.length !== target.length) {
    return false
  }
  const targetIds = new Set(target)
  return props.selectedIds.every((id) => targetIds.has(id))
}

function onGroupCheckboxChange(
  actor: string,
  items: readonly CuratedFrameGridItem[],
  event: Event,
) {
  const checked = (event.target as HTMLInputElement).checked
  emit("groupSelectionChange", actor, items, checked)
}

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
  <div class="flex flex-col gap-8">
    <section v-for="[actor, items] in actorGroups" :key="actor">
      <div class="mb-3">
        <label
          v-if="batchMode"
          class="flex cursor-pointer items-start gap-2.5"
        >
          <input
            type="checkbox"
            class="mt-1 size-4 shrink-0 cursor-pointer rounded accent-primary"
            :disabled="items.length === 0"
            :checked="groupIsFullySelected(items)"
            :aria-label="selectGroupAriaLabel"
            @change="onGroupCheckboxChange(actor, items, $event)"
          />
          <span class="min-w-0 text-lg font-semibold leading-snug">{{ actor }}</span>
        </label>
        <h2
          v-else
          class="text-lg font-semibold"
        >
          {{ actor }}
        </h2>
      </div>
      <CuratedFrameGrid
        :items="items"
        :batch-mode="batchMode"
        :selected-ids="selectedIds"
        :near-duplicate-ids="nearDuplicateIds"
        :section-actor="actor"
        @toggle-selection="forwardToggleSelection"
        @contextmenu="forwardContextMenu"
        @open="forwardOpen"
      />
    </section>
  </div>
</template>

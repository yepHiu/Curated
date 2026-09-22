<script setup lang="ts">
import { computed, ref } from "vue"
import { useResizeObserver } from "@vueuse/core"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { WishlistItem } from "@/domain/wishlist/types"
import { useLibraryService } from "@/services/library-service"
import { useLibraryScrollPreserve } from "@/composables/use-library-scroll-preserve"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"
import WishlistCard from "./WishlistCard.vue"

const props = defineProps<{ items: WishlistItem[]; scrollPreserveKey: string }>()
defineEmits<{ open: [id: string] }>()
const service = useLibraryService().wishlist
const probe = ref<HTMLElement | null>(null)
const scroller = ref<InstanceType<typeof DynamicScroller> | null>(null)
const columns = ref(1)
const gridStyle = buildMovieGridChunkStyle({
  minTrackWidth: "var(--movie-grid-min-track)",
  gap: "var(--movie-grid-gap)",
})
useResizeObserver(probe, () => {
  if (!probe.value) return
  const tracks = getComputedStyle(probe.value).gridTemplateColumns
  if (tracks && tracks !== "none") columns.value = Math.max(1, tracks.split(/\s+/).length)
})
const rows = computed(() => {
  const result: { id: string; items: WishlistItem[] }[] = []
  for (let i = 0; i < props.items.length; i += columns.value) {
    result.push({ id: `${columns.value}-${props.items[i]!.id}`, items: props.items.slice(i, i + columns.value) })
  }
  return result
})
const scrollEl = computed(() => (scroller.value?.$el as HTMLElement | undefined) ?? null)
useLibraryScrollPreserve({ scrollElRef: scrollEl, preserveKey: computed(() => props.scrollPreserveKey) })
</script>

<template>
  <DynamicScroller
    ref="scroller"
    :items="rows"
    :min-item-size="360"
    key-field="id"
    class="h-full min-h-0 overflow-y-auto pr-2"
  >
    <template #before>
      <div ref="probe" class="invisible grid h-0 w-full" :style="{ gridTemplateColumns: gridStyle.gridTemplateColumns, columnGap: gridStyle.columnGap }" aria-hidden="true" />
    </template>
    <template #default="{ item: row, index, active }">
      <DynamicScrollerItem :item="row" :active="active" :data-index="index" :size-dependencies="[columns, items]">
        <div class="grid w-full" :style="gridStyle">
          <div v-for="item in (row as { items: WishlistItem[] }).items" :key="item.id" class="flex min-w-0 justify-center">
            <WishlistCard
              class="w-full max-w-[var(--movie-card-max-width)]"
              :item="item"
              :image="item.assets[0] ? service.assetUrl(item.assets[0].thumbnailUrl) : undefined"
              @open="$emit('open', $event)"
            />
          </div>
        </div>
      </DynamicScrollerItem>
    </template>
  </DynamicScroller>
</template>

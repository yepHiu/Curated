<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useMediaQuery, useResizeObserver } from "@vueuse/core"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { ComicBook } from "@/domain/comic/types"
import ComicCard from "@/components/jav-library/comics/ComicCard.vue"
import {
  estimateVirtualMovieChunkHeight,
  getVirtualMovieFocusChunkIndex,
  resolveVirtualMoviePosterLoadPolicy,
} from "@/lib/library-virtual-scroll"
import {
  BOOK_VIRTUAL_BUFFER_CHUNKS,
  BOOK_VIRTUAL_BUFFER_PX,
  BOOK_VIRTUAL_ROWS_PER_CHUNK,
  buildBookVirtualChunks,
  parseBookGridColumnCount,
  type BookVirtualChunk,
} from "@/lib/book-virtual-scroll"
import {
  RETINA_DESKTOP_DENSITY_QUERY,
  resolveMovieGridDensity,
} from "@/lib/display-density"
import { buildMovieGridChunkStyle } from "@/lib/movie-grid-template"

const props = withDefaults(
  defineProps<{
    comics: readonly ComicBook[]
    selectedComicId?: string
    batchMode?: boolean
    batchSelectedIds?: readonly string[]
  }>(),
  {
    batchMode: false,
    batchSelectedIds: () => [],
  },
)

const emit = defineEmits<{
  openDetails: [comicId: string]
  openReader: [comicId: string, pageIndex: number]
  toggleFavorite: [payload: { comicId: string; nextValue: boolean }]
  toggleBatchSelect: [comicId: string]
}>()

const batchSelectedSet = computed(() => new Set(props.batchSelectedIds ?? []))
const rootEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const containerWidth = ref(typeof window !== "undefined" ? window.innerWidth : 1200)
const containerHeight = ref(typeof window !== "undefined" ? window.innerHeight : 900)
const retinaDesktopCompact = useMediaQuery(RETINA_DESKTOP_DENSITY_QUERY)
const comicGridDensity = computed(() => resolveMovieGridDensity(retinaDesktopCompact.value))
const comicGridChunkStyle = computed(() =>
  buildMovieGridChunkStyle({
    minTrackWidth: comicGridDensity.value.minTrackWidth,
    gap: comicGridDensity.value.gap,
  }),
)
const comicCardFrameStyle = computed(() => ({
  maxWidth: comicGridDensity.value.cardMaxWidth,
}))

useResizeObserver(rootEl, (entries) => {
  const rect = entries[0]?.contentRect
  if (!rect) return
  if (rect.width > 0) {
    containerWidth.value = rect.width
  }
  if (rect.height > 0) {
    containerHeight.value = rect.height
  }
})

const columnCountFallback = computed(() =>
  Math.max(
    1,
    Math.floor(
      (containerWidth.value + comicGridDensity.value.gapPxEstimate) /
        (comicGridDensity.value.minTrackPx + comicGridDensity.value.gapPxEstimate),
    ),
  ),
)

/** 首块布局后从 getComputedStyle 读取，与 auto-fill 真实列数对齐。 */
const measuredGridColumns = ref(0)

/** 把块网格节点的真实列数写回，后续分块容量跟随实测列数。 */
function onChunkGridRef(el: unknown) {
  const node =
    el && typeof el === "object" && el !== null && "$el" in el
      ? (el as { $el: unknown }).$el
      : el
  if (!node || !(node instanceof HTMLElement)) return
  void nextTick(() => {
    const n = parseBookGridColumnCount(node)
    if (n > 0 && measuredGridColumns.value !== n) {
      measuredGridColumns.value = n
    }
  })
}

const effectiveColumnCount = computed(() =>
  measuredGridColumns.value > 0 ? measuredGridColumns.value : columnCountFallback.value,
)

const chunkCapacity = computed(() =>
  Math.max(1, effectiveColumnCount.value * BOOK_VIRTUAL_ROWS_PER_CHUNK),
)

const estimatedChunkHeight = computed(() =>
  estimateVirtualMovieChunkHeight({
    containerWidth: containerWidth.value,
    columnCount: effectiveColumnCount.value,
    rowsPerChunk: BOOK_VIRTUAL_ROWS_PER_CHUNK,
    gapPx: comicGridDensity.value.gapPxEstimate,
  }),
)

/** 用 id / 收藏 / 标签数组成测量缓存键，避免无关重渲染反复测高。 */
function comicChunkSizeKey(comic: ComicBook): string {
  return `${comic.id}:${comic.isFavorite ? 1 : 0}:${comic.tags.length}`
}

const comicChunks = computed(() =>
  buildBookVirtualChunks(props.comics, chunkCapacity.value, comicChunkSizeKey),
)

/** 收窄 DynamicScroller 槽位类型，无效块渲染为空数组。 */
function isComicChunk(value: unknown): value is BookVirtualChunk<ComicBook> {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    "items" in value &&
    "sizeKey" in value &&
    Array.isArray((value as BookVirtualChunk<ComicBook>).items)
  )
}

/** 取出当前块；槽位异常时返回空块以免模板崩溃。 */
function getChunk(value: unknown): BookVirtualChunk<ComicBook> {
  return isComicChunk(value)
    ? value
    : {
        id: "invalid-chunk",
        items: [],
        sizeKey: "",
      }
}

const focusChunkIndex = computed(() =>
  getVirtualMovieFocusChunkIndex({
    scrollTop: scrollTop.value,
    viewportHeight: containerHeight.value,
    chunkHeight: estimatedChunkHeight.value,
  }),
)

/** 按块与焦点距离决定封面 loading / fetchpriority。 */
function posterLoadPolicyForChunk(index: number) {
  return resolveVirtualMoviePosterLoadPolicy(index, focusChunkIndex.value)
}

/** 记录滚动位置，供焦点块海报策略使用。 */
function onScrollerScroll(event: Event) {
  const target = event.target
  if (target instanceof HTMLElement) {
    scrollTop.value = target.scrollTop
  }
}
</script>

<template>
  <div
    ref="rootEl"
    class="relative h-full min-h-0"
  >
    <DynamicScroller
      v-if="props.comics.length"
      data-comic-grid-scroller
      :items="comicChunks"
      key-field="id"
      :min-item-size="estimatedChunkHeight"
      :buffer="BOOK_VIRTUAL_BUFFER_PX"
      :pool-size="BOOK_VIRTUAL_BUFFER_CHUNKS * 2 + 7"
      class="h-full min-h-0 overflow-y-auto pr-2"
      @scroll="onScrollerScroll"
    >
      <template #default="{ item, index, active }">
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :data-index="index"
          :size-dependencies="[getChunk(item).sizeKey]"
          :min-size="estimatedChunkHeight"
        >
          <div
            data-virtual-comic-grid
            class="grid w-full overflow-x-hidden"
            :ref="(el) => onChunkGridRef(el)"
            :style="comicGridChunkStyle"
          >
            <div
              v-for="comic in getChunk(item).items"
              :key="comic.id"
              data-comic-card-shell
              class="flex min-w-0 justify-center"
            >
              <div
                data-comic-card-frame
                class="w-full min-w-0"
                :style="comicCardFrameStyle"
              >
                <ComicCard
                  :comic="comic"
                  :selected="comic.id === props.selectedComicId"
                  :batch-mode="props.batchMode"
                  :batch-checked="batchSelectedSet.has(comic.id)"
                  :poster-loading="posterLoadPolicyForChunk(index).loading"
                  :poster-fetch-priority="posterLoadPolicyForChunk(index).fetchPriority"
                  @open-details="emit('openDetails', $event)"
                  @open-reader="(comicId, pageIndex) => emit('openReader', comicId, pageIndex)"
                  @toggle-favorite="emit('toggleFavorite', $event)"
                  @toggle-batch-select="emit('toggleBatchSelect', $event)"
                />
              </div>
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>

    <div
      v-else
      data-comic-grid-scroller
      class="h-full min-h-0 overflow-y-auto pr-2"
    >
      <div
        data-virtual-comic-grid
        class="grid w-full overflow-x-hidden"
        :style="comicGridChunkStyle"
      />
    </div>
  </div>
</template>

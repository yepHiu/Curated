<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useMediaQuery, useResizeObserver } from "@vueuse/core"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { PhotoBook } from "@/domain/photo/types"
import PhotoCard from "@/components/jav-library/photos/PhotoCard.vue"
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

const props = defineProps<{
  photos: readonly PhotoBook[]
}>()

const emit = defineEmits<{
  openDetails: [photoId: string]
  openViewer: [photoId: string, pageIndex: number]
}>()

const rootEl = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const containerWidth = ref(typeof window !== "undefined" ? window.innerWidth : 1200)
const containerHeight = ref(typeof window !== "undefined" ? window.innerHeight : 900)
const retinaDesktopCompact = useMediaQuery(RETINA_DESKTOP_DENSITY_QUERY)
const photoGridDensity = computed(() => resolveMovieGridDensity(retinaDesktopCompact.value))
const photoGridChunkStyle = computed(() =>
  buildMovieGridChunkStyle({
    minTrackWidth: photoGridDensity.value.minTrackWidth,
    gap: photoGridDensity.value.gap,
  }),
)
const photoCardFrameStyle = computed(() => ({
  maxWidth: photoGridDensity.value.cardMaxWidth,
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
      (containerWidth.value + photoGridDensity.value.gapPxEstimate) /
        (photoGridDensity.value.minTrackPx + photoGridDensity.value.gapPxEstimate),
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
    gapPx: photoGridDensity.value.gapPxEstimate,
  }),
)

/** 用 id / 收藏 / 标签数组成测量缓存键。 */
function photoChunkSizeKey(photo: PhotoBook): string {
  return `${photo.id}:${photo.isFavorite ? 1 : 0}:${photo.tags.length}`
}

const photoChunks = computed(() =>
  buildBookVirtualChunks(props.photos, chunkCapacity.value, photoChunkSizeKey),
)

/** 收窄 DynamicScroller 槽位类型，无效块渲染为空数组。 */
function isPhotoChunk(value: unknown): value is BookVirtualChunk<PhotoBook> {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    "items" in value &&
    "sizeKey" in value &&
    Array.isArray((value as BookVirtualChunk<PhotoBook>).items)
  )
}

/** 取出当前块；槽位异常时返回空块以免模板崩溃。 */
function getChunk(value: unknown): BookVirtualChunk<PhotoBook> {
  return isPhotoChunk(value)
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
      v-if="props.photos.length"
      data-photo-grid-scroller
      :items="photoChunks"
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
            data-virtual-photo-grid
            class="grid w-full overflow-x-hidden"
            :ref="(el) => onChunkGridRef(el)"
            :style="photoGridChunkStyle"
          >
            <div
              v-for="photo in getChunk(item).items"
              :key="photo.id"
              data-photo-card-shell
              class="flex min-w-0 justify-center"
            >
              <div
                data-photo-card-frame
                class="w-full min-w-0"
                :style="photoCardFrameStyle"
              >
                <PhotoCard
                  :photo="photo"
                  :poster-loading="posterLoadPolicyForChunk(index).loading"
                  :poster-fetch-priority="posterLoadPolicyForChunk(index).fetchPriority"
                  @open-details="emit('openDetails', $event)"
                  @open-viewer="(photoId, pageIndex) => emit('openViewer', photoId, pageIndex)"
                />
              </div>
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>

    <div
      v-else
      data-photo-grid-scroller
      class="h-full min-h-0 overflow-y-auto pr-2"
    >
      <div
        data-virtual-photo-grid
        class="grid w-full overflow-x-hidden"
        :style="photoGridChunkStyle"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useResizeObserver } from "@vueuse/core"
import { DynamicScroller, DynamicScrollerItem } from "vue-virtual-scroller"
import type { Movie } from "@/domain/movie/types"
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import MovieCard from "@/components/jav-library/MovieCard.vue"

interface MovieChunk {
  id: string
  items: Movie[]
  /** 稳定串，供虚拟列表测量缓存；避免每次渲染新建数组触发反复重算 */
  sizeKey: string
}

const props = withDefaults(
  defineProps<{
    movies: readonly Movie[]
    selectedMovieId?: string
    emptyTitle?: string
    emptyDescription?: string
  }>(),
  {
    emptyTitle: "No matches found",
    emptyDescription:
      "Try another query or switch to a different library tab.",
  },
)

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

/**
 * 虚拟块高度 = 固定行数 × 行高；块内卡片数 = 列数 × 行数，保证除最后一块外每块都是「整行满列」，
 * 避免固定 12 张/块在 8 列下变成 8+4 后下一块另起网格造成「阶梯断行」。
 */
const ROWS_PER_CHUNK = 4
/** 与 grid minmax 下限大致对齐（≈11.25rem），用于估算列数 */
const MIN_TRACK_PX = 188
/** 与 rowGap/columnGap 的 clamp 中间值接近 */
const GAP_PX_ESTIMATE = 20
/** 卡片略放大后虚拟行高略增 */
const ESTIMATED_CARD_HEIGHT = 300
const ESTIMATED_GAP = 22

/** 虚拟滚动缓冲区：上下各多渲染的块数，防止快速滚动白屏（与文档「上下各缓冲 5」对齐） */
const BUFFER_CHUNKS = 5
/** 虚拟滚动缓冲区像素：额外像素缓冲，与块数缓冲叠加使用 */
const BUFFER_PX = 600

const rootEl = ref<HTMLElement | null>(null)
const containerWidth = ref(typeof window !== "undefined" ? window.innerWidth : 1200)

useResizeObserver(rootEl, (entries) => {
  const w = entries[0]?.contentRect.width
  if (w != null && w > 0) {
    containerWidth.value = w
  }
})

const columnCountFallback = computed(() =>
  Math.max(1, Math.floor((containerWidth.value + GAP_PX_ESTIMATE) / (MIN_TRACK_PX + GAP_PX_ESTIMATE))),
)

/** 首块（或任意可见块）布局后从 getComputedStyle 读取，与 auto-fill 真实列数对齐 */
const measuredGridColumns = ref(0)

function parseGridColumnCount(el: HTMLElement): number {
  const raw = getComputedStyle(el).gridTemplateColumns
  if (!raw || raw === "none") return 0
  const trimmed = raw.trim()
  const repeatMatch = trimmed.match(/repeat\s*\(\s*(\d+)/i)
  if (repeatMatch) {
    return Math.max(1, parseInt(repeatMatch[1]!, 10))
  }
  const parts = trimmed.split(/\s+/).filter((p) => p.length > 0)
  return parts.length
}

function onChunkGridRef(el: unknown) {
  const node =
    el && typeof el === "object" && el !== null && "$el" in el
      ? (el as { $el: unknown }).$el
      : el
  if (!node || !(node instanceof HTMLElement)) return
  void nextTick(() => {
    const n = parseGridColumnCount(node)
    if (n > 0 && measuredGridColumns.value !== n) {
      measuredGridColumns.value = n
    }
  })
}

const effectiveColumnCount = computed(() =>
  measuredGridColumns.value > 0 ? measuredGridColumns.value : columnCountFallback.value,
)

const chunkCapacity = computed(() => Math.max(1, effectiveColumnCount.value * ROWS_PER_CHUNK))

const estimatedChunkHeight = ROWS_PER_CHUNK * (ESTIMATED_CARD_HEIGHT + ESTIMATED_GAP)

/** 网格列最小宽度：窄屏不过小、随视口变宽、大屏有上限 */
const GRID_COL_MIN = "clamp(11.25rem, 9vw + 8.75rem, 15rem)"
/** 单卡最大宽度：避免超宽屏单卡拉得过大；与列 min 解耦 */
const CARD_MAX_WIDTH = "min(100%, clamp(12rem, 28vw, 19rem))"
/** 间距随视口在区间内变化 */
const GRID_GAP = "clamp(1rem, 2vw, 1.5rem)"

function buildChunkSizeKey(items: Movie[]): string {
  if (items.length === 0) {
    return ""
  }
  let key = ""
  for (let i = 0; i < items.length; i++) {
    const m = items[i]!
    key += `${m.id}:${m.isFavorite ? 1 : 0}:${m.tags.length}:${m.userTags.length}:`
  }
  return key
}

const movieChunks = computed<MovieChunk[]>(() => {
  const { movies } = props
  const total = movies.length
  if (total === 0) {
    return []
  }
  const size = chunkCapacity.value
  const chunks: MovieChunk[] = []
  for (let offset = 0; offset < total; offset += size) {
    const items = movies.slice(offset, offset + size)
    chunks.push({
      id: `chunk-${offset}-${size}`,
      items,
      sizeKey: buildChunkSizeKey(items),
    })
  }
  return chunks
})

const isMovieChunk = (value: unknown): value is MovieChunk =>
  typeof value === "object" &&
  value !== null &&
  "id" in value &&
  "items" in value &&
  "sizeKey" in value &&
  Array.isArray((value as MovieChunk).items)

const getChunk = (value: unknown): MovieChunk =>
  isMovieChunk(value)
    ? value
    : {
        id: "invalid-chunk",
        items: [],
        sizeKey: "",
      }
</script>

<template>
  <div v-if="props.movies.length" ref="rootEl" class="h-full min-h-0">
    <DynamicScroller
      :items="movieChunks"
      key-field="id"
      :min-item-size="estimatedChunkHeight"
      :buffer="BUFFER_PX"
      :pool-size="BUFFER_CHUNKS * 2 + 5"
      class="h-full min-h-0 overflow-y-auto pr-2"
      list-class="flex flex-col gap-5"
      item-class="pb-5"
    >
      <template #default="{ item, index, active }">
        <DynamicScrollerItem
          :item="item"
          :active="active"
          :data-index="index"
          :size-dependencies="[getChunk(item).sizeKey]"
          :min-size="estimatedChunkHeight"
        >
          <!-- 列宽 min 用 clamp 响应式；1fr 吃满行；卡片区 max-width 限制单卡上限并居中 -->
          <div
            class="grid w-full"
            :ref="(el) => onChunkGridRef(el)"
            :style="{
              gridTemplateColumns: `repeat(auto-fill, minmax(min(100%, ${GRID_COL_MIN}), 1fr))`,
              columnGap: GRID_GAP,
              rowGap: GRID_GAP,
            }"
          >
            <div
              v-for="movie in getChunk(item).items"
              :key="movie.id"
              class="flex min-w-0 justify-center"
            >
              <div class="w-full" :style="{ maxWidth: CARD_MAX_WIDTH }">
              <MovieCard
                :movie="movie"
                :selected="movie.id === props.selectedMovieId"
                :show-favorite="false"
                @select="emit('select', $event)"
                @open-details="emit('openDetails', $event)"
                @open-player="emit('openPlayer', $event)"
                @toggle-favorite="emit('toggleFavorite', $event)"
              />
              </div>
            </div>
          </div>
        </DynamicScrollerItem>
      </template>
    </DynamicScroller>
  </div>

  <Card v-else class="rounded-3xl border-border/70 bg-card/80">
    <CardHeader>
      <CardTitle>{{ props.emptyTitle }}</CardTitle>
      <CardDescription>
        {{ props.emptyDescription }}
      </CardDescription>
    </CardHeader>
  </Card>
</template>

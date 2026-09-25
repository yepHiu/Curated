<script lang="ts">
/** Wait to distinguish a single click (details) from a double-click (player). */
export const MOVIE_CARD_OPEN_DETAILS_DELAY_MS = 300

let pendingOpenDetailsTimer: ReturnType<typeof setTimeout> | null = null
let pendingOpenDetailsMovieId = ""

function clearPendingOpenDetails(movieId?: string) {
  if (movieId && pendingOpenDetailsMovieId !== movieId) {
    return
  }
  if (pendingOpenDetailsTimer == null) {
    return
  }
  clearTimeout(pendingOpenDetailsTimer)
  pendingOpenDetailsTimer = null
  pendingOpenDetailsMovieId = ""
}

function scheduleOpenDetails(movieId: string, open: () => void) {
  clearPendingOpenDetails()
  pendingOpenDetailsMovieId = movieId
  pendingOpenDetailsTimer = setTimeout(() => {
    pendingOpenDetailsTimer = null
    pendingOpenDetailsMovieId = ""
    open()
  }, MOVIE_CARD_OPEN_DETAILS_DELAY_MS)
}
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Heart, Star } from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { Toggle } from "@/components/ui/toggle"
import MediaStill from "@/components/jav-library/MediaStill.vue"
import {
  getProgress,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import { isImageUrlLoaded } from "@/lib/image-load-cache"
import { buildVersionedImageUrl, getMovieImageVersion } from "@/lib/image-version"

const props = withDefaults(
  defineProps<{
    movie: Movie
    showFavorite?: boolean
    batchMode?: boolean
    batchChecked?: boolean
    posterLoading?: "lazy" | "eager"
    posterFetchPriority?: "high" | "low" | "auto"
  }>(),
  {
    batchMode: false,
    batchChecked: false,
    posterLoading: "eager",
    posterFetchPriority: "auto",
  },
)

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
  contextMenu: [event: MouseEvent]
  toggleBatchSelect: [payload: { movieId: string; shiftKey: boolean }]
}>()

const { t } = useI18n()

const MAX_CARD_TAGS = 3

type CardTagEntry = { text: string; source: "user" | "metadata" }

/**
 * 先铺用户标签（去重），未满 3 个再用 NFO/元数据标签补足（跳过已与已展示重复的标签）。
 * 槽位用尽后未展示的标签计入 overflow（含多出来的用户标签与元数据标签）。
 */
const cardTagsDisplay = computed(() => {
  const user = props.movie.userTags
  const meta = props.movie.tags
  let slots = MAX_CARD_TAGS
  const occupied = new Set<string>()
  const tags: CardTagEntry[] = []
  let overflow = 0

  for (const t of user) {
    if (slots <= 0) {
      overflow++
      continue
    }
    if (occupied.has(t)) continue
    occupied.add(t)
    tags.push({ text: t, source: "user" })
    slots--
  }
  for (const t of meta) {
    if (occupied.has(t)) continue
    if (slots <= 0) {
      overflow++
      continue
    }
    occupied.add(t)
    tags.push({ text: t, source: "metadata" })
    slots--
  }

  return { tags, overflow }
})

/** 列表有效分满档（0–5 制，含浮点） */
const isFullRating = computed(() => props.movie.rating >= 4.99)

/** 列表优先用缩略图，减轻带宽；无则回落封面 */
const posterSrc = computed(() => props.movie.thumbUrl || props.movie.coverUrl || "")

/** 图片版本号 - 用于强制刷新重新搜刮后的海报 */
const imageVersion = computed(() => getMovieImageVersion(props.movie.id))
const versionedPosterSrc = computed(() => buildVersionedImageUrl(posterSrc.value, imageVersion.value))
const javdbSearchUrl = computed(
  () => `https://javdb.com/search?q=${encodeURIComponent(props.movie.code.trim())}&f=all`,
)

/** 渐变叠层仅在海报解码完成后显示，避免盖住骨架屏（见 MediaStill 内 Skeleton） */
const posterImageLoaded = ref(false)
watch(
  () => `${props.movie.id}\0${versionedPosterSrc.value ?? ""}`,
  () => {
    posterImageLoaded.value = isImageUrlLoaded(versionedPosterSrc.value)
  },
  { immediate: true },
)

/** 与 PlaybackHistoryCard 相同算法；依赖 revision 以便播放器回写进度后卡片更新 */
const progressPercent = computed(() => {
  void playbackProgressRevision.value
  const row = getProgress(props.movie.id)
  if (!row) return null
  const dur = row.durationSec
  if (!dur || dur <= 0) return null
  return Math.min(100, Math.max(0, (row.positionSec / dur) * 100))
})

const fullStarBadgePositionClass = computed(() => {
  const hasFav = props.showFavorite !== false
  const hasProg = progressPercent.value != null
  const h = hasFav ? "right-[3.75rem]" : "right-2.5"
  const v = hasProg ? "bottom-3" : "bottom-2.5"
  return `${h} ${v}`
})

const handleCardClick = (event: MouseEvent) => {
  if (props.batchMode) {
    clearPendingOpenDetails(props.movie.id)
    emit("toggleBatchSelect", { movieId: props.movie.id, shiftKey: event.shiftKey })
    return
  }

  if (event.detail >= 2) {
    clearPendingOpenDetails()
    emit("openPlayer", props.movie.id)
    return
  }

  if (event.detail === 0) {
    clearPendingOpenDetails()
    emit("openDetails", props.movie.id)
    return
  }

  const movieId = props.movie.id
  scheduleOpenDetails(movieId, () => {
    emit("openDetails", movieId)
  })
}

watch(
  () => props.movie.id,
  (_id, previousId) => {
    clearPendingOpenDetails(previousId)
  },
)

onBeforeUnmount(() => {
  clearPendingOpenDetails(props.movie.id)
})

const handleFavoriteChange = (nextValue: boolean) => {
  emit("toggleFavorite", { movieId: props.movie.id, nextValue })
}
</script>

<template>
  <Card
    class="group relative gap-0 overflow-hidden rounded-[1.2rem] bg-card/80 py-0 shadow-md shadow-black/5 transition-[box-shadow,border-color] duration-150 motion-reduce:transition-none"
    :class="
      props.batchChecked
        ? 'border-2 border-primary shadow-lg shadow-primary/20'
        : 'border-0 hover:shadow-lg'
    "
    :data-movie-card-id="movie.id"
    :data-movie-card-selected="props.batchChecked ? 'true' : undefined"
  >
    <button
      type="button"
      class="flex w-full flex-col text-left select-none focus-visible:outline-none"
      :data-movie-batch-toggle="props.batchMode ? '' : undefined"
      :aria-pressed="props.batchMode ? props.batchChecked : undefined"
      :aria-label="props.batchMode ? t('library.batchCardToggleAria') : undefined"
      @click="handleCardClick"
      @dblclick.prevent
      @contextmenu.prevent="emit('contextMenu', $event)"
    >
      <div class="p-[var(--movie-card-padding)] pb-0">
        <div
          class="relative flex w-full items-start overflow-hidden rounded-[0.95rem] border border-border/60 aspect-[358/537]"
          :class="posterSrc ? 'bg-muted/30' : `bg-gradient-to-br p-2.5 ${movie.tone}`"
        >
          <MediaStill
            v-if="posterSrc"
            :src="posterSrc"
            :alt="movie.code"
            :version="imageVersion"
            class="absolute inset-0 z-[1]"
            :loading="props.posterLoading"
            :fetch-priority="props.posterFetchPriority"
            @load="posterImageLoaded = true"
            @error="posterImageLoaded = false"
          />
          <div
            v-if="posterImageLoaded"
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/50 via-transparent to-black/25"
            aria-hidden="true"
          />

          <div
            v-if="isFullRating"
            class="pointer-events-none absolute z-[2] flex items-center gap-0.5 rounded-full border border-primary/35 bg-background/88 px-1.5 py-0.5 text-[10px] font-medium tabular-nums text-primary shadow-sm backdrop-blur-sm"
            :class="fullStarBadgePositionClass"
            role="img"
            :aria-label="t('library.ratingFiveStarsBadgeAria')"
          >
            <span aria-hidden="true">5</span>
            <Star
              class="size-3 shrink-0 fill-primary text-primary"
              aria-hidden="true"
            />
          </div>

          <Toggle
            v-if="props.showFavorite !== false"
            :pressed="props.movie.isFavorite"
            variant="outline"
            size="sm"
            data-movie-favorite-toggle
            class="absolute right-2.5 bottom-2.5 z-[2] size-11 rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
            @update:pressed="handleFavoriteChange(Boolean($event))"
            @click.stop
          >
            <Heart />
          </Toggle>

          <div
            v-if="progressPercent != null"
            class="absolute right-0 bottom-0 left-0 z-[2] h-1 bg-black/50"
            aria-hidden="true"
          >
            <div
              class="h-full bg-primary transition-[width] duration-300 motion-reduce:transition-none"
              :style="{ width: `${progressPercent}%` }"
            />
          </div>
        </div>
      </div>

      <CardContent class="flex min-h-[var(--movie-card-body-min-height)] flex-col justify-between gap-[var(--movie-card-body-gap)] p-[var(--movie-card-padding)]">
        <div class="flex min-h-0 min-w-0 flex-col justify-start gap-0.5">
          <CardTitle class="truncate text-[13px]">{{ movie.title }}</CardTitle>
          <CardDescription class="truncate text-[11px]">
            {{ movie.actors.join(" · ") }}
          </CardDescription>
        </div>

        <!-- Badge 默认含 py-0.5 + 边框，高度常 > h-5；勿用固定矮行 + overflow-hidden 以免裁切 -->
        <div data-movie-tag-row class="flex min-h-6 min-w-0 items-center gap-1 overflow-hidden">
          <Badge
            v-for="(item, i) in cardTagsDisplay.tags"
            :key="`${i}-${item.text}`"
            :variant="item.source === 'user' ? 'outline' : 'secondary'"
            data-movie-card-tag
            :class="
              item.source === 'user'
                ? 'min-w-0 max-w-[4.75rem] shrink justify-start rounded-full border-primary/40 px-1.5 text-[10px] leading-tight text-primary'
                : 'min-w-0 max-w-[4.75rem] shrink justify-start rounded-full border border-border/60 bg-secondary/70 px-1.5 text-[10px] leading-tight'
            "
          >
            <span dir="ltr" class="block min-w-0 truncate text-left">{{ item.text }}</span>
          </Badge>
          <Badge
            v-if="cardTagsDisplay.overflow > 0"
            variant="outline"
            data-movie-tag-overflow
            class="shrink-0 rounded-full border-muted-foreground/35 px-1.5 text-[10px] leading-tight text-muted-foreground"
          >
            +{{ cardTagsDisplay.overflow }}
          </Badge>
        </div>
      </CardContent>
    </button>
    <div class="pointer-events-none absolute inset-x-0 top-0 z-[3] p-[var(--movie-card-padding)]">
      <Badge
        :as="props.batchMode ? 'span' : 'a'"
        variant="outline"
        :href="props.batchMode ? undefined : javdbSearchUrl"
        :target="props.batchMode ? undefined : '_blank'"
        :rel="props.batchMode ? undefined : 'noopener noreferrer'"
        data-movie-code-link
        class="m-[var(--movie-card-padding)] h-5 w-fit select-none truncate rounded-full border-border/40 bg-background/90 px-1.5 text-[10px] text-foreground shadow-sm backdrop-blur-sm focus-visible:ring-offset-2 focus-visible:ring-offset-background"
        :class="props.batchMode ? 'pointer-events-none' : 'pointer-events-auto hover:bg-background/95'"
      >
        {{ movie.code }}
      </Badge>
    </div>
  </Card>
</template>

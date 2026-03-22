<script setup lang="ts">
import { computed } from "vue"
import { Heart } from "lucide-vue-next"
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

const props = defineProps<{
  movie: Movie
  selected?: boolean
  showFavorite?: boolean
}>()

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

const visibleTags = computed(() => props.movie.tags.slice(0, 2))
const userTagCount = computed(() => props.movie.userTags.length)

/** 列表优先用缩略图，减轻带宽；无则回落封面 */
const posterSrc = computed(() => props.movie.thumbUrl || props.movie.coverUrl || "")

/** 与 PlaybackHistoryCard 相同算法；依赖 revision 以便播放器回写进度后卡片更新 */
const progressPercent = computed(() => {
  void playbackProgressRevision.value
  const row = getProgress(props.movie.id)
  if (!row) return null
  const dur = row.durationSec
  if (!dur || dur <= 0) return null
  return Math.min(100, Math.max(0, (row.positionSec / dur) * 100))
})

const handleOpenDetails = () => {
  emit("select", props.movie.id)
  emit("openDetails", props.movie.id)
}

const handleFavoriteChange = (nextValue: boolean) => {
  emit("toggleFavorite", { movieId: props.movie.id, nextValue })
}
</script>

<template>
  <Card
    class="group gap-0 overflow-hidden rounded-[1.2rem] border-border/70 bg-card/80 py-0 shadow-md shadow-black/5 transition-[box-shadow,border-color] duration-150 hover:border-primary/25 hover:shadow-lg motion-reduce:transition-none"
  >
    <button
      type="button"
      class="flex w-full flex-col text-left focus-visible:outline-none"
      @click="handleOpenDetails"
    >
      <div class="p-2.5 pb-0">
        <div
          class="relative flex w-full items-start overflow-hidden rounded-[0.95rem] border border-border/60 aspect-[358/537]"
          :class="posterSrc ? 'bg-muted/30' : `bg-gradient-to-br p-2.5 ${movie.tone}`"
        >
          <MediaStill
            v-if="posterSrc"
            :src="posterSrc"
            :alt="movie.code"
            class="absolute inset-0 z-0"
          />
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/50 via-transparent to-black/25"
            aria-hidden="true"
          />

          <Badge
            class="relative z-[2] m-2.5 h-5 w-fit rounded-full border border-border/40 bg-background/85 px-1.5 text-[10px] text-foreground shadow-sm backdrop-blur-sm"
          >
            {{ movie.code }}
          </Badge>

          <Toggle
            v-if="props.showFavorite !== false"
            :pressed="props.movie.isFavorite"
            variant="outline"
            size="sm"
            class="absolute right-2.5 bottom-2.5 z-[2] rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
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

      <CardContent class="flex min-h-[5.25rem] flex-col justify-between gap-1.5 p-2.5">
        <div class="flex min-h-0 min-w-0 flex-col justify-start gap-0.5">
          <CardTitle class="truncate text-[13px]">{{ movie.title }}</CardTitle>
          <CardDescription class="truncate text-[11px]">
            {{ movie.actors.join(" · ") }}
          </CardDescription>
        </div>

        <!-- Badge 默认含 py-0.5 + 边框，高度常 > h-5；勿用固定矮行 + overflow-hidden 以免裁切 -->
        <div class="flex min-h-6 items-center gap-1">
          <Badge
            v-for="tag in visibleTags"
            :key="tag"
            variant="secondary"
            class="max-w-[4.75rem] truncate rounded-full border border-border/60 bg-secondary/70 px-1.5 text-[10px] leading-tight"
          >
            {{ tag }}
          </Badge>
          <Badge
            v-if="userTagCount > 0"
            variant="outline"
            class="shrink-0 rounded-full border-primary/40 px-1.5 text-[10px] leading-tight text-primary"
          >
            我的 {{ userTagCount }}
          </Badge>
        </div>
      </CardContent>
    </button>
  </Card>
</template>

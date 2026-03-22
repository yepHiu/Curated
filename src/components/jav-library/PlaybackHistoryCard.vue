<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { Movie } from "@/domain/movie/types"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"

const { t } = useI18n()

const props = defineProps<{
  movie: Movie
  entry: PlaybackProgressEntry
}>()

const emit = defineEmits<{
  click: []
}>()

/** 宽版海报：优先封面图（常为整幅封套/横长图），缩略图多为竖版裁切 */
const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl)

const useWideCoverCrop = computed(() => Boolean(props.movie.coverUrl))

const progressPercent = computed(() => {
  const dur = props.entry.durationSec
  if (!dur || dur <= 0) return null
  return Math.min(100, Math.max(0, (props.entry.positionSec / dur) * 100))
})

const actorsLine = computed(() => {
  const list = props.movie.actors ?? []
  if (!list.length) return "—"
  return list.slice(0, 4).join(" · ")
})
</script>

<template>
  <button
    type="button"
    class="group w-full rounded-2xl border border-border/60 bg-card/90 p-3 text-left shadow-sm transition-[border-color,box-shadow] hover:border-primary/35 hover:shadow-md motion-reduce:transition-none"
    @click="emit('click')"
  >
    <div class="flex items-center gap-4">
      <div class="flex min-w-0 flex-1 flex-col justify-center gap-1.5 py-0.5 pr-1">
        <p class="line-clamp-2 text-sm font-semibold leading-snug sm:text-base">
          {{ movie.title }}
        </p>
        <p class="line-clamp-2 text-xs text-muted-foreground sm:text-sm">
          {{ actorsLine }}
        </p>
        <p class="text-[11px] text-muted-foreground/80 sm:text-xs">
          {{ movie.code }}
        </p>
      </div>

      <div
        class="relative aspect-[2.25/1] w-[min(46%,15.5rem)] shrink-0 overflow-hidden rounded-xl bg-muted sm:aspect-[2.4/1] sm:w-[min(44%,17rem)]"
        :title="t('curated.widePoster')"
      >
        <img
          v-if="posterSrc"
          :src="posterSrc"
          :alt="movie.title"
          class="size-full object-cover"
          :class="useWideCoverCrop ? 'object-[76%_center] sm:object-right' : 'object-center'"
          loading="lazy"
        />
        <div
          v-else
          class="flex size-full items-center justify-center bg-gradient-to-br from-primary/25 via-accent/30 to-card text-[10px] text-muted-foreground"
        >
          {{ t("common.noArt") }}
        </div>
        <div
          class="absolute right-0 bottom-0 left-0 h-1 bg-black/50"
          aria-hidden="true"
        >
          <div
            class="h-full bg-primary transition-[width] duration-300 motion-reduce:transition-none"
            :style="{ width: progressPercent != null ? `${progressPercent}%` : '0%' }"
          />
        </div>
      </div>
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { Movie } from "@/domain/movie/types"

const { t } = useI18n()

const props = defineProps<{
  movie: Movie
  current?: boolean
}>()

const emit = defineEmits<{
  select: []
}>()

const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl)
const useWideCoverCrop = computed(() => Boolean(props.movie.coverUrl))
const actorsLine = computed(() => {
  const list = props.movie.actors ?? []
  if (!list.length) return "—"
  return list.slice(0, 4).join(" · ")
})
</script>

<template>
  <button
    type="button"
    class="flex w-full overflow-hidden rounded-[1.1rem] border text-left transition-[border-color,background-color] duration-150 motion-reduce:transition-none"
    :class="
      props.current
        ? 'border-primary/45 bg-primary/10'
        : 'border-border/70 bg-card/80 hover:border-primary/25'
    "
    :data-player-playlist-item="movie.id"
    :aria-current="props.current ? 'true' : undefined"
    @click="emit('select')"
  >
    <div
      data-player-playlist-poster
      class="relative aspect-[2.25/1] w-[min(68%,19.5rem)] shrink-0 overflow-hidden bg-muted/30"
    >
      <img
        v-if="posterSrc"
        :src="posterSrc"
        :alt="movie.title"
        class="absolute inset-0 size-full object-cover"
        :class="useWideCoverCrop ? 'object-[76%_center] sm:object-right' : 'object-center'"
        loading="lazy"
      />
      <div
        v-else
        class="absolute inset-0 flex items-center justify-center bg-muted text-[10px] text-muted-foreground"
      >
        {{ t("common.noArt") }}
      </div>
    </div>
    <div class="flex min-w-0 flex-1 flex-col justify-center gap-1 px-3 py-2.5">
      <p class="line-clamp-2 text-sm font-semibold leading-snug text-foreground">
        {{ movie.title }}
      </p>
      <p class="truncate text-[11px] text-muted-foreground">
        {{ movie.code }}
      </p>
      <p class="line-clamp-2 text-xs text-muted-foreground">
        {{ actorsLine }}
      </p>
    </div>
  </button>
</template>

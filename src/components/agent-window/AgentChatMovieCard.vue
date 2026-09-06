<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { AIAgentMovieCardDTO } from "@/api/types"
import { resolveMediaUrl } from "@/api/media-url"

const props = defineProps<{
  movie: AIAgentMovieCardDTO
}>()

const emit = defineEmits<{
  open: [movieId: string]
}>()

const { t } = useI18n()

const posterSrc = computed(() => {
  const src = (props.movie.thumbUrl || props.movie.coverUrl || "").trim()
  return src ? resolveMediaUrl(src) : ""
})

const actorsLine = computed(() => {
  const list = props.movie.actors ?? []
  if (!list.length) return ""
  return list.slice(0, 3).join(" · ")
})
</script>

<template>
  <button
    type="button"
    class="flex min-h-11 w-full overflow-hidden rounded-xl border border-border bg-card text-left text-card-foreground transition-[border-color] duration-150 hover:border-primary/30 motion-reduce:transition-none"
    :data-agent-movie-card="movie.movieId"
    :aria-label="t('agentWindow.openMovie')"
    @click="emit('open', movie.movieId)"
  >
    <div class="relative aspect-[2/3] w-14 shrink-0 overflow-hidden bg-muted/40">
      <img
        v-if="posterSrc"
        :src="posterSrc"
        :alt="movie.title || movie.code || movie.movieId"
        class="absolute inset-0 size-full object-cover"
        loading="lazy"
        decoding="async"
        referrerpolicy="no-referrer"
      />
      <span
        v-else
        class="absolute inset-0 flex items-center justify-center px-1 text-center text-[9px] leading-tight text-muted-foreground"
      >
        {{ t("common.noArt") }}
      </span>
    </div>
    <div class="flex min-w-0 flex-1 flex-col justify-center gap-0.5 px-3 py-2">
      <p class="line-clamp-1 text-sm font-medium leading-snug">
        {{ movie.title || movie.code || movie.movieId }}
      </p>
      <p v-if="movie.code || actorsLine" class="truncate text-[11px] text-muted-foreground">
        {{ [movie.code, actorsLine].filter(Boolean).join(" · ") }}
      </p>
      <p v-if="movie.reason" class="line-clamp-2 text-xs text-muted-foreground">
        {{ movie.reason }}
      </p>
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { Button } from "@/components/ui/button"
import type { AIAgentMovieCardDTO } from "@/api/types"
import { agentConfirmCopy, savedViewFiltersFromConfirm } from "@/lib/agent-confirm-copy"
import { getProgress, playbackProgressRevision } from "@/lib/playback-progress-storage"
import { hasPlayedMovie, playedMovieCount } from "@/lib/played-movies-storage"
import {
  listMoviesMatchingSavedView,
  splitSavedViewConfirmMovies,
} from "@/lib/saved-view-preview"
import { useLibraryService } from "@/services/library-service"
import AgentChatMovieCard from "./AgentChatMovieCard.vue"
import type { AgentChatEntry } from "./types"

const props = defineProps<{
  entry: Extract<AgentChatEntry, { kind: "confirm" }>
}>()

const emit = defineEmits<{
  apply: []
  discard: []
  openMovie: [movieId: string]
}>()

const { t } = useI18n()
const libraryService = useLibraryService()
const moviesExpanded = ref(false)

const copy = computed(() =>
  agentConfirmCopy(props.entry, (key, values) => String(t(key, values as never))),
)

watch(
  () => props.entry.id,
  () => {
    moviesExpanded.value = false
  },
)

const matchingMovies = computed(() => {
  void playbackProgressRevision.value
  void playedMovieCount.value
  if (props.entry.name !== "create_saved_view") return []
  const filters = savedViewFiltersFromConfirm(props.entry)
  if (!filters) return []
  return listMoviesMatchingSavedView({
    movies: libraryService.movies.value,
    trashedMovies: libraryService.trashedMovies.value,
    filters,
    hasPlayedMovie,
    getProgress,
  })
})

const moviePreview = computed(() => splitSavedViewConfirmMovies(matchingMovies.value, moviesExpanded.value))

const previewCards = computed((): AIAgentMovieCardDTO[] =>
  moviePreview.value.visible.map((movie) => ({
    movieId: movie.id,
    title: movie.title,
    code: movie.code,
    actors: movie.actors,
    coverUrl: movie.coverUrl,
    thumbUrl: movie.thumbUrl,
  })),
)

function asText(value: unknown) {
  if (value == null) return ""
  if (typeof value === "object") {
    try {
      return JSON.stringify(value, null, 2)
    } catch {
      return ""
    }
  }
  return String(value)
}
</script>

<template>
  <div
    class="space-y-3 rounded-xl border border-border/60 bg-muted/30 px-3 py-3"
    data-agent-confirm-card
  >
    <p class="text-sm font-medium">{{ copy.title }}</p>
    <div
      v-if="copy.paragraphs.length"
      class="space-y-2 text-sm leading-relaxed"
      data-agent-confirm-narrative
    >
      <p v-for="(paragraph, index) in copy.paragraphs" :key="index">
        {{ paragraph }}
      </p>
    </div>
    <div
      v-if="props.entry.name === 'create_saved_view'"
      class="space-y-2"
      data-agent-confirm-movies
    >
      <p class="text-sm text-muted-foreground">
        {{
          moviePreview.total === 0
            ? t("agentWindow.confirmCreateViewMoviesEmpty")
            : t("agentWindow.confirmCreateViewMovies", { count: moviePreview.total })
        }}
      </p>
      <div v-if="previewCards.length" class="space-y-2">
        <AgentChatMovieCard
          v-for="movie in previewCards"
          :key="movie.movieId"
          :movie="movie"
          @open="emit('openMovie', $event)"
        />
      </div>
      <Button
        v-if="moviePreview.canExpand"
        type="button"
        variant="ghost"
        size="sm"
        data-agent-confirm-show-more
        @click="moviesExpanded = true"
      >
        {{ t("agentWindow.confirmCreateViewShowMore", { count: moviePreview.foldedCount }) }}
      </Button>
      <Button
        v-else-if="moviePreview.canCollapse"
        type="button"
        variant="ghost"
        size="sm"
        data-agent-confirm-show-less
        @click="moviesExpanded = false"
      >
        {{ t("agentWindow.confirmCreateViewShowLess") }}
      </Button>
      <p
        v-if="moviesExpanded && moviePreview.remainderAfterExpand > 0"
        class="text-xs text-muted-foreground"
        data-agent-confirm-movies-remainder
      >
        {{ t("agentWindow.confirmCreateViewMoreHidden", { count: moviePreview.remainderAfterExpand }) }}
      </p>
    </div>
    <template v-if="copy.showRawChanges">
      <div
        v-for="(change, index) in props.entry.changes"
        :key="`${change.path}-${index}`"
        class="grid gap-2 text-sm sm:grid-cols-2"
      >
        <div class="rounded-lg bg-background/80 p-2">
          <p class="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">
            {{ t("agentWindow.confirmBefore") }}
          </p>
          <p class="whitespace-pre-wrap text-muted-foreground">{{ asText(change.before) || t("agentWindow.confirmEmpty") }}</p>
        </div>
        <div class="rounded-lg bg-background p-2">
          <p class="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">
            {{ t("agentWindow.confirmAfter") }}
          </p>
          <p class="whitespace-pre-wrap">{{ asText(change.after) || t("agentWindow.confirmEmpty") }}</p>
        </div>
      </div>
    </template>
    <p v-if="props.entry.status === 'applied'" class="text-sm text-muted-foreground">
      {{ copy.appliedLabel }}
    </p>
    <p v-else-if="props.entry.status === 'discarded'" class="text-sm text-muted-foreground">
      {{ copy.discardedLabel }}
    </p>
    <p v-else-if="props.entry.error" class="text-sm text-destructive">{{ props.entry.error }}</p>
    <div v-else class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
      <Button
        type="button"
        variant="outline"
        :disabled="props.entry.status === 'applying'"
        data-agent-confirm-discard
        @click="emit('discard')"
      >
        {{ t("agentWindow.confirmDiscard") }}
      </Button>
      <Button
        type="button"
        :disabled="props.entry.status === 'applying'"
        data-agent-confirm-apply
        @click="emit('apply')"
      >
        {{ copy.applyLabel }}
      </Button>
    </div>
  </div>
</template>

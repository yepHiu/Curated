<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import PlaybackHistoryCard from "@/components/jav-library/PlaybackHistoryCard.vue"
import type { Movie } from "@/domain/movie/types"
import type { HomepageContinueEntry } from "@/lib/homepage-portal"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"

const props = defineProps<{
  entries: HomepageContinueEntry[]
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()

const { t } = useI18n()

interface ContinueCardRow {
  movie: Movie
  movieId: string
  progressEntry: PlaybackProgressEntry
}

function buildPlaybackEntry(entry: HomepageContinueEntry): PlaybackProgressEntry {
  const remainingSec = Math.max(60, entry.remainingMinutes * 60)
  const progressRatio = Math.min(0.94, Math.max(0.01, entry.progressPercent / 100))
  const estimatedDuration = Math.max(
    remainingSec + 1,
    Math.round(remainingSec / Math.max(0.01, 1 - progressRatio)),
  )
  const positionSec = Math.max(1, estimatedDuration - remainingSec)

  return {
    movieId: entry.movie.id,
    positionSec,
    durationSec: estimatedDuration,
    updatedAt: entry.updatedAt,
  }
}

const cards = computed<ContinueCardRow[]>(() =>
  props.entries.map((entry) => ({
    movie: entry.movie,
    movieId: entry.movie.id,
    progressEntry: buildPlaybackEntry(entry),
  })),
)
</script>

<template>
  <section class="space-y-4">
    <div class="space-y-1">
      <h2 class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
        {{ t("home.sectionContinueTitle") }}
      </h2>
      <p class="text-sm text-muted-foreground">
        {{ t("home.sectionContinueBody") }}
      </p>
    </div>

    <div class="grid gap-3 lg:grid-cols-2">
      <article
        v-for="entry in cards"
        :key="entry.movieId"
        class="contents"
      >
        <PlaybackHistoryCard
          :movie="entry.movie"
          :entry="entry.progressEntry"
          :show-remove-action="false"
          @click="emit('openPlayer', entry.movieId)"
        />
      </article>
    </div>
  </section>
</template>

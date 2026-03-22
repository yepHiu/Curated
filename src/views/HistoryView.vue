<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { RouterLink, useRouter } from "vue-router"
import PlaybackHistoryCard from "@/components/jav-library/PlaybackHistoryCard.vue"
import { Button } from "@/components/ui/button"
import type { Movie } from "@/domain/movie/types"
import { groupPlaybackRowsByLocalDay } from "@/lib/playback-history-groups"
import { buildPlayerRouteFromHistory } from "@/lib/player-route"
import type { PlaybackProgressEntry } from "@/lib/playback-progress-storage"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"

const { t, locale } = useI18n()
const router = useRouter()
const libraryService = useLibraryService()

interface HistoryRow {
  entry: PlaybackProgressEntry
  movie: Movie
  updatedAt: string
}

const historyRows = computed((): HistoryRow[] => {
  void playbackProgressRevision.value
  const sorted = listSortedByUpdatedDesc()
  const out: HistoryRow[] = []
  for (const entry of sorted) {
    const movie = libraryService.getMovieById(entry.movieId)
    if (!movie) continue
    out.push({ entry, movie, updatedAt: entry.updatedAt })
  }
  return out
})

const dayBuckets = computed(() => {
  const loc = locale.value
  return groupPlaybackRowsByLocalDay(historyRows.value, {
    locale: loc,
    labels: {
      today: t("history.today"),
      yesterday: t("history.yesterday"),
    },
  })
})

const isEmpty = computed(() => historyRows.value.length === 0)

async function openFromHistory(row: HistoryRow) {
  const pos = Math.max(0, Math.floor(row.entry.positionSec))
  await router.push(buildPlayerRouteFromHistory(row.movie.id, pos))
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-y-auto pb-6">
    <div
      class="mx-auto flex w-full max-w-4xl flex-col gap-6 px-3 sm:px-6 lg:px-8"
    >
    <header class="flex flex-col gap-1">
      <h1 class="text-2xl font-semibold tracking-tight">{{ t("history.title") }}</h1>
      <p class="text-sm text-muted-foreground">
        {{ t("history.subtitle") }}
      </p>
    </header>

    <div
      v-if="isEmpty"
      class="flex flex-col items-center justify-center gap-4 rounded-3xl border border-dashed border-border/70 bg-card/50 px-6 py-16 text-center"
    >
      <p class="max-w-sm text-sm text-muted-foreground">
        {{ t("history.empty") }}
      </p>
      <Button as-child variant="secondary" class="rounded-2xl">
        <RouterLink :to="{ name: 'library' }">{{ t("history.goLibrary") }}</RouterLink>
      </Button>
    </div>

    <template v-else>
      <section
        v-for="bucket in dayBuckets"
        :key="bucket.dayKey"
        class="flex flex-col gap-3"
      >
        <h2
          class="sticky top-0 z-[1] bg-background/90 px-1 py-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase backdrop-blur-sm"
        >
          {{ bucket.label }}
        </h2>
        <!-- 单列瀑布流：按时间自上而下逐条排列，卡片占满内容区宽度 -->
        <div class="flex w-full flex-col gap-4">
          <PlaybackHistoryCard
            v-for="row in bucket.rows"
            :key="row.entry.movieId + row.entry.updatedAt"
            :movie="row.movie"
            :entry="row.entry"
            @click="openFromHistory(row)"
          />
        </div>
      </section>
    </template>
    </div>
  </div>
</template>

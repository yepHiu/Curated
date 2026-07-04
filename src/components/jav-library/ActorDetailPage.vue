<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import ActorProfileCard from "@/components/jav-library/ActorProfileCard.vue"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"
import { pushAppToast } from "@/composables/use-app-toast"
import type { Movie } from "@/domain/movie/types"
import { getSelectedMovieQuery } from "@/lib/library-query"
import {
  buildDetailRouteFromActor,
  buildPlayerRouteFromActorIntent,
} from "@/lib/navigation-intent"
import { useLibraryService } from "@/services/library-service"

const props = defineProps<{
  actorName: string
}>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const actorDisplayName = computed(() => props.actorName.trim())

const actorMovies = computed(() => {
  const actor = actorDisplayName.value
  if (!actor) {
    return [] as Movie[]
  }
  return libraryService.movies.value.filter((movie) => movie.actors.includes(actor))
})

const selectedMovieId = computed(() => {
  const selected = getSelectedMovieQuery(route.query)
  if (!selected) {
    return undefined
  }
  return actorMovies.value.some((movie) => movie.id === selected) ? selected : undefined
})

const scrollPreserveKey = computed(() =>
  actorDisplayName.value ? `actor-detail:${actorDisplayName.value}` : "actor-detail",
)

async function selectMovie(movieId?: string) {
  const id = movieId?.trim()
  const actor = actorDisplayName.value
  if (!id || !actor) {
    return
  }
  await router.replace({
    name: "actor-detail",
    params: { actorName: actor },
    query: {
      ...route.query,
      selected: id,
    },
  })
}

async function openDetails(movieId?: string) {
  const id = movieId?.trim()
  const actor = actorDisplayName.value
  if (!id || !actor) {
    return
  }
  await router.push(buildDetailRouteFromActor(id, actor))
}

async function openPlayer(movieId?: string) {
  const id = movieId?.trim() || actorMovies.value[0]?.id
  const actor = actorDisplayName.value
  if (!id || !actor) {
    return
  }
  await router.push(buildPlayerRouteFromActorIntent(id, actor))
}

async function toggleFavorite(payload: { movieId: string; nextValue: boolean }) {
  try {
    await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
  } catch (err) {
    pushAppToast(t("library.favoriteToggleFailed"), { variant: "destructive" })
    console.error("[ActorDetailPage] toggle favorite failed", err)
  }
}
</script>

<template>
  <div
    data-actor-detail-page
    class="flex h-full min-h-0 min-w-0 flex-col gap-4 overflow-hidden px-[var(--app-page-px)] py-[var(--app-page-py)] sm:px-[var(--app-page-px-sm)] lg:px-[var(--app-page-px-lg)] lg:py-[var(--app-page-py-lg)] xl:px-[var(--app-page-px-xl)]"
  >
    <header class="flex shrink-0 flex-col gap-1">
      <p class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
        {{ t("actors.detailEyebrow") }}
      </p>
      <h1 class="truncate text-2xl font-semibold tracking-tight sm:text-3xl">
        {{ actorDisplayName }}
      </h1>
      <p class="text-sm text-muted-foreground">
        {{ t("actors.detailSubtitle") }}
      </p>
    </header>

    <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
      <ActorProfileCard
        :actor-name="actorDisplayName"
        :show-clear-filter="false"
      />

      <section class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
        <div class="flex shrink-0 flex-wrap items-end justify-between gap-2">
          <h2 class="text-lg font-semibold tracking-tight">
            {{ t("actors.detailMovieSection") }}
          </h2>
          <p class="text-sm text-muted-foreground">
            {{ t("actors.movieCount", { n: actorMovies.length }) }}
          </p>
        </div>

        <div class="min-h-0 flex-1 overflow-hidden">
          <VirtualMovieMasonry
            :movies="actorMovies"
            :selected-movie-id="selectedMovieId"
            :empty-title="t('actors.detailEmptyTitle')"
            :empty-description="t('actors.detailEmptyDesc')"
            :scroll-preserve-key="scrollPreserveKey"
            @select="selectMovie"
            @open-details="openDetails"
            @open-player="openPlayer"
            @toggle-favorite="toggleFavorite"
          />
        </div>
      </section>
    </div>
  </div>
</template>

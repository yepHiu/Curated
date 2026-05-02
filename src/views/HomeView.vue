<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useHomepageDailyRecommendations } from "@/composables/use-homepage-daily-recommendations"
import { armHomeDetailReturnRestore } from "@/composables/use-home-scroll-preserve"
import { useRouter } from "vue-router"
import HomepageEmptyState from "@/components/jav-library/HomepageEmptyState.vue"
import HomepagePortal from "@/components/jav-library/HomepagePortal.vue"
import HomepagePortalSkeleton from "@/components/jav-library/HomepagePortalSkeleton.vue"
import type { HomepageTasteEntry } from "@/lib/homepage-portal"
import { buildHomepagePortalModel } from "@/lib/homepage-portal"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"

const libraryService = useLibraryService()
const router = useRouter()
const homepageDailyRecommendations = useHomepageDailyRecommendations()
const showHomepageSkeleton = computed(() => !libraryService.moviesLoaded.value)
const showHomepageEmptyState = computed(
  () => libraryService.moviesLoaded.value && libraryService.movies.value.length === 0,
)

const PORTAL_PLAYBACK_PROGRESS_DEBOUNCE_MS = 5_000
const portalPlaybackProgressRevision = ref(playbackProgressRevision.value)
let portalPlaybackProgressTimer: ReturnType<typeof setTimeout> | null = null

watch(playbackProgressRevision, (revision) => {
  if (portalPlaybackProgressTimer) {
    clearTimeout(portalPlaybackProgressTimer)
  }
  portalPlaybackProgressTimer = setTimeout(() => {
    portalPlaybackProgressTimer = null
    portalPlaybackProgressRevision.value = revision
  }, PORTAL_PLAYBACK_PROGRESS_DEBOUNCE_MS)
})

onBeforeUnmount(() => {
  if (portalPlaybackProgressTimer) {
    clearTimeout(portalPlaybackProgressTimer)
    portalPlaybackProgressTimer = null
  }
})

const portalModel = computed(() => {
  void portalPlaybackProgressRevision.value

  return buildHomepagePortalModel({
    movies: libraryService.movies.value,
    playbackEntries: listSortedByUpdatedDesc(),
    dailyRecommendations: homepageDailyRecommendations.snapshot.value ?? undefined,
    heroLimit: 8,
  })
})

function openDetails(movieId: string) {
  armHomeDetailReturnRestore()
  void router.push({
    name: "detail",
    params: { id: movieId },
    query: { back: "home" },
  })
}

function openPlayer(movieId: string) {
  void router.push({
    name: "player",
    params: { id: movieId },
    query: {
      autoplay: "1",
      back: "home",
    },
  })
}

function browseTaste(payload: { kind: HomepageTasteEntry["kind"]; label: string }) {
  const label = payload.label.trim()
  if (!label) return

  if (payload.kind === "tag") {
    void router.push({
      name: "tags",
      query: { tag: label },
    })
    return
  }

  void router.push({
    name: "library",
    query: payload.kind === "actor" ? { actor: label } : { studio: label },
  })
}

function refreshRecommendations() {
  void homepageDailyRecommendations.refreshRecommendationsOnly({
    preserveHeroMovieIds: portalModel.value.heroMovies.map((movie) => movie.id),
    excludeRecommendationMovieIds: portalModel.value.recommendations.map((entry) => entry.movie.id),
  })
}
</script>

<template>
  <HomepagePortalSkeleton v-if="showHomepageSkeleton" />
  <HomepageEmptyState v-else-if="showHomepageEmptyState" />
  <HomepagePortal
    v-else
    :model="portalModel"
    :recommendations-refreshing="homepageDailyRecommendations.loading.value"
    @open-details="openDetails"
    @open-player="openPlayer"
    @browse-taste="browseTaste"
    @refresh-recommendations="refreshRecommendations"
  />
</template>

<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { CreateRecommendationFeedbackBody, RecommendationFeedbackDTO } from "@/api/types"
import { useHomepageDailyRecommendations } from "@/composables/use-homepage-daily-recommendations"
import { armHomeDetailReturnRestore } from "@/composables/use-home-scroll-preserve"
import { useRouter } from "vue-router"
import HomepageEmptyState from "@/components/jav-library/HomepageEmptyState.vue"
import HomepagePortal from "@/components/jav-library/HomepagePortal.vue"
import HomepagePortalSkeleton from "@/components/jav-library/HomepagePortalSkeleton.vue"
import { buildHomepagePortalModel } from "@/lib/homepage-portal"
import { openLibraryFromHomeKey } from "@/lib/home-library-navigation"
import {
  listSortedByUpdatedDesc,
  playbackProgressRevision,
} from "@/lib/playback-progress-storage"
import { useLibraryService } from "@/services/library-service"
import { pushAppToast } from "@/composables/use-app-toast"

const libraryService = useLibraryService()
const router = useRouter()
const openLibraryFromHome = inject(openLibraryFromHomeKey, () => {
  void router.push({ name: "library" })
})
const { t } = useI18n()
const homepageDailyRecommendations = useHomepageDailyRecommendations()
const showHomepageSkeleton = computed(() => !libraryService.moviesLoaded.value)
const showHomepageEmptyState = computed(
  () => libraryService.moviesLoaded.value && libraryService.movies.value.length === 0,
)

const PORTAL_PLAYBACK_PROGRESS_DEBOUNCE_MS = 5_000
const portalPlaybackProgressRevision = ref(playbackProgressRevision.value)
let portalPlaybackProgressTimer: ReturnType<typeof setTimeout> | null = null
const recommendationFeedback = ref<RecommendationFeedbackDTO[]>([])
const recommendationFeedbackBusy = ref(false)

async function refreshRecommendationFeedback() {
  try {
    recommendationFeedback.value = (
      await libraryService.listHomepageRecommendationFeedback()
    ).items
  } catch (error) {
    pushAppToast(error instanceof Error ? error.message : t("home.recommendationFeedbackFailed"), {
      variant: "destructive",
    })
  }
}

onMounted(() => {
  void refreshRecommendationFeedback()
})

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

function refreshRecommendations() {
  void homepageDailyRecommendations.refreshRecommendationsOnly({
    preserveHeroMovieIds: portalModel.value.heroMovies.map((movie) => movie.id),
    excludeRecommendationMovieIds: portalModel.value.recommendations.map((entry) => entry.movie.id),
  })
}

async function submitRecommendationFeedback(body: CreateRecommendationFeedbackBody) {
  if (recommendationFeedbackBusy.value) return
  recommendationFeedbackBusy.value = true
  try {
    await libraryService.createHomepageRecommendationFeedback(body)
    await refreshRecommendationFeedback()
    pushAppToast(t("home.recommendationFeedbackSaved"), { variant: "success" })
  } catch (error) {
    pushAppToast(error instanceof Error ? error.message : t("home.recommendationFeedbackFailed"), {
      variant: "destructive",
    })
  } finally {
    recommendationFeedbackBusy.value = false
  }
}

async function deleteRecommendationFeedback(feedbackId: string) {
  if (recommendationFeedbackBusy.value) return
  recommendationFeedbackBusy.value = true
  try {
    await libraryService.deleteHomepageRecommendationFeedback(feedbackId)
    await refreshRecommendationFeedback()
    pushAppToast(t("home.recommendationFeedbackRemoved"), { variant: "success" })
  } catch (error) {
    pushAppToast(error instanceof Error ? error.message : t("home.recommendationFeedbackFailed"), {
      variant: "destructive",
    })
  } finally {
    recommendationFeedbackBusy.value = false
  }
}
</script>

<template>
  <HomepagePortalSkeleton v-if="showHomepageSkeleton" />
  <HomepageEmptyState v-else-if="showHomepageEmptyState" />
  <HomepagePortal
    v-else
    :model="portalModel"
    :recommendations-refreshing="homepageDailyRecommendations.loading.value"
    :recommendation-feedback="recommendationFeedback"
    :recommendation-feedback-busy="recommendationFeedbackBusy"
    @open-details="openDetails"
    @open-player="openPlayer"
    @browse-library="openLibraryFromHome"
    @refresh-recommendations="refreshRecommendations"
    @submit-recommendation-feedback="submitRecommendationFeedback"
    @delete-recommendation-feedback="deleteRecommendationFeedback"
  />
</template>

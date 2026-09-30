<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { RotateCcw } from "lucide-vue-next"
import type { RecommendationFeedbackDTO } from "@/api/types"
import type { HomepageRecommendationEntry } from "@/lib/homepage-portal"
import MovieCard from "@/components/jav-library/MovieCard.vue"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  entry: HomepageRecommendationEntry
  feedback: RecommendationFeedbackDTO[]
  busy?: boolean
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  deleteFeedback: [feedbackId: string]
}>()

const { t } = useI18n()
const movie = computed(() => props.entry.movie)
const activeFeedback = computed(() =>
  props.feedback.find((item) => item.sourceMovieId === movie.value.id),
)
</script>

<template>
  <article data-home-recommendation-card class="flex min-w-0 flex-col gap-2">
    <MovieCard
      :movie="movie"
      :show-favorite="false"
      poster-loading="lazy"
      poster-fetch-priority="low"
      @open-details="emit('openDetails', $event)"
      @open-player="emit('openPlayer', $event)"
    />

    <div
      v-if="activeFeedback"
      data-home-recommendation-meta
      class="flex min-w-0 items-center justify-end gap-1.5 px-0.5"
    >
      <div data-home-recommendation-actions class="flex shrink-0 items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="min-h-11 sm:min-h-8"
          :disabled="busy"
          @click="emit('deleteFeedback', activeFeedback.id)"
        >
          <RotateCcw data-icon="inline-start" aria-hidden="true" />
          {{ t("home.recommendationUndo") }}
        </Button>
      </div>
    </div>
  </article>
</template>

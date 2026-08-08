<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Clock3, MoreHorizontal, RotateCcw, ThumbsDown, UserRound, Tags } from "lucide-vue-next"
import type {
  CreateRecommendationFeedbackBody,
  RecommendationFeedbackDTO,
} from "@/api/types"
import type { HomepageRecommendationEntry } from "@/lib/homepage-portal"
import MovieCard from "@/components/jav-library/MovieCard.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

const props = defineProps<{
  entry: HomepageRecommendationEntry
  feedback: RecommendationFeedbackDTO[]
  busy?: boolean
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  submitFeedback: [body: CreateRecommendationFeedbackBody]
  deleteFeedback: [feedbackId: string]
}>()

const { t } = useI18n()
const movie = computed(() => props.entry.movie)
const tags = computed(() => [...new Set([...movie.value.userTags, ...movie.value.tags])].slice(0, 5))
const activeFeedback = computed(() =>
  props.feedback.find((item) => item.sourceMovieId === movie.value.id),
)

function submit(
  action: CreateRecommendationFeedbackBody["action"],
  targetType: CreateRecommendationFeedbackBody["targetType"],
  targetValue: string,
  durationDays?: number,
) {
  emit("submitFeedback", {
    action,
    targetType,
    targetValue,
    sourceMovieId: movie.value.id,
    ...(durationDays === undefined ? {} : { durationDays }),
  })
}

function reasonLabel(index: number) {
  const reason = props.entry.reasons[index]
  return reason
    ? t(`home.recommendationReason.${reason.code}`, { value: reason.entityValue ?? "" })
    : ""
}
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
      data-home-recommendation-meta
      class="flex min-w-0 items-center gap-1.5 px-0.5"
    >
      <div
        data-home-recommendation-tags
        class="flex min-h-6 min-w-0 flex-1 flex-wrap gap-1.5"
      >
        <Badge
          v-for="(_, index) in entry.reasons.slice(0, 2)"
          :key="`${movie.id}-${index}`"
          variant="secondary"
          class="max-w-full truncate"
        >
          {{ reasonLabel(index) }}
        </Badge>
      </div>

      <div data-home-recommendation-actions class="flex shrink-0 items-center gap-1">
        <Button
          v-if="activeFeedback"
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

        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              class="min-h-11 min-w-11 sm:min-h-8 sm:min-w-8"
              :disabled="busy"
              :aria-label="t('home.recommendationAdjust')"
            >
              <MoreHorizontal data-icon="inline-start" aria-hidden="true" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" class="w-64 rounded-xl">
            <DropdownMenuLabel>{{ t("home.recommendationAdjust") }}</DropdownMenuLabel>
            <DropdownMenuGroup>
              <DropdownMenuItem @click="submit('not_interested', 'movie', movie.id)">
                <ThumbsDown aria-hidden="true" />
                {{ t("home.recommendationNotInterested") }}
              </DropdownMenuItem>
              <DropdownMenuSub>
                <DropdownMenuSubTrigger>
                  <Clock3 aria-hidden="true" />
                  {{ t("home.recommendationSnooze") }}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent class="w-48 rounded-xl">
                  <DropdownMenuGroup>
                    <DropdownMenuItem @click="submit('snooze', 'movie', movie.id, 7)">
                      {{ t("home.recommendationSnoozeDays", { days: 7 }) }}
                    </DropdownMenuItem>
                    <DropdownMenuItem @click="submit('snooze', 'movie', movie.id, 30)">
                      {{ t("home.recommendationSnoozeDays", { days: 30 }) }}
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            </DropdownMenuGroup>

            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuSub v-if="movie.actors.length > 0">
                <DropdownMenuSubTrigger>
                  <UserRound aria-hidden="true" />
                  {{ t("home.recommendationLessActor") }}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent class="w-52 rounded-xl">
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      v-for="actor in movie.actors.slice(0, 5)"
                      :key="actor"
                      @click="submit('less', 'actor', actor)"
                    >
                      {{ actor }}
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                </DropdownMenuSubContent>
              </DropdownMenuSub>
              <DropdownMenuItem
                v-if="movie.studio.trim()"
                @click="submit('less', 'studio', movie.studio)"
              >
                {{ t("home.recommendationLessStudio", { value: movie.studio }) }}
              </DropdownMenuItem>
              <DropdownMenuSub v-if="tags.length > 0">
                <DropdownMenuSubTrigger>
                  <Tags aria-hidden="true" />
                  {{ t("home.recommendationLessTag") }}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent class="w-52 rounded-xl">
                  <DropdownMenuGroup>
                    <DropdownMenuItem
                      v-for="tag in tags"
                      :key="tag"
                      @click="submit('less', 'tag', tag)"
                    >
                      {{ tag }}
                    </DropdownMenuItem>
                  </DropdownMenuGroup>
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  </article>
</template>

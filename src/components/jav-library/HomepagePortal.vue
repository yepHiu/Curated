<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import { Loader2, RefreshCw, SlidersHorizontal, Trash2 } from "lucide-vue-next"
import type { CreateRecommendationFeedbackBody, RecommendationFeedbackDTO } from "@/api/types"
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from "reka-ui"
import type { HomepagePortalModel } from "@/lib/homepage-portal"
import HomeContinueRow from "@/components/jav-library/HomeContinueRow.vue"
import HomeHeroCarousel from "@/components/jav-library/HomeHeroCarousel.vue"
import HomeSectionRow from "@/components/jav-library/HomeSectionRow.vue"
import HomeRecommendationCard from "@/components/jav-library/HomeRecommendationCard.vue"
import { useHomeScrollPreserve } from "@/composables/use-home-scroll-preserve"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

const props = defineProps<{
  model: HomepagePortalModel
  recommendationsRefreshing?: boolean
  recommendationFeedback?: RecommendationFeedbackDTO[]
  recommendationFeedbackBusy?: boolean
}>()

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  refreshRecommendations: []
  submitRecommendationFeedback: [body: CreateRecommendationFeedbackBody]
  deleteRecommendationFeedback: [feedbackId: string]
}>()

const { t } = useI18n()
const homeScrollRegionRef = ref<HTMLElement | null>(null)
const feedbackDialogOpen = ref(false)
const { persist } = useHomeScrollPreserve({ scrollElRef: homeScrollRegionRef })

const recommendationsRefreshLabel = computed(() =>
  props.recommendationsRefreshing
    ? t("home.refreshingRecommendations")
    : t("home.refreshRecommendations"),
)

const recommendationMovies = computed(() =>
  props.model.recommendations.map((entry) => entry.movie),
)
const recommendationByMovieID = computed(() =>
  new Map(props.model.recommendations.map((entry) => [entry.movie.id, entry] as const)),
)

function feedbackActionLabel(item: RecommendationFeedbackDTO) {
  return t(`home.recommendationFeedbackAction.${item.action}`, {
    value: item.targetValue,
  })
}

function onHomeScroll() {
  persist()
}
</script>

<template>
  <div
    ref="homeScrollRegionRef"
    data-home-scroll-region
    class="h-full min-h-0 overflow-y-auto bg-background text-foreground"
    @scroll.passive="onHomeScroll"
  >
    <HomeHeroCarousel
      :movies="model.heroMovies"
      @open-details="emit('openDetails', $event)"
      @open-player="emit('openPlayer', $event)"
    />

    <div
      class="mx-auto flex w-full max-w-[1680px] flex-col gap-8 px-4 py-6 sm:px-5 lg:gap-10 lg:px-6 lg:py-8 xl:px-8"
    >
      <HomeSectionRow
        :title="t('home.sectionRecentTitle')"
        :movies="model.recentMovies"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />

      <HomeSectionRow
        :title="t('home.sectionRecommendTitle')"
        :movies="recommendationMovies"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      >
        <template #item="{ movie }">
          <HomeRecommendationCard
            v-if="recommendationByMovieID.get(movie.id)"
            :entry="recommendationByMovieID.get(movie.id)!"
            :feedback="recommendationFeedback ?? []"
            :busy="recommendationFeedbackBusy"
            @open-details="emit('openDetails', $event)"
            @open-player="emit('openPlayer', $event)"
            @submit-feedback="emit('submitRecommendationFeedback', $event)"
            @delete-feedback="emit('deleteRecommendationFeedback', $event)"
          />
        </template>
        <template #action>
          <div class="flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              class="min-h-11 min-w-11 sm:min-h-8 sm:min-w-8"
              :aria-label="t('home.recommendationManage')"
              @click="feedbackDialogOpen = true"
            >
              <SlidersHorizontal data-icon="inline-start" aria-hidden="true" />
            </Button>
            <TooltipProvider :delay-duration="240">
              <TooltipRoot>
                <TooltipTrigger as-child>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    class="min-h-11 min-w-11 sm:min-h-8 sm:min-w-8"
                    :disabled="recommendationsRefreshing"
                    :aria-label="recommendationsRefreshLabel"
                    data-home-refresh-recommendations
                    @click="emit('refreshRecommendations')"
                  >
                    <Loader2
                      v-if="recommendationsRefreshing"
                      data-icon="inline-start"
                      class="motion-safe:animate-spin"
                      style="animation-duration: 4s"
                      aria-hidden="true"
                    />
                    <RefreshCw
                      v-else
                      data-icon="inline-start"
                      aria-hidden="true"
                    />
                  </Button>
                </TooltipTrigger>
                <TooltipPortal>
                  <TooltipContent
                    side="top"
                    :side-offset="6"
                    class="rounded-md border border-border bg-popover px-2 py-1 text-xs text-popover-foreground shadow-md"
                  >
                    {{ recommendationsRefreshLabel }}
                  </TooltipContent>
                </TooltipPortal>
              </TooltipRoot>
            </TooltipProvider>
          </div>
        </template>
      </HomeSectionRow>

      <HomeContinueRow
        v-if="model.continueWatching.length > 0"
        :entries="model.continueWatching"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
      />

    </div>
  </div>

  <Dialog v-model:open="feedbackDialogOpen">
    <DialogContent class="max-h-[min(80dvh,42rem)] overflow-y-auto sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t("home.recommendationManageTitle") }}</DialogTitle>
        <DialogDescription>{{ t("home.recommendationManageDescription") }}</DialogDescription>
      </DialogHeader>

      <div v-if="(recommendationFeedback ?? []).length > 0" class="flex flex-col gap-2">
        <div
          v-for="item in recommendationFeedback"
          :key="item.id"
          class="flex min-w-0 items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/35 p-3"
        >
          <div class="min-w-0">
            <p class="truncate text-sm font-medium text-foreground">
              {{ feedbackActionLabel(item) }}
            </p>
            <p class="truncate text-xs text-muted-foreground">
              {{ item.targetValue }}
            </p>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            class="min-h-11 min-w-11 shrink-0 sm:min-h-8 sm:min-w-8"
            :disabled="recommendationFeedbackBusy"
            :aria-label="t('home.recommendationRemoveFeedback', { value: item.targetValue })"
            @click="emit('deleteRecommendationFeedback', item.id)"
          >
            <Trash2 data-icon="inline-start" aria-hidden="true" />
          </Button>
        </div>
      </div>
      <p v-else class="text-sm text-muted-foreground">
        {{ t("home.recommendationFeedbackEmpty") }}
      </p>
    </DialogContent>
  </Dialog>
</template>

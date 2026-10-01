<script setup lang="ts">
import { computed, ref, useId, watch } from "vue"
import { usePreferredReducedMotion } from "@vueuse/core"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { GitMerge, Images } from "lucide-vue-next"
import ActorMergeDialog from "@/components/jav-library/ActorMergeDialog.vue"
import ActorProfileCard from "@/components/jav-library/ActorProfileCard.vue"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"
import CuratedFramesSection from "@/components/jav-library/CuratedFramesSection.vue"
import { Button } from "@/components/ui/button"
import { pushAppToast } from "@/composables/use-app-toast"
import type { Movie } from "@/domain/movie/types"
import { compareByReleaseDateDesc } from "@/lib/movie-sort"
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

const resolvedActorName = ref(props.actorName.trim())
const mergeDialogOpen = ref(false)
const profileRevision = ref(0)
const actorDisplayName = computed(() => resolvedActorName.value || props.actorName.trim())
const framesRegion = ref<HTMLElement | null>(null)
const framesRegionId = useId()
const reducedMotion = usePreferredReducedMotion()

function jumpToFrames() {
  const region = framesRegion.value
  if (!region) return
  region.focus({ preventScroll: true })
  region.scrollIntoView({ block: "start", behavior: reducedMotion.value === "reduce" ? "instant" : "smooth" })
}

watch(
  () => props.actorName,
  (name) => {
    resolvedActorName.value = name.trim()
  },
)

async function onActorNameResolved(name: string) {
  const canonical = name.trim()
  if (!canonical) return
  resolvedActorName.value = canonical
  if (canonical !== props.actorName.trim()) {
    await router.replace({
      name: "actor-detail",
      params: { actorName: canonical },
      query: route.query,
    })
  }
}

async function onActorMerged(targetName: string) {
  mergeDialogOpen.value = false
  resolvedActorName.value = targetName.trim()
  profileRevision.value += 1
  await router.replace({
    name: "actor-detail",
    params: { actorName: targetName.trim() },
    query: route.query,
  })
}

const actorMovies = computed(() => {
  const actor = actorDisplayName.value
  if (!actor) {
    return [] as Movie[]
  }
  return libraryService.movies.value
    .filter((movie) => movie.actors.includes(actor))
    .sort(compareByReleaseDateDesc)
})

const scrollPreserveKey = computed(() =>
  actorDisplayName.value ? `actor-detail:${actorDisplayName.value}` : "actor-detail",
)

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
    <h1 class="sr-only">{{ actorDisplayName }}</h1>

    <div class="min-h-0 flex-1 overflow-hidden">
      <VirtualMovieMasonry
        :movies="actorMovies"
        :empty-title="t('mediaEmpty.title')"
        :empty-description="t('actors.detailEmptyDesc')"
        :scroll-preserve-key="scrollPreserveKey"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
      >
        <template #header>
          <div class="space-y-4">
            <ActorProfileCard
              :key="profileRevision"
              :actor-name="actorDisplayName"
              :show-clear-filter="false"
              @resolved-name="onActorNameResolved"
            >
              <template #actions>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="min-h-11 rounded-full sm:min-h-8"
                  :aria-controls="framesRegionId"
                  @click="jumpToFrames"
                >
                  <Images data-icon="inline-start" aria-hidden="true" />
                  {{ t("curated.title") }}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  class="min-h-11 rounded-full sm:min-h-8"
                  @click="mergeDialogOpen = true"
                >
                  <GitMerge data-icon="inline-start" aria-hidden="true" />
                  {{ t("actors.merge.openAction") }}
                </Button>
              </template>
            </ActorProfileCard>

            <div class="flex shrink-0 flex-wrap items-end justify-between gap-2">
              <h2 class="text-lg font-semibold tracking-tight">
                {{ t("actors.detailMovieSection") }}
              </h2>
              <p class="text-sm text-muted-foreground">
                {{ t("actors.movieCount", { n: actorMovies.length }) }}
              </p>
            </div>
          </div>
        </template>
        <template #footer>
          <section
            :id="framesRegionId"
            ref="framesRegion"
            data-actor-frames-region
            tabindex="-1"
            :aria-label="t('curated.title')"
            class="scroll-mt-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <CuratedFramesSection :key="profileRevision" :actor-name="actorDisplayName" unframed />
          </section>
        </template>
      </VirtualMovieMasonry>
    </div>

    <ActorMergeDialog
      v-model:open="mergeDialogOpen"
      :source-name="actorDisplayName"
      @merged="onActorMerged"
    />
  </div>
</template>

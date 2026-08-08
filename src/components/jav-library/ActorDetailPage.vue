<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"
import { GitMerge } from "lucide-vue-next"
import ActorMergeDialog from "@/components/jav-library/ActorMergeDialog.vue"
import ActorProfileCard from "@/components/jav-library/ActorProfileCard.vue"
import VirtualMovieMasonry from "@/components/jav-library/VirtualMovieMasonry.vue"
import { Button } from "@/components/ui/button"
import { pushAppToast } from "@/composables/use-app-toast"
import type { Movie } from "@/domain/movie/types"
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
const actorDisplayName = computed(() => resolvedActorName.value || props.actorName.trim())

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
  return libraryService.movies.value.filter((movie) => movie.actors.includes(actor))
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
    <header class="flex shrink-0 flex-col gap-1">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div class="min-w-0">
          <p class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            {{ t("actors.detailEyebrow") }}
          </p>
          <h1 class="truncate text-2xl font-semibold tracking-tight sm:text-3xl">
            {{ actorDisplayName }}
          </h1>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="min-h-11 sm:min-h-8"
          @click="mergeDialogOpen = true"
        >
          <GitMerge data-icon="inline-start" aria-hidden="true" />
          {{ t("actors.merge.openAction") }}
        </Button>
      </div>
      <p class="text-sm text-muted-foreground">
        {{ t("actors.detailSubtitle") }}
      </p>
    </header>

    <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden">
      <ActorProfileCard
        :actor-name="actorDisplayName"
        :show-clear-filter="false"
        @resolved-name="onActorNameResolved"
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
            :empty-title="t('actors.detailEmptyTitle')"
            :empty-description="t('actors.detailEmptyDesc')"
            :scroll-preserve-key="scrollPreserveKey"
            @open-details="openDetails"
            @open-player="openPlayer"
            @toggle-favorite="toggleFavorite"
          />
        </div>
      </section>
    </div>

    <ActorMergeDialog
      v-model:open="mergeDialogOpen"
      :source-name="actorDisplayName"
      @merged="onActorMerged"
    />
  </div>
</template>

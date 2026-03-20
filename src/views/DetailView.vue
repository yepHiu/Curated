<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { HttpClientError } from "@/api/http-client"
import DetailPage from "@/components/jav-library/DetailPage.vue"
import NotFoundState from "@/components/jav-library/NotFoundState.vue"
import { buildMovieRouteQuery, getBrowseSourceMode, mergeLibraryQuery } from "@/lib/library-query"
import { loadMovieDetail } from "@/services/adapters/web/web-library-service"
import { useLibraryService } from "@/services/library-service"
import type { Movie } from "@/domain/movie/types"

const USE_WEB_API = import.meta.env.VITE_USE_WEB_API === "true"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const movieId = computed(() =>
  typeof route.params.id === "string" ? route.params.id : undefined,
)

const detailMovie = shallowRef<Movie | undefined>()
const detailLoading = ref(false)
const deleteBusy = ref(false)
const deleteError = ref("")

watch(
  () => movieId.value,
  async (id) => {
    detailMovie.value = undefined
    if (!id) {
      return
    }
    if (USE_WEB_API) {
      detailLoading.value = true
      try {
        detailMovie.value =
          (await loadMovieDetail(id)) ?? libraryService.getMovieById(id)
      } finally {
        detailLoading.value = false
      }
    } else {
      detailMovie.value = libraryService.getMovieById(id)
    }
  },
  { immediate: true },
)

const relatedMovies = computed(() =>
  detailMovie.value ? libraryService.getRelatedMovies(detailMovie.value.id) : [],
)

const selectMovie = async (nextMovieId: string) => {
  await router.replace({
    name: "detail",
    params: { id: nextMovieId },
  })
}

const openDetails = async (nextMovieId: string) => {
  await router.push({
    name: "detail",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const openPlayer = async (nextMovieId: string) => {
  await router.push({
    name: "player",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, getBrowseSourceMode(route.query), nextMovieId),
  })
}

const toggleFavorite = (payload: { movieId: string; nextValue: boolean }) => {
  libraryService.toggleFavorite(payload.movieId, payload.nextValue)
}

const handleDeleteMovie = async (id: string) => {
  deleteError.value = ""
  deleteBusy.value = true
  try {
    await libraryService.deleteMovie(id)
    await router.replace({
      name: getBrowseSourceMode(route.query),
      query: mergeLibraryQuery(route.query, { selected: undefined }),
    })
  } catch (err) {
    const message =
      err instanceof HttpClientError
        ? (err.apiError?.message ?? err.message)
        : err instanceof Error
          ? err.message
          : "删除失败"
    deleteError.value = message
    console.error("[DetailView] delete movie failed", err)
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <div class="h-full overflow-y-auto pr-2">
    <div
      v-if="detailLoading"
      class="rounded-3xl border border-border/70 bg-card/80 p-8 text-sm text-muted-foreground"
    >
      正在加载详情与预览图…
    </div>
    <template v-else-if="detailMovie">
      <p
        v-if="deleteBusy"
        class="mb-3 rounded-2xl border border-border/70 bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
      >
        正在删除影片…
      </p>
      <p
        v-if="deleteError"
        class="mb-3 rounded-2xl border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive"
      >
        {{ deleteError }}
      </p>
      <DetailPage
        :movie="detailMovie"
        :related-movies="relatedMovies"
        @select="selectMovie"
        @open-details="openDetails"
        @open-player="openPlayer"
        @toggle-favorite="toggleFavorite"
        @delete-movie="handleDeleteMovie"
      />
    </template>
    <NotFoundState
      v-else
      title="未找到影片"
      description="当前库中不存在该条目，或详情接口暂时不可用。"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import LibraryPage from "@/components/jav-library/LibraryPage.vue"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import {
  buildMovieRouteQuery,
  getLibrarySearchQuery,
  getLibraryTabQuery,
  getSelectedMovieQuery,
  isLibraryRouteName,
  mergeLibraryQuery,
} from "@/lib/library-query"
import { isMovieRecentlyAdded } from "@/lib/library-stats"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const libraryMode = computed<LibraryMode>(() => {
  return isLibraryRouteName(route.name) ? route.name : "library"
})

const libraryMovies = computed(() => libraryService.movies.value)
const searchQuery = computed(() => getLibrarySearchQuery(route.query))
const activeTab = computed<LibraryTab>(() => getLibraryTabQuery(route.query))
const selectedMovieId = computed(() => getSelectedMovieQuery(route.query))

const queryFilteredMovies = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  let result = [...libraryMovies.value]

  if (libraryMode.value === "favorites") {
    result = result.filter((movie) => movie.isFavorite)
  }

  if (libraryMode.value === "recent") {
    result = result
      .filter((movie) => isMovieRecentlyAdded(movie.addedAt))
      .sort((left, right) => right.addedAt.localeCompare(left.addedAt))
  }

  if (libraryMode.value === "tags") {
    result = result.sort((left, right) => left.tags.join("").localeCompare(right.tags.join("")))
  }

  if (!query) {
    return result
  }

  return result.filter((movie) =>
    [movie.title, movie.code, movie.studio, movie.actors.join(" "), movie.tags.join(" ")]
      .join(" ")
      .toLowerCase()
      .includes(query),
  )
})

const visibleMovies = computed(() => {
  switch (activeTab.value) {
    case "new":
      return queryFilteredMovies.value.filter((movie) => movie.year >= 2025)
    case "favorites":
      return queryFilteredMovies.value.filter((movie) => movie.isFavorite)
    case "top-rated":
      return queryFilteredMovies.value.filter((movie) => movie.rating >= 4.6)
    default:
      return queryFilteredMovies.value
  }
})

const selectedMovie = computed(() => {
  if (selectedMovieId.value) {
    const routeMovie = visibleMovies.value.find((movie) => movie.id === selectedMovieId.value)
    if (routeMovie) {
      return routeMovie
    }
  }

  return visibleMovies.value[0] ?? undefined
})

const replaceQuery = async (
  nextQuery: Partial<Record<"q" | "tab" | "selected" | "from", string | undefined>>,
) => {
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, nextQuery),
  })
}

watch(
  [selectedMovie, () => route.query.selected],
  ([movie, currentSelected]) => {
    const nextSelected = movie?.id
    const normalizedSelected = typeof currentSelected === "string" ? currentSelected : undefined

    if (nextSelected === normalizedSelected) {
      return
    }

    void replaceQuery({
      selected: nextSelected,
    })
  },
  { immediate: true },
)

const updateActiveTab = async (value: LibraryTab) => {
  await replaceQuery({
    tab: value,
    selected: selectedMovie.value?.id ?? visibleMovies.value[0]?.id,
  })
}

const selectMovie = async (movieId: string) => {
  await replaceQuery({
    selected: movieId,
  })
}

const openDetails = async (movieId: string) => {
  await router.push({
    name: "detail",
    params: { id: movieId },
    query: buildMovieRouteQuery(route.query, libraryMode.value, movieId),
  })
}

const openPlayer = async (movieId?: string) => {
  const nextMovieId = movieId ?? selectedMovie.value?.id

  if (!nextMovieId) {
    return
  }

  await router.push({
    name: "player",
    params: { id: nextMovieId },
    query: buildMovieRouteQuery(route.query, libraryMode.value, nextMovieId),
  })
}

const toggleFavorite = (payload: { movieId: string; nextValue: boolean }) => {
  libraryService.toggleFavorite(payload.movieId, payload.nextValue)
}
</script>

<template>
  <LibraryPage
    :mode="libraryMode"
    :all-movies="libraryMovies"
    :visible-movies="visibleMovies"
    :selected-movie="selectedMovie"
    :active-tab="activeTab"
    @update:active-tab="updateActiveTab"
    @select="selectMovie"
    @open-details="openDetails"
    @open-player="openPlayer"
    @toggle-favorite="toggleFavorite"
  />
</template>

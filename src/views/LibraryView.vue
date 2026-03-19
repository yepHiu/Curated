<script setup lang="ts">
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"
import LibraryPage from "@/components/jav-library/LibraryPage.vue"
import type { LibraryMode, LibraryTab } from "@/lib/jav-library"
import { getMovieById, movies } from "@/lib/jav-library"

const route = useRoute()
const router = useRouter()

const validTabs: LibraryTab[] = ["all", "new", "favorites", "top-rated"]

const libraryMode = computed<LibraryMode>(() => {
  switch (route.name) {
    case "favorites":
      return "favorites"
    case "recent":
      return "recent"
    case "tags":
      return "tags"
    default:
      return "library"
  }
})

const searchQuery = computed(() =>
  typeof route.query.q === "string" ? route.query.q : "",
)

const activeTab = computed<LibraryTab>(() => {
  const value = typeof route.query.tab === "string" ? route.query.tab : "all"
  return validTabs.includes(value as LibraryTab) ? (value as LibraryTab) : "all"
})

const selectedMovieId = computed(() =>
  typeof route.query.selected === "string" ? route.query.selected : movies[0].id,
)

const queryFilteredMovies = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  let result = [...movies]

  if (libraryMode.value === "favorites") {
    result = result.filter((movie) => movie.isFavorite)
  }

  if (libraryMode.value === "recent") {
    result = result.sort((left, right) => right.addedAt.localeCompare(left.addedAt))
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
  const routeMovie = visibleMovies.value.find((movie) => movie.id === selectedMovieId.value)
  return routeMovie ?? visibleMovies.value[0] ?? getMovieById(selectedMovieId.value)
})

const replaceQuery = async (nextQuery: Record<string, string | undefined>) => {
  const mergedQuery = {
    ...route.query,
    ...nextQuery,
  }

  await router.replace({
    name: route.name ?? "library",
    query: mergedQuery,
  })
}

const updateActiveTab = async (value: LibraryTab) => {
  await replaceQuery({
    tab: value === "all" ? undefined : value,
    selected: selectedMovie.value.id,
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
  })
}

const openPlayer = async (movieId?: string) => {
  await router.push({
    name: "player",
    params: { id: movieId ?? selectedMovie.value.id },
  })
}
</script>

<template>
  <LibraryPage
    :mode="libraryMode"
    :all-movies="movies"
    :visible-movies="visibleMovies"
    :selected-movie="selectedMovie"
    :active-tab="activeTab"
    @update:active-tab="updateActiveTab"
    @select="selectMovie"
    @open-details="openDetails"
    @open-player="openPlayer"
  />
</template>

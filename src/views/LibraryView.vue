<script setup lang="ts">
import { computed, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import LibraryPage from "@/components/jav-library/LibraryPage.vue"
import type { LibraryMode, LibraryTab } from "@/domain/library/types"
import type { Movie } from "@/domain/movie/types"
import {
  buildMovieRouteQuery,
  getLibraryActorExactQuery,
  getLibrarySearchQuery,
  getLibraryTabQuery,
  getLibraryTagExactQuery,
  getSelectedMovieQuery,
  isLibraryRouteName,
  mergeLibraryQuery,
} from "@/lib/library-query"
import { buildPlayerRouteFromBrowse } from "@/lib/player-route"
import { isMovieRecentlyAdded } from "@/lib/library-stats"
import { movieSearchHaystack } from "@/lib/movie-search"
import { compareByReleaseDateDesc } from "@/lib/movie-sort"
import { useLibraryService } from "@/services/library-service"

const route = useRoute()
const router = useRouter()
const libraryService = useLibraryService()

const libraryMode = computed<LibraryMode>(() => {
  return isLibraryRouteName(route.name) ? route.name : "library"
})

const libraryMovies = computed(() => libraryService.movies.value)
const searchQuery = computed(() => getLibrarySearchQuery(route.query))
const tagExactQuery = computed(() => getLibraryTagExactQuery(route.query).trim())
const actorExactQuery = computed(() => getLibraryActorExactQuery(route.query).trim())
const activeTab = computed<LibraryTab>(() => getLibraryTabQuery(route.query))
const selectedMovieId = computed(() => getSelectedMovieQuery(route.query))

const queryFilteredMovies = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  const mode = libraryMode.value
  const raw = libraryMovies.value

  let list: Movie[]
  if (mode === "favorites") {
    list = raw.filter((movie) => movie.isFavorite)
  } else if (mode === "recent") {
    list = raw
      .filter((movie) => isMovieRecentlyAdded(movie.addedAt))
      .slice()
      .sort((left, right) => right.addedAt.localeCompare(left.addedAt))
  } else if (mode === "tags") {
    list = raw.slice().sort((left, right) => left.tags.join("").localeCompare(right.tags.join("")))
  } else {
    list = [...raw]
  }

  if (query) {
    list = list.filter((movie) => movieSearchHaystack(movie).includes(query))
  }

  const tagExact = tagExactQuery.value
  if (tagExact) {
    list = list.filter(
      (movie) => movie.tags.includes(tagExact) || movie.userTags.includes(tagExact),
    )
  }

  const actorExact = actorExactQuery.value
  if (actorExact) {
    list = list.filter((movie) => movie.actors.includes(actorExact))
  }

  return list
})

const visibleMovies = computed(() => {
  switch (activeTab.value) {
    case "new":
      return queryFilteredMovies.value.slice().sort(compareByReleaseDateDesc)
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

  await router.push(buildPlayerRouteFromBrowse(nextMovieId, route.query, libraryMode.value))
}

const toggleFavorite = async (payload: { movieId: string; nextValue: boolean }) => {
  try {
    await libraryService.toggleFavorite(payload.movieId, payload.nextValue)
  } catch (err) {
    console.error("[LibraryView] toggle favorite failed", err)
  }
}

/** Tags 页标签云：与详情 `browseByTag` 一致 */
const browseByExactTag = async (tag: string) => {
  const t = tag.trim()
  if (!t) return
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, {
      tag: t,
      q: undefined,
      actor: undefined,
      tab: "all",
      selected: undefined,
    }),
  })
}

const clearExactTagFilter = async () => {
  await router.replace({
    name: libraryMode.value,
    query: mergeLibraryQuery(route.query, { tag: undefined }),
  })
}
</script>

<template>
  <LibraryPage
    :mode="libraryMode"
    :all-movies="libraryMovies"
    :visible-movies="visibleMovies"
    :selected-movie="selectedMovie"
    :active-tab="activeTab"
    :active-tag-filter="tagExactQuery"
    @update:active-tab="updateActiveTab"
    @select="selectMovie"
    @open-details="openDetails"
    @open-player="openPlayer"
    @toggle-favorite="toggleFavorite"
    @browse-by-exact-tag="browseByExactTag"
    @clear-exact-tag-filter="clearExactTagFilter"
  />
</template>

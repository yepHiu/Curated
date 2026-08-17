import { computed, ref, watch, type ComputedRef, type Ref } from "vue"
import { useRoute } from "vue-router"
import type { Movie } from "@/domain/movie/types"
import {
  findPlaylistIndex,
  listPlayerPlaylistMovies,
  readPlaylistAutoAdvance,
  recenterPlaylistWindow,
  resolvePlayerPlaylistSource,
  slidePlaylistWindow,
  writePlaylistAutoAdvance,
  type PlaylistWindow,
} from "@/lib/player-playlist"
import { getProgress, playbackProgressRevision } from "@/lib/playback-progress-storage"
import { hasPlayedMovie, playedMovieCount } from "@/lib/played-movies-storage"
import { useLibraryService } from "@/services/library-service"

const sharedPanelOpen = ref(false)

export function usePlayerPlaylist(movieId: ComputedRef<string>) {
  const route = useRoute()
  const libraryService = useLibraryService()
  const panelOpen = sharedPanelOpen
  const autoAdvance = ref(readPlaylistAutoAdvance())
  const windowRange = ref<PlaylistWindow>({ start: 0, end: -1 })

  const source = computed(() => resolvePlayerPlaylistSource(route.query))

  const items = computed(() => {
    void playbackProgressRevision.value
    void playedMovieCount.value
    return listPlayerPlaylistMovies({
      source: source.value,
      movies: libraryService.movies.value,
      trashedMovies: libraryService.trashedMovies.value,
      query: route.query,
      hasPlayedMovie,
      getProgress,
    })
  })

  const currentIndex = computed(() => findPlaylistIndex(items.value, movieId.value))
  const active = computed(() => currentIndex.value >= 0)
  const current = computed(() => (active.value ? items.value[currentIndex.value] : undefined))
  const previous = computed(() =>
    active.value && currentIndex.value > 0 ? items.value[currentIndex.value - 1] : undefined,
  )
  const next = computed(() =>
    active.value && currentIndex.value < items.value.length - 1
      ? items.value[currentIndex.value + 1]
      : undefined,
  )

  const visibleItems = computed(() => {
    if (!active.value) {
      return [] as Movie[]
    }
    const { start, end } = windowRange.value
    if (end < start) {
      return [] as Movie[]
    }
    return items.value.slice(start, end + 1)
  })

  watch(
    [currentIndex, () => items.value.length],
    ([index, total]) => {
      windowRange.value = recenterPlaylistWindow(index, total)
      if (index < 0) {
        panelOpen.value = false
      }
    },
    { immediate: true },
  )

  function setAutoAdvance(enabled: boolean) {
    autoAdvance.value = enabled
    writePlaylistAutoAdvance(enabled)
  }

  function openPanel() {
    if (!active.value) {
      return
    }
    panelOpen.value = true
  }

  function closePanel() {
    panelOpen.value = false
  }

  function slideWindow(direction: "up" | "down") {
    if (!active.value) {
      return
    }
    windowRange.value = slidePlaylistWindow(
      windowRange.value,
      direction,
      currentIndex.value,
      items.value.length,
    )
  }

  return {
    active,
    source,
    items,
    visibleItems,
    currentIndex,
    current,
    previous,
    next,
    panelOpen: panelOpen as Ref<boolean>,
    autoAdvance: autoAdvance as Ref<boolean>,
    windowStart: computed(() => windowRange.value.start),
    hiddenBeforeCount: computed(() => Math.max(0, windowRange.value.start)),
    hiddenAfterCount: computed(() =>
      active.value ? Math.max(0, items.value.length - 1 - windowRange.value.end) : 0,
    ),
    setAutoAdvance,
    openPanel,
    closePanel,
    slideWindow,
  }
}

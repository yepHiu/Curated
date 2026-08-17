<script setup lang="ts">
import { nextTick, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { Movie } from "@/domain/movie/types"
import PlayerPlaylistCard from "@/components/jav-library/PlayerPlaylistCard.vue"
import PlayerPlaylistRevealTab from "@/components/jav-library/PlayerPlaylistRevealTab.vue"
import { Switch } from "@/components/ui/switch"

const { t } = useI18n()

const props = defineProps<{
  movies: readonly Movie[]
  currentMovieId: string
  autoAdvance: boolean
  hiddenBeforeCount: number
  hiddenAfterCount: number
  totalCount: number
  currentIndex: number
}>()

const emit = defineEmits<{
  close: []
  select: [movieId: string]
  "update:autoAdvance": [value: boolean]
  "slide-window": [direction: "up" | "down"]
}>()

const scrollerRef = ref<HTMLElement | null>(null)
const slidingWindow = ref(false)

watch(
  () => props.currentMovieId,
  async () => {
    await nextTick()
    const current = scrollerRef.value?.querySelector("[aria-current='true']")
    if (current instanceof HTMLElement) {
      current.scrollIntoView({ block: "nearest" })
    }
  },
  { immediate: true },
)

function captureScrollAnchor(root: HTMLElement) {
  const cards = Array.from(root.querySelectorAll("[data-player-playlist-item]"))
  const visible = cards.find((node) => {
    if (!(node instanceof HTMLElement)) {
      return false
    }
    return node.offsetTop + node.offsetHeight > root.scrollTop
  })
  const anchor = visible instanceof HTMLElement ? visible : null
  if (!anchor) {
    return null
  }
  return {
    id: anchor.dataset.playerPlaylistItem ?? "",
    offset: anchor.offsetTop - root.scrollTop,
  }
}

function restoreScrollAnchor(
  root: HTMLElement,
  anchor: { id: string; offset: number } | null,
) {
  if (!anchor?.id) {
    return
  }
  const next = root.querySelector(`[data-player-playlist-item="${anchor.id}"]`)
  if (!(next instanceof HTMLElement)) {
    return
  }
  root.scrollTop = next.offsetTop - anchor.offset
}

async function extendWindow(direction: "up" | "down") {
  if (slidingWindow.value) {
    return
  }
  slidingWindow.value = true
  try {
    for (let step = 0; step < 8; step += 1) {
      const root = scrollerRef.value
      if (!root) {
        return
      }
      if (direction === "up") {
        if (props.hiddenBeforeCount <= 0 || root.scrollTop >= 48) {
          return
        }
      } else if (
        props.hiddenAfterCount <= 0 ||
        root.scrollHeight - root.scrollTop - root.clientHeight >= 48
      ) {
        return
      }

      const beforeStart = props.movies[0]?.id
      const beforeEnd = props.movies[props.movies.length - 1]?.id
      const anchor = captureScrollAnchor(root)
      emit("slide-window", direction)
      await nextTick()
      restoreScrollAnchor(root, anchor)

      if (
        beforeStart === props.movies[0]?.id &&
        beforeEnd === props.movies[props.movies.length - 1]?.id
      ) {
        return
      }
    }
  } finally {
    slidingWindow.value = false
  }
}

function onScroll() {
  const root = scrollerRef.value
  if (!root || slidingWindow.value) {
    return
  }
  if (root.scrollTop < 48 && props.hiddenBeforeCount > 0) {
    void extendWindow("up")
    return
  }
  const remaining = root.scrollHeight - root.scrollTop - root.clientHeight
  if (remaining < 48 && props.hiddenAfterCount > 0) {
    void extendWindow("down")
  }
}
</script>

<template>
  <aside
    data-player-playlist-panel
    class="absolute inset-y-0 right-0 z-40 flex h-full w-[min(28rem,92vw)] flex-col border-l border-border bg-background text-foreground shadow-[-16px_0_40px_rgba(0,0,0,0.35)] sm:w-[min(28rem,46%)]"
    @click.stop
  >
    <PlayerPlaylistRevealTab
      expanded
      class="absolute top-1/2 left-0 z-10 -translate-x-full -translate-y-1/2"
      @close="emit('close')"
    />

    <div class="flex shrink-0 items-center gap-2 border-b border-border px-3 py-2.5">
      <p class="min-w-0 flex-1 truncate text-sm font-semibold">
        {{ t("player.playlistTitle") }}
      </p>
      <span class="shrink-0 text-xs tabular-nums text-muted-foreground">
        {{ currentIndex + 1 }}/{{ totalCount }}
      </span>
      <label class="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
        {{ t("player.playlistAutoAdvance") }}
        <Switch
          data-player-playlist-auto-advance
          :model-value="autoAdvance"
          :aria-label="t('player.playlistAutoAdvanceAria')"
          @update:model-value="emit('update:autoAdvance', $event === true)"
        />
      </label>
    </div>

    <div
      ref="scrollerRef"
      class="min-h-0 flex-1 space-y-2 overflow-y-auto p-2"
      @scroll.passive="onScroll"
    >
      <PlayerPlaylistCard
        v-for="movie in movies"
        :key="movie.id"
        :movie="movie"
        :current="movie.id === currentMovieId"
        @select="emit('select', movie.id)"
      />
    </div>
  </aside>
</template>

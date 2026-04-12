<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import type { Movie } from "@/domain/movie/types"
import MediaStill from "@/components/jav-library/MediaStill.vue"

const props = withDefaults(
  defineProps<{
    movies: Movie[]
    autoplayMs?: number
    autoplayTransitionMs?: number
    manualTransitionMs?: number
  }>(),
  {
    autoplayMs: 6000,
    autoplayTransitionMs: 760,
    manualTransitionMs: 420,
  },
)

const emit = defineEmits<{
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
}>()

const { t } = useI18n()

const activeIndex = ref(0)
const activeTrackIndex = ref(0)
const trackTransitionEnabled = ref(true)
const trackTransitionMs = ref(props.autoplayTransitionMs)
let timer: ReturnType<typeof setInterval> | undefined
let snapTimer: ReturnType<typeof setTimeout> | undefined

const safeMovies = computed(() => props.movies.filter((movie) => movie))
const total = computed(() => safeMovies.value.length)
const displaySlides = computed(() => {
  const movies = safeMovies.value
  if (movies.length <= 0) return []
  if (movies.length === 1) {
    return [
      {
        key: movies[0].id,
        movie: movies[0],
        movieIndex: 0,
        trackIndex: 0,
        clone: null as "head" | "tail" | null,
      },
    ]
  }

  const lastIndex = movies.length - 1
  return [
    {
      key: `hero-head-${movies[lastIndex]!.id}`,
      movie: movies[lastIndex]!,
      movieIndex: lastIndex,
      trackIndex: 0,
      clone: "head" as const,
    },
    ...movies.map((movie, index) => ({
      key: `hero-${movie.id}`,
      movie,
      movieIndex: index,
      trackIndex: index + 1,
      clone: null as "head" | "tail" | null,
    })),
    {
      key: `hero-tail-${movies[0]!.id}`,
      movie: movies[0]!,
      movieIndex: 0,
      trackIndex: movies.length + 1,
      clone: "tail" as const,
    },
  ]
})

function normalizeIndex(index: number): number {
  if (total.value <= 0) return 0
  return (index + total.value) % total.value
}

const activeMovie = computed(() => safeMovies.value[activeIndex.value])
const trackStyle = computed(() => ({
  transform: `translateX(calc((100% - var(--hero-card-width)) / 2 - ${activeTrackIndex.value} * (var(--hero-card-width) + var(--hero-card-gap))))`,
  transitionDuration: `${trackTransitionMs.value}ms`,
}))

function stopSnapReset() {
  if (!snapTimer) return
  clearTimeout(snapTimer)
  snapTimer = undefined
}

function stopAutoplay() {
  if (!timer) return
  clearInterval(timer)
  timer = undefined
}

function startAutoplay() {
  stopAutoplay()
  if (total.value <= 1) return
  timer = setInterval(() => {
    setActiveMovieIndex(normalizeIndex(activeIndex.value + 1), false, "autoplay")
  }, props.autoplayMs)
}

function selectIndex(index: number) {
  if (index < 0 || index >= total.value) return
  setActiveMovieIndex(index, true, "manual")
}

function syncTrackIndexWithActive() {
  activeTrackIndex.value = total.value > 1 ? activeIndex.value + 1 : activeIndex.value
}

function queueTrackSnap(trackIndex: number) {
  stopSnapReset()
  snapTimer = setTimeout(() => {
    trackTransitionEnabled.value = false
    activeTrackIndex.value = trackIndex
    setTimeout(() => {
      trackTransitionEnabled.value = true
    }, 16)
  }, trackTransitionMs.value + 40)
}

function setActiveMovieIndex(
  index: number,
  restartAutoplay: boolean,
  trigger: "manual" | "autoplay",
) {
  if (index < 0 || index >= total.value) return

  const currentIndex = activeIndex.value
  activeIndex.value = index
  trackTransitionMs.value =
    trigger === "manual" ? props.manualTransitionMs : props.autoplayTransitionMs

  if (total.value <= 1) {
    activeTrackIndex.value = index
  } else {
    let targetTrackIndex = index + 1
    let snapTrackIndex: number | undefined

    if (currentIndex === total.value - 1 && index === 0) {
      targetTrackIndex = total.value + 1
      snapTrackIndex = 1
    } else if (currentIndex === 0 && index === total.value - 1) {
      targetTrackIndex = 0
      snapTrackIndex = total.value
    }

    trackTransitionEnabled.value = true
    activeTrackIndex.value = targetTrackIndex

    if (snapTrackIndex !== undefined) {
      queueTrackSnap(snapTrackIndex)
    } else {
      stopSnapReset()
    }
  }

  if (restartAutoplay) {
    startAutoplay()
  }
}

function getSlideState(movieIndex: number): "active" | "prev" | "next" | "hidden" {
  if (total.value <= 0) return "hidden"
  if (movieIndex === activeIndex.value) return "active"
  if (total.value > 1 && movieIndex === normalizeIndex(activeIndex.value - 1)) return "prev"
  if (total.value > 1 && movieIndex === normalizeIndex(activeIndex.value + 1)) return "next"
  return "hidden"
}

function getSlideCardClass(movieIndex: number): string {
  const state = getSlideState(movieIndex)
  if (state === "active") {
    return "scale-100 opacity-100 ring-1 ring-white/12 shadow-2xl shadow-foreground/10 dark:shadow-black/35"
  }
  if (state === "prev" || state === "next") {
    return "scale-[0.91] opacity-82 brightness-75 saturate-[0.82] shadow-xl shadow-foreground/8 dark:shadow-black/24"
  }
  return "scale-[0.86] opacity-44 brightness-[0.62] saturate-[0.72]"
}

function getSlideImageClass(movieIndex: number): string {
  return getSlideState(movieIndex) === "active" ? "scale-100" : "scale-105"
}

function onSlideClick(index: number) {
  if (index === activeIndex.value) return
  selectIndex(index)
}

watch(total, (nextTotal) => {
  if (nextTotal === 0) {
    activeIndex.value = 0
    activeTrackIndex.value = 0
    trackTransitionMs.value = props.autoplayTransitionMs
    stopAutoplay()
    stopSnapReset()
    return
  }
  if (activeIndex.value >= nextTotal) {
    activeIndex.value = 0
  }
  syncTrackIndexWithActive()
  startAutoplay()
}, { immediate: true })

onMounted(() => {
  startAutoplay()
})

onBeforeUnmount(() => {
  stopAutoplay()
  stopSnapReset()
})
</script>

<template>
  <section
    data-home-hero
    class="relative isolate bg-[radial-gradient(circle_at_top,_hsl(var(--primary)/0.24),_transparent_52%),linear-gradient(180deg,hsl(var(--background)),hsl(var(--background)))]"
  >
    <div
      data-home-hero-shell
      class="flex w-full justify-center px-0 py-4 lg:py-5"
    >
      <div
        class="w-full"
      >
        <div
          data-home-hero-frame
          class="w-full overflow-visible"
        >
          <div
            data-home-hero-stage
            class="relative h-[clamp(22rem,44vw,40rem)] w-full overflow-hidden bg-[radial-gradient(circle_at_top,_hsl(var(--primary)/0.24),_transparent_68%),linear-gradient(180deg,rgba(8,10,14,0.92),rgba(8,10,14,0.98))] sm:h-[clamp(25rem,46vw,44rem)]"
          >
            <template v-if="activeMovie">
              <div
                class="relative h-full overflow-hidden [--hero-card-gap:0.75rem] [--hero-card-width:82%] sm:[--hero-card-gap:0.9rem] sm:[--hero-card-width:76%] lg:[--hero-card-gap:1rem] lg:[--hero-card-width:68%] xl:[--hero-card-width:64%]"
              >
                <div
                  data-home-hero-track
                  class="flex h-full gap-[var(--hero-card-gap)] will-change-transform"
                  :class="
                    trackTransitionEnabled
                      ? 'transition-transform duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]'
                      : 'transition-none'
                  "
                  :style="trackStyle"
                >
                  <article
                    v-for="slide in displaySlides"
                    :key="slide.key"
                    data-home-hero-slide
                    :data-hero-slide-state="getSlideState(slide.movieIndex)"
                    :data-hero-slide-clone="slide.clone ?? undefined"
                    class="group relative h-full w-[var(--hero-card-width)] shrink-0 overflow-hidden rounded-[1.35rem] border border-black/8 bg-background/8 transition-[transform,opacity,box-shadow,filter] duration-700 ease-[cubic-bezier(0.22,1,0.36,1)] dark:border-white/10 dark:bg-black/25 sm:rounded-[1.6rem]"
                    :class="getSlideCardClass(slide.movieIndex)"
                    @click="onSlideClick(slide.movieIndex)"
                  >
                    <MediaStill
                      v-if="slide.movie.coverUrl || slide.movie.thumbUrl"
                      :src="slide.movie.coverUrl || slide.movie.thumbUrl || ''"
                      :alt="slide.movie.code"
                      class="absolute inset-0 h-full w-full object-cover object-center transition-transform duration-700 ease-[cubic-bezier(0.22,1,0.36,1)]"
                      :class="getSlideImageClass(slide.movieIndex)"
                      loading="eager"
                      fetch-priority="high"
                    />
                    <div
                      v-else
                      class="absolute inset-0 bg-gradient-to-br from-primary/30 via-accent/20 to-card"
                      aria-hidden="true"
                    />
                    <div
                      data-hero-slide-overlay
                      class="absolute inset-0 bg-[linear-gradient(180deg,rgba(8,10,14,0.04)_0%,rgba(8,10,14,0.22)_58%,rgba(8,10,14,0.72)_100%)]"
                      aria-hidden="true"
                    />
                    <div class="absolute inset-x-0 bottom-0 flex items-end justify-between gap-4 p-4 sm:p-5 lg:p-6">
                      <div class="min-w-0 space-y-2 text-white">
                        <p
                          data-hero-slide-code
                          class="inline-flex max-w-full rounded-full border border-white/20 bg-black/44 px-3 py-1 text-[0.68rem] font-semibold tracking-[0.24em] text-white uppercase shadow-lg shadow-black/20 backdrop-blur-md"
                        >
                          {{ slide.movie.code }}
                        </p>
                        <h2
                          data-hero-slide-title
                          class="truncate text-base font-semibold sm:text-lg lg:text-[1.35rem]"
                        >
                          {{ slide.movie.title }}
                        </h2>
                      </div>

                      <div
                        v-if="slide.movieIndex === activeIndex"
                        class="hidden shrink-0 items-center gap-2 sm:flex"
                      >
                        <button
                          type="button"
                          class="rounded-full bg-white px-4 py-2 text-sm font-medium text-black transition-colors hover:bg-white/90"
                          @click.stop="emit('openDetails', slide.movie.id)"
                        >
                          {{ t("home.heroDetailsAction") }}
                        </button>
                        <button
                          type="button"
                          class="rounded-full border border-white/20 bg-white/10 px-4 py-2 text-sm font-medium text-white backdrop-blur-sm transition-colors hover:bg-white/16"
                          @click.stop="emit('openPlayer', slide.movie.id)"
                        >
                          {{ t("home.heroPlayAction") }}
                        </button>
                      </div>
                    </div>
                  </article>
                </div>
              </div>
            </template>

            <div
              v-else
              class="relative flex h-full items-end px-5 py-5 sm:px-6 sm:py-6 lg:px-8 lg:py-7 xl:px-10"
            >
              <div class="max-w-xl space-y-3">
                <p class="text-[0.68rem] font-medium tracking-[0.32em] text-muted-foreground uppercase">
                  {{ t("home.heroEyebrow") }}
                </p>
                <h1 class="text-3xl leading-none font-semibold sm:text-4xl lg:text-6xl">
                  {{ t("home.emptyTitle") }}
                </h1>
                <p class="text-sm leading-6 text-muted-foreground sm:text-base">
                  {{ t("home.emptyBody") }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div
          data-home-hero-progress-rail
          class="relative mx-auto mt-3 w-[calc(100%-2rem)] max-w-[54rem] rounded-[1.4rem] bg-background/72 px-5 py-4 backdrop-blur-md sm:w-[calc(100%-2.5rem)] sm:px-7 lg:w-[calc(100%-3rem)] lg:px-10 xl:w-[calc(100%-4rem)] xl:px-14"
        >
          <div class="grid grid-cols-8 gap-2">
            <button
              v-for="(_, index) in safeMovies"
              :key="index"
              type="button"
              data-hero-progress-item
              :data-hero-progress-item-active="index === activeIndex ? 'true' : undefined"
              class="h-1.5 rounded-full transition-colors duration-200"
              :class="
                index === activeIndex
                  ? 'bg-primary'
                  : 'bg-foreground/12 hover:bg-foreground/24'
              "
              :aria-label="`${t('home.heroProgressLabel')} ${index + 1}`"
              @click="selectIndex(index)"
            />
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

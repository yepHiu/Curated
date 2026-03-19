<script setup lang="ts">
import { ref } from "vue"
import {
  Maximize2,
  Pause,
  Play,
  SkipBack,
  SkipForward,
  Volume2,
} from "lucide-vue-next"
import type { Movie } from "@/lib/jav-library"
import { formatRuntime } from "@/lib/jav-library"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Slider } from "@/components/ui/slider"

defineProps<{
  movie: Movie
}>()

const playbackProgress = ref(43)
const volume = ref([68])
</script>

<template>
  <div class="flex h-full min-h-0 flex-col p-1 sm:p-2">
    <div
      class="relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-[1.75rem] border border-border/50 bg-gradient-to-br from-black via-zinc-950 to-card"
    >
      <div class="absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 bg-gradient-to-b from-black/85 via-black/40 to-transparent p-4 sm:p-5">
        <div class="flex flex-col items-start gap-2 text-left">
          <Badge variant="secondary" class="rounded-full border border-border/60 bg-background/30">
            {{ movie.code }}
          </Badge>
          <div class="flex flex-col gap-1">
            <p class="text-lg font-semibold text-white sm:text-xl">{{ movie.title }}</p>
            <p class="text-sm text-white/65">{{ movie.location }}</p>
          </div>
        </div>
      </div>

      <div class="flex min-h-0 flex-1 items-center justify-center p-4 sm:p-6 lg:p-8">
        <div class="flex flex-col items-center gap-4 text-center">
          <Button size="icon-lg" class="size-16 rounded-full sm:size-18">
            <Play />
          </Button>
          <div class="flex flex-col gap-1">
            <p class="text-xl font-semibold text-white sm:text-2xl">Video surface placeholder</p>
            <p class="max-w-xl text-sm text-white/65">
              Future mpv output and playback surface will attach here.
            </p>
          </div>
        </div>
      </div>

      <div class="absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-black/90 via-black/65 to-transparent p-4 sm:p-5">
        <div class="flex w-full flex-col gap-4">
          <div class="flex items-center justify-between gap-3 text-sm text-white/70">
            <span>00:58:21</span>
            <span>{{ formatRuntime(movie.runtimeMinutes) }}</span>
          </div>

          <Progress v-model="playbackProgress" class="h-2.5 bg-white/10" />

          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-2">
              <Button variant="secondary" size="icon" class="rounded-full bg-white/10 text-white hover:bg-white/20">
                <SkipBack />
              </Button>
              <Button size="icon-lg" class="rounded-full">
                <Pause />
              </Button>
              <Button variant="secondary" size="icon" class="rounded-full bg-white/10 text-white hover:bg-white/20">
                <SkipForward />
              </Button>
            </div>

            <div class="flex flex-wrap items-center gap-3">
              <div class="flex min-w-[14rem] items-center gap-3 rounded-full bg-white/8 px-4 py-2 text-white/80 backdrop-blur">
                <Volume2 />
                <Slider v-model="volume" :max="100" :step="1" class="flex-1" />
                <span class="w-10 text-right text-sm">{{ volume[0] }}%</span>
              </div>

              <Button variant="secondary" class="rounded-2xl bg-white/10 text-white hover:bg-white/20">
                <Maximize2 data-icon="inline-start" />
                Full screen
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

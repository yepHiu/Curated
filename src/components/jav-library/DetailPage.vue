<script setup lang="ts">
import { computed } from "vue"
import type { Movie } from "@/domain/movie/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import DetailPanel from "@/components/jav-library/DetailPanel.vue"
import MovieGrid from "@/components/jav-library/MovieGrid.vue"
import MediaStill from "@/components/jav-library/MediaStill.vue"

const props = defineProps<{
  movie: Movie
  relatedMovies: Movie[]
}>()

const previewImages = computed(() => props.movie.previewImages?.slice(0, 18) ?? [])
const hasPreviews = computed(() => previewImages.value.length > 0)

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()
</script>

<template>
  <div class="flex flex-col gap-6">
    <DetailPanel
      :movie="movie"
      @open-player="emit('openPlayer', $event)"
      @toggle-favorite="emit('toggleFavorite', $event)"
    />

    <Card class="rounded-3xl border-border/70 bg-card/85">
      <CardHeader>
        <CardTitle>Preview gallery</CardTitle>
        <CardDescription>
          来自元数据刮削的样本图；若源站限制外链，图片可能无法显示（可后续接后端代理）。
        </CardDescription>
      </CardHeader>
      <CardContent
        v-if="hasPreviews"
        class="grid w-full gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <div
          v-for="(url, index) in previewImages"
          :key="`${url}-${index}`"
          class="relative aspect-[16/9] overflow-hidden rounded-[1.25rem] border border-border/70 bg-muted/30"
        >
          <MediaStill
            :src="url"
            :alt="`${movie.code} sample ${index + 1}`"
            class="absolute inset-0 z-0"
          />
        </div>
      </CardContent>
      <CardContent v-else class="grid w-full gap-4 sm:grid-cols-3">
        <div
          v-for="index in 3"
          :key="index"
          class="aspect-[16/9] rounded-[1.25rem] border border-dashed border-border/70 bg-muted/20"
        />
        <p class="col-span-full text-sm text-muted-foreground">
          当前条目没有样本图 URL（例如仅刮削到文本、或提供商未返回预览图）。
        </p>
      </CardContent>
    </Card>

    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <h3 class="text-xl font-semibold">Related titles</h3>
        <p class="text-sm text-muted-foreground">
          More cards using the same movie grid system and selection contract.
        </p>
      </div>

      <MovieGrid
        :movies="relatedMovies"
        :selected-movie-id="movie.id"
        @select="emit('select', $event)"
        @open-details="emit('openDetails', $event)"
        @open-player="emit('openPlayer', $event)"
        @toggle-favorite="emit('toggleFavorite', $event)"
      />
    </div>
  </div>
</template>

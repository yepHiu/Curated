<script setup lang="ts">
import { computed } from "vue"
import { Heart } from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { Toggle } from "@/components/ui/toggle"

const props = defineProps<{
  movie: Movie
  selected?: boolean
  showFavorite?: boolean
}>()

const emit = defineEmits<{
  select: [movieId: string]
  openDetails: [movieId: string]
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

const visibleTags = computed(() => props.movie.tags.slice(0, 2))

const handleOpenDetails = () => {
  emit("select", props.movie.id)
  emit("openDetails", props.movie.id)
}

const handleFavoriteChange = (nextValue: boolean) => {
  emit("toggleFavorite", { movieId: props.movie.id, nextValue })
}
</script>

<template>
  <Card
    class="group gap-0 overflow-hidden rounded-[1.2rem] border-border/70 bg-card/80 py-0 shadow-lg shadow-black/5 transition-transform duration-200 hover:-translate-y-1"
  >
    <button
      type="button"
      class="flex w-full flex-col text-left focus-visible:outline-none"
      @click="handleOpenDetails"
    >
      <div class="p-2.5 pb-0">
        <div
          class="relative flex w-full items-start overflow-hidden rounded-[0.95rem] border border-border/60 bg-gradient-to-br p-2.5 aspect-[358/537]"
          :class="movie.tone"
        >
          <Badge
            class="h-5 w-fit rounded-full bg-background/80 px-1.5 text-[10px] text-foreground hover:bg-background/80"
          >
            {{ movie.code }}
          </Badge>

          <Toggle
            v-if="props.showFavorite !== false"
            :pressed="props.movie.isFavorite"
            variant="outline"
            size="sm"
            class="absolute right-2.5 bottom-2.5 z-10 rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
            @update:pressed="handleFavoriteChange(Boolean($event))"
            @click.stop
          >
            <Heart />
          </Toggle>
        </div>
      </div>

      <CardContent class="flex h-[4.75rem] flex-col justify-between gap-1.5 p-2.5">
        <div class="flex min-w-0 min-h-0 flex-col justify-start gap-0.5">
          <CardTitle class="truncate text-[13px]">{{ movie.title }}</CardTitle>
          <CardDescription class="truncate text-[11px]">
            {{ movie.actors.join(" · ") }}
          </CardDescription>
        </div>

        <div class="flex h-5 items-start gap-1 overflow-hidden">
          <Badge
            v-for="tag in visibleTags"
            :key="tag"
            variant="secondary"
            class="max-w-[4.75rem] truncate rounded-full border border-border/60 bg-secondary/70 px-1.5 text-[10px]"
          >
            {{ tag }}
          </Badge>
        </div>
      </CardContent>
    </button>
  </Card>
</template>

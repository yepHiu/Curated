<script setup lang="ts">
import { Heart, PlayCircle, Star } from "lucide-vue-next"
import type { Movie } from "@/domain/movie/types"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Toggle } from "@/components/ui/toggle"

const props = withDefaults(
  defineProps<{
    movie: Movie
    compact?: boolean
    showActions?: boolean
  }>(),
  {
    compact: false,
    showActions: true,
  },
)

const emit = defineEmits<{
  openPlayer: [movieId: string]
  toggleFavorite: [payload: { movieId: string; nextValue: boolean }]
}>()

const actorInitials = (name: string) =>
  name
    .split(" ")
    .slice(0, 2)
    .map((part) => part.charAt(0))
    .join("")
    .toUpperCase()

const handleFavoriteChange = (nextValue: boolean) => {
  emit("toggleFavorite", { movieId: props.movie.id, nextValue })
}
</script>

<template>
  <Card class="rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
    <CardContent
      class="grid gap-6 p-5 sm:p-6"
      :class="
        props.compact
          ? 'lg:grid-cols-[minmax(11rem,12.5rem)_minmax(0,1fr)]'
          : 'lg:grid-cols-[minmax(13rem,15rem)_minmax(0,1fr)]'
      "
    >
      <div class="mx-auto w-full" :class="props.compact ? 'max-w-[12.5rem]' : 'max-w-[15rem]'">
        <div
          class="relative aspect-[358/537] overflow-hidden rounded-[1.5rem] border border-border/60 bg-gradient-to-br p-4"
          :class="movie.tone"
        >
          <div class="flex h-full flex-col justify-between">
            <Badge class="w-fit rounded-full bg-background/80 text-foreground hover:bg-background/80">
              {{ movie.code }}
            </Badge>
          </div>

          <Toggle
            :pressed="props.movie.isFavorite"
            variant="outline"
            size="sm"
            class="absolute right-2.5 bottom-2.5 z-10 rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
            @update:pressed="handleFavoriteChange(Boolean($event))"
          >
            <Heart />
          </Toggle>
        </div>

        <div class="mt-3 rounded-2xl border border-border/70 bg-background/50 p-4">
          <p class="text-sm text-muted-foreground">Rating</p>
          <p class="mt-2 flex items-center gap-2 text-lg font-semibold">
            <Star class="text-primary" />
            {{ movie.rating.toFixed(1) }}/5
          </p>
        </div>
      </div>

      <div class="flex min-w-0 flex-col gap-5">
        <div class="flex min-w-0 flex-col gap-2">
          <CardTitle :class="props.compact ? 'text-2xl' : 'text-2xl sm:text-3xl'">
            {{ movie.title }}
          </CardTitle>
          <CardDescription class="text-sm text-muted-foreground sm:text-base">
            {{ movie.studio }} · {{ movie.year }} · {{ movie.resolution }}
          </CardDescription>
        </div>

        <p class="text-sm leading-6 text-muted-foreground">{{ movie.summary }}</p>

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">Tags</p>
          <div class="flex flex-wrap gap-2">
            <Badge
              v-for="tag in movie.tags"
              :key="tag"
              variant="secondary"
              class="rounded-full border border-border/60 bg-secondary/70"
            >
              {{ tag }}
            </Badge>
          </div>
        </div>

        <Separator />

        <div class="flex flex-col gap-3">
          <p class="text-sm font-medium">Cast</p>
          <div class="flex flex-wrap gap-3">
            <div
              v-for="actor in movie.actors"
              :key="actor"
              class="flex w-[15rem] max-w-full items-center gap-3 rounded-2xl border border-border/70 bg-background/50 p-3"
            >
              <Avatar class="size-10 border border-border/70">
                <AvatarFallback class="bg-primary/15 text-primary">
                  {{ actorInitials(actor) }}
                </AvatarFallback>
              </Avatar>
              <div class="flex min-w-0 flex-col gap-1">
                <span class="truncate text-sm font-medium">{{ actor }}</span>
                <span class="truncate text-sm text-muted-foreground">
                  Matched from local metadata cache
                </span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="props.showActions" class="flex flex-wrap items-center gap-3">
          <Button class="rounded-2xl" @click="emit('openPlayer', movie.id)">
            <PlayCircle data-icon="inline-start" />
            Play selected title
          </Button>
          <Button variant="secondary" class="rounded-2xl">
            Refresh metadata
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

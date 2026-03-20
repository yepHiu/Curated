<script setup lang="ts">
import { computed } from "vue"
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
import MediaStill from "@/components/jav-library/MediaStill.vue"

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

/** 详情页优先展示封面，其次缩略图 */
const posterSrc = computed(() => props.movie.coverUrl || props.movie.thumbUrl || "")
</script>

<template>
  <Card class="rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
    <CardContent
      class="grid gap-6 p-5 sm:p-6"
      :class="
        props.compact
          ? 'lg:grid-cols-[minmax(11rem,12.5rem)_minmax(0,1fr)]'
          : 'lg:grid-cols-[minmax(18rem,30rem)_minmax(0,1fr)] xl:grid-cols-[minmax(20rem,34rem)_minmax(0,1fr)]'
      "
    >
      <div
        class="mx-auto w-full shrink-0"
        :class="
          props.compact
            ? 'max-w-[12.5rem]'
            : 'w-full max-w-[min(100%,30rem)] xl:max-w-[min(100%,34rem)]'
        "
      >
        <!-- 不锁死竖版比例：横版整碟封套 / 竖版封面都由图片 intrinsic 高度决定，避免上下黑边 -->
        <div
          class="relative isolate w-full overflow-hidden rounded-[1.5rem] border border-border/60"
          :class="
            posterSrc
              ? 'bg-zinc-950/90'
              : `aspect-[358/537] min-h-[14rem] bg-gradient-to-br p-4 ${movie.tone}`
          "
        >
          <MediaStill
            v-if="posterSrc"
            :src="posterSrc"
            :alt="`${movie.code} cover`"
            layout="intrinsic"
            class="relative z-0"
          />
          <div
            class="pointer-events-none absolute inset-0 z-[1] bg-gradient-to-t from-black/55 via-transparent to-black/30"
            aria-hidden="true"
          />

          <div class="pointer-events-none absolute inset-x-0 top-0 z-[2] flex justify-start p-4">
            <Badge
              class="pointer-events-auto w-fit rounded-full border border-border/40 bg-background/90 text-foreground shadow-sm backdrop-blur-sm hover:bg-background/90"
            >
              {{ movie.code }}
            </Badge>
          </div>

          <Toggle
            :pressed="props.movie.isFavorite"
            variant="outline"
            size="sm"
            class="absolute right-2.5 bottom-2.5 z-[2] rounded-full border-border/60 bg-background/80 px-0 shadow-sm backdrop-blur hover:bg-background/90 data-[state=on]:border-primary data-[state=on]:bg-primary data-[state=on]:text-primary-foreground"
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

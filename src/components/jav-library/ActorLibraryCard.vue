<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { useRouter } from "vue-router"
import type { ActorListItemDTO } from "@/api/types"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card"

const props = defineProps<{
  actor: ActorListItemDTO
}>()

const { t } = useI18n()
const router = useRouter()

const initials = computed(() => {
  const n = props.actor.name.trim()
  if (!n) return "?"
  const parts = n.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0]![0]! + parts[1]![0]!).toUpperCase()
  }
  return n.slice(0, 2).toUpperCase()
})

function goFilmography() {
  void router.push({
    name: "actor-detail",
    params: { actorName: props.actor.name },
  })
}
</script>

<template>
  <Card
    class="group flex h-full min-w-0 w-full max-w-full flex-col gap-0 overflow-hidden rounded-2xl border-border/70 bg-card/90 py-0 shadow-sm transition-[box-shadow,border-color,transform] hover:-translate-y-0.5 hover:border-primary/25 hover:shadow-md"
  >
    <button
      type="button"
      class="flex h-full min-w-0 flex-col items-center gap-3 px-3 py-4 text-center focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:outline-none"
      @click="goFilmography"
    >
      <Avatar
        class="size-24 shrink-0 rounded-2xl border border-border/70 bg-muted shadow-sm transition-transform duration-200 group-hover:scale-[1.02]"
      >
        <AvatarImage
          v-if="actor.avatarUrl"
          :src="actor.avatarUrl"
          :alt="actor.name"
          class="object-cover"
        />
        <AvatarFallback
          class="rounded-2xl bg-muted text-lg font-semibold text-muted-foreground"
        >
          {{ initials }}
        </AvatarFallback>
      </Avatar>

      <CardContent class="flex min-w-0 flex-1 flex-col items-center gap-1 px-1 py-0">
        <CardTitle class="line-clamp-2 text-base font-semibold leading-snug">
          {{ actor.name }}
        </CardTitle>
        <CardDescription class="text-xs leading-tight">
          {{ t("actors.movieCount", { n: actor.movieCount }) }}
        </CardDescription>
      </CardContent>
    </button>
  </Card>
</template>

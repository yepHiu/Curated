<script setup lang="ts">
import { PlayCircle, SlidersHorizontal } from "lucide-vue-next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const props = defineProps<{
  title: string
  description: string
}>()

const emit = defineEmits<{
  openPlayer: []
}>()
</script>

<template>
  <div
    class="flex flex-col gap-4 rounded-3xl border border-border/70 bg-card/90 p-6 shadow-xl shadow-black/10 lg:flex-row lg:items-center lg:justify-between"
  >
    <div class="flex min-w-0 flex-col gap-3">
      <Badge class="w-fit rounded-full bg-primary/15 text-primary hover:bg-primary/15">
        Cinema shell
      </Badge>
      <h2 class="text-3xl font-semibold tracking-tight">{{ props.title }}</h2>
    </div>

    <div class="flex w-full max-w-xl flex-col gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <Button variant="secondary" class="rounded-2xl">
          <SlidersHorizontal data-icon="inline-start" />
          Filters
        </Button>
        <Button class="rounded-2xl" @click="emit('openPlayer')">
          <PlayCircle data-icon="inline-start" />
          Open player view
        </Button>
      </div>
    </div>
  </div>
</template>

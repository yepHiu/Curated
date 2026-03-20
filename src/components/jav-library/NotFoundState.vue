<script setup lang="ts">
import { AlertCircle } from "lucide-vue-next"
import { RouterLink } from "vue-router"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

withDefaults(
  defineProps<{
    title?: string
    description?: string
    backLabel?: string
  }>(),
  {
    title: "Content not found",
    description: "The requested item does not exist or is no longer available in the current library snapshot.",
    backLabel: "Back to library",
  },
)
</script>

<template>
  <div class="flex h-full min-h-0 items-center justify-center">
    <Card class="w-full max-w-xl rounded-3xl border-border/70 bg-card/85 shadow-xl shadow-black/10">
      <CardHeader class="items-center text-center">
        <div class="flex size-14 items-center justify-center rounded-full bg-primary/12 text-primary">
          <AlertCircle />
        </div>
        <CardTitle>{{ title }}</CardTitle>
        <CardDescription class="max-w-md">
          {{ description }}
        </CardDescription>
      </CardHeader>
      <CardContent class="flex justify-center">
        <Button as-child class="rounded-2xl">
          <RouterLink :to="{ name: 'library' }">
            {{ backLabel }}
          </RouterLink>
        </Button>
      </CardContent>
    </Card>
  </div>
</template>

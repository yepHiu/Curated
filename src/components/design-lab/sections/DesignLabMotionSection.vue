<script setup lang="ts">
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const motionPresets = [
  {
    name: "Fade",
    timing: "220ms",
    easing: "cubic-bezier(0.32, 0.72, 0, 1)",
    previewClass: "animate-in fade-in duration-300",
  },
  {
    name: "Slide",
    timing: "260ms",
    easing: "cubic-bezier(0.24, 0.84, 0.18, 1)",
    previewClass: "animate-in slide-in-from-bottom-3 duration-300",
  },
  {
    name: "Zoom",
    timing: "240ms",
    easing: "cubic-bezier(0.2, 0.8, 0.2, 1)",
    previewClass: "animate-in zoom-in-95 duration-300",
  },
] as const
</script>

<template>
  <section id="motion" class="scroll-mt-24">
    <Card class="rounded-2xl border border-border bg-card shadow-sm">
      <CardHeader>
        <CardTitle>Motion</CardTitle>
        <CardDescription>
          Entrance and emphasis motion should stay restrained, token-driven, and safe under reduced-motion settings.
        </CardDescription>
      </CardHeader>
      <CardContent class="grid gap-4 xl:grid-cols-3">
        <article
          v-for="preset in motionPresets"
          :key="preset.name"
          class="rounded-2xl border border-border/70 p-4"
        >
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-semibold text-foreground">{{ preset.name }}</h3>
            <span class="font-mono text-xs text-muted-foreground">{{ preset.timing }}</span>
          </div>

          <div class="mt-4 rounded-2xl border border-dashed border-border/70 bg-background/80 p-4">
            <div class="flex h-24 items-center justify-center rounded-2xl bg-surface">
              <div
                class="rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground motion-reduce:animate-none"
                :class="preset.previewClass"
              >
                {{ preset.name }} preview
              </div>
            </div>
          </div>

          <dl class="mt-4 space-y-2 text-sm leading-6 text-muted-foreground">
            <div class="flex items-center justify-between gap-3">
              <dt>Easing</dt>
              <dd class="font-mono text-xs text-foreground">{{ preset.easing }}</dd>
            </div>
            <div class="rounded-xl bg-muted/50 px-3 py-2 text-xs leading-5">
              Reduced motion: keep visibility changes instant or near-instant when `prefers-reduced-motion` is active.
            </div>
          </dl>
        </article>
      </CardContent>
    </Card>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

const { t } = useI18n()
</script>

<template>
  <div
    data-homepage-empty-state
    class="flex h-full min-h-0 items-center justify-center overflow-y-auto bg-background px-4 py-10 text-foreground sm:px-6"
  >
    <Card class="w-full max-w-2xl rounded-[1.75rem] border border-dashed border-border/70 bg-card/60 text-center shadow-sm">
      <CardHeader class="space-y-3 pb-2">
        <p class="text-[0.68rem] font-medium tracking-[0.32em] text-muted-foreground uppercase">
          {{ t("home.heroEyebrow") }}
        </p>
        <CardTitle class="text-3xl leading-none sm:text-4xl">
          {{ t("home.emptyTitle") }}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p class="mx-auto max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
          {{ t("home.emptyBody") }}
        </p>
      </CardContent>
    </Card>
  </div>
</template>

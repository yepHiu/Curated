<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { Layers } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"

defineProps<{
  organizeLibrary: boolean
  organizeLibrarySaving: boolean
  organizeLibraryError: string
}>()

const emit = defineEmits<{
  changeOrganizeLibrary: [value: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <div class="break-inside-avoid">
    <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="space-y-3 pb-2">
        <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <Layers class="size-[1.15rem]" />
          </span>
          {{ t("settings.organizeTitle") }}
        </CardTitle>
        <CardDescription
          class="text-xs leading-relaxed text-pretty text-muted-foreground"
        >
          {{ t("settings.organizeDesc") }}
        </CardDescription>
      </CardHeader>
      <CardContent class="flex flex-col gap-3 pt-2">
        <div
          class="flex items-center justify-between gap-3 rounded-2xl border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
          :aria-busy="organizeLibrarySaving"
        >
          <div class="flex min-w-0 flex-1 flex-col gap-3">
            <p class="text-sm font-semibold text-foreground">
              {{ t("settings.organizeSwitch") }}
            </p>
            <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
              {{ t("settings.organizeHint") }}
            </p>
            <p
              v-if="organizeLibrarySaving"
              class="text-xs text-muted-foreground motion-safe:animate-pulse"
            >
              {{ t("settings.organizeSyncing") }}
            </p>
          </div>
          <Switch
            class="motion-safe:transition-colors motion-safe:duration-200"
            :model-value="organizeLibrary"
            @update:model-value="emit('changeOrganizeLibrary', $event)"
          />
        </div>
        <p v-if="organizeLibraryError" class="text-sm text-destructive">
          {{ organizeLibraryError }}
        </p>
      </CardContent>
    </Card>
  </div>
</template>

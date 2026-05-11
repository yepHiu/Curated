<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { CircleHelp, Layers } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import {
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from "reka-ui"

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
    <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
      <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <Layers class="size-[1.15rem]" />
        </span>
        <CardTitle class="min-w-0 text-lg tracking-tight">
          {{ t("settings.organizeTitle") }}
        </CardTitle>
        <CardDescription
          class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
        >
          {{ t("settings.organizeDesc") }}
        </CardDescription>
      </CardHeader>
      <CardContent class="flex flex-col gap-3 pt-0">
        <div
          class="flex items-center justify-between gap-3 rounded-2xl border border-border/50 bg-muted/5 p-4 shadow-sm shadow-black/5"
          :aria-busy="organizeLibrarySaving"
        >
          <div class="flex min-w-0 flex-1 flex-col gap-3">
            <div class="flex flex-wrap items-center gap-1.5">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.organizeSwitch") }}
              </p>
              <TooltipProvider :delay-duration="280">
                <TooltipRoot>
                  <TooltipTrigger as-child>
                    <button
                      type="button"
                      data-organize-library-help
                      class="inline-flex size-8 shrink-0 items-center justify-center rounded-full text-muted-foreground transition hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                    >
                      <CircleHelp class="size-4" aria-hidden="true" />
                      <span class="sr-only">{{ t("settings.organizeHelpAria") }}</span>
                    </button>
                  </TooltipTrigger>
                  <TooltipPortal>
                    <TooltipContent
                      side="top"
                      :side-offset="6"
                      class="z-50 max-w-[min(22rem,calc(100vw-2rem))] rounded-xl border border-border/50 bg-popover px-3 py-2 text-xs leading-relaxed text-pretty text-popover-foreground shadow-lg"
                    >
                      {{ t("settings.organizeHint") }}
                    </TooltipContent>
                  </TooltipPortal>
                </TooltipRoot>
              </TooltipProvider>
            </div>
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

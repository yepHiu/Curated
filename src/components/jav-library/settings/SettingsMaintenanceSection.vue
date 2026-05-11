<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { BookOpen, RefreshCw, ScanSearch, Wrench } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

defineProps<{
  fullScanBusy: boolean
}>()

const emit = defineEmits<{
  runFullScan: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="flex w-full flex-col gap-6">
    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <Wrench class="size-[1.15rem]" />
          </span>
          <CardTitle class="min-w-0 text-lg tracking-tight">
            {{ t("settings.manualCardTitle") }}
          </CardTitle>
          <CardDescription
            class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
          >
            {{ t("settings.manualCardDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <div
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0 flex flex-col gap-3">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.triggerFullScan") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.triggerFullScanHint") }}
              </p>
            </div>
            <Button
              type="button"
              class="h-11 shrink-0 rounded-2xl px-5 font-medium"
              :disabled="fullScanBusy"
              data-settings-full-scan
              @click="emit('runFullScan')"
            >
              <ScanSearch
                data-icon="inline-start"
                class="size-4"
                :class="fullScanBusy ? 'animate-pulse' : ''"
              />
              {{ t("common.run") }}
            </Button>
          </div>

          <div
            class="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/5 p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0 flex flex-col gap-3">
              <p class="text-sm font-semibold text-foreground">
                {{ t("settings.rebuildCache") }}
              </p>
              <p class="text-xs leading-relaxed text-muted-foreground sm:text-sm">
                {{ t("settings.rebuildCacheHint") }}
              </p>
            </div>
            <Button variant="secondary" class="h-11 shrink-0 rounded-2xl px-5 font-medium">
              <RefreshCw data-icon="inline-start" class="size-4" />
              {{ t("common.run") }}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <div class="break-inside-avoid">
      <Card class="gap-2 rounded-xl border border-border bg-card shadow-sm">
        <CardHeader class="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2.5 gap-y-1 pb-0">
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
            aria-hidden="true"
          >
            <BookOpen class="size-[1.15rem]" />
          </span>
          <CardTitle class="min-w-0 text-lg tracking-tight">
            {{ t("settings.configCardTitle") }}
          </CardTitle>
          <CardDescription
            class="col-start-2 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm"
          >
            {{ t("settings.configCardDesc") }}
          </CardDescription>
        </CardHeader>
        <CardContent class="pt-0 text-xs leading-relaxed text-pretty text-muted-foreground sm:text-sm sm:leading-6">
          {{ t("settings.configCardBody") }}
        </CardContent>
      </Card>
    </div>
  </div>
</template>

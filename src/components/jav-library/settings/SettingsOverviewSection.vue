<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { LayoutDashboard } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import SettingsWatchTimeHeatmap from "@/components/jav-library/settings/SettingsWatchTimeHeatmap.vue"
import type {
  DailyWatchTimeEntry,
  DailyWatchTimeSummary,
} from "@/lib/watch-time-heatmap"

type DashboardStat = {
  labelKey: string
  value: string | number
  detailKey?: string
}

defineProps<{
  dashboardStats: readonly DashboardStat[]
  watchTimeDays?: readonly DailyWatchTimeEntry[]
  watchTimeSummary?: DailyWatchTimeSummary | null
  watchTimeLoading?: boolean
  watchTimeError?: string
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
            <LayoutDashboard class="size-[1.15rem]" />
          </span>
          <CardTitle class="min-w-0 text-lg tracking-tight">
            {{ t("settings.navOverview") }}
          </CardTitle>
        </CardHeader>
        <CardContent class="flex flex-col gap-3 pt-0">
          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <div
              v-for="stat in dashboardStats"
              :key="stat.labelKey"
              class="rounded-lg border border-border/50 bg-muted/5 px-3.5 py-3"
            >
              <p class="text-xs font-medium text-muted-foreground">{{ t(stat.labelKey) }}</p>
              <p class="mt-1 text-xl font-semibold tabular-nums">{{ stat.value }}</p>
              <p v-if="stat.detailKey" class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                {{ t(stat.detailKey) }}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>

    <SettingsWatchTimeHeatmap
      :days="watchTimeDays ?? []"
      :summary="watchTimeSummary"
      :loading="watchTimeLoading"
      :error="watchTimeError"
    />
  </div>
</template>

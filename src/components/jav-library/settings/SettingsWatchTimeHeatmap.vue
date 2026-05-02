<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { CalendarDays } from "lucide-vue-next"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  buildWatchTimeSummary,
  formatWatchTimeDuration,
  type DailyWatchTimeEntry,
  type DailyWatchTimeSummary,
} from "@/lib/watch-time-heatmap"

const props = withDefaults(
  defineProps<{
    days: readonly DailyWatchTimeEntry[]
    summary?: DailyWatchTimeSummary | null
    loading?: boolean
    error?: string
    today?: Date
  }>(),
  {
    summary: null,
    loading: false,
    error: "",
    today: () => new Date(),
  },
)

const { t } = useI18n()

const displaySummary = computed(() =>
  props.summary ?? buildWatchTimeSummary(props.days, { today: props.today }),
)

const metricItems = computed(() => [
  {
    key: "this-week",
    label: t("settings.watchTimeThisWeek"),
    value: formatWatchTimeDuration(displaySummary.value.thisWeekWatchedSec),
  },
  {
    key: "past-three-months",
    label: t("settings.watchTimePastThreeMonths"),
    value: formatWatchTimeDuration(displaySummary.value.totalWatchedSec),
  },
  {
    key: "longest-streak",
    label: t("settings.watchTimeLongestStreak"),
    value: t("settings.watchTimeDaysValue", {
      days: displaySummary.value.longestStreakDays,
    }),
  },
  {
    key: "max-day",
    label: t("settings.watchTimeMaxDay"),
    value: formatWatchTimeDuration(displaySummary.value.maxDayWatchedSec),
  },
])
</script>

<template>
  <Card class="gap-4 rounded-xl border border-border bg-card shadow-sm">
    <CardHeader class="gap-3 pb-2">
      <CardTitle class="flex items-center gap-2.5 text-lg font-semibold tracking-tight">
        <span
          class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10 text-primary"
          aria-hidden="true"
        >
          <CalendarDays class="size-[1.15rem]" />
        </span>
        {{ t("settings.watchTimeTitle") }}
      </CardTitle>
      <CardDescription>{{ t("settings.watchTimeDesc") }}</CardDescription>
    </CardHeader>

    <CardContent class="flex flex-col gap-4 pt-2">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="item in metricItems"
          :key="item.key"
          class="rounded-lg border border-border/50 bg-muted/5 px-3.5 py-3"
        >
          <p class="text-xs font-medium text-muted-foreground">{{ item.label }}</p>
          <p class="mt-1 text-xl font-semibold tabular-nums">{{ item.value }}</p>
        </div>
      </div>

      <div v-if="loading" class="flex flex-col gap-3" aria-live="polite">
        <p class="text-sm text-muted-foreground">{{ t("settings.watchTimeLoading") }}</p>
        <div class="grid grid-cols-4 gap-2">
          <Skeleton v-for="i in 8" :key="i" class="h-8 rounded-md" />
        </div>
      </div>

      <div
        v-else-if="error"
        class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
        role="alert"
      >
        {{ error }}
      </div>

      <div v-else>
        <p v-if="displaySummary.activeDays <= 0" class="text-sm text-muted-foreground">
          {{ t("settings.watchTimeEmpty") }}
        </p>
      </div>
    </CardContent>
  </Card>
</template>

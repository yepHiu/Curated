<script setup lang="ts">
import type { Component } from "vue"
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  ChartNoAxesColumnIncreasing,
  CircleCheckBig,
  Clock3,
  Film,
  Play,
  RotateCw,
  Star,
} from "lucide-vue-next"
import { Sparkles } from "lucide-vue-next"
import type {
  PersonalInsightsBreakdownDTO,
  PersonalInsightsDimension,
  PersonalInsightsOverviewDTO,
  PersonalInsightsRange,
} from "@/api/types"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { useExperimentalAgent } from "@/lib/experimental-agent"
import { useAIActionRequest } from "@/composables/use-ai-action-request"
import { useAIService } from "@/services/ai-service"
import { AIServiceError } from "@/services/contracts/ai-service"
import { useLibraryService } from "@/services/library-service"

interface MetricCard {
  key: string
  label: string
  value: string
  hint: string
  icon: Component
}

const rangeOptions: Array<{ value: PersonalInsightsRange; labelKey: string }> = [
  { value: "30d", labelKey: "insights.range30d" },
  { value: "90d", labelKey: "insights.range90d" },
  { value: "365d", labelKey: "insights.range365d" },
  { value: "all", labelKey: "insights.rangeAll" },
]

const breakdownDefinitions: Array<{
  dimension: PersonalInsightsDimension
  titleKey: string
  descriptionKey: string
}> = [
  { dimension: "actor", titleKey: "insights.actorTitle", descriptionKey: "insights.actorDescription" },
  { dimension: "studio", titleKey: "insights.studioTitle", descriptionKey: "insights.studioDescription" },
  { dimension: "tag", titleKey: "insights.tagTitle", descriptionKey: "insights.tagDescription" },
]

const { t, locale } = useI18n()
const libraryService = useLibraryService()
const aiService = useAIService()
const { run: runAIAction, pending: aiActionPending, cancel: cancelAIAction } = useAIActionRequest(aiService)
const { enabled: agentEnabled } = useExperimentalAgent()
const selectedRange = ref<PersonalInsightsRange>("30d")
const loading = ref(true)
const loadError = ref(false)
const overview = ref<PersonalInsightsOverviewDTO | null>(null)
const breakdowns = ref<Record<PersonalInsightsDimension, PersonalInsightsBreakdownDTO | null>>({
  actor: null,
  studio: null,
  tag: null,
})
const narrative = ref("")
const narrativeBusy = ref(false)
const narrativeError = ref("")
let requestSequence = 0

const timezone = (() => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
  } catch {
    return "UTC"
  }
})()

function numberFormatter(options?: Intl.NumberFormatOptions) {
  return new Intl.NumberFormat(locale.value, options)
}

function formatInteger(value: number): string {
  return numberFormatter({ maximumFractionDigits: 0 }).format(value)
}

function formatDuration(seconds: number): string {
  const totalMinutes = Math.max(0, Math.floor(seconds / 60))
  if (seconds > 0 && totalMinutes === 0) return t("insights.durationLessMinute")
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours > 0 && minutes > 0) {
    return t("insights.durationHoursMinutes", {
      hours: formatInteger(hours),
      minutes: formatInteger(minutes),
    })
  }
  if (hours > 0) return t("insights.durationHours", { hours: formatInteger(hours) })
  return t("insights.durationMinutes", { minutes: formatInteger(minutes) })
}

function formatPercent(value: number): string {
  return numberFormatter({ style: "percent", maximumFractionDigits: 1 }).format(value)
}

function formatRating(value: number | null): string {
  if (value === null) return "—"
  return `${numberFormatter({ maximumFractionDigits: 1 }).format(value)} / 5`
}

const metricCards = computed((): MetricCard[] => {
  const value = overview.value
  if (!value) return []
  return [
    {
      key: "watched",
      label: t("insights.watchedTime"),
      value: formatDuration(value.watchedSeconds),
      hint: t("insights.watchedHint"),
      icon: Clock3,
    },
    {
      key: "started",
      label: t("insights.startedMovies"),
      value: formatInteger(value.startedMovies),
      hint: t("insights.startedHint"),
      icon: Play,
    },
    {
      key: "completed",
      label: t("insights.completedMovies"),
      value: formatInteger(value.completedMovies),
      hint: t("insights.completedHint", { threshold: formatPercent(value.completionThreshold) }),
      icon: CircleCheckBig,
    },
    {
      key: "completion-rate",
      label: t("insights.completionRate"),
      value: value.completionRate === null ? "—" : formatPercent(value.completionRate),
      hint: value.completionRate === null
        ? t("insights.noCompletionDenominator")
        : t("insights.completionRateHint"),
      icon: ChartNoAxesColumnIncreasing,
    },
    {
      key: "rated",
      label: t("insights.ratedMovies"),
      value: formatInteger(value.ratedMovies),
      hint: t("insights.ratedHint"),
      icon: Film,
    },
    {
      key: "average-rating",
      label: t("insights.averageRating"),
      value: formatRating(value.averageUserRating),
      hint: value.averageUserRating === null
        ? t("insights.noRatingDenominator")
        : t("insights.averageRatingHint"),
      icon: Star,
    },
  ]
})

const isEmpty = computed(() => overview.value?.watchedSeconds === 0)

async function loadInsights() {
  const sequence = ++requestSequence
  loading.value = true
  loadError.value = false
  overview.value = null
  breakdowns.value = { actor: null, studio: null, tag: null }
  narrative.value = ""
  narrativeError.value = ""
  try {
    const params = { range: selectedRange.value, timezone }
    const [nextOverview, actor, studio, tag] = await Promise.all([
      libraryService.getPersonalInsightsOverview(params),
      libraryService.getPersonalInsightsBreakdown({ ...params, dimension: "actor", limit: 10 }),
      libraryService.getPersonalInsightsBreakdown({ ...params, dimension: "studio", limit: 10 }),
      libraryService.getPersonalInsightsBreakdown({ ...params, dimension: "tag", limit: 10 }),
    ])
    if (sequence !== requestSequence) return
    overview.value = nextOverview
    breakdowns.value = { actor, studio, tag }
  } catch {
    if (sequence !== requestSequence) return
    loadError.value = true
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

watch(selectedRange, () => void loadInsights(), { immediate: true })
onBeforeUnmount(() => { requestSequence += 1 })

watch(selectedRange, cancelAIAction)
watch(agentEnabled, cancelAIAction)

async function generateNarrative() {
  if (!agentEnabled.value || narrativeBusy.value || !overview.value) {
    return
  }
  if (isEmpty.value) {
    narrativeError.value = t("insights.aiReadoutEmpty")
    return
  }
  narrativeBusy.value = true
  narrativeError.value = ""
  try {
    const dto = await runAIAction("insights_narrative", {
      range: selectedRange.value,
      timezone,
      locale: locale.value,
    })
    narrative.value = dto.proposedText?.trim() ?? ""
    if (!narrative.value) {
      narrativeError.value = t("insights.aiReadoutError")
    }
  } catch (err) {
    if (err instanceof AIServiceError && err.code === "AI_CANCELLED") return
    if (err instanceof AIServiceError && err.code === "AI_PROVIDER_UNAVAILABLE") {
      narrativeError.value = t("insights.aiReadoutUnconfigured")
    } else {
      narrativeError.value = t("insights.aiReadoutError")
    }
  } finally {
    narrativeBusy.value = false
  }
}
</script>

<template>
  <div
    data-personal-insights-page
    class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-y-auto"
  >
    <main class="mx-auto flex w-full max-w-7xl flex-col gap-6 px-3 pb-8 sm:px-6 lg:px-8">
      <header class="flex flex-col gap-5">
        <div class="flex max-w-3xl flex-col gap-2">
          <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
            {{ t("insights.title") }}
          </h1>
          <p class="text-sm leading-relaxed text-muted-foreground sm:text-base">
            {{ t("insights.subtitle") }}
          </p>
        </div>

        <fieldset class="grid grid-cols-2 gap-2 sm:grid-cols-4" data-insights-range-selector>
          <legend class="sr-only">{{ t("insights.rangeLegend") }}</legend>
          <label
            v-for="option in rangeOptions"
            :key="option.value"
            class="relative flex min-h-11 cursor-pointer items-center justify-center rounded-2xl border border-border/70 bg-card px-3 text-center text-sm font-medium transition-colors hover:bg-accent/70 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring/60 has-[:checked]:border-primary/60 has-[:checked]:bg-primary has-[:checked]:text-primary-foreground"
          >
            <input
              v-model="selectedRange"
              class="sr-only"
              type="radio"
              name="personal-insights-range"
              :value="option.value"
            />
            <span>{{ t(option.labelKey) }}</span>
          </label>
        </fieldset>

        <div v-if="agentEnabled" class="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            class="min-h-11 rounded-lg md:h-8 md:min-h-8"
            :disabled="narrativeBusy || loading || !overview"
            data-insights-ai-readout
            @click="generateNarrative"
          >
            <Sparkles class="size-4" />
            {{ t("insights.aiReadout") }}
          </Button>
          <Button v-if="aiActionPending" type="button" variant="ghost" size="sm" data-ai-action-cancel @click="cancelAIAction">{{ t("common.cancel") }}</Button>
        </div>
        <p v-if="narrativeError" class="text-sm text-destructive">{{ narrativeError }}</p>
        <Card v-if="narrative" data-insights-ai-narrative class="border-border/70">
          <CardHeader class="pb-2">
            <CardTitle class="text-base">{{ t("insights.aiReadoutTitle") }}</CardTitle>
          </CardHeader>
          <CardContent class="whitespace-pre-wrap text-sm leading-relaxed">
            {{ narrative }}
          </CardContent>
        </Card>

        <p v-if="overview" class="text-xs leading-relaxed text-muted-foreground">
          {{ t("insights.rangeSummary", { from: overview.from, to: overview.to, timezone: overview.timezone }) }}
          <template v-if="overview.dataSince">
            · {{ t("insights.dataSince", { date: overview.dataSince }) }}
          </template>
        </p>
      </header>

      <div v-if="loading" class="flex flex-col gap-6" role="status" aria-live="polite">
        <span class="sr-only">{{ t("insights.loading") }}</span>
        <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
          <Card v-for="index in 6" :key="index" class="gap-3">
            <CardHeader class="gap-2 pb-0">
              <Skeleton class="h-4 w-24" />
              <Skeleton class="h-7 w-20" />
            </CardHeader>
            <CardContent><Skeleton class="h-3 w-full" /></CardContent>
          </Card>
        </div>
        <div class="grid gap-4 lg:grid-cols-3">
          <Card v-for="index in 3" :key="index" class="gap-3">
            <CardHeader class="gap-2"><Skeleton class="h-5 w-28" /><Skeleton class="h-3 w-full" /></CardHeader>
            <CardContent class="flex flex-col gap-3">
              <Skeleton v-for="row in 4" :key="row" class="h-10 w-full" />
            </CardContent>
          </Card>
        </div>
      </div>

      <Card v-else-if="loadError" role="alert" class="border-border/80">
        <CardHeader>
          <CardTitle>{{ t("insights.errorTitle") }}</CardTitle>
          <CardDescription>{{ t("insights.errorDescription") }}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button type="button" variant="secondary" class="min-h-11 rounded-2xl" @click="loadInsights">
            <RotateCw data-icon="inline-start" />
            {{ t("insights.retry") }}
          </Button>
        </CardContent>
      </Card>

      <template v-else-if="overview">
        <section class="flex flex-col gap-3" aria-labelledby="insights-summary-heading">
          <h2 id="insights-summary-heading" class="text-lg font-semibold tracking-tight">
            {{ t("insights.summaryTitle") }}
          </h2>
          <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
            <Card
              v-for="metric in metricCards"
              :key="metric.key"
              :data-insights-metric="metric.key"
              class="min-w-0 gap-3"
            >
              <CardHeader class="flex-row items-start justify-between gap-2 pb-0">
                <div class="flex min-w-0 flex-col gap-2">
                  <CardDescription class="text-xs">{{ metric.label }}</CardDescription>
                  <CardTitle class="break-words text-xl sm:text-2xl">{{ metric.value }}</CardTitle>
                </div>
                <component :is="metric.icon" class="size-5 shrink-0 text-primary" aria-hidden="true" />
              </CardHeader>
              <CardContent class="text-xs leading-relaxed text-muted-foreground">
                {{ metric.hint }}
              </CardContent>
            </Card>
          </div>
        </section>

        <Card v-if="isEmpty" class="border-dashed">
          <CardHeader>
            <CardTitle>{{ t("insights.emptyTitle") }}</CardTitle>
            <CardDescription class="max-w-2xl leading-relaxed">
              {{ t("insights.emptyDescription") }}
            </CardDescription>
          </CardHeader>
        </Card>

        <section class="flex flex-col gap-3" aria-labelledby="insights-breakdown-heading">
          <div class="flex flex-col gap-1">
            <h2 id="insights-breakdown-heading" class="text-lg font-semibold tracking-tight">
              {{ t("insights.breakdownTitle") }}
            </h2>
            <p class="text-sm leading-relaxed text-muted-foreground">
              {{ t("insights.attributionNote") }}
            </p>
          </div>

          <div class="grid gap-4 lg:grid-cols-3">
            <Card
              v-for="definition in breakdownDefinitions"
              :key="definition.dimension"
              :data-insights-breakdown="definition.dimension"
              class="min-w-0 gap-3"
            >
              <CardHeader class="gap-1 pb-0">
                <CardTitle class="text-base">{{ t(definition.titleKey) }}</CardTitle>
                <CardDescription>{{ t(definition.descriptionKey) }}</CardDescription>
              </CardHeader>
              <CardContent class="flex flex-col gap-4">
                <p
                  v-if="breakdowns[definition.dimension]?.items.length === 0"
                  class="py-6 text-center text-sm text-muted-foreground"
                >
                  {{ t("insights.noBreakdown") }}
                </p>
                <div
                  v-for="item in breakdowns[definition.dimension]?.items ?? []"
                  :key="item.name"
                  class="flex min-w-0 flex-col gap-2"
                >
                  <div class="flex min-w-0 items-start justify-between gap-3">
                    <span class="min-w-0 truncate text-sm font-medium" :title="item.name">{{ item.name }}</span>
                    <span class="shrink-0 text-xs text-muted-foreground">{{ formatPercent(item.shareOfTotal) }}</span>
                  </div>
                  <Progress
                    :model-value="Math.min(100, Math.max(0, item.shareOfTotal * 100))"
                    :aria-label="`${item.name}: ${formatPercent(item.shareOfTotal)}`"
                  />
                  <p class="text-xs leading-relaxed text-muted-foreground">
                    {{ formatDuration(item.watchedSeconds) }} · {{ t("insights.movieCount", { count: formatInteger(item.movieCount) }) }}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

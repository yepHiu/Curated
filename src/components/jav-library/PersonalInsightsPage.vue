<script setup lang="ts">
import type { Component } from "vue"
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import {
  ChartNoAxesColumnIncreasing,
  ChevronDown,
  CircleCheckBig,
  Clock3,
  Film,
  Play,
  RotateCw,
  Sparkles,
  Star,
} from "lucide-vue-next"
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
  CardFooter,
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
  emphasis: boolean
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
const breakdownErrors = ref<Record<PersonalInsightsDimension, boolean>>({
  actor: false,
  studio: false,
  tag: false,
})
const breakdownPending = ref<Record<PersonalInsightsDimension, boolean>>({
  actor: false,
  studio: false,
  tag: false,
})
const narrative = ref("")
const narrativeBusy = ref(false)
const narrativeError = ref("")
const expandedBreakdowns = ref<Record<PersonalInsightsDimension, boolean>>({
  actor: false,
  studio: false,
  tag: false,
})
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
      hint: t("insights.watchedShort"),
      icon: Clock3,
      emphasis: true,
    },
    {
      key: "started",
      label: t("insights.startedMovies"),
      value: formatInteger(value.startedMovies),
      hint: t("insights.startedShort"),
      icon: Play,
      emphasis: true,
    },
    {
      key: "completion-rate",
      label: t("insights.completionRate"),
      value: value.completionRate === null ? "—" : formatPercent(value.completionRate),
      hint: value.completionRate === null
        ? t("insights.noCompletionDenominator")
        : t("insights.completionRateShort"),
      icon: ChartNoAxesColumnIncreasing,
      emphasis: true,
    },
    {
      key: "completed",
      label: t("insights.completedMovies"),
      value: formatInteger(value.completedMovies),
      hint: t("insights.completedShort", { threshold: formatPercent(value.completionThreshold) }),
      icon: CircleCheckBig,
      emphasis: false,
    },
    {
      key: "rated",
      label: t("insights.ratedMovies"),
      value: formatInteger(value.ratedMovies),
      hint: t("insights.ratedShort"),
      icon: Film,
      emphasis: false,
    },
    {
      key: "average-rating",
      label: t("insights.averageRating"),
      value: formatRating(value.averageUserRating),
      hint: value.averageUserRating === null
        ? t("insights.noRatingDenominator")
        : t("insights.averageRatingShort"),
      icon: Star,
      emphasis: false,
    },
  ]
})

const isEmpty = computed(() => overview.value?.watchedSeconds === 0)
const displayedRangeIsStale = computed(() => overview.value !== null && overview.value.range !== selectedRange.value)
const factualSummary = computed(() => {
  const value = overview.value
  const topActor = breakdowns.value.actor?.items[0]
  if (
    !value || displayedRangeIsStale.value || value.startedMovies < 5 ||
    value.watchedSeconds < 3600 || !topActor || topActor.shareOfTotal < 0.5
  ) return ""
  return t("insights.topActorFact", { name: topActor.name, share: formatPercent(topActor.shareOfTotal) })
})

function visibleBreakdownItems(dimension: PersonalInsightsDimension) {
  const items = breakdowns.value[dimension]?.items ?? []
  return expandedBreakdowns.value[dimension] ? items : items.slice(0, 5)
}

function toggleBreakdown(dimension: PersonalInsightsDimension) {
  expandedBreakdowns.value[dimension] = !expandedBreakdowns.value[dimension]
}

async function loadInsights() {
  const sequence = ++requestSequence
  loading.value = true
  loadError.value = false
  breakdownPending.value = { actor: false, studio: false, tag: false }
  expandedBreakdowns.value = { actor: false, studio: false, tag: false }
  narrative.value = ""
  narrativeError.value = ""
  const params = { range: selectedRange.value, timezone }
  const breakdownRequests = breakdownDefinitions.map(({ dimension }) =>
    libraryService.getPersonalInsightsBreakdown({ ...params, dimension, limit: 10 }).then(
      (dto) => ({ dimension, dto, failed: false }),
      () => ({ dimension, dto: null, failed: true }),
    ),
  )
  try {
    const nextOverview = await libraryService.getPersonalInsightsOverview(params)
    if (sequence !== requestSequence) return
    overview.value = nextOverview
    breakdowns.value = { actor: null, studio: null, tag: null }
    breakdownErrors.value = { actor: false, studio: false, tag: false }
    breakdownPending.value = { actor: true, studio: true, tag: true }
    for (const request of breakdownRequests) {
      void request.then(({ dimension, dto, failed }) => {
        if (sequence !== requestSequence) return
        breakdowns.value = { ...breakdowns.value, [dimension]: dto }
        breakdownErrors.value = { ...breakdownErrors.value, [dimension]: failed }
        breakdownPending.value = { ...breakdownPending.value, [dimension]: false }
      })
    }
  } catch {
    if (sequence !== requestSequence) return
    loadError.value = true
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

async function retryBreakdown(dimension: PersonalInsightsDimension) {
  const range = overview.value?.range
  if (!range || breakdownPending.value[dimension] || displayedRangeIsStale.value) return
  const sequence = requestSequence
  breakdownPending.value = { ...breakdownPending.value, [dimension]: true }
  breakdownErrors.value = { ...breakdownErrors.value, [dimension]: false }
  try {
    const dto = await libraryService.getPersonalInsightsBreakdown({ range, timezone, dimension, limit: 10 })
    if (sequence !== requestSequence) return
    breakdowns.value = { ...breakdowns.value, [dimension]: dto }
  } catch {
    if (sequence !== requestSequence) return
    breakdownErrors.value = { ...breakdownErrors.value, [dimension]: true }
  } finally {
    if (sequence === requestSequence) {
      breakdownPending.value = { ...breakdownPending.value, [dimension]: false }
    }
  }
}

watch(selectedRange, () => void loadInsights(), { immediate: true })
onBeforeUnmount(() => { requestSequence += 1 })

watch(selectedRange, cancelAIAction)
watch(agentEnabled, cancelAIAction)

async function generateNarrative() {
  if (!agentEnabled.value || narrativeBusy.value || !overview.value || loading.value || displayedRangeIsStale.value) {
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
    <main class="mx-auto flex w-full max-w-7xl flex-col gap-6 px-3 pb-8 pt-5 sm:px-6 lg:px-8">
      <header class="flex flex-col gap-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="flex max-w-3xl flex-col gap-2">
            <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">
              {{ t("insights.title") }}
            </h1>
          </div>
          <div v-if="agentEnabled" class="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              class="min-h-11 rounded-xl sm:min-h-9"
              :disabled="narrativeBusy || loading || loadError || !overview || displayedRangeIsStale"
              data-insights-ai-readout
              @click="generateNarrative"
            >
              <Sparkles data-icon="inline-start" />
              {{ t("insights.aiReadout") }}
            </Button>
            <Button v-if="aiActionPending" type="button" variant="ghost" size="sm" data-ai-action-cancel @click="cancelAIAction">{{ t("common.cancel") }}</Button>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-x-5 gap-y-3 rounded-2xl border border-border/70 bg-card/70 p-3 sm:p-4">
          <span class="text-sm font-medium text-muted-foreground">{{ t("insights.rangeControl") }}</span>
          <fieldset class="grid w-full grid-cols-2 gap-1 rounded-xl border border-border/70 bg-background/70 p-1 sm:w-auto sm:grid-cols-4" data-insights-range-selector>
            <legend class="sr-only">{{ t("insights.rangeLegend") }}</legend>
            <label
              v-for="option in rangeOptions"
              :key="option.value"
              class="relative flex min-h-11 cursor-pointer items-center justify-center rounded-lg px-3 text-center text-sm font-medium text-muted-foreground transition-colors hover:bg-accent/70 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring/60 has-[:checked]:bg-primary has-[:checked]:text-primary-foreground sm:min-h-9"
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
          <p v-if="overview" class="min-w-0 text-xs leading-relaxed text-muted-foreground xl:ml-auto">
            <span class="font-medium text-foreground">{{ t("insights.rangeDates", { from: overview.from, to: overview.to }) }}</span>
            <br class="hidden xl:block" />
            <span>{{ t("insights.rangeTimezone", { timezone: overview.timezone }) }}</span>
            <template v-if="overview.dataSince"> · {{ t("insights.dataSince", { date: overview.dataSince }) }}</template>
          </p>
        </div>

        <p v-if="narrativeError" class="text-sm text-destructive" role="alert">{{ narrativeError }}</p>
        <Card v-if="narrative" data-insights-ai-narrative class="gap-2 border-border/70">
          <CardHeader class="pb-0">
            <CardTitle class="text-base">{{ t("insights.aiReadoutTitle") }}</CardTitle>
          </CardHeader>
          <CardContent class="whitespace-pre-wrap text-sm leading-relaxed">
            {{ narrative }}
          </CardContent>
        </Card>
      </header>

      <div
        v-if="overview && (loading || loadError)"
        data-insights-update-status
        :role="loadError ? 'alert' : 'status'"
        class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/30 px-4 py-3 text-sm"
      >
        <span>{{ loadError
          ? t("insights.previousRangeFailed")
          : displayedRangeIsStale
            ? t("insights.showingPreviousRange")
            : t("insights.updatingRange") }}</span>
        <Button v-if="loadError" type="button" variant="secondary" size="sm" class="min-h-11 sm:min-h-9" data-insights-retry-overview @click="loadInsights">
          <RotateCw data-icon="inline-start" />
          {{ t("insights.retry") }}
        </Button>
      </div>

      <div v-if="loading && !overview" class="flex flex-col gap-6" role="status" aria-live="polite">
        <span class="sr-only">{{ t("insights.loading") }}</span>
        <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
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

      <Card v-else-if="loadError && !overview" role="alert" class="border-border/80">
        <CardHeader>
          <CardTitle>{{ t("insights.errorTitle") }}</CardTitle>
          <CardDescription>{{ t("insights.errorDescription") }}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button type="button" variant="secondary" class="min-h-11 rounded-2xl" data-insights-retry-overview @click="loadInsights">
            <RotateCw data-icon="inline-start" />
            {{ t("insights.retry") }}
          </Button>
        </CardContent>
      </Card>

      <template v-else-if="overview">
        <section class="flex flex-col gap-3" aria-labelledby="insights-summary-heading">
          <div class="flex flex-wrap items-baseline justify-between gap-2">
            <h2 id="insights-summary-heading" class="text-lg font-semibold tracking-tight">
              {{ t("insights.summaryTitle") }}
            </h2>
            <p class="text-xs text-muted-foreground">{{ t("insights.summaryScope") }}</p>
          </div>
          <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
            <Card
              v-for="metric in metricCards"
              :key="metric.key"
              :data-insights-metric="metric.key"
              :data-emphasis="metric.emphasis ? '' : undefined"
              class="min-w-0 gap-2 p-4 data-[emphasis]:border-primary/30 data-[emphasis]:bg-primary/5 sm:p-5"
            >
              <CardHeader class="grid-cols-[minmax(0,1fr)_auto] items-start gap-2 p-0">
                <CardDescription class="text-xs font-medium sm:text-sm">{{ metric.label }}</CardDescription>
                <div class="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <component :is="metric.icon" class="size-4" aria-hidden="true" />
                </div>
              </CardHeader>
              <CardContent class="min-w-0 p-0">
                <CardTitle class="break-words text-2xl leading-tight sm:text-3xl">{{ metric.value }}</CardTitle>
                <p class="mt-2 text-xs leading-relaxed text-muted-foreground">{{ metric.hint }}</p>
              </CardContent>
            </Card>
          </div>
          <p v-if="factualSummary" data-insights-fact class="rounded-xl border border-border/60 bg-muted/30 px-4 py-3 text-sm leading-relaxed text-foreground">
            {{ factualSummary }}
          </p>
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
          <h2 id="insights-breakdown-heading" class="text-lg font-semibold tracking-tight">
            {{ t("insights.breakdownTitle") }}
          </h2>

          <div class="grid gap-4 lg:grid-cols-3">
            <Card
              v-for="definition in breakdownDefinitions"
              :key="definition.dimension"
              :data-insights-breakdown="definition.dimension"
              class="min-w-0 gap-3"
            >
              <CardHeader class="grid-cols-[minmax(0,1fr)_auto] items-start gap-2 pb-0">
                <div class="flex min-w-0 flex-col gap-1">
                  <CardTitle class="text-base">{{ t(definition.titleKey) }}</CardTitle>
                  <CardDescription class="text-xs">{{ t(definition.descriptionKey) }}</CardDescription>
                </div>
                <span v-if="breakdowns[definition.dimension]" class="shrink-0 text-xs text-muted-foreground">
                  {{ t("insights.itemCount", { count: breakdowns[definition.dimension]?.items.length ?? 0 }) }}
                </span>
              </CardHeader>
              <CardContent class="flex flex-col gap-3">
                <div v-if="breakdownPending[definition.dimension]" class="flex flex-col gap-3" role="status">
                  <span class="sr-only">{{ t("insights.breakdownLoading", { dimension: t(definition.titleKey) }) }}</span>
                  <Skeleton v-for="row in 5" :key="row" class="h-10 w-full" />
                </div>
                <div v-else-if="breakdownErrors[definition.dimension]" class="flex flex-col items-start gap-3 py-4" role="alert">
                  <p class="text-sm text-muted-foreground">{{ t("insights.breakdownError", { dimension: t(definition.titleKey) }) }}</p>
                  <Button type="button" variant="secondary" size="sm" class="min-h-11 sm:min-h-9" :data-insights-retry-breakdown="definition.dimension" @click="retryBreakdown(definition.dimension)">
                    <RotateCw data-icon="inline-start" />
                    {{ t("insights.retry") }}
                  </Button>
                </div>
                <p
                  v-else-if="breakdowns[definition.dimension]?.items.length === 0"
                  class="py-6 text-center text-sm text-muted-foreground"
                >
                  {{ t("insights.noBreakdown") }}
                </p>
                <div
                  v-for="item in breakdownPending[definition.dimension] || breakdownErrors[definition.dimension] ? [] : visibleBreakdownItems(definition.dimension)"
                  :key="item.name"
                  class="flex min-w-0 flex-col gap-2 border-t border-border/50 pt-3 first:border-t-0 first:pt-0"
                >
                  <div class="flex min-w-0 items-start justify-between gap-3">
                    <span class="min-w-0 truncate text-sm font-medium" :title="item.name">{{ item.name }}</span>
                    <span class="shrink-0 text-xs text-muted-foreground">{{ formatPercent(item.shareOfTotal) }}</span>
                  </div>
                  <Progress
                    class="h-1.5"
                    :model-value="Math.min(100, Math.max(0, item.shareOfTotal * 100))"
                    :aria-label="`${item.name}: ${formatPercent(item.shareOfTotal)}`"
                  />
                  <p class="text-xs leading-relaxed text-muted-foreground">
                    {{ formatDuration(item.watchedSeconds) }} · {{ t("insights.movieCount", { count: formatInteger(item.movieCount) }) }}
                  </p>
                </div>
              </CardContent>
              <CardFooter v-if="(breakdowns[definition.dimension]?.items.length ?? 0) > 5" class="pt-0">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  class="min-h-11 w-full rounded-xl sm:min-h-9"
                  :aria-expanded="expandedBreakdowns[definition.dimension]"
                  :data-insights-expand="definition.dimension"
                  @click="toggleBreakdown(definition.dimension)"
                >
                  {{ expandedBreakdowns[definition.dimension]
                    ? t("insights.showLess")
                    : t("insights.showAll", { count: breakdowns[definition.dimension]?.items.length ?? 0 }) }}
                  <ChevronDown data-icon="inline-end" :class="{ 'rotate-180': expandedBreakdowns[definition.dimension] }" />
                </Button>
              </CardFooter>
            </Card>
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

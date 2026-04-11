<script setup lang="ts">
import { computed } from "vue"
import { ChevronDown, ChevronUp, ClipboardCopy, Cpu, Pause, Play, RotateCcw, X } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { useDevPerformanceMonitor } from "@/composables/use-dev-performance-monitor"
import { devPerformanceBarHidden, setDevPerformanceBarHidden } from "@/lib/dev-performance/visibility"

const monitor = useDevPerformanceMonitor()
const hidden = devPerformanceBarHidden

function formatNumber(value: number | null | undefined, digits = 1, suffix = ""): string {
  if (value == null) {
    return "n/a"
  }
  return `${Number(value.toFixed(digits))}${suffix}`
}

const backendStatusLabel = computed(() => {
  if (monitor.backendHealthStatus.value === "mock") {
    return "mock"
  }
  if (monitor.backendHealthStatus.value === "online") {
    return `online ${monitor.backendHealthLatencyMs.value ?? "n/a"}ms`
  }
  if (monitor.backendHealthStatus.value === "checking") {
    return "checking"
  }
  return "offline"
})

const systemCpuLabel = computed(() =>
  monitor.backendPerformance.value?.supported
    ? formatNumber(monitor.backendPerformance.value.systemCpuPercent ?? null, 1, "%")
    : "n/a",
)

const backendCpuLabel = computed(() =>
  monitor.backendPerformance.value?.supported
    ? formatNumber(monitor.backendPerformance.value.backendCpuPercent ?? null, 1, "%")
    : "n/a",
)

const decodeLabel = computed(() => {
  const video = monitor.frontendSnapshot.value.video
  if (!video.available) {
    return "decode n/a"
  }
  return `decode ${formatNumber(video.estimatedFps, 2)}fps / wait ${video.waitingCount30s}`
})

const requestSummaryLabel = computed(
  () =>
    `${monitor.requestSnapshot.value.requestCount30s} req / ${monitor.requestSnapshot.value.failedRequestCount30s} fail`,
)

const routeLabel = computed(() => monitor.frontendSnapshot.value.routeName || "unknown")

const recentRequests = computed(() => monitor.requestSnapshot.value.recentRequests.slice(0, 12))
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed inset-x-0 bottom-0 z-[95] px-2 pb-2 sm:px-3">
      <div class="pointer-events-auto mx-auto flex w-full max-w-[1200px] flex-col gap-2">
        <section
          v-if="!hidden && monitor.expanded.value"
          class="overflow-hidden rounded-[1.4rem] border border-slate-900/10 bg-background/94 shadow-[0_-14px_40px_rgba(15,23,42,0.16)] backdrop-blur-md dark:border-white/10"
        >
          <div class="flex flex-wrap items-start justify-between gap-3 border-b border-border/70 px-4 py-3">
            <div class="min-w-0">
              <p class="text-sm font-semibold text-foreground">Dev Performance Monitor</p>
              <p class="text-xs text-muted-foreground">
                {{ routeLabel }} · {{ monitor.backendVersion.value ?? "backend version n/a" }}
              </p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="secondary"
                size="sm"
                class="rounded-full"
                @click.stop="monitor.togglePaused"
              >
                <Pause v-if="!monitor.paused.value" class="size-4" />
                <Play v-else class="size-4" />
                {{ monitor.paused.value ? "Resume" : "Pause" }}
              </Button>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                class="rounded-full"
                @click.stop="monitor.clearStats"
              >
                <RotateCcw class="size-4" />
                Clear
              </Button>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                class="rounded-full"
                @click.stop="monitor.copySummary"
              >
                <ClipboardCopy class="size-4" />
                Copy Summary
              </Button>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                class="rounded-full"
                aria-label="Hide performance monitor"
                @click.stop="setDevPerformanceBarHidden(true)"
              >
                <X class="size-4" />
                Hide
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                class="rounded-full"
                aria-label="Collapse monitor"
                @click.stop="monitor.toggleExpanded"
              >
                <ChevronDown class="size-4" />
              </Button>
            </div>
          </div>

          <div class="grid gap-3 px-4 py-4 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-2xl border border-border/70 bg-background/70 p-3">
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Frontend</p>
              <div class="mt-3 space-y-1.5 text-sm">
                <p>FPS: {{ formatNumber(monitor.frontendSnapshot.value.fps, 1) }}</p>
                <p>Long Tasks 30s: {{ monitor.frontendSnapshot.value.longTaskCount30s }}</p>
                <p>Memory: {{ formatNumber(monitor.frontendSnapshot.value.memoryUsedMB, 1, " MB") }}</p>
                <p>Route Switch: {{ formatNumber(monitor.frontendSnapshot.value.lastRouteChangeMs, 0, " ms") }}</p>
              </div>
            </div>

            <div class="rounded-2xl border border-border/70 bg-background/70 p-3">
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Decode</p>
              <div class="mt-3 space-y-1.5 text-sm">
                <p>Video Metrics: {{ monitor.frontendSnapshot.value.video.available ? "available" : "n/a" }}</p>
                <p>Video FPS: {{ formatNumber(monitor.frontendSnapshot.value.video.estimatedFps, 2) }}</p>
                <p>
                  Dropped Frames:
                  {{ monitor.frontendSnapshot.value.video.droppedFrames ?? "n/a" }}/{{ monitor.frontendSnapshot.value.video.totalFrames ?? "n/a" }}
                </p>
                <p>
                  Dropped Ratio:
                  {{ formatNumber(monitor.frontendSnapshot.value.video.droppedFrameRatePercent, 2, "%") }}
                </p>
                <p>Waiting 30s: {{ monitor.frontendSnapshot.value.video.waitingCount30s }}</p>
              </div>
            </div>

            <div class="rounded-2xl border border-border/70 bg-background/70 p-3">
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Requests</p>
              <div class="mt-3 space-y-1.5 text-sm">
                <p>Total 30s: {{ monitor.requestSnapshot.value.requestCount30s }}</p>
                <p>Failed 30s: {{ monitor.requestSnapshot.value.failedRequestCount30s }}</p>
                <p>Active: {{ monitor.requestSnapshot.value.activeRequestCount }}</p>
                <p>Avg Latency: {{ formatNumber(monitor.requestSnapshot.value.avgLatencyMs30s, 0, " ms") }}</p>
              </div>
            </div>

            <div class="rounded-2xl border border-border/70 bg-background/70 p-3">
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Backend</p>
              <div class="mt-3 space-y-1.5 text-sm">
                <p>Status: {{ backendStatusLabel }}</p>
                <p>System CPU: {{ systemCpuLabel }}</p>
                <p>Backend CPU: {{ backendCpuLabel }}</p>
                <p>Mode: {{ monitor.useWebApi ? "web api" : "mock" }}</p>
              </div>
            </div>
          </div>

          <div class="border-t border-border/70 px-4 py-4">
            <div class="mb-2 flex items-center justify-between gap-3">
              <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
                Recent Requests
              </p>
              <p class="text-xs text-muted-foreground">
                latest {{ recentRequests.length }} / {{ monitor.requestSnapshot.value.recentRequests.length }}
              </p>
            </div>

            <div
              v-if="recentRequests.length"
              class="max-h-56 overflow-auto rounded-2xl border border-border/70"
            >
              <div
                v-for="request in recentRequests"
                :key="request.id"
                class="grid grid-cols-[auto_minmax(0,1fr)_auto_auto] items-center gap-3 border-b border-border/60 px-3 py-2 text-xs last:border-b-0"
              >
                <span class="rounded-full bg-muted px-2 py-0.5 font-mono font-semibold text-foreground">
                  {{ request.method }}
                </span>
                <span class="truncate font-mono text-muted-foreground" :title="request.path">
                  {{ request.path }}
                </span>
                <span
                  class="rounded-full px-2 py-0.5 font-mono"
                  :class="request.failed ? 'bg-red-500/12 text-red-700 dark:text-red-300' : 'bg-emerald-500/12 text-emerald-700 dark:text-emerald-300'"
                >
                  {{ request.status ?? "ERR" }}
                </span>
                <span class="font-mono text-foreground">{{ request.durationMs }}ms</span>
              </div>
            </div>

            <div
              v-else
              class="rounded-2xl border border-dashed border-border/70 px-3 py-5 text-center text-sm text-muted-foreground"
            >
              No captured requests yet.
            </div>
          </div>
        </section>

        <div v-if="!hidden" class="flex items-center gap-2">
          <button
            type="button"
            class="flex h-10 min-w-0 flex-1 items-center justify-between gap-3 rounded-full border border-slate-900/10 bg-background/92 px-4 text-left shadow-[0_10px_30px_rgba(15,23,42,0.12)] backdrop-blur-md transition-colors hover:border-primary/40 dark:border-white/10"
            :aria-expanded="monitor.expanded.value"
            @click="monitor.toggleExpanded"
          >
            <div class="flex min-w-0 items-center gap-3 text-sm">
              <span class="inline-flex items-center gap-2 font-semibold text-foreground">
                <Cpu class="size-4 text-primary" />
                Perf
              </span>
              <span class="truncate text-muted-foreground">{{ routeLabel }}</span>
              <span class="hidden text-muted-foreground sm:inline">
                {{ requestSummaryLabel }}
              </span>
              <span class="hidden text-muted-foreground md:inline">
                {{ backendStatusLabel }}
              </span>
              <span class="hidden text-muted-foreground xl:inline">
                {{ decodeLabel }}
              </span>
            </div>

            <div class="flex items-center gap-3 text-xs text-muted-foreground">
              <span>CPU {{ systemCpuLabel }}</span>
              <span>FPS {{ formatNumber(monitor.frontendSnapshot.value.fps, 1) }}</span>
              <span>{{ monitor.paused.value ? "paused" : "live" }}</span>
              <ChevronUp v-if="monitor.expanded.value" class="size-4" />
              <ChevronUp v-else class="size-4 rotate-180" />
            </div>
          </button>

          <button
            type="button"
            class="flex size-10 shrink-0 items-center justify-center rounded-full border border-slate-900/10 bg-background/92 text-muted-foreground shadow-[0_10px_30px_rgba(15,23,42,0.12)] backdrop-blur-md transition-colors hover:border-primary/40 hover:text-foreground dark:border-white/10"
            aria-label="Hide performance monitor"
            title="Hide performance monitor"
            @click="setDevPerformanceBarHidden(true)"
          >
            <X class="size-4" />
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

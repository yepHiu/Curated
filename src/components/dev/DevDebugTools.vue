<script setup lang="ts">
import { computed, ref } from "vue"
import { ClipboardCopy, Pause, Play, RotateCcw } from "lucide-vue-next"
import { Button } from "@/components/ui/button"
import { useDevPerformanceMonitor } from "@/composables/use-dev-performance-monitor"

import DevDebugDialog from "./DevDebugDialog.vue"

const debugOpen = defineModel<boolean>("debugOpen", { default: false })
const debugTab = ref("actions")
const monitor = useDevPerformanceMonitor()

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

const routeLabel = computed(() => monitor.frontendSnapshot.value.routeName || "unknown")

const recentRequests = computed(() => monitor.requestSnapshot.value.recentRequests.slice(0, 12))
</script>

<template>
  <DevDebugDialog v-model:open="debugOpen" v-model:tab="debugTab">
    <template #performance>
      <section
        class="overflow-hidden rounded-xl border border-border bg-card"
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
          </div>
        </div>

        <div class="grid gap-3 px-4 py-4 md:grid-cols-2">
          <div class="rounded-lg border border-border/70 bg-background/70 p-3">
            <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Frontend</p>
            <div class="mt-3 space-y-1.5 text-sm">
              <p>FPS: {{ formatNumber(monitor.frontendSnapshot.value.fps, 1) }}</p>
              <p>Long Tasks 30s: {{ monitor.frontendSnapshot.value.longTaskCount30s }}</p>
              <p>Memory: {{ formatNumber(monitor.frontendSnapshot.value.memoryUsedMB, 1, " MB") }}</p>
              <p>Route Switch: {{ formatNumber(monitor.frontendSnapshot.value.lastRouteChangeMs, 0, " ms") }}</p>
            </div>
          </div>

          <div class="rounded-lg border border-border/70 bg-background/70 p-3">
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

          <div class="rounded-lg border border-border/70 bg-background/70 p-3">
            <p class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">Requests</p>
            <div class="mt-3 space-y-1.5 text-sm">
              <p>Total 30s: {{ monitor.requestSnapshot.value.requestCount30s }}</p>
              <p>Failed 30s: {{ monitor.requestSnapshot.value.failedRequestCount30s }}</p>
              <p>Active: {{ monitor.requestSnapshot.value.activeRequestCount }}</p>
              <p>Avg Latency: {{ formatNumber(monitor.requestSnapshot.value.avgLatencyMs30s, 0, " ms") }}</p>
            </div>
          </div>

          <div class="rounded-lg border border-border/70 bg-background/70 p-3">
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
            class="max-h-56 overflow-auto rounded-lg border border-border/70"
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
            class="rounded-lg border border-dashed border-border/70 px-3 py-5 text-center text-sm text-muted-foreground"
          >
            No captured requests yet.
          </div>
        </div>
      </section>
    </template>
  </DevDebugDialog>
</template>

export interface DevPerformanceSummaryInput {
  routeName: string
  fps: number | null
  longTaskCount30s: number
  memoryUsedMB: number | null
  requestCount30s: number
  failedRequestCount30s: number
  activeRequestCount: number
  avgLatencyMs30s: number | null
  backendHealthStatus: string
  backendHealthLatencyMs: number | null
  backendVersion: string | null
  systemCpuPercent: number | null
  backendCpuPercent: number | null
  videoAvailable: boolean
  videoWaitingCount30s: number
  videoDroppedFrames: number | null
  videoTotalFrames: number | null
  videoDroppedFrameRatePercent: number | null
  videoEstimatedFps: number | null
}

function formatNumber(value: number | null, digits = 1): string {
  return value == null ? "n/a" : Number(value.toFixed(digits)).toString()
}

function formatPercent(value: number | null, digits = 1): string {
  return value == null ? "n/a" : `${formatNumber(value, digits)}%`
}

export function buildDevPerformanceSummaryText(input: DevPerformanceSummaryInput): string {
  const decodeSummary = input.videoAvailable
    ? `Decode: fps=${formatNumber(input.videoEstimatedFps, 2)}, waiting30s=${input.videoWaitingCount30s}, dropped=${input.videoDroppedFrames ?? 0}/${input.videoTotalFrames ?? 0} (${formatPercent(input.videoDroppedFrameRatePercent, 2)})`
    : `Decode: unavailable, waiting30s=${input.videoWaitingCount30s}`

  return [
    `Route: ${input.routeName}`,
    `Frontend: fps=${formatNumber(input.fps, 1)}, longTasks30s=${input.longTaskCount30s}, memoryMB=${formatNumber(input.memoryUsedMB, 1)}`,
    `Requests: total30s=${input.requestCount30s}, failed30s=${input.failedRequestCount30s}, active=${input.activeRequestCount}, avgMs=${input.avgLatencyMs30s ?? "n/a"}`,
    `Backend: status=${input.backendHealthStatus}, latencyMs=${input.backendHealthLatencyMs ?? "n/a"}, version=${input.backendVersion ?? "n/a"}`,
    `CPU: system=${formatPercent(input.systemCpuPercent, 1)}, backend=${formatPercent(input.backendCpuPercent, 1)}`,
    decodeSummary,
  ].join("\n")
}

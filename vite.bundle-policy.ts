import { readFileSync, readdirSync, statSync } from "node:fs"
import path from "node:path"

export const metricKeys = ["initialRaw", "initialGzip", "jsRaw", "jsGzip", "cssRaw", "distRaw"] as const
export type Metric = typeof metricKeys[number]
export type Metrics = Record<Metric, number>
export interface Snapshot {
  schemaVersion: 2
  mode: "web" | "mock"
  metrics: Metrics
}
export interface Policy {
  schemaVersion: 1
  metrics: Record<Metric, { label: string; warn: number; fail: number; growthFloor: number }>
  growth: { recentRatio: number; reviewedRatio: number; reviewedFloorMultiplier: number }
  chunkWarningBytes: number
}
export interface Finding { level: "warning" | "error"; message: string }
export interface AssetSize { fileName: string; rawBytes: number }

/** 验证可比较的快照；损坏或旧版本数据不可被当成零基线。 */
export function isSnapshot(value: unknown): value is Snapshot {
  if (!value || typeof value !== "object") return false
  const snapshot = value as Partial<Snapshot>
  return snapshot.schemaVersion === 2 && (snapshot.mode === "web" || snapshot.mode === "mock") &&
    metricKeys.every((key) => {
      // 每个指标都必须存在且为有限的非负字节数。
      const size = snapshot.metrics?.[key]
      return typeof size === "number" && Number.isFinite(size) && size >= 0
    })
}

/** 加载版本化策略并拒绝颠倒或非法阈值，避免配置错误悄悄关闭监管。 */
export function loadPolicy(root: string): Policy {
  const policy: Policy = JSON.parse(readFileSync(path.join(root, "bundle-policy.json"), "utf8"))
  if (policy.schemaVersion !== 1) throw new Error("Unsupported bundle policy schema")
  for (const key of metricKeys) {
    const rule = policy.metrics?.[key]
    if (!rule || !rule.label || !Number.isFinite(rule.warn) || !Number.isFinite(rule.fail) ||
      !Number.isFinite(rule.growthFloor) || rule.warn <= 0 || rule.fail <= rule.warn || rule.growthFloor <= 0) {
      throw new Error(`Invalid bundle policy metric: ${key}`)
    }
  }
  for (const value of [policy.growth?.recentRatio, policy.growth?.reviewedRatio,
    policy.growth?.reviewedFloorMultiplier, policy.chunkWarningBytes]) {
    if (!Number.isFinite(value) || value <= 0) throw new Error("Invalid bundle growth policy")
  }
  return policy
}

/** 基线缺失、损坏或构建模式不同只影响比较；绝对上限仍然执行。 */
export function loadSnapshot(file: string, mode: Snapshot["mode"]): Snapshot | undefined {
  try {
    const value: unknown = JSON.parse(readFileSync(file, "utf8"))
    return isSnapshot(value) && value.mode === mode
      ? { schemaVersion: 2, mode: value.mode, metrics: value.metrics } : undefined
  } catch {
    return undefined
  }
}

/** 使用十进制字节单位，便于与 Vite 构建输出直接比较。 */
export function formatBytes(bytes: number): string {
  return bytes >= 1_000_000 ? `${(bytes / 1_000_000).toFixed(2)} MB` : `${(bytes / 1_000).toFixed(2)} kB`
}

/** 分级评估：只有绝对严重上限阻断，单次与累计增长均只提醒。 */
export function evaluateSize(current: Snapshot, policy: Policy, recent?: Snapshot, reviewed?: Snapshot): Finding[] {
  if (!isSnapshot(current)) throw new Error("Invalid current bundle metrics")
  const findings: Finding[] = []
  for (const key of metricKeys) {
    const value = current.metrics[key]
    const rule = policy.metrics[key]
    if (value > rule.fail) {
      findings.push({ level: "error", message: `${rule.label} ${formatBytes(value)} 超过严重上限 ${formatBytes(rule.fail)}，请减少体积后构建。` })
    } else if (value > rule.warn) {
      findings.push({ level: "warning", message: `${rule.label} ${formatBytes(value)} 超过关注线 ${formatBytes(rule.warn)}，请安排体积优化。` })
    }
    for (const [label, baseline, ratio, floor] of [
      ["最近主分支成功构建", recent, policy.growth.recentRatio, rule.growthFloor],
      ["人工确认基线", reviewed, policy.growth.reviewedRatio, rule.growthFloor * policy.growth.reviewedFloorMultiplier],
    ] as const) {
      if (!baseline || baseline.mode !== current.mode) continue
      const delta = value - baseline.metrics[key]
      if (delta > Math.max(baseline.metrics[key] * ratio, floor)) {
        findings.push({ level: "warning", message: `${rule.label} 相比${label}增长 ${formatBytes(delta)}，超过增长观察线；请检查新增依赖或资源。` })
      }
    }
  }
  return findings
}

/** 遍历最终输出，包含 public 复制文件，排除监管报告自身，避免自我累计。 */
export function collectAssets(directory: string, prefix = ""): AssetSize[] {
  const rows: AssetSize[] = []
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const fileName = prefix + entry.name
    const absolute = path.join(directory, entry.name)
    if (entry.isDirectory()) rows.push(...collectAssets(absolute, `${fileName}/`))
    else if (entry.isFile() && !["bundle-analysis.json", "bundle-report.md"].includes(fileName)) {
      rows.push({ fileName, rawBytes: statSync(absolute).size })
    }
  }
  return rows.sort((left, right) => {
    // 大资源优先展示，帮助定位字体、图片和依赖占用。
    return right.rawBytes - left.rawBytes
  })
}

/** 展示绝对线与两种基线，提醒必须给出可采取的下一步。 */
export function renderSummary(current: Snapshot, policy: Policy, findings: Finding[], assets: AssetSize[], recent?: Snapshot, reviewed?: Snapshot): string {
  const lines = ["# Curated 构建体积报告", "", `构建模式：${current.mode}。仅严重超限阻断；增长提醒不阻断。`, "",
    "| 指标 | 当前 | 最近主分支 | 人工确认基线 | 关注线 | 严重上限 |", "|---|---:|---:|---:|---:|---:|"]
  for (const key of metricKeys) {
    const rule = policy.metrics[key]
    lines.push(`| ${rule.label} | ${formatBytes(current.metrics[key])} | ${recent ? formatBytes(recent.metrics[key]) : "不可用"} | ${reviewed ? formatBytes(reviewed.metrics[key]) : "不可用"} | ${formatBytes(rule.warn)} | ${formatBytes(rule.fail)} |`)
  }
  lines.push("", "## 检查结果", "")
  if (!recent) lines.push("- 最近主分支基线不可用：本次仍检查绝对上限和人工确认基线。")
  if (!reviewed) lines.push("- 同模式人工确认基线不可用：累计增长比较未执行。")
  if (findings.length === 0) lines.push("- 体积处于允许范围，无需因本次体积增长进行优化。")
  for (const finding of findings) lines.push(`- **${finding.level === "error" ? "阻断" : "提醒"}**：${finding.message}`)
  lines.push("", "## 最大资源", "", "| 文件 | 原始大小 |", "|---|---:|")
  for (const asset of assets.slice(0, 15)) lines.push(`| ${asset.fileName} | ${formatBytes(asset.rawBytes)} |`)
  lines.push("", "完整分块与模块占用见 bundle-analysis.json。优先检查新增依赖、重复资源和首屏静态导入；合理功能增长无需立即压缩。人工基线只在审阅后运行 pnpm bundle:baseline 更新，CI 不会自动抬高严重上限。", "",
    "范围：Vue 前端输出；gzip 是逐 JS 文件压缩估算，不代表真实网络传输。Electron、Go、FFmpeg 和安装包体积不包含在此报告中。", "")
  return lines.join("\n")
}

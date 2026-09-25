// @vitest-environment node
import { mkdtempSync, mkdirSync, readFileSync, realpathSync, rmSync, writeFileSync } from "node:fs"
import os from "node:os"
import path from "node:path"
import { afterEach, describe, expect, it } from "vitest"
import { build } from "vite"
import { bundleBudgetPlugin } from "../../vite.bundle-budget"
import { evaluateSize, loadPolicy, loadSnapshot, metricKeys } from "../../vite.bundle-policy"
import type { Metrics, Snapshot } from "../../vite.bundle-policy"

const policy = loadPolicy(process.cwd())
const temporaryDirectories: string[] = []

/** 创建低于所有观察线的完整快照，以单独验证某项增长。 */
function snapshot(overrides: Partial<Metrics> = {}, mode: Snapshot["mode"] = "web"): Snapshot {
  return { schemaVersion: 2, mode, metrics: {
    initialRaw: 400_000, initialGzip: 140_000, jsRaw: 2_000_000,
    jsGzip: 650_000, cssRaw: 300_000, distRaw: 30_000_000, ...overrides,
  } }
}

/** 在系统临时目录搭建独立构建，避免污染真实产物和基线。 */
function fixture() {
  const root = realpathSync(mkdtempSync(path.join(os.tmpdir(), "curated-bundle-test-")))
  temporaryDirectories.push(root)
  mkdirSync(path.join(root, "public"))
  writeFileSync(path.join(root, "bundle-policy.json"), JSON.stringify(policy), "utf8")
  writeFileSync(path.join(root, ".env"), "VITE_USE_WEB_API=true\n", "utf8")
  writeFileSync(path.join(root, "index.html"), '<script type="module" src="/main.js"></script>', "utf8")
  writeFileSync(path.join(root, "main.js"), 'import "./static.js"; import "./style.css"; window.loadFeature = () => import("./feature.js");', "utf8")
  writeFileSync(path.join(root, "static.js"), 'window.staticValue = "eager";', "utf8")
  writeFileSync(path.join(root, "feature.js"), 'export const feature = "lazy";', "utf8")
  writeFileSync(path.join(root, "style.css"), "body { color: red; }", "utf8")
  writeFileSync(path.join(root, "public", "sample.bin"), Buffer.alloc(5_000, 42))
  return root
}

/** 所有测试结束均释放临时目录，包括预期失败的构建。 */
afterEach(() => {
  for (const root of temporaryDirectories.splice(0)) rmSync(root, { recursive: true, force: true })
})

/** 覆盖增长门槛、模式隔离和真实 Vite 生命周期。 */
describe("tiered bundle governance", () => {
  /** 合理增长、缩小及绝对字节门槛以下的噪声都不提示。 */
  it("allows modest growth and small absolute changes", () => {
    const baseline = snapshot()
    expect(evaluateSize(snapshot({ jsRaw: 2_200_000 }), policy, baseline, baseline)).toEqual([])
    expect(evaluateSize(snapshot({ jsRaw: 1_500_000 }), policy, baseline, baseline)).toEqual([])
    expect(evaluateSize(snapshot({ initialRaw: 120_000 }), policy, snapshot({ initialRaw: 50_000 }))).toEqual([])
  })

  /** 单次与累计增长只产生 warning，连续小增长仍会命中人工基线。 */
  it("warns on spikes and accumulated drift without failing", () => {
    const spike = evaluateSize(snapshot({ jsRaw: 2_500_000 }), policy, snapshot())
    expect(spike).toEqual([expect.objectContaining({ level: "warning", message: expect.stringContaining("最近主分支") })])
    const drift = evaluateSize(snapshot({ jsRaw: 3_100_000 }), policy, snapshot({ jsRaw: 3_000_000 }), snapshot())
    expect(drift).toEqual([expect.objectContaining({ level: "warning", message: expect.stringContaining("人工确认基线") })])
  })

  /** 对全部指标验证严格大于阈值才触发，严重上限不会随基线漂移。 */
  it("enforces absolute limits for every metric independently of baseline", () => {
    for (const key of metricKeys) {
      expect(evaluateSize(snapshot({ [key]: policy.metrics[key].warn }), policy)).toEqual([])
      expect(evaluateSize(snapshot({ [key]: policy.metrics[key].warn + 1 }), policy)[0]?.level).toBe("warning")
      expect(evaluateSize(snapshot({ [key]: policy.metrics[key].fail }), policy)[0]?.level).toBe("warning")
      const current = snapshot({ [key]: policy.metrics[key].fail + 1 })
      expect(evaluateSize(current, policy, current, current)[0]?.level).toBe("error")
    }
  })

  /** 缺失、旧版、损坏或不同服务模式的基线不应制造误报警。 */
  it("rejects incompatible baselines and invalid policy", () => {
    const root = fixture()
    const file = path.join(root, "baseline.json")
    expect(loadSnapshot(file, "web")).toBeUndefined()
    for (const content of ["invalid json", '{"schemaVersion":1}', JSON.stringify(snapshot({}, "mock"))]) {
      writeFileSync(file, content, "utf8")
      expect(loadSnapshot(file, "web")).toBeUndefined()
    }
    writeFileSync(file, JSON.stringify({ ...snapshot(), comparisons: { historical: "should not recurse" } }), "utf8")
    expect(loadSnapshot(file, "web")).toEqual(snapshot())
    expect(evaluateSize(snapshot(), policy, snapshot({ jsRaw: 1 }, "mock"))).toEqual([])
    writeFileSync(path.join(root, "bundle-policy.json"), JSON.stringify({ ...policy, metrics: {} }), "utf8")
    expect(() => {
      // 配置损坏必须显式报错，不静默跳过指标。
      loadPolicy(root)
    }).toThrow("Invalid bundle policy metric")
  })

  /** 实际构建统计 public/CSS/动态块；严重超限仍先留下两个报告。 */
  it("writes diagnostics before failing a real oversized Vite build", async () => {
    const root = fixture()
    const warningPolicy = structuredClone(policy)
    warningPolicy.metrics.distRaw.warn = 100
    writeFileSync(path.join(root, "bundle-policy.json"), JSON.stringify(warningPolicy), "utf8")
    await build({ root, configFile: false, logLevel: "silent", plugins: [bundleBudgetPlugin()] })
    const output = path.join(root, "dist")
    const report = JSON.parse(readFileSync(path.join(output, "bundle-analysis.json"), "utf8"))
    expect(report.mode).toBe("web")
    expect(report.findings).toContainEqual(expect.objectContaining({ level: "warning" }))
    expect(report.metrics.distRaw).toBeGreaterThan(5_000)
    expect(report.metrics.cssRaw).toBeGreaterThan(0)
    expect(report.assets).toContainEqual({ fileName: "sample.bin", rawBytes: 5_000 })
    expect(report.metrics.initialRaw).toBeLessThan(report.metrics.jsRaw)
    const strict = structuredClone(policy)
    strict.metrics.distRaw.warn = 100
    strict.metrics.distRaw.fail = 1_000
    writeFileSync(path.join(root, "bundle-policy.json"), JSON.stringify(strict), "utf8")
    await expect(build({ root, configFile: false, logLevel: "silent", plugins: [bundleBudgetPlugin()] })).rejects.toThrow("构建体积超过严重上限")
    const failedReport = JSON.parse(readFileSync(path.join(output, "bundle-analysis.json"), "utf8"))
    expect(failedReport.findings).toContainEqual(expect.objectContaining({ level: "error" }))
    expect(readFileSync(path.join(output, "bundle-report.md"), "utf8")).toContain("阻断")
    expect(failedReport.assets).not.toContainEqual(expect.objectContaining({ fileName: "bundle-analysis.json" }))
  })
})

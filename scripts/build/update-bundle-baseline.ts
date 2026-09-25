import { readFileSync, writeFileSync } from "node:fs"
import { execFileSync } from "node:child_process"
import { fileURLToPath } from "node:url"
import path from "node:path"
import { evaluateSize, isSnapshot, loadPolicy } from "../../vite.bundle-policy.ts"

/** 只接受真实 Web 构建报告；人工更新累计基线不会改变绝对阈值。 */
function updateBaseline() {
  const root = fileURLToPath(new URL("../../", import.meta.url))
  const report: unknown = JSON.parse(readFileSync(path.join(root, "dist/bundle-analysis.json"), "utf8"))
  if (!isSnapshot(report) || report.mode !== "web") throw new Error("Run a production Web API build before updating the baseline")
  const errors = evaluateSize(report, loadPolicy(root)).filter((finding) => {
    // 不允许通过更新基线洗掉绝对严重超限。
    return finding.level === "error"
  })
  if (errors.length) throw new Error("Cannot baseline an oversized build")
  const baseline = { schemaVersion: 2, mode: report.mode, metrics: report.metrics,
    generatedAt: new Date().toISOString(), sourceCommit: execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim() }
  writeFileSync(path.join(root, "bundle-baseline.json"), `${JSON.stringify(baseline, null, 2)}\n`, "utf8")
  console.log("Updated bundle-baseline.json. Review and commit it explicitly; absolute limits are unchanged.")
}
updateBaseline()

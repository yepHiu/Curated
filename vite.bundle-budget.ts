import path from "node:path"
import { appendFileSync, readFileSync, writeFileSync } from "node:fs"
import { gzipSync } from "node:zlib"
import type { Plugin, ResolvedConfig } from "vite"
import { collectAssets, evaluateSize, formatBytes, loadPolicy, loadSnapshot, renderSummary } from "./vite.bundle-policy.ts"
import type { Snapshot } from "./vite.bundle-policy.ts"

interface Size { rawBytes: number; gzipBytes: number }
interface Chunk extends Size {
  name: string
  fileName: string
  isEntry: boolean
  isDynamicEntry: boolean
  imports: string[]
  dynamicImports: string[]
  topModules: { id: string; renderedBytes: number }[]
}
interface Analysis {
  totals: { initialJs: Size; totalJs: Size }
  initialChunks: string[]
  chunks: Chunk[]
}

/** 将模块路径归一化，报告不包含开发机的绝对目录。 */
function safeModuleId(id: string, root: string): string {
  const normalized = id.replaceAll("\\", "/").replace(/^\0/, "virtual:")
  const nodeModulesIndex = normalized.lastIndexOf("/node_modules/")
  if (nodeModulesIndex >= 0) return normalized.slice(nodeModulesIndex + 1)
  const relative = path.relative(root, normalized).replaceAll("\\", "/")
  return relative.startsWith("../") ? path.basename(normalized) : relative
}

/** 汇总逐文件 raw/gzip 数据，gzip 不假定跨文件共享压缩字典。 */
function sumSizes(chunks: Chunk[]): Size {
  return chunks.reduce((total, chunk) => {
    // 累计实际产出的每个 JavaScript 文件。
    return { rawBytes: total.rawBytes + chunk.rawBytes, gzipBytes: total.gzipBytes + chunk.gzipBytes }
  }, { rawBytes: 0, gzipBytes: 0 })
}

/** 生成体积报告并执行分级监管；错误在报告落盘后抛出，便于 CI 留存诊断。 */
export function bundleBudgetPlugin(): Plugin {
  let config: ResolvedConfig
  let analysis: Analysis | undefined
  return {
    name: "curated-bundle-budget",
    apply: "build",
    /** 使用 Vite 已解析环境区分 Web/Mock，避免跨模式错误比较。 */
    configResolved(resolved) {
      config = resolved
    },
    /** 收集 JS 模块与入口静态依赖闭包，不把动态导入算入首屏。 */
    generateBundle(_outputOptions, bundle) {
      const chunks: Chunk[] = []
      for (const output of Object.values(bundle)) {
        if (output.type !== "chunk") continue
        chunks.push({
          name: output.name, fileName: output.fileName,
          rawBytes: Buffer.byteLength(output.code), gzipBytes: gzipSync(output.code).length,
          isEntry: output.isEntry, isDynamicEntry: output.isDynamicEntry,
          imports: [...output.imports].sort(), dynamicImports: [...output.dynamicImports].sort(),
          topModules: Object.entries(output.modules).map(([id, details]) => {
            // 仅保留可用于排查依赖的相对模块名及渲染大小。
            return { id: safeModuleId(id, config.root), renderedBytes: details.renderedLength }
          }).sort((left, right) => {
            // 每块优先报告体积最大的模块。
            return right.renderedBytes - left.renderedBytes
          }).slice(0, 12),
        })
      }
      chunks.sort((left, right) => {
        // 与报告资源排行保持一致，最大分块排在前面。
        return right.rawBytes - left.rawBytes
      })
      const chunkByFileName = new Map(chunks.map((chunk) => {
        // 按构建文件名建立静态导入图索引。
        return [chunk.fileName, chunk] as const
      }))
      const initialFileNames = new Set<string>()
      /** 遍历静态依赖并去重，支持循环依赖与多个入口。 */
      function addStaticImports(fileName: string) {
        if (initialFileNames.has(fileName)) return
        const chunk = chunkByFileName.get(fileName)
        if (!chunk) return
        initialFileNames.add(fileName)
        for (const imported of chunk.imports) addStaticImports(imported)
      }
      for (const chunk of chunks) if (chunk.isEntry) addStaticImports(chunk.fileName)
      analysis = {
        totals: { initialJs: sumSizes(chunks.filter((chunk) => {
          // 动态加载功能不计入首屏 JS。
          return initialFileNames.has(chunk.fileName)
        })), totalJs: sumSizes(chunks) },
        initialChunks: [...initialFileNames].sort(), chunks,
      }
    },
    /** 输出写完后计入 public、字体等资源，并先保存报告再决定构建是否失败。 */
    writeBundle(outputOptions) {
      if (!analysis) throw new Error("Bundle analysis is missing")
      const directory = path.resolve(config.root, outputOptions.dir ?? config.build.outDir)
      const assets = collectAssets(directory)
      for (const chunk of analysis.chunks) {
        // 以最终落盘字节为准，计入构建器后续加入的注释或换行。
        const content = readFileSync(path.join(directory, chunk.fileName))
        chunk.rawBytes = content.length
        chunk.gzipBytes = gzipSync(content).length
      }
      analysis.totals.totalJs = sumSizes(analysis.chunks)
      analysis.totals.initialJs = sumSizes(analysis.chunks.filter((chunk) => {
        // 沿用生成阶段解析的静态依赖集合，使用最终文件大小重新汇总。
        return analysis!.initialChunks.includes(chunk.fileName)
      }))
      const snapshot: Snapshot = {
        schemaVersion: 2,
        mode: config.env.VITE_USE_WEB_API === "true" ? "web" : "mock",
        metrics: {
          initialRaw: analysis.totals.initialJs.rawBytes, initialGzip: analysis.totals.initialJs.gzipBytes,
          jsRaw: analysis.totals.totalJs.rawBytes, jsGzip: analysis.totals.totalJs.gzipBytes,
          cssRaw: 0, distRaw: 0,
        },
      }
      for (const asset of assets) {
        snapshot.metrics.distRaw += asset.rawBytes
        if (asset.fileName.endsWith(".css")) snapshot.metrics.cssRaw += asset.rawBytes
      }
      const policy = loadPolicy(config.root)
      const reviewed = loadSnapshot(path.join(config.root, "bundle-baseline.json"), snapshot.mode)
      const recent = loadSnapshot(path.join(config.root, ".workspace/bundle-baseline/bundle-analysis.json"), snapshot.mode)
      const findings = evaluateSize(snapshot, policy, recent, reviewed)
      for (const chunk of analysis.chunks) {
        if (chunk.rawBytes > policy.chunkWarningBytes) findings.push({ level: "warning", message: `分块 ${chunk.name} 为 ${formatBytes(chunk.rawBytes)}；可评估懒加载，无逐文件硬限制。` })
        if (chunk.name === "hls-player" && analysis.initialChunks.includes(chunk.fileName)) findings.push({ level: "warning", message: "HLS 播放器进入首屏静态依赖，请检查是否仍可延迟加载。" })
      }
      const summary = renderSummary(snapshot, policy, findings, assets, recent, reviewed)
      const report = { ...snapshot, ...analysis, policy, assets, findings,
        comparisons: { recent: recent ?? null, reviewed: reviewed ?? null } }
      writeFileSync(path.join(directory, "bundle-analysis.json"), `${JSON.stringify(report, null, 2)}\n`, "utf8")
      writeFileSync(path.join(directory, "bundle-report.md"), summary, "utf8")
      if (process.env.GITHUB_STEP_SUMMARY) appendFileSync(process.env.GITHUB_STEP_SUMMARY, summary, "utf8")
      for (const finding of findings) {
        this.warn(finding.message)
        if (process.env.GITHUB_ACTIONS === "true") console.log(`::${finding.level === "error" ? "error" : "warning"} title=Bundle size::${finding.message}`)
      }
      this.info(`体积报告：${path.join(directory, "bundle-report.md")}；${findings.length} 项提醒/超限。`)
      if (findings.some((finding) => {
        // 仅绝对严重上限触发失败，增长类提醒不会阻断新功能。
        return finding.level === "error"
      })) this.error("Curated 构建体积超过严重上限，请查看 bundle-report.md 并减少体积。")
    },
  }
}

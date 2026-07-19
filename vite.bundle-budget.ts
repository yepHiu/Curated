import path from "node:path"
import { gzipSync } from "node:zlib"
import type { Plugin } from "vite"

interface SizeBudget {
  rawBytes: number
  gzipBytes: number
}

interface NamedChunkBudget extends SizeBudget {
  name: string
}

const INITIAL_JS_BUDGET: SizeBudget = {
  rawBytes: 500_000,
  gzipBytes: 165_000,
}

const TOTAL_JS_BUDGET: SizeBudget = {
  rawBytes: 2_250_000,
  gzipBytes: 750_000,
}

const NAMED_CHUNK_BUDGETS: NamedChunkBudget[] = [
  { name: "hls-player", rawBytes: 525_000, gzipBytes: 165_000 },
  { name: "pinyin-search", rawBytes: 305_000, gzipBytes: 150_000 },
  { name: "index", rawBytes: 215_000, gzipBytes: 72_000 },
  { name: "SettingsView", rawBytes: 205_000, gzipBytes: 45_000 },
]

function safeModuleId(id: string): string {
  const normalized = id.replaceAll("\\", "/").replace(/^\0/, "virtual:")
  const nodeModulesIndex = normalized.lastIndexOf("/node_modules/")
  if (nodeModulesIndex >= 0) {
    return normalized.slice(nodeModulesIndex + 1)
  }
  const relative = path.relative(process.cwd(), normalized).replaceAll("\\", "/")
  return relative.startsWith("../") ? path.basename(normalized) : relative
}

function formatBytes(value: number): string {
  return `${(value / 1_000).toFixed(2)} kB`
}

function assertWithinBudget(
  label: string,
  actual: SizeBudget,
  budget: SizeBudget,
  failures: string[],
) {
  if (actual.rawBytes > budget.rawBytes) {
    failures.push(`${label} raw ${formatBytes(actual.rawBytes)} > ${formatBytes(budget.rawBytes)}`)
  }
  if (actual.gzipBytes > budget.gzipBytes) {
    failures.push(`${label} gzip ${formatBytes(actual.gzipBytes)} > ${formatBytes(budget.gzipBytes)}`)
  }
}

export function bundleBudgetPlugin(): Plugin {
  return {
    name: "curated-bundle-budget",
    apply: "build",
    generateBundle(_outputOptions, bundle) {
      const chunks = Object.values(bundle)
        .filter((output) => output.type === "chunk")
        .map((chunk) => ({
          chunk,
          fileName: chunk.fileName,
          gzipBytes: gzipSync(chunk.code).length,
          rawBytes: Buffer.byteLength(chunk.code),
        }))
        .sort((left, right) => right.rawBytes - left.rawBytes)

      const chunkByFileName = new Map(chunks.map((row) => [row.fileName, row]))
      const initialFileNames = new Set<string>()

      function addStaticImports(fileName: string) {
        if (initialFileNames.has(fileName)) return
        const row = chunkByFileName.get(fileName)
        if (!row) return
        initialFileNames.add(fileName)
        for (const importedFileName of row.chunk.imports) {
          addStaticImports(importedFileName)
        }
      }

      for (const row of chunks) {
        if (row.chunk.isEntry) {
          addStaticImports(row.fileName)
        }
      }

      const initialChunks = chunks.filter((row) => initialFileNames.has(row.fileName))
      const initialTotals = initialChunks.reduce(
        (totals, row) => ({
          rawBytes: totals.rawBytes + row.rawBytes,
          gzipBytes: totals.gzipBytes + row.gzipBytes,
        }),
        { rawBytes: 0, gzipBytes: 0 },
      )
      const totalJs = chunks.reduce(
        (totals, row) => ({
          rawBytes: totals.rawBytes + row.rawBytes,
          gzipBytes: totals.gzipBytes + row.gzipBytes,
        }),
        { rawBytes: 0, gzipBytes: 0 },
      )

      const report = {
        schemaVersion: 1,
        budgets: {
          initialJs: INITIAL_JS_BUDGET,
          totalJs: TOTAL_JS_BUDGET,
          namedChunks: NAMED_CHUNK_BUDGETS,
        },
        totals: {
          initialJs: initialTotals,
          totalJs,
        },
        initialChunks: initialChunks.map((row) => row.fileName).sort(),
        chunks: chunks.map((row) => ({
          name: row.chunk.name,
          fileName: row.fileName,
          rawBytes: row.rawBytes,
          gzipBytes: row.gzipBytes,
          isEntry: row.chunk.isEntry,
          isDynamicEntry: row.chunk.isDynamicEntry,
          imports: [...row.chunk.imports].sort(),
          dynamicImports: [...row.chunk.dynamicImports].sort(),
          topModules: Object.entries(row.chunk.modules)
            .map(([id, details]) => ({
              id: safeModuleId(id),
              renderedBytes: details.renderedLength,
            }))
            .sort((left, right) => right.renderedBytes - left.renderedBytes)
            .slice(0, 12),
        })),
      }

      this.emitFile({
        type: "asset",
        fileName: "bundle-analysis.json",
        source: `${JSON.stringify(report, null, 2)}\n`,
      })

      const failures: string[] = []
      assertWithinBudget("initial JS", initialTotals, INITIAL_JS_BUDGET, failures)
      assertWithinBudget("total JS", totalJs, TOTAL_JS_BUDGET, failures)

      for (const budget of NAMED_CHUNK_BUDGETS) {
        const row = chunks.find((candidate) => candidate.chunk.name === budget.name)
        if (!row) {
          failures.push(`required chunk ${budget.name} is missing`)
          continue
        }
        assertWithinBudget(`chunk ${budget.name}`, row, budget, failures)
      }

      if (failures.length > 0) {
        this.error(`Curated bundle budget exceeded:\n- ${failures.join("\n- ")}`)
      }
    },
  }
}

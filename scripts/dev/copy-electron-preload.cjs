const fs = require("node:fs")
const path = require("node:path")

const repoRoot = path.resolve(__dirname, "..", "..")
const source = path.join(repoRoot, "electron", "preload.cjs")
const targetDir = path.join(repoRoot, "electron-dist")
const target = path.join(targetDir, "preload.cjs")

fs.mkdirSync(targetDir, { recursive: true })
fs.copyFileSync(source, target)

const versionState = JSON.parse(fs.readFileSync(path.join(repoRoot, "scripts/release/versions/desktop.json"), "utf8"))
const parts = [versionState.current?.major, versionState.current?.minor, versionState.current?.patch]
if (versionState.schema !== 1 || !parts.every((part) => Number.isSafeInteger(part) && part >= 0)) {
  throw new Error("Invalid Desktop version source")
}
const config = JSON.parse(fs.readFileSync(path.join(repoRoot, "electron/release-config.json"), "utf8"))
if (!["legacy", "desktop"].includes(config.distribution) || !(config.updateFeed === null || typeof config.updateFeed === "string")) {
  throw new Error("Invalid Desktop release configuration")
}
fs.writeFileSync(path.join(targetDir, "desktop-release.json"), JSON.stringify({
  schema: 1,
  version: parts.join("."),
  buildStamp: new Date().toISOString().replace(/[-:]/g, "").replace("T", ".").slice(0, 15),
  distribution: config.distribution,
  updateFeed: config.updateFeed,
}, null, 2) + "\n")

for (const file of ["connections-preload.cjs", "connections.html", "connections.css", "connections-ui.js"]) {
  fs.copyFileSync(path.join(repoRoot, "electron", file), path.join(targetDir, file))
}
// Reuse the canonical light/dark semantic palette without bundling the business UI.
const theme = fs.readFileSync(path.join(repoRoot, "src/style.css"), "utf8")
const light = theme.match(/:root\s*\{([^}]+)\}/)?.[1]
const dark = theme.match(/\.dark\s*\{([^}]+)\}/)?.[1]
if (!light || !dark) throw new Error("Missing canonical connection-page theme tokens")
fs.writeFileSync(path.join(targetDir, "connections-tokens.css"), `:root {${light}}\n@media (prefers-color-scheme: dark) { :root {${dark}} }\n`)

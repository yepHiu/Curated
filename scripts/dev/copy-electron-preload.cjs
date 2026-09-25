const fs = require("node:fs")
const path = require("node:path")

const repoRoot = path.resolve(__dirname, "..", "..")
const source = path.join(repoRoot, "electron", "preload.cjs")
const targetDir = path.join(repoRoot, "electron-dist")
const target = path.join(targetDir, "preload.cjs")

fs.mkdirSync(targetDir, { recursive: true })
fs.copyFileSync(source, target)

fs.copyFileSync(path.join(repoRoot, "electron", "launcher-preload.cjs"), path.join(targetDir, "launcher-preload.cjs"))

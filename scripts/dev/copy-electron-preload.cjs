const fs = require("node:fs")
const path = require("node:path")
const { execFileSync } = require("node:child_process")

const repoRoot = path.resolve(__dirname, "..", "..")
const source = path.join(repoRoot, "electron", "preload.cjs")
const targetDir = path.join(repoRoot, "electron-dist")
const target = path.join(targetDir, "preload.cjs")

fs.mkdirSync(targetDir, { recursive: true })
fs.copyFileSync(source, target)

fs.copyFileSync(path.join(repoRoot, "electron", "launcher-preload.cjs"), path.join(targetDir, "launcher-preload.cjs"))

// macOS Dock 读取 bundle 名称；只调整本项目开发运行时，不改应用标识或配置目录。
if (process.platform === "darwin") {
  const electronBinary = require("electron")
  const plist = path.resolve(path.dirname(electronBinary), "..", "Info.plist")
  for (const key of ["CFBundleName", "CFBundleDisplayName"]) {
    execFileSync("/usr/libexec/PlistBuddy", ["-c", `Set :${key} Curated`, plist])
  }
}

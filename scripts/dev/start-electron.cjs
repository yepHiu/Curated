const fs = require("node:fs")
const path = require("node:path")
const { pathToFileURL } = require("node:url")
const { execFileSync, spawn } = require("node:child_process")

const repoRoot = path.resolve(__dirname, "..", "..")
let binary = require("electron")

// Dock 以应用 bundle 的名称和身份显示入口，不能仅靠 app.setName。
// 为每个 Electron 版本缓存独立开发副本，避免修改 node_modules 运行时。
if (process.platform === "darwin") {
  const version = require("electron/package.json").version
  const destination = path.join(repoRoot, ".workspace", "electron-dev", version, "Curated.app")
  const source = path.resolve(path.dirname(binary), "..", "..")
  if (!fs.existsSync(destination)) {
    fs.mkdirSync(path.dirname(destination), { recursive: true })
    execFileSync("/bin/cp", ["-cR", source, destination])
  }
  const plist = path.join(destination, "Contents", "Info.plist")
  for (const [key, value] of Object.entries({
    CFBundleName: "Curated",
    CFBundleDisplayName: "Curated",
    CFBundleIdentifier: "app.curated.desktop.dev",
  })) {
    execFileSync("/usr/libexec/PlistBuddy", ["-c", `Set :${key} ${value}`, plist])
  }
  // macOS 登录项不保留 CLI 项目参数，因此 bundle 必须有自己的默认入口。
  const appResources = path.join(destination, "Contents", "Resources", "app")
  fs.mkdirSync(appResources, { recursive: true })
  const projectPackage = require(path.join(repoRoot, "package.json"))
  fs.writeFileSync(path.join(appResources, "package.json"), JSON.stringify({ name: projectPackage.name, version: projectPackage.version, main: "main.mjs", type: "module" }))
  fs.writeFileSync(path.join(appResources, "main.mjs"), `await import(${JSON.stringify(pathToFileURL(path.join(repoRoot, "electron-dist", "main.js")).href)});\n`)
  fs.copyFileSync(path.join(repoRoot, "public", "Curated-icon.png"), path.join(appResources, "curated.png"))
  binary = path.join(destination, "Contents", "MacOS", "Electron")
}

// macOS 正常启动也走 bundle 默认入口，与系统登录启动保持一致。
const child = spawn(binary, process.platform === "darwin" ? [] : [repoRoot], { cwd: repoRoot, stdio: "inherit" })
// 将启动失败返回给开发命令，避免报告未实际打开的窗口。
child.on("error", error => { console.error(error); process.exitCode = 1 })
// 保留 Electron 的退出结果。
child.on("exit", code => { process.exitCode = code ?? 1 })
// 终止开发命令时只结束它启动的 Desktop。
process.on("SIGINT", () => child.kill("SIGINT"))
// 不影响独立运行的 Server。
process.on("SIGTERM", () => child.kill("SIGTERM"))

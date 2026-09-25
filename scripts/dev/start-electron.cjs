const fs = require("node:fs")
const path = require("node:path")
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
  binary = path.join(destination, "Contents", "MacOS", "Electron")
}

const child = spawn(binary, [repoRoot], { cwd: repoRoot, stdio: "inherit" })
// 将启动失败返回给开发命令，避免报告未实际打开的窗口。
child.on("error", error => { console.error(error); process.exitCode = 1 })
// 保留 Electron 的退出结果。
child.on("exit", code => { process.exitCode = code ?? 1 })
// 终止开发命令时只结束它启动的 Desktop。
process.on("SIGINT", () => child.kill("SIGINT"))
// 不影响独立运行的 Server。
process.on("SIGTERM", () => child.kill("SIGTERM"))

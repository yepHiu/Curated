const path = require("node:path")
const { spawn } = require("node:child_process")
const root = path.resolve(__dirname, "..", "..")
const child = spawn(require("electron"), [path.join(root, ".workspace/native-player-dist/native-player-prototype.js")], {
  cwd: root, stdio: "inherit", env: process.env,
})
// 报告启动与退出结果，不操作正式 Desktop 的进程。
child.on("error", () => { console.error("Native prototype could not start"); process.exitCode = 1 })
// 透传退出状态给开发命令。
child.on("exit", (code) => { process.exitCode = code ?? 1 })
// 只关闭该命令创建的原型进程。
process.on("SIGINT", () => child.kill("SIGINT"))

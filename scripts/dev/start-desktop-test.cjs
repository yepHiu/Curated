const fs = require("node:fs")
const path = require("node:path")
const { spawn, spawnSync } = require("node:child_process")

const root = path.resolve(__dirname, "..", "..")
const workspace = path.join(root, ".workspace")
const frontend = path.join(workspace, "desktop-test-ui")
const profile = path.join(workspace, "desktop-native-test-profile")

/** 仅在显式准备测试版本时编译；平时启动直接复用已有产物。 */
function buildTestDesktop() {
  const env = { ...process.env, VITE_USE_WEB_API: "true", VITE_API_BASE_URL: "" }
  const pnpmCli = process.env.npm_execpath
  if (!pnpmCli || !fs.existsSync(pnpmCli)) throw new Error("请通过 pnpm desktop:test:build 准备测试版本。")
  for (const args of [["build", "--outDir", ".workspace/desktop-test-ui"], ["build:electron"]]) {
    const result = spawnSync(process.execPath, [pnpmCli, ...args], {
      cwd: root, env, stdio: "inherit",
    })
    if (result.error) throw result.error
    if (result.status !== 0) throw new Error(`测试版构建失败：pnpm ${args[0]}`)
  }
}

/** 启动独立测试配置的 Desktop；静态页面由 Electron 自己托管。 */
function startTestDesktop() {
  if (!fs.existsSync(path.join(frontend, "index.html")) || !fs.existsSync(path.join(root, "electron-dist", "main.js"))) {
    throw new Error("测试产物尚未准备，请先运行 pnpm desktop:test:build。")
  }
  fs.mkdirSync(profile, { recursive: true })
  const env = {
    ...process.env,
    CURATED_ELECTRON_BACKEND_URL: "http://127.0.0.1:8080",
    CURATED_ELECTRON_FRONTEND_URL: "http://127.0.0.1:5183",
    CURATED_ELECTRON_FRONTEND_DIR: frontend,
  }
  delete env.ELECTRON_RUN_AS_NODE
  const stdout = fs.openSync(path.join(workspace, "desktop-native-test.log"), "a")
  const stderr = fs.openSync(path.join(workspace, "desktop-native-test-error.log"), "a")
  const child = spawn(require("electron"), ["--user-data-dir=" + profile, "--remote-debugging-port=19016", root], {
    cwd: root, env, detached: true, windowsHide: true, stdio: ["ignore", stdout, stderr],
  })
  // 进程创建失败时返回错误，不报告一个不存在的测试窗口。
  child.on("error", error => { console.error(error); process.exitCode = 1 })
  child.unref()
  fs.closeSync(stdout)
  fs.closeSync(stderr)
  fs.writeFileSync(path.join(workspace, "desktop-native-test-process.json"), JSON.stringify({
    pid: child.pid, profile, server: env.CURATED_ELECTRON_BACKEND_URL,
    frontend: env.CURATED_ELECTRON_FRONTEND_URL, frontendMode: "prebuilt", frontendDirectory: frontend,
  }, null, 2) + "\n")
  console.log(JSON.stringify({ pid: child.pid, frontendMode: "prebuilt", profile }))
}

try {
  if (process.argv.includes("--build-only")) buildTestDesktop()
  else startTestDesktop()
} catch (error) {
  console.error(error.message)
  process.exitCode = 1
}

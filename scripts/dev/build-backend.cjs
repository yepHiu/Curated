const { spawnSync } = require("node:child_process")
const fs = require("node:fs")
const path = require("node:path")

/**
 * 选择开发后端文件名。Windows 固定为 curated-dev.exe，避免和发布包的 curated.exe 抢同一路径。
 * @param {string} platform Node 的 process.platform
 * @returns {string} 写到 backend/runtime 下的文件名
 */
function devBackendFileName(platform) {
  return platform === "win32" ? "curated-dev.exe" : "curated-dev"
}

/**
 * 在 backend/ 执行不带 release tag 的 go build，产物给 Electron 优先启动。
 * 失败时返回非 0；找不到 go 时返回 1。
 * @param {NodeJS.ProcessEnv} env 传给 go 的环境
 * @param {string} platform 决定输出文件名
 * @returns {number} 进程退出码
 */
function buildDevBackend(env = process.env, platform = process.platform) {
  const repoRoot = path.resolve(__dirname, "..", "..")
  const backendDir = path.join(repoRoot, "backend")
  const outputName = devBackendFileName(platform)
  if (outputName.toLowerCase() === "curated.exe") {
    console.error("Refusing to write a development backend named curated.exe")
    return 1
  }
  const runtimeDir = path.join(backendDir, "runtime")
  fs.mkdirSync(runtimeDir, { recursive: true })
  const outputPath = path.join(runtimeDir, outputName)
  const result = spawnSync("go", ["build", "-o", outputPath, "./cmd/curated"], {
    cwd: backendDir,
    env,
    stdio: "inherit",
  })
  if (result.error) {
    console.error(result.error.message)
    return 1
  }
  return result.status ?? 1
}

if (require.main === module) {
  process.exit(buildDevBackend())
}

module.exports = { buildDevBackend, devBackendFileName }

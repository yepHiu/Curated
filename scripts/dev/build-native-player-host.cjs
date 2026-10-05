const { spawnSync } = require("node:child_process")
const fs = require("node:fs")
const path = require("node:path")
const root = path.resolve(__dirname, "..", "..")
// 非 Windows 不调用 Win32 编译器。正式入口可显式指定输出目录。
if (process.platform === "win32") {
  const output = process.argv[2] ? path.resolve(root, process.argv[2]) : path.join(root, ".workspace", "native-player-dist")
  fs.mkdirSync(output, { recursive: true })
  const compiler = process.env.CURATED_NATIVE_CC || spawnSync("where.exe", ["gcc"], { encoding: "utf8", windowsHide: true }).stdout?.trim().split(/\r?\n/)[0]
  if (!compiler) throw new Error("NATIVE_HOST_COMPILER_MISSING: install Windows MinGW gcc or set CURATED_NATIVE_CC to its absolute path")
  const result = spawnSync(compiler, [
    "-std=c11", "-O2", "-Wall", "-Wextra", "-Werror", "-municode", "-mwindows", "-static-libgcc",
    path.join(root, "electron", "native-player-host.c"), "-o", path.join(output, "native-player-host.exe"),
    "-luser32", "-lgdi32", "-lshell32", "-ldwmapi",
  ], { cwd: root, stdio: "inherit", windowsHide: true,
    env: { ...process.env, PATH: `${path.dirname(compiler)};${process.env.PATH || ""}` } })
  if (result.error) throw new Error("NATIVE_HOST_COMPILER_MISSING: install a Windows MinGW gcc or set CURATED_NATIVE_CC")
  if (result.status !== 0) throw new Error(`NATIVE_HOST_BUILD_FAILED: compiler exit ${result.status}, signal ${result.signal || "none"}`)
}

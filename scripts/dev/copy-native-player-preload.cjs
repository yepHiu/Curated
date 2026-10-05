const fs = require("node:fs")
const path = require("node:path")
const root = path.resolve(__dirname, "..", "..")
// 独立原型构建才复制原型 preload；正常 Desktop 构建不增加此桥接。
const output = path.join(root, ".workspace", "native-player-dist")
fs.mkdirSync(output, { recursive: true })
fs.copyFileSync(path.join(root, "electron", "native-player-preload.cjs"), path.join(output, "native-player-preload.cjs"))

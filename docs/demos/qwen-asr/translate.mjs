/**
 * 把文件转写的日语句子译成简体中文，时间码原样保留。
 * 用法：node translate.mjs --file work/filetrans-42-45.txt
 */
import { readFile, writeFile } from "node:fs/promises"
import { dirname, resolve } from "node:path"
import { fileURLToPath } from "node:url"

const demoDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(demoDir, "../../..")
const envPath = resolve(repoRoot, ".workspace/qwen-asr.env")

/**
 * 读取本地 env，不打印密钥。
 * @param {string} path 密钥文件
 * @returns {Promise<Record<string, string>>}
 */
async function readEnvFile(path) {
  const text = await readFile(path, "utf8")
  const values = {}
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith("#")) continue
    const separator = trimmed.indexOf("=")
    if (separator <= 0) continue
    values[trimmed.slice(0, separator).trim()] = trimmed.slice(separator + 1).trim()
  }
  return values
}

/**
 * 把时间轴文本拆成可翻译的句子，空行和纯时间码不进模型。
 * @param {string} text 转写文本
 * @returns {{ kind: "cue" | "text", value: string }[]}
 */
function splitCues(text) {
  return text.split(/\r?\n/).map((line) => {
    if (/^\d{2}:\d{2}:\d{2}\.\d{3} --> /.test(line)) return { kind: "cue", value: line }
    return { kind: "text", value: line }
  })
}

/**
 * 调用百炼兼容接口翻译日语对白。失败时把响应正文带回，便于判断密钥是否只能用于语音模型。
 * @param {string} apiKey 百炼密钥
 * @param {string[]} lines 日语句子
 * @returns {Promise<string[]>}
 */
async function translateLines(apiKey, lines) {
  const response = await fetch("https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions", {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      model: "qwen-plus",
      temperature: 0,
      messages: [
        {
          role: "system",
          content: "把每行日语对白译成简体中文。保持行数和顺序，不要编号，不要解释，听不清的口语按原意直译。",
        },
        { role: "user", content: lines.join("\n") },
      ],
    }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`翻译失败：${JSON.stringify(body)}`)
  const content = body.choices?.[0]?.message?.content
  if (typeof content !== "string") throw new Error(`没有译文：${JSON.stringify(body)}`)
  return content.split(/\r?\n/).map((line) => line.trim()).filter((line) => line.length > 0)
}

/**
 * 入口：读转写文本，写出带同一时间码的中文文本。
 * @returns {Promise<void>}
 */
async function main() {
  const fileFlag = process.argv.indexOf("--file")
  const filePath = resolve(fileFlag >= 0 ? process.argv[fileFlag + 1] : "")
  const env = await readEnvFile(envPath)
  const apiKey = env.DASHSCOPE_API_KEY
  if (!apiKey) throw new Error("qwen-asr.env 里没有 DASHSCOPE_API_KEY")
  const blocks = splitCues(await readFile(filePath, "utf8"))
  const sourceLines = blocks.filter((block) => block.kind === "text" && block.value.trim()).map((block) => block.value.trim())
  const translated = await translateLines(apiKey, sourceLines)
  if (translated.length !== sourceLines.length) {
    throw new Error(`译文 ${translated.length} 行，原文 ${sourceLines.length} 行，没有落盘`)
  }
  let cursor = 0
  const output = blocks.map((block) => {
    if (block.kind === "text" && block.value.trim()) {
      const line = translated[cursor]
      cursor += 1
      return line
    }
    return block.value
  }).join("\n")
  const outputPath = filePath.replace(/\.txt$/i, ".zh.txt")
  await writeFile(outputPath, `${output.trim()}\n`, "utf8")
  console.log(output)
  console.log(`已写入 ${outputPath}`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})

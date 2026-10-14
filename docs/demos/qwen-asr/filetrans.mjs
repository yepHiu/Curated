/**
 * 用 qwen-audio-3.1-asr-flash-filetrans 把本地音频转成带时间戳的文本。
 * 先申请百炼临时上传凭证，再提交异步转写并轮询结果。
 * 用法：node filetrans.mjs --file 音频路径 [--offset-sec 2520] [--language ja]
 */
import { readFile, writeFile } from "node:fs/promises"
import { basename, dirname, resolve } from "node:path"
import { fileURLToPath } from "node:url"

const demoDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(demoDir, "../../..")
const envPath = resolve(repoRoot, ".workspace/qwen-asr.env")
const model = "qwen-audio-3.1-asr-flash-filetrans"
const apiRoot = "https://dashscope.aliyuncs.com/api/v1"

/**
 * 读取本地 env。忽略空行和注释，不打印密钥。
 * @param {string} path 密钥文件绝对路径
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
 * 解析命令行。offset-sec 用来把片段内时间加回影片时间。
 * @param {string[]} argv process.argv 去掉 node 与脚本路径后的部分
 * @returns {{ file: string, offsetSec: number, language: string }}
 */
function parseArgs(argv) {
  let file = ""
  let offsetSec = 0
  let language = "ja"
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index]
    if (token === "--file") {
      file = argv[index + 1] ?? ""
      index += 1
    } else if (token === "--offset-sec") {
      offsetSec = Number(argv[index + 1] ?? 0)
      index += 1
    } else if (token === "--language") {
      language = argv[index + 1] ?? language
      index += 1
    } else if (!token.startsWith("--") && !file) {
      file = token
    }
  }
  if (!file) throw new Error("请用 --file 指定音频")
  return { file, offsetSec, language }
}

/**
 * 申请与目标模型绑定的临时上传凭证。
 * @param {string} apiKey 百炼密钥
 * @returns {Promise<Record<string, string>>}
 */
async function getUploadPolicy(apiKey) {
  const url = new URL(`${apiRoot}/uploads`)
  url.searchParams.set("action", "getPolicy")
  url.searchParams.set("model", model)
  const response = await fetch(url, { headers: { Authorization: `Bearer ${apiKey}` } })
  const body = await response.json()
  if (!response.ok) throw new Error(`获取上传凭证失败：${JSON.stringify(body)}`)
  return body.data
}

/**
 * 把本地文件传到临时 OSS，返回 48 小时内有效的 oss:// 地址。
 * @param {string} apiKey 百炼密钥
 * @param {string} filePath 本地音频
 * @returns {Promise<string>}
 */
async function uploadAudio(apiKey, filePath) {
  const policy = await getUploadPolicy(apiKey)
  const fileName = basename(filePath)
  const key = `${policy.upload_dir}/${fileName}`
  const bytes = await readFile(filePath)
  const form = new FormData()
  form.append("OSSAccessKeyId", policy.oss_access_key_id)
  form.append("Signature", policy.signature)
  form.append("policy", policy.policy)
  form.append("x-oss-object-acl", policy.x_oss_object_acl)
  form.append("x-oss-forbid-overwrite", policy.x_oss_forbid_overwrite)
  form.append("key", key)
  form.append("success_action_status", "200")
  form.append("file", new Blob([bytes]), fileName)
  const response = await fetch(policy.upload_host, { method: "POST", body: form })
  if (!response.ok) throw new Error(`上传失败：${response.status} ${await response.text()}`)
  return `oss://${key}`
}

/**
 * 提交异步转写任务。
 * @param {string} apiKey 百炼密钥
 * @param {string} ossUrl 临时音频地址
 * @param {string} language 语种提示
 * @returns {Promise<string>}
 */
async function submitTask(apiKey, ossUrl, language) {
  const response = await fetch(`${apiRoot}/services/audio/asr/transcription`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
      "X-DashScope-Async": "enable",
      "X-DashScope-OssResourceResolve": "enable",
    },
    body: JSON.stringify({
      model,
      input: { file_urls: [ossUrl] },
      parameters: { channel_id: [0], language_hints: [language] },
    }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`提交转写失败：${JSON.stringify(body)}`)
  const taskId = body.output?.task_id
  if (!taskId) throw new Error(`没有任务编号：${JSON.stringify(body)}`)
  return taskId
}

/**
 * 轮询任务直到成功、失败或超时。
 * @param {string} apiKey 百炼密钥
 * @param {string} taskId 任务编号
 * @returns {Promise<Record<string, unknown>>}
 */
async function waitTask(apiKey, taskId) {
  const deadline = Date.now() + 4 * 60 * 1000
  while (Date.now() < deadline) {
    const response = await fetch(`${apiRoot}/tasks/${taskId}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    })
    const body = await response.json()
    if (!response.ok) throw new Error(`查询任务失败：${JSON.stringify(body)}`)
    const status = body.output?.task_status
    console.log(`任务 ${status}`)
    if (status === "SUCCEEDED") return body.output
    if (status === "FAILED" || status === "UNKNOWN") {
      throw new Error(`转写失败：${JSON.stringify(body.output)}`)
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 2000))
  }
  throw new Error("轮询超过 4 分钟")
}

/**
 * 把毫秒格式化成影片时间码。
 * @param {number} milliseconds 相对片段起点的毫秒，再加上片段在影片中的偏移
 * @returns {string}
 */
function formatClock(milliseconds) {
  const total = Math.max(0, Math.round(milliseconds))
  const hours = Math.floor(total / 3_600_000)
  const minutes = Math.floor((total % 3_600_000) / 60_000)
  const seconds = Math.floor((total % 60_000) / 1000)
  const millis = total % 1000
  const pad = (value, width = 2) => String(value).padStart(width, "0")
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}.${pad(millis, 3)}`
}

/**
 * 从转写 JSON 抽出句子，并按影片偏移写成可读文本。
 * @param {unknown} result 下载到的转写 JSON
 * @param {number} offsetSec 片段在影片中的起点秒数
 * @returns {string}
 */
function renderSentences(result, offsetSec) {
  const transcripts = result?.transcripts ?? result?.transcript ?? []
  const lines = []
  const list = Array.isArray(transcripts) ? transcripts : [transcripts]
  for (const transcript of list) {
    const sentences = transcript?.sentences ?? []
    for (const sentence of sentences) {
      const start = formatClock((sentence.begin_time ?? 0) + offsetSec * 1000)
      const end = formatClock((sentence.end_time ?? 0) + offsetSec * 1000)
      const text = (sentence.text ?? "").trim()
      if (!text) continue
      lines.push(`${start} --> ${end}`)
      lines.push(text)
      lines.push("")
    }
  }
  return lines.join("\n")
}

/**
 * 脚本入口：上传、转写并落盘 JSON 与时间轴文本。
 * @returns {Promise<void>}
 */
async function main() {
  const options = parseArgs(process.argv.slice(2))
  const env = await readEnvFile(envPath)
  const apiKey = env.DASHSCOPE_API_KEY
  if (!apiKey) throw new Error("qwen-asr.env 里没有 DASHSCOPE_API_KEY")
  const filePath = resolve(options.file)
  console.log(`上传 ${filePath}`)
  const ossUrl = await uploadAudio(apiKey, filePath)
  console.log("已上传，提交转写")
  const taskId = await submitTask(apiKey, ossUrl, options.language)
  console.log(`任务 ${taskId}`)
  const output = await waitTask(apiKey, taskId)
  const resultItem = output.results?.[0]
  if (!resultItem || resultItem.subtask_status !== "SUCCEEDED") {
    throw new Error(`子任务失败：${JSON.stringify(resultItem ?? output)}`)
  }
  const transcriptResponse = await fetch(resultItem.transcription_url)
  const transcript = await transcriptResponse.json()
  const workDir = dirname(filePath)
  const jsonPath = resolve(workDir, "filetrans-42-45.json")
  const textPath = resolve(workDir, "filetrans-42-45.txt")
  await writeFile(jsonPath, `${JSON.stringify(transcript, null, 2)}\n`, "utf8")
  const text = renderSentences(transcript, options.offsetSec)
  await writeFile(textPath, text, "utf8")
  console.log(text)
  console.log(`已写入 ${textPath}`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})

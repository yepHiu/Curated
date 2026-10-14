/**
 * 用 qwen3-asr-flash-filetrans 转写同一段 42–45 分钟样本，便于和 3.1 对照。
 * 千问 3 的提交字段是 input.file_url，结果在 output.result.transcription_url。
 * 用法：node filetrans-qwen3.mjs --file 音频路径 [--offset-sec 2520]
 */
import { readFile, writeFile } from "node:fs/promises"
import { basename, dirname, resolve } from "node:path"
import { fileURLToPath } from "node:url"

const demoDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(demoDir, "../../..")
const envPath = resolve(repoRoot, ".workspace/qwen-asr.env")
const model = "qwen3-asr-flash-filetrans"
const apiRoot = "https://dashscope.aliyuncs.com/api/v1"

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
 * 解析音频路径和影片时间偏移。
 * @param {string[]} argv 命令行参数
 * @returns {{ file: string, offsetSec: number }}
 */
function parseArgs(argv) {
  let file = ""
  let offsetSec = 0
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index]
    if (token === "--file") {
      file = argv[index + 1] ?? ""
      index += 1
    } else if (token === "--offset-sec") {
      offsetSec = Number(argv[index + 1] ?? 0)
      index += 1
    } else if (!token.startsWith("--") && !file) {
      file = token
    }
  }
  if (!file) throw new Error("请用 --file 指定音频")
  return { file, offsetSec }
}

/**
 * 申请与千问 3 文件转写模型绑定的上传凭证。
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
 * 上传本地音频并返回 oss:// 地址。
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
 * 按千问 3 的 file_url 字段提交异步转写。
 * @param {string} apiKey 百炼密钥
 * @param {string} ossUrl 临时地址
 * @returns {Promise<string>}
 */
async function submitTask(apiKey, ossUrl) {
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
      input: { file_url: ossUrl },
      parameters: {
        channel_id: [0],
        language: "ja",
        enable_words: false,
      },
    }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`提交转写失败：${JSON.stringify(body)}`)
  const taskId = body.output?.task_id
  if (!taskId) throw new Error(`没有任务编号：${JSON.stringify(body)}`)
  return taskId
}

/**
 * 轮询任务，成功时返回完整 output，里面含计费秒数。
 * @param {string} apiKey 百炼密钥
 * @param {string} taskId 任务编号
 * @returns {Promise<Record<string, unknown>>}
 */
async function waitTask(apiKey, taskId) {
  const deadline = Date.now() + 6 * 60 * 1000
  while (Date.now() < deadline) {
    const response = await fetch(`${apiRoot}/tasks/${taskId}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    })
    const body = await response.json()
    if (!response.ok) throw new Error(`查询任务失败：${JSON.stringify(body)}`)
    const status = body.output?.task_status
    console.log(`任务 ${status}`)
    if (status === "SUCCEEDED") return body
    if (status === "FAILED" || status === "UNKNOWN") {
      throw new Error(`转写失败：${JSON.stringify(body.output)}`)
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 2000))
  }
  throw new Error("轮询超过 6 分钟")
}

/**
 * 从任务响应里找出转写结果地址。千问 3 用 result，3.1 用 results。
 * @param {Record<string, unknown>} taskBody 查询接口响应
 * @returns {string}
 */
function transcriptionUrl(taskBody) {
  const output = taskBody.output ?? {}
  const single = output.result?.transcription_url
  if (typeof single === "string") return single
  const listed = output.results?.[0]?.transcription_url
  if (typeof listed === "string") return listed
  throw new Error(`没有转写地址：${JSON.stringify(output)}`)
}

/**
 * 把毫秒格式化成影片时间码。
 * @param {number} milliseconds 片段内毫秒加上影片偏移
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
 * 抽出句子文本。情绪标签有则附在行尾，方便和 3.1 对照。
 * @param {unknown} result 转写 JSON
 * @param {number} offsetSec 片段在影片中的起点
 * @returns {string}
 */
function renderSentences(result, offsetSec) {
  const transcripts = result?.transcripts ?? []
  const lines = []
  for (const transcript of transcripts) {
    for (const sentence of transcript.sentences ?? []) {
      const text = String(sentence.text ?? "").trim()
      if (!text) continue
      const start = formatClock((sentence.begin_time ?? 0) + offsetSec * 1000)
      const end = formatClock((sentence.end_time ?? 0) + offsetSec * 1000)
      const emotion = sentence.emotion ? ` [${sentence.emotion}]` : ""
      lines.push(`${start} --> ${end}`, `${text}${emotion}`, "")
    }
  }
  return lines.join("\n")
}

/**
 * 入口：上传、转写，并落盘去掉临时地址后的 JSON 与文本。
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
  const taskId = await submitTask(apiKey, ossUrl)
  console.log(`任务 ${taskId}`)
  const taskBody = await waitTask(apiKey, taskId)
  const usage = taskBody.usage ?? null
  console.log(`用量 ${JSON.stringify(usage)}`)
  const transcriptResponse = await fetch(transcriptionUrl(taskBody))
  const transcript = await transcriptResponse.json()
  if (transcript.file_url) transcript.file_url = "[redacted temporary oss url]"
  const workDir = dirname(filePath)
  const jsonPath = resolve(workDir, "filetrans-42-45-qwen3.json")
  const textPath = resolve(workDir, "filetrans-42-45-qwen3.txt")
  await writeFile(jsonPath, `${JSON.stringify({ usage, transcript }, null, 2)}\n`, "utf8")
  const text = renderSentences(transcript, options.offsetSec)
  await writeFile(textPath, text, "utf8")
  console.log(text)
  console.log(`已写入 ${textPath}`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})

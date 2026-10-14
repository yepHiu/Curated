/**
 * 把整部影片的 5 分钟音轨依次转写成带时间码的日语，再译成简体中文。
 * 已完成的分段会跳过，方便中断后接着跑。
 * 用法：node movie-translate.mjs
 */
import { execFile } from "node:child_process"
import { readdir, readFile, writeFile, mkdir } from "node:fs/promises"
import { basename, dirname, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { promisify } from "node:util"

const execFileAsync = promisify(execFile)
const demoDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(demoDir, "../../..")
const envPath = resolve(repoRoot, ".workspace/qwen-asr.env")
const movieDir = resolve(repoRoot, ".workspace/qwen-asr-demo/work/movie")
const chunkDir = resolve(movieDir, "chunks")
const partDir = resolve(movieDir, "parts")
const model = "qwen-audio-3.1-asr-flash-filetrans"
const apiRoot = "https://dashscope.aliyuncs.com/api/v1"
const asrConcurrency = 4
const translateBatchSize = 40

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
 * 用 ffprobe 读取一个分段的秒数，作为下一段的时间偏移。
 * @param {string} filePath 分段音频
 * @returns {Promise<number>}
 */
async function probeDuration(filePath) {
  const { stdout } = await execFileAsync("ffprobe", [
    "-v", "error",
    "-show_entries", "format=duration",
    "-of", "csv=p=0",
    filePath,
  ])
  const duration = Number(String(stdout).trim())
  if (!Number.isFinite(duration)) throw new Error(`无法读取时长：${filePath}`)
  return duration
}

/**
 * 列出分段并累计每段在整片中的起点。
 * @returns {Promise<{ file: string, offsetSec: number, durationSec: number }[]>}
 */
async function listChunks() {
  const names = (await readdir(chunkDir)).filter((name) => name.endsWith(".mp3")).sort()
  const chunks = []
  let offsetSec = 0
  for (const name of names) {
    const file = resolve(chunkDir, name)
    const durationSec = await probeDuration(file)
    chunks.push({ file, offsetSec, durationSec })
    offsetSec += durationSec
  }
  return chunks
}

/**
 * 有限并发执行。每个任务失败只影响自己的结果，不取消其余任务。
 * @template T, R
 * @param {T[]} items 待处理项
 * @param {number} limit 同时进行的数量
 * @param {(item: T, index: number) => Promise<R>} worker 单项处理函数
 * @returns {Promise<R[]>}
 */
async function mapPool(items, limit, worker) {
  const results = new Array(items.length)
  let next = 0
  /**
   * 从共享序号里领取下一项，直到队列空。
   * @returns {Promise<void>}
   */
  async function run() {
    while (next < items.length) {
      const index = next
      next += 1
      results[index] = await worker(items[index], index)
    }
  }
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, () => run()))
  return results
}

/**
 * 申请与转写模型绑定的临时上传凭证。
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
 * 上传一个分段，返回 oss:// 临时地址。
 * @param {string} apiKey 百炼密钥
 * @param {string} filePath 本地分段
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
 * 提交一个分段的异步转写。
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
      input: { file_urls: [ossUrl] },
      parameters: { channel_id: [0], language_hints: ["ja"] },
    }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`提交转写失败：${JSON.stringify(body)}`)
  const taskId = body.output?.task_id
  if (!taskId) throw new Error(`没有任务编号：${JSON.stringify(body)}`)
  return taskId
}

/**
 * 轮询一个转写任务。
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
    if (status === "SUCCEEDED") return body.output
    if (status === "FAILED" || status === "UNKNOWN") {
      throw new Error(`转写失败：${JSON.stringify(body.output)}`)
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 2000))
  }
  throw new Error("轮询超过 6 分钟")
}

/**
 * 去掉临时下载地址和字级明细，只保留句子时间码。
 * @param {Record<string, unknown>} transcript 原始转写
 * @returns {Record<string, unknown>}
 */
function slimTranscript(transcript) {
  const tracks = Array.isArray(transcript.transcripts) ? transcript.transcripts : []
  return {
    properties: transcript.properties ?? null,
    transcripts: tracks.map((track) => ({
      channel_id: track.channel_id,
      content_duration_in_milliseconds: track.content_duration_in_milliseconds,
      sentences: (track.sentences ?? []).map((sentence) => ({
        begin_time: sentence.begin_time,
        end_time: sentence.end_time,
        text: sentence.text,
      })),
    })),
  }
}

/**
 * 转写一个分段。已有结果文件时直接复用。
 * @param {string} apiKey 百炼密钥
 * @param {{ file: string, offsetSec: number }} chunk 分段及其在整片中的起点
 * @param {number} index 分段序号
 * @returns {Promise<{ offsetSec: number, sentences: { begin: number, end: number, text: string }[], error?: string }>}
 */
async function transcribeChunk(apiKey, chunk, index) {
  const savedPath = resolve(partDir, `${basename(chunk.file, ".mp3")}.json`)
  try {
    const saved = JSON.parse(await readFile(savedPath, "utf8"))
    console.log(`分段 ${index + 1} 已有结果，跳过`)
    return { offsetSec: chunk.offsetSec, sentences: saved.sentences ?? [] }
  } catch {
    // 没有可复用的结果，继续上传转写。
  }
  try {
    console.log(`分段 ${index + 1} 上传 ${basename(chunk.file)}`)
    const ossUrl = await uploadAudio(apiKey, chunk.file)
    const taskId = await submitTask(apiKey, ossUrl)
    const output = await waitTask(apiKey, taskId)
    const resultItem = output.results?.[0]
    if (!resultItem || resultItem.subtask_status !== "SUCCEEDED") {
      throw new Error(`子任务失败：${JSON.stringify(resultItem ?? output)}`)
    }
    const transcriptResponse = await fetch(resultItem.transcription_url)
    const transcript = slimTranscript(await transcriptResponse.json())
    const sentences = []
    for (const track of transcript.transcripts) {
      for (const sentence of track.sentences) {
        const text = String(sentence.text ?? "").trim()
        if (!text) continue
        sentences.push({
          begin: (sentence.begin_time ?? 0) + chunk.offsetSec * 1000,
          end: (sentence.end_time ?? 0) + chunk.offsetSec * 1000,
          text,
        })
      }
    }
    await writeFile(savedPath, `${JSON.stringify({ offsetSec: chunk.offsetSec, sentences, transcript }, null, 2)}\n`, "utf8")
    console.log(`分段 ${index + 1} 完成，${sentences.length} 句`)
    return { offsetSec: chunk.offsetSec, sentences }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.error(`分段 ${index + 1} 失败：${message}`)
    return { offsetSec: chunk.offsetSec, sentences: [], error: message }
  }
}

/**
 * 把毫秒格式化成时间码。srt 为 true 时用逗号分隔毫秒。
 * @param {number} milliseconds 影片内毫秒
 * @param {boolean} srt 是否使用 SRT 分隔符
 * @returns {string}
 */
function formatClock(milliseconds, srt = false) {
  const total = Math.max(0, Math.round(milliseconds))
  const hours = Math.floor(total / 3_600_000)
  const minutes = Math.floor((total % 3_600_000) / 60_000)
  const seconds = Math.floor((total % 60_000) / 1000)
  const millis = total % 1000
  const pad = (value, width = 2) => String(value).padStart(width, "0")
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}${srt ? "," : "."}${pad(millis, 3)}`
}

/**
 * 调用文本模型翻译一批日语句子，要求返回同样行数。
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
          content: "把每行日语对白译成简体中文。保持行数和顺序，不要编号，不要解释，不要合并或拆行。人名按读音音译并保持前后一致，例如佐々木译作佐佐木，不要换成别的姓。口语直译。",
        },
        { role: "user", content: lines.join("\n") },
      ],
    }),
  })
  const body = await response.json()
  if (!response.ok) throw new Error(`翻译失败：${JSON.stringify(body)}`)
  const content = body.choices?.[0]?.message?.content
  if (typeof content !== "string") throw new Error(`没有译文：${JSON.stringify(body)}`)
  return content.split(/\r?\n/).map((line) => line.replace(/^\d+[.、)\]]\s*/, "").trim()).filter((line) => line.length > 0)
}

/**
 * 翻译一批句子。行数对不上时对半重试，单句失败则保留原文并标明未译。
 * @param {string} apiKey 百炼密钥
 * @param {string[]} lines 日语句子
 * @returns {Promise<string[]>}
 */
async function translateBatch(apiKey, lines) {
  if (lines.length === 0) return []
  try {
    const translated = await translateLines(apiKey, lines)
    if (translated.length === lines.length) return translated
    throw new Error(`译文 ${translated.length} 行，原文 ${lines.length} 行`)
  } catch (error) {
    if (lines.length === 1) {
      const message = error instanceof Error ? error.message : String(error)
      console.error(`单句未译：${message}`)
      return [`［未译］${lines[0]}`]
    }
    const middle = Math.ceil(lines.length / 2)
    const left = await translateBatch(apiKey, lines.slice(0, middle))
    const right = await translateBatch(apiKey, lines.slice(middle))
    return left.concat(right)
  }
}

/**
 * 把全部句子按批次翻译，并写出原文、译文和 SRT。
 * @param {string} apiKey 百炼密钥
 * @param {{ begin: number, end: number, text: string }[]} sentences 按时间排序的句子
 * @returns {Promise<void>}
 */
async function writeTranslations(apiKey, sentences) {
  const translated = []
  for (let start = 0; start < sentences.length; start += translateBatchSize) {
    const batch = sentences.slice(start, start + translateBatchSize).map((sentence) => sentence.text)
    console.log(`翻译 ${start + 1}–${start + batch.length} / ${sentences.length}`)
    const batchText = await translateBatch(apiKey, batch)
    translated.push(...batchText)
  }
  const jaLines = []
  const zhLines = []
  const srt = []
  sentences.forEach((sentence, index) => {
    const cue = `${formatClock(sentence.begin)} --> ${formatClock(sentence.end)}`
    jaLines.push(cue, sentence.text, "")
    zhLines.push(cue, translated[index] ?? "", "")
    srt.push(String(index + 1), `${formatClock(sentence.begin, true)} --> ${formatClock(sentence.end, true)}`, translated[index] ?? "", "")
  })
  await writeFile(resolve(movieDir, "ebwh-273.ja.txt"), `${jaLines.join("\n").trim()}\n`, "utf8")
  await writeFile(resolve(movieDir, "ebwh-273.zh.txt"), `${zhLines.join("\n").trim()}\n`, "utf8")
  await writeFile(resolve(movieDir, "ebwh-273.zh.srt"), `${srt.join("\n").trim()}\n`, "utf8")
}

/**
 * 入口：转写全部分段，再翻译成简体中文。
 * @returns {Promise<void>}
 */
async function main() {
  const env = await readEnvFile(envPath)
  const apiKey = env.DASHSCOPE_API_KEY
  if (!apiKey) throw new Error("qwen-asr.env 里没有 DASHSCOPE_API_KEY")
  await mkdir(partDir, { recursive: true })
  const chunks = await listChunks()
  console.log(`共 ${chunks.length} 段，合计 ${chunks.at(-1) ? (chunks.at(-1).offsetSec + chunks.at(-1).durationSec).toFixed(1) : 0} 秒`)
  const parts = await mapPool(chunks, asrConcurrency, (chunk, index) => transcribeChunk(apiKey, chunk, index))
  const failures = parts.filter((part) => part.error)
  const sentences = parts.flatMap((part) => part.sentences).sort((left, right) => left.begin - right.begin)
  await writeFile(resolve(movieDir, "ebwh-273.sentences.json"), `${JSON.stringify({ failures, sentences }, null, 2)}\n`, "utf8")
  console.log(`转写句子 ${sentences.length}，失败分段 ${failures.length}`)
  if (sentences.length === 0) throw new Error("没有可翻译的句子")
  await writeTranslations(apiKey, sentences)
  console.log(`已写入 ${resolve(movieDir, "ebwh-273.zh.srt")}`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})

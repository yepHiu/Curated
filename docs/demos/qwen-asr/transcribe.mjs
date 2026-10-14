/**
 * 千问 qwen3-asr-flash-realtime 本地试听脚本。
 * 密钥仍读仓库根目录 .workspace/qwen-asr.env，产物写入 .workspace/qwen-asr-demo/work。
 * 用法：node transcribe.mjs [--language ja|zh] [--vad] [--file 音频路径]
 */
import { spawn } from "node:child_process"
import { readFile, writeFile, mkdir } from "node:fs/promises"
import { dirname, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { setTimeout as delay } from "node:timers/promises"

const demoDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(demoDir, "../../..")
const envPath = resolve(repoRoot, ".workspace/qwen-asr.env")
const workDir = resolve(repoRoot, ".workspace/qwen-asr-demo/work")

/**
 * 读取本地 env 文件。忽略空行和注释，不打印密钥。
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
 * 解析命令行。未给文件时由脚本自己准备一段试音。
 * @param {string[]} argv process.argv 去掉 node 与脚本路径后的部分
 * @returns {{ language: string, file: string }}
 */
function parseArgs(argv) {
  let language = "ja"
  let file = ""
  let vad = false
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index]
    if (token === "--language") {
      language = argv[index + 1] ?? language
      index += 1
    } else if (token === "--file") {
      file = argv[index + 1] ?? ""
      index += 1
    } else if (token === "--vad") {
      vad = true
    } else if (!token.startsWith("--") && !file) {
      file = token
    }
  }
  return { language, file, vad }
}

/**
 * 运行外部命令并收集退出码。失败时抛出包含 stderr 的错误。
 * @param {string} command 可执行文件
 * @param {string[]} args 参数
 * @returns {Promise<void>}
 */
function runCommand(command, args) {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(command, args, { windowsHide: true })
    let stderr = ""
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString()
    })
    child.on("error", reject)
    child.on("close", (code) => {
      if (code === 0) resolvePromise()
      else reject(new Error(`${command} 退出码 ${code}: ${stderr.trim()}`))
    })
  })
}

/**
 * 用 Windows 中文语音合成一段 wav，供没有日语声线时验证接口。
 * @param {string} wavPath 输出 wav
 * @param {string} text 要朗读的文本
 * @returns {Promise<void>}
 */
function synthesizeChinese(wavPath, text) {
  const script = `
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
$synth.SelectVoice('Microsoft Huihui Desktop')
$synth.SetOutputToWaveFile(${JSON.stringify(wavPath)})
$synth.Speak(${JSON.stringify(text)})
$synth.Dispose()
`
  return runCommand("powershell", ["-NoProfile", "-Command", script])
}

/**
 * 尝试用 edge-tts 合成日语。未安装时返回 false，由调用方改用中文样本。
 * @param {string} mp3Path 输出 mp3
 * @param {string} text 日语文本
 * @returns {Promise<boolean>}
 */
async function trySynthesizeJapanese(mp3Path, text) {
  try {
    await runCommand("python", [
      "-m",
      "edge_tts",
      "--voice",
      "ja-JP-NanamiNeural",
      "--text",
      text,
      "--write-media",
      mp3Path,
    ])
    return true
  } catch {
    return false
  }
}

/**
 * 把任意 ffmpeg 可读音频转成 16 kHz、16-bit、单声道 PCM。
 * @param {string} inputPath 源音频
 * @param {string} pcmPath 原始 PCM 输出
 * @returns {Promise<void>}
 */
function convertToPcm(inputPath, pcmPath) {
  return runCommand("ffmpeg", [
    "-y",
    "-i",
    inputPath,
    "-ac",
    "1",
    "-ar",
    "16000",
    "-f",
    "s16le",
    "-acodec",
    "pcm_s16le",
    pcmPath,
  ])
}

/**
 * 准备要发送的 PCM，并返回实际使用的语种。
 * @param {{ language: string, file: string }} options 命令行选项
 * @returns {Promise<{ pcm: Buffer, language: string, source: string }>}
 */
async function prepareAudio(options) {
  await mkdir(workDir, { recursive: true })
  if (options.file) {
    const pcmPath = resolve(workDir, "input.pcm")
    await convertToPcm(resolve(options.file), pcmPath)
    return { pcm: await readFile(pcmPath), language: options.language, source: options.file }
  }

  if (options.language === "ja") {
    const mp3Path = resolve(workDir, "sample-ja.mp3")
    const ready = await trySynthesizeJapanese(mp3Path, "今日はいい天気です。音声認識のテストです。")
    if (ready) {
      const pcmPath = resolve(workDir, "sample-ja.pcm")
      await convertToPcm(mp3Path, pcmPath)
      return { pcm: await readFile(pcmPath), language: "ja", source: mp3Path }
    }
  }

  const wavPath = resolve(workDir, "sample-zh.wav")
  await synthesizeChinese(wavPath, "今天天气很好。这是语音识别测试。")
  const pcmPath = resolve(workDir, "sample-zh.pcm")
  await convertToPcm(wavPath, pcmPath)
  return { pcm: await readFile(pcmPath), language: "zh", source: wavPath }
}

/**
 * 按北京公共域名或业务空间域名拼出 WebSocket 地址。
 * @param {Record<string, string>} env 本地配置
 * @returns {string}
 */
function buildSocketUrl(env) {
  const model = env.QWEN_ASR_MODEL || "qwen3-asr-flash-realtime"
  const workspaceId = env.QWEN_ASR_WORKSPACE_ID
  const base = workspaceId
    ? `wss://${workspaceId}.cn-beijing.maas.aliyuncs.com/api-ws/v1/realtime`
    : env.QWEN_ASR_URL || "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
  return `${base}?model=${encodeURIComponent(model)}`
}

/**
 * 把 PCM 按约 100 毫秒一片发给已配置好的会话，并收集最终文本。
 * @param {Record<string, string>} env 含 API Key 的配置
 * @param {Buffer} pcm 16 kHz 单声道 PCM
 * @param {string} language 识别语种代码
 * @param {boolean} vad 为真时由服务端按静音断句；否则整段提交
 * @returns {Promise<string[]>}
 */
function recognize(env, pcm, language, vad) {
  const apiKey = env.DASHSCOPE_API_KEY
  if (!apiKey) throw new Error("qwen-asr.env 里没有 DASHSCOPE_API_KEY")
  const url = buildSocketUrl(env)
  let eventSerial = 0
  /** 生成本次连接内唯一的事件编号。 */
  const nextEventId = () => {
    eventSerial += 1
    return `event_${eventSerial}`
  }

  return new Promise((resolvePromise, reject) => {
    const socket = new WebSocket(url, {
      headers: { Authorization: `Bearer ${apiKey}` },
    })
    const finals = []
    let settled = false
    const audioSeconds = pcm.length / 2 / 16000
    const timeoutMs = Math.max(45_000, Math.ceil(audioSeconds * 1000) + 90_000)
    const timeout = setTimeout(() => {
      finish(new Error(`等待识别结果超过 ${Math.round(timeoutMs / 1000)} 秒`))
    }, timeoutMs)

    /**
     * 结束本次连接。成功和失败都只结算一次。
     * @param {Error | null} error 失败原因；成功时为空
     * @param {string[]} [texts] 最终识别句
     */
    function finish(error, texts) {
      if (settled) return
      settled = true
      clearTimeout(timeout)
      socket.close()
      if (error) reject(error)
      else resolvePromise(texts ?? finals)
    }

    socket.addEventListener("open", () => {
      console.log(`已连接 ${url.replace(/\/\/[^/]+/, "//…")}`)
      socket.send(JSON.stringify({
        event_id: nextEventId(),
        type: "session.update",
        session: {
          modalities: ["text"],
          input_audio_format: "pcm",
          sample_rate: 16000,
          input_audio_transcription: { language },
          turn_detection: vad
            ? { type: "server_vad", threshold: 0.2, silence_duration_ms: 800 }
            : null,
        },
      }))
    })

    socket.addEventListener("message", async (event) => {
      const payload = JSON.parse(event.data.toString())
      if (payload.type === "error") {
        finish(new Error(payload.error?.message || JSON.stringify(payload.error ?? payload)))
        return
      }
      if (payload.type === "session.updated") {
        await sendAudio(socket, pcm, nextEventId, vad)
        return
      }
      if (payload.type === "conversation.item.input_audio_transcription.text") {
        const partial = payload.transcript ?? payload.text ?? payload.stash ?? ""
        if (partial) process.stdout.write(`\r中间结果 ${partial}`)
      }
      if (payload.type === "conversation.item.input_audio_transcription.completed") {
        const text = payload.transcript ?? payload.text ?? ""
        if (text) {
          finals.push(text)
          console.log(`\n最终结果 ${text}`)
        }
      }
      if (payload.type === "session.finished") finish(null, finals)
    })

    socket.addEventListener("error", () => {
      finish(new Error("WebSocket 连接失败。若密钥属于业务空间，请在 qwen-asr.env 填写 QWEN_ASR_WORKSPACE_ID。"))
    })
    socket.addEventListener("close", () => {
      if (!settled) finish(null, finals)
    })
  })
}

/**
 * 在 Manual 模式下送完音频、提交缓冲区并结束会话。
 * @param {WebSocket} socket 已打开的连接
 * @param {Buffer} pcm 原始音频
 * @param {() => string} nextEventId 事件编号生成器
 * @param {boolean} vad 服务端断句时不再发送 commit
 * @returns {Promise<void>}
 */
async function sendAudio(socket, pcm, nextEventId, vad) {
  const chunkBytes = 3200
  for (let offset = 0; offset < pcm.length; offset += chunkBytes) {
    const chunk = pcm.subarray(offset, Math.min(offset + chunkBytes, pcm.length))
    socket.send(JSON.stringify({
      event_id: nextEventId(),
      type: "input_audio_buffer.append",
      audio: chunk.toString("base64"),
    }))
    await delay(20)
  }
  if (!vad) {
    socket.send(JSON.stringify({ event_id: nextEventId(), type: "input_audio_buffer.commit" }))
  }
  socket.send(JSON.stringify({ event_id: nextEventId(), type: "session.finish" }))
}

/**
 * 脚本入口：准备音频、调用识别并写出文本。
 * @returns {Promise<void>}
 */
async function main() {
  const options = parseArgs(process.argv.slice(2))
  const env = await readEnvFile(envPath)
  const audio = await prepareAudio(options)
  const seconds = (audio.pcm.length / 2 / 16000).toFixed(1)
  console.log(`音频 ${audio.source}`)
  console.log(`语种 ${audio.language}，时长约 ${seconds} 秒`)
  console.log(options.vad ? "模式 服务端断句" : "模式 整段提交")
  const lines = await recognize(env, audio.pcm, audio.language, options.vad)
  const outputPath = resolve(workDir, "transcript.txt")
  await writeFile(outputPath, `${lines.join("\n")}\n`, "utf8")
  console.log(lines.length ? `已写入 ${outputPath}` : "会话结束，没有识别文本")
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error)
  process.exitCode = 1
})

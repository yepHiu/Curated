# 日语字幕 ASR 调研与 2026-10-15 实验记录

日期：2026-10-15
状态：research
关联需求：无

本文记录两件事：一是 MiMo 与其它日语识别方案的契约差距；二是当晚用百炼接口对一部本地日语影片做的识别和翻译实验。没有批准实施，没有新增 PRD、API 或配置。后续需求设计以「今晚实验记录」为准，前面的模型表只说明当时为什么换方案。

演示脚本在 `docs/demos/qwen-asr/`，可以随仓库提交。密钥仍在 `.workspace/qwen-asr.env`。音频、转写 JSON 和字幕仍在 `.workspace/qwen-asr-demo/work/`。`/.gitignore` 忽略 `/.workspace/`，密钥和成品不会进入 Git。清理该目录或换机器会丢掉字幕成品，本文只保留方法、用量和质量结论，不粘贴整片对白，也不记录密钥。

官方材料（2026-07 更新）：

- 用法：<https://mimo.mi.com/docs/zh-CN/quick-start/usage-guide/audio/Speech-Recognition>
- 接口：<https://mimo.mi.com/docs/zh-CN/api/audio/Speech-Recognition>
- 模型列表与限流：<https://mimo.mi.com/docs/zh-CN/quick-start/summary/model>
- 模型卡：<https://mimo.mi.com/models/zh-CN/mimo-v2.5-asr>
- 计费：<https://mimo.mi.com/docs/zh-CN/price/pay-as-you-go>

## 结论

MiMo 文档里的「流式调用」不能直接当成实时翻译字幕服务。

它是一次 HTTP 请求：客户端先把整段 wav/mp3 做成 Base64 放进请求体，服务端再以 SSE 逐段吐出识别文本。音频不能边录边送，响应里没有时间戳，也没有译文。语种契约只有中文、英文和中文方言。

Curated 片源以日语对白为主，播放器目前只在诊断里显示字幕轨数量，没有字幕渲染。现有 AI 客户端只发文本 chat completion，不能提交 `input_audio`。

因此这套模型至多覆盖「中/英音频、整段或短切片转写」。播放中的日语实时翻译要另选识别模型；翻译本身还要第二次文本模型调用。

## 模型实际能力

| 项 | 契约 |
|---|---|
| 模型 | 仅 `mimo-v2.5-asr` |
| 地址 | `POST https://api.xiaomimimo.com/v1/chat/completions` |
| 认证 | 请求头 `api-key` |
| 音频 | 仅 wav、mp3；整段 Base64。Data URL 的 MIME：wav 为 `audio/wav`，mp3 为 `audio/mpeg` 或 `audio/mp3` |
| 体积 | Base64 字符串上限 10 MB |
| 语种参数 | `asr_options.language`：`auto` / `zh` / `en`，默认 `auto`，只允许单一语种 |
| 宣传语种 | 中英双语、中英混说，以及粤语、吴语、闽南语、四川话 |
| `stream=true` | 文本用 SSE 增量返回（`chat.completion.chunk` 的 `choices[].delta.content`） |
| 非流式 | `choices[].message.content` 为整段纯文本 |
| 上下文 / 输出 | 8k / 2k token |
| 限流 | 100 RPM，10k TPM |
| 计费 | 按输入音频时长，精确到秒再折算小时。国内 ¥0.5/小时，海外 $0.074/小时 |
| 用量 | `usage.seconds`、`usage.prompt_tokens_details.audio_tokens` |
| 结束原因 | `stop`、`length`、`content_filter` |

官方示例即使勾了流式，也是先读完整本地文件再 Base64。没有 WebSocket、没有分块音频上传、没有预连接。响应字段里没有词级或句级时间戳，没有说话人，没有译文。

采样率、声道和单次时长上限官方未写。10 MB Base64 约等于 7.5 MB 原始音频，粗算上限：

- 16 kHz、16-bit、单声道 WAV：约 4 分钟
- 24 kHz、16-bit、单声道 WAV：约 2.7 分钟
- 64 kbps MP3：约 16 分钟
- 128 kbps MP3：约 8 分钟

输出上限 2k token。超长对白会以 `finish_reason=length` 截断，不能把整部影片一次送入。

社区实现 `mimo-asr-vllm` 称单次超过约 2 分钟后字错率明显变差，并会重采样到 24 kHz、按静音切成不超过 60 秒的片段。这不是小米 API 的承诺，只能当作切片实验的参考。

## 和「实时翻译」差在哪里

本地点播不必做成麦克风式 WebSocket。文件已经在磁盘上，可以按播放时钟提前切出后面几秒音频，在播放头到达前提前识别。MiMo 的文件进、文本出形态在这一点上能用，但只解决「整段上传后的文本更快出现」，不解决下面四件事。

```
播放时钟
    │  提前切 [t, t+W)
    ▼
FFmpeg 抽出单声道 mp3/wav
    │  Base64，须 < 10 MB
    ▼
mimo-v2.5-asr          ← 只有原文，时间只能借用切片窗口
    │
    ▼
文本模型翻译            ← ASR 接口不翻译
    │
    ▼
字幕 cue 叠加到播放器   ← 当前没有这条渲染路径
```

1. **语种。** 资料库对白主要是日语。官方语种表和 `language` 枚举都没有日语。未用日语音频实测，不能断言模型完全无输出，但不能按日语识别服务来设计。
2. **翻译是第二跳。** ASR 只回原文。译文需要另一次文本模型（平台上有 `mimo-v2.6-pro` / `mimo-v2.6-flash`，与 ASR 计费和限流都不同）。延迟、失败和费用叠在识别之后。
3. **没有时间轴。** 一句对白落在哪一秒，只能等于这次切片的起止，或者再接强制对齐。窗口太长，字幕整块出现；窗口太短，词被切断，重叠部分会按全长再计费。
4. **延迟下限是窗口长度。** 服务端要等这一小段音频到齐才开始识别。SSE 只缩短「文本逐字出来」的时间，缩短不了「等音频 + 上传 + 识别 + 翻译」。体感实时通常要求落后人声 2–3 秒以内，所以窗口本身就要落在这个量级，并靠预取把等待藏到播放头前面。

暂停、拖动、倍速、换片、换分部都会使在途请求作废。迟到结果不能写进新的时间轴。

成人对白还有 `content_filter` 风险。接口会整段拦截，但是否拦截这类语音没有样本，不能事先当成可用或不可用。

## Curated 现状

- 播放描述符已有 `audioTracks` / `subtitleTracks`，`PlayerPage` 只在诊断里显示数量，没有字幕轨选择，也没有 cue 叠加。
- Web 播放走直连或 HLS；Windows / macOS 原生播放走 mpv。原生窗口的音轨、字幕仍是后续项。截图明确不包含字幕。
- Server 已有 FFmpeg，能按时间切音频，但没有 ASR 任务或字幕产物。
- `backend/internal/llm` 的消息是文本。MiMo 的 Base URL 虽是 OpenAI 兼容形态，现有客户端不能带 `input_audio` 和 `asr_options`。
- 现有 `aiProvider` 服务于 Agent 文本。语音识别的密钥、模型、按秒计费和内容审核不应复用这套配置。
- 音频离开本机才会到达小米云端。这和本地资料库的默认边界不同，必须由用户显式打开，密钥只留在 Server。

## 若继续做，服务要满足的需求

这些是产品需求，不是本模型已经具备的能力。

**识别与翻译**

- 源语言至少包含日语；目标语言为简体中文。中/英仅作为附加源语言。
- 同一时间轴上同时保留原文和译文，用户可只看译文、只看原文或双语。
- 每条 cue 有起止时间。误差目标需要单独定，建议先以「一句对白落在所说的那几秒内」为验收，而不是词级卡拉 OK。
- 背景音乐、喘息和非语言声音应可跳过或标成无对白，避免把噪声译成句子。
- 识别失败、审核拦截、超时和空结果不中断播放，字幕区显示可理解的失败，并允许重试这一段。

**跟随时钟**

- 以当前 `movieId` + `fileId` 和媒体时间为准，不跟墙钟。
- 播放前预取播放头之后一个短窗口；暂停停止新请求；seek 丢弃旧窗口结果。
- 倍速变化后，预取范围按媒体时间重算。
- 同一文件再次播放优先用已保存 cue，避免重复计费。

**边界与费用**

- 单次音频遵守供应商体积、时长和输出上限；超限由切片解决，不把截断文本当成完整一句。
- 切片重叠只为避免切词，重叠策略要可计算重复计费。
- 密钥、原文、译文和用量只在 Server 持久化。渲染进程不接收密钥，不接收完整音频。
- 远程客户端默认不能把 Server 上的影片音频转发给第三方，除非以后单独定义这项权限。
- 用量至少能按次看到音频秒数和失败原因，便于对账 ¥0.5/小时。

**播放器**

- Web 与原生窗口用同一份 cue，而不是各做一套识别。
- 字幕层不挡住圆形控件的命中区，窄窗口可隐藏或收成一行。
- 切换 Web / 原生播放时，已生成的 cue 仍然可用。

## 采用该模型前必须先测的样本

未做这些试验，不应进入方案或 PRD。

1. 一段清晰日语对白、一段中英对白，各 10 秒和 60 秒，看 `language=auto` 的原文是否可用。
2. 一段明确成人对白，看是否返回 `content_filter` 或空文本。
3. 3 秒、8 秒单声道 MP3 的端到端时间：上传完成、首个 SSE 字、全文结束。确认能否压进播放预取。
4. 同一句切在两个重叠窗口里，是否重复计 `usage.seconds`，译文是否互相打架。
5. 16 kHz 与 24 kHz、单声道与立体声，确认哪种输入被接受以及质量差异。官方未写采样率。

日语样本不可用时，换识别引擎；翻译模型可以之后再选。中/英样本可用、日语不可用，则该模型只适合资料库里的中英片段，不能作为默认实时字幕。

## 日语识别可选方案（2026-10-15）

Curated 要的是本地点播字幕：指定日语、句级时间戳、播放头前预取，音频默认不出本机。按这个条件，本机 Kotoba-Whisper 优先；要托管流式接口时用通义 Fun-ASR-Realtime。翻译仍是识别之后的另一次文本调用。

| 方案 | 日语 | 时间戳 | 调用形态 | 和本需求的关系 |
|---|---|---|---|---|
| Kotoba-Whisper v2.0 / v2.2 | 专为日语，教师模型是 Whisper large-v3，训练语料为 ReazonSpeech 电视音频 | 分段时间戳；v2.2 管道另含标点和说话人分离 | 本机整段或按 15–30 秒切片 | 首选。音频留在本机，无内容审核拦截 |
| Whisper large-v3（faster-whisper / whisper.cpp） | 多语种，日语可用 | 分段时间戳 | 同上 | 对照基线。Kotoba 官方称域内字错率更低，速度约 6.3 倍 |
| Fun-ASR-Realtime `fun-asr-realtime`、`fun-asr-realtime-2026-02-28` | `language_hints=ja` | 句级和字级，毫秒 | WebSocket，可边送 PCM | 云端首选。适合真流式；音频离开本机 |
| Qwen-Audio-3.1-ASR-Flash-Streaming | `language_hints` 含 `ja` | 句级和字级 | WebSocket | 与 Fun-ASR 同类，作备选 |
| 千问 `qwen3-asr-flash` | `language=ja` | 文件转写 `qwen3-asr-flash-filetrans` 有字级时间戳 | HTTP，整段或 URL | 适合离线转写，不适合边播边出 |
| `qwen3-asr-flash-realtime` | 含日语 | 官方写明当前不返回时间戳 | WebSocket | 字幕对齐不够用 |
| SenseVoice `sensevoice-v1` / 开源 SenseVoice-Small | `ja` | 云端为句级毫秒；开源模型按静音切段 | 录音文件 HTTP，非双向流 | 速度好。第三方比较里日语字错率略高于 Whisper large-v3 |
| MiMo-V2.5-ASR | 契约无日语 | 无 | 整段 Base64 后文本 SSE | 不采用 |

Kotoba-Whisper v2.0 保留 large-v3 编码器，解码器压成两层，16 kHz。官方评测写明：ReazonSpeech 留出集优于 large-v3，JSUT 与 Common Voice 日语子集与 large-v3 接近。v2.2 在同一识别结果上加标点和说话人分离。权重是 Hugging Face Transformers 格式；放进 faster-whisper 或 whisper.cpp 前要做对应转换，不能把未核对的第三方转换包当成官方产物。

公开可下载的是权重，不是按量调用的识别服务：

- `kotoba-tech/kotoba-whisper-v2.0`、`kotoba-tech/kotoba-whisper-v2.2` 在 Hugging Face，用 Transformers 在本机跑。
- 公司另有一套实时语音 API，文档在 <https://docs.kotoba.tech>，状态为私有 alpha，联系 `kotoba_product@kotoba.tech` 申请。能力是 WebSocket 流式识别、REST 文件转写（可要分段时间戳）、语音到语音翻译和日语 TTS。输入语种写明英、日、韩、中。SDK 为 `kotoba-sdk`。文档没有把这套接口写成 Kotoba-Whisper v2.2 的托管版。
- `kotoba-sdk` 还列出 fal 上的 `Kotoba-Technologies/kotoba-stt` 与 `kotoba-streaming-stt` 部署地址，并写明 fal 不支持其批量任务接口。这是自备密钥后的部署目标，不是打开即用的公开价目页。

Fun-ASR 实时主版本和 `2026-02-28` 快照都接受日语提示，默认返回句子起止和字级时间。8 kHz 的 `fun-asr-flash-8k-realtime` 只有中文，不能拿来识别日语。

云端接口仍要单独用成人对白样本看是否被审核截断。本机 Whisper 系没有这条拦截，但会把音乐、喘息和非语言声音听写成句子，字幕前需要按置信度或静音丢掉这类片段。

## 今晚实验记录（2026-10-15）

当晚的目标从「边播边出实时翻译」改成了验证一条更稳的离线链路：音频文件 → 带时间戳的日语文本 → 同一时间码的简体中文。实时接口只作为对照，没有拿来出整片字幕。Curated 播放器仍没有字幕渲染，这些实验没有接入产品。

### 样本

本地文件 `D:\asr-test\EBWH-273.mp4`。时长 02:01:28.45（7288.4485 秒），约 6.6 GB，H.264 1920×1080 59.94 fps，AAC 48 kHz 立体声。对照段是 42:00–45:00，因为这里有课堂讲解、人名、拉面和香水等可核对的对白；片中其它 12 秒抽查大多只有「あ」「うん」。

音轨处理：FFmpeg 只映射音轨，压成 16 kHz、单声道、64 kbps MP3，约 56 MB。整片再按约 5 分钟切成 25 段。MP3 容器起点约有 69 毫秒偏移，分段时长用 ffprobe 累加，不假设每段正好 300 秒。

### 试过的调用

| 顺序 | 接口 | 样本 | 结果 |
|---|---|---|---|
| 1 | `qwen3-asr-flash-realtime` | edge-tts 合成的一句日语 | 原文完全正确 |
| 2 | 同上，服务端 VAD | 片中若干 12 秒窗口，以及 42:00–45:00 | 有对白的窗口能出日语；非对白被写成「あ」「うん」。180 秒约 56 秒返回。没有时间戳 |
| 3 | `qwen-audio-3.1-asr-flash-filetrans` | 同一段 180 秒 | 约 7 秒返回。句级和字级时间戳。任务 `0d6dd1c7-de4f-419e-aae0-2606ac30eb2e` |
| 4 | `qwen-plus` 文本翻译 | 第 3 步的句子 | 时间码保持不变。未加译名约束时「佐々木」被译成佐藤 |
| 5 | 3.1 文件转写 + `qwen-plus` | 整片 25 段 | 25 段全部成功，660 句。转写加翻译约 199 秒。0 句未译 |
| 6 | `qwen3-asr-flash-filetrans` | 同一段 180 秒 | 约 5 秒返回。用量 90 秒。任务 `647b5929-a8dc-43b0-8fc4-ea34453b867b` |

实时地址用的是公开北京入口 `wss://dashscope.aliyuncs.com/api-ws/v1/realtime`，模型 `qwen3-asr-flash-realtime`。密钥前缀是工作空间受限的 `sk-ws-`，没有从密钥里猜测 WorkspaceId；这个公开入口可以直接用。音频为 16 kHz、16-bit、单声道 PCM。手工模式要 `commit`，VAD 模式不要。该模型官方说明不返回时间戳。

文件转写地址是 `POST https://dashscope.aliyuncs.com/api/v1/services/audio/asr/transcription`，请求头必须有 `X-DashScope-Async: enable`。本地路径不能放进 `file_urls`。做法是先 `GET /api/v1/uploads?action=getPolicy&model=<与后续调用相同的模型>`，把文件 POST 到返回的临时 OSS，得到 48 小时有效的 `oss://` 地址，调用时再加 `X-DashScope-OssResourceResolve: enable`。上传凭证按主账号加模型限流 100 QPS，文档写明不适合生产。3.1 的字段是 `input.file_urls`（数组，一次一个）和 `parameters.language_hints`；千问 3 的字段是 `input.file_url`（单个字符串）和 `parameters.language`。结果地址也不同：3.1 在 `output.results[].transcription_url`，千问 3 在 `output.result.transcription_url`。下载到的 JSON 里有带签名的临时文件地址，落盘前已抹掉。

同一密钥还能调用兼容接口 `POST https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions` 的 `qwen-plus`。翻译要求逐行对齐；行数不符就对半重试，单句失败才保留原文并标「未译」。整片这次没有走到单句失败。

### 费用（北京标价，按返回的 usage 计算）

| 模型 | 标价 | 这段 180 秒的 usage | 这段费用 |
|---|---|---|---|
| 千问 3 文件转写 | ¥0.00022 / 秒 | `seconds: 90` | 约 ¥0.0198 |
| 千问 3.1 文件转写 | 输入 ¥0.8 / 百万 token，输出 ¥2.7 / 百万 token | 语音 69 秒；输入 3128 token，输出 209 token | 约 ¥0.0031 |
| 千问 3 实时 | 文档为 ¥0.00033 / 秒 | 当晚没有保存该次 usage | 未按账单复核 |

3.1 的模型卡用 token 标价，看起来比「每秒不到两毫」贵。这段实测输入大约每秒语音 45 个 token，所以 3.1 更便宜。两边都只计被判定为语音的时长，不把整段 180 秒都算进去。若整片语音占比接近这段，外推大约是 3.1 ¥0.12、千问 3 ¥0.80。这不是整片账单，整片也没有再用千问 3 重跑。

### 同一段 42:00–45:00 的质量

三边都没有被内容审核截成空文本。实时接口把开头非对白写成很长的「うん」，课堂从半句「三ルート」才接上，香水对白没有出现。两个文件模型都跳过了开头约 21 秒，并写出了「この香り」「香水」。

文件模型之间：

- 千问 3 听对了「皆さん解けましたか」。3.1 听成「溶けましたか」。
- 3.1 句子更短，标点和数字更像字幕：「57ページ」「2の2乗」。千问 3 写成「五十七ページ」「二の二乗」，并把多句并成一段。默认没有做逆文本归一化。
- 3.1 多出「大丈夫かな」和「カバン消してくれない」附近几句，但「カバン」一句听着仍不可靠。千问 3 把「悩めるお年頃」听成「悩める私ごろ」，走廊里的「佐々木」写成平假名。
- 千问 3 每句带 `emotion`（这段出现了 neutral、happy、angry、sad）。这些标签不能当成说话人情绪的事实。
- 3.1 的 JSON 固定带字级时间和置信度。千问 3 默认只有句级；字级要另传 `enable_words: true`，日语在支持范围内。当晚千问 3 这次没有打开字级。

翻译会把听写错误原样译出去。第一版译文没有译名约束，「佐々木」变成了「佐藤」。整片翻译的提示里要求按读音音译并保持一致，结果是「佐々木」译成佐佐木；`00:45:47` 的「さとしさん」仍被猜成「佐藤先生」，片中人名没有核对。

### 整片成品

3.1 转写加 `qwen-plus` 翻译，25 段无失败。660 句，译文行数与原文一致。第一句 `00:00:29.900`「我来打个电话试试。」，最后一句 `02:00:28.614`「这种事情，就我们两个人的秘密哦。」影片在 02:01:28 结束。

本机文件：

- `.workspace/qwen-asr-demo/work/movie/ebwh-273.ja.txt`：整片日语，带时间码
- `.workspace/qwen-asr-demo/work/movie/ebwh-273.zh.txt`：整片简体中文，同一时间码
- `.workspace/qwen-asr-demo/work/movie/ebwh-273.zh.srt`：可挂到播放器的中文字幕
- `.workspace/qwen-asr-demo/work/movie/parts/`：25 段的句子 JSON，可断点续跑
- `.workspace/qwen-asr-demo/work/filetrans-42-45.txt`：3.1 的 180 秒对照
- `.workspace/qwen-asr-demo/work/filetrans-42-45-qwen3.txt`：千问 3 的 180 秒对照
- `.workspace/qwen-asr-demo/work/transcript-42-45.txt`：实时接口的 180 秒对照

演示脚本在 `docs/demos/qwen-asr/`：`transcribe.mjs`（实时）、`filetrans.mjs`（3.1）、`filetrans-qwen3.mjs`（千问 3）、`translate.mjs`（单段翻译）、`movie-translate.mjs`（整片，已有分段会跳过）。运行方式见该目录的 `README.md`。

切段是为了避开 3.1 模型卡上的 1024 token 输出上限。180 秒那次返回的 JSON 已经明显超过 1024 token，所以上限未必真的截断文件转写结果；5 分钟一段是保守做法。代价是切点可能把一句对白拆开。整片没有做段间重叠，也没有人工校对。

### 敏感词过滤

3.1、Fun-ASR 和 Paraformer 在不传 `special_word_filter` 时启用百炼内置词表，命中词替换成等长 `*`。`system_reserved_filter` 默认 `true`。设为 `false`，并把 `filter_with_signed` 与 `filter_with_empty` 的词表留空，即可关掉系统词表、也不追加自定义词。当晚已经生成的 180 秒原文和整片原文里没有 `*`，说明默认词表没有改写这些结果。还没有用 `system_reserved_filter: false` 重跑，所以只确认了文档开关，没有做开关前后的差异对照。

千问 3 文件转写的文档没有把这套默认过滤写成它的能力。它另有固定开启的 7 类情绪。

### 留给后续需求设计的事实

这些是当晚已经看到的约束，不是已批准的方案。

- 离线「整段音频 → 时间戳原文 → 时间戳译文」在这部日语片上跑通了，比实时识别更适合做可保存、可重放的字幕。
- 实时模型能识别日语，但没有时间戳，并且会把非对白写成「あ」「うん」，不适合直接当字幕源。
- 识别和翻译是两次调用。听写错会进入译文；人名要有稳定译名，不能让模型每次重猜。
- 3.1 在这段样本上又好又便宜：句子边界和数字更适合字幕，费用约为千问 3 的六分之一。千问 3 的优势是个别同音词更准，以及自带情绪字段。
- 音频会离开本机，进入百炼临时空间，48 小时后失效。正式使用要换成自有存储上的稳定地址，不能依赖上传凭证接口。
- 密钥不能进渲染进程，也不能复用现有 `aiProvider`。语音按秒或按 token 计费，和 Agent 文本不是同一笔账。
- 播放器还没有字幕层。Web 与原生窗口要共用同一份 cue。已生成的字幕应能再次播放时直接用，避免重复计费。
- 5 分钟硬切会拆句。以后如果改切片，重叠部分的重复计费要单独算。
- 这部片的云端调用没有被内容审核截断，但不能据此写成「成人对白永远不会被拦」。3.1 的系统敏感词表仍默认开启。
- 本机 Kotoba-Whisper 当晚没有跑。它仍是音频不出机器时的候选，和这次云端实验是两条路。

### 当晚明确没做

- 没有把整片再交给千问 3 或实时接口。
- 没有打开千问 3 的字级时间戳，没有开 3.1 的说话人分离、热词或上下文。
- 没有用关闭敏感词过滤的参数重跑。
- 没有人工听校整片，没有把字幕接进 Curated。
- 没有新增 PRD、HTTP API、`library-config` 字段或数据库迁移。
- 没有把密钥、音频或整片对白写入 Git。脚本后来放到 `docs/demos/qwen-asr/`，只有这份代码可以提交。

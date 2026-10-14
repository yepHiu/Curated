# 百炼日语转写与翻译演示

2026-10-15 的本地实验脚本。用百炼把日语音频转成带时间码的原文，再译成简体中文。实验结论在 [调研记录](../../plan/2026-10-15-realtime-translation-asr.md) 的「今晚实验记录」。

在仓库根目录运行。需要 Node.js 和 FFmpeg。密钥文件保持为 `.workspace/qwen-asr.env`，至少包含 `DASHSCOPE_API_KEY`。脚本只读取它，不会把密钥写进本目录。

产物写到 `.workspace/qwen-asr-demo/work/`，或写到 `--file` 音频所在目录。不要把这些成品提交到 Git。

| 脚本 | 作用 |
|---|---|
| `transcribe.mjs` | `qwen3-asr-flash-realtime`。可选 `--vad`、`--file`、`--language` |
| `filetrans.mjs` | `qwen-audio-3.1-asr-flash-filetrans`。`--file`、`--offset-sec`、`--language` |
| `filetrans-qwen3.mjs` | `qwen3-asr-flash-filetrans`。`--file`、`--offset-sec` |
| `translate.mjs` | 用 `qwen-plus` 把时间轴文本译成简体中文。`--file` |
| `movie-translate.mjs` | 转写 `.workspace/qwen-asr-demo/work/movie/chunks/` 里已切好的分段，再翻译整片。已有分段结果会跳过 |

示例：

```bash
node docs/demos/qwen-asr/filetrans.mjs --file .workspace/qwen-asr-demo/work/ebwh-273-42-45.wav --offset-sec 2520 --language ja
node docs/demos/qwen-asr/translate.mjs --file .workspace/qwen-asr-demo/work/filetrans-42-45.txt
```

文件转写会先把音频上传到百炼临时空间，地址约 48 小时有效。3.1 默认开启系统敏感词过滤；要关闭，在 `filetrans.mjs` 的 `parameters` 里加入 `special_word_filter.system_reserved_filter: false`。

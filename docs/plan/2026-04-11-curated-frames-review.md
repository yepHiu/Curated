# 2026-04-11 萃取帧功能现状梳理与改进建议

> 更新于 2026-09-06：本次代码复核与性能 / 截图 UI 建议见 §11。§1–10 保留为 4 月历史分析，其中“尚无分页、二进制上传、缩略图”等描述已过时，不应再作为当前待办。

## 1. 结论先行

当前项目里的“萃取帧”已经不是一个单点截图按钮，而是一条相对完整的能力链：

- 播放页支持在当前时间点截取画面，并把帧与影片、演员、时间点关联起来。
- 萃取帧库页支持集中浏览、按时间/演员/影片分组查看、标签管理、从帧跳回播放页、单张或批量导出。
- 在 `VITE_USE_WEB_API=true` 时，数据持久化到后端 SQLite，并支持带元数据的 WebP / PNG 导出。
- 在 Mock 模式下，数据走浏览器 IndexedDB，本地体验完整，但后端导出能力不对等。

从产品形态看，这个功能已经具备“收藏关键镜头并二次利用”的雏形。  
从工程实现看，它的主链路已经打通，但目前仍偏“单机图库 + 小规模数据”设计，随着帧数量增长、跨设备使用、导出诉求变复杂，现有结构会出现性能、语义和扩展性问题。

## 2. 功能范围与入口

### 2.1 用户入口

用户主要从两个位置触发这项功能：

- 播放页快捷键 `C`，对应 `PlayerPage.vue` 中的 `KeyC -> runCuratedCapture()`。
- 播放页工具栏上的 `Curated` 操作，文案在设置页与本地化中已明确说明。

相关代码：

- `src/components/jav-library/PlayerPage.vue`
- `src/lib/curated-frames/save-capture.ts`
- `src/locales/zh-CN.json`
- `src/components/jav-library/SettingsPage.vue`

### 2.2 路由与页面

萃取帧库是一个独立页面路由：

- 路由名：`curated-frames`
- 页面：`src/views/CuratedFramesView.vue`
- 主体列表组件：`src/components/jav-library/CuratedFramesLibrary.vue`
- 批量操作底栏：`src/components/jav-library/CuratedFramesBatchActionBar.vue`

此外，侧边栏与设置页概览也会显示萃取帧数量：

- `src/components/jav-library/AppSidebar.vue`
- `src/services/adapters/web/web-library-service.ts`
- `src/services/adapters/mock/mock-library-service.ts`

### 2.3 双模式行为

这个功能和项目其他用户态数据一样，分成两套存储路径：

- Web API 模式：前端经 `/api/curated-frames` 调用后端，元数据与图像保存到 SQLite。
- Mock 模式：前端把图像和元数据写入 IndexedDB，本地可浏览、可标签管理，但没有后端导出等价物。

相关实现：

- `src/lib/curated-frames/db.ts`
- `backend/internal/server/playback_curated_handlers.go`
- `backend/internal/storage/playback_curated.go`

## 3. 当前实现怎么工作

### 3.1 播放页截帧链路

播放页按下 `C` 后，`runCuratedCapture()` 会：

1. 检查当前是否有可用视频源。
2. 触发快门反馈动效。
3. 调用 `saveCuratedCaptureFromVideo()`。
4. 成功后显示 `+1` 提示，失败则显示错误。

其中 `saveCuratedCaptureFromVideo()` 做了几件事：

1. 用 `canvas.drawImage(video, ...)` 把当前帧绘制为 PNG Blob。
2. 计算这张帧对应的播放时间点。
3. 组装一条帧记录：`id / movieId / title / code / actors / positionSec / capturedAt / tags`。
4. Web API 模式下把 Blob 转成 base64，调用 `POST /api/curated-frames`。
5. Mock 模式下直接写 IndexedDB。
6. 再根据用户设置决定是否额外下载到浏览器，或写入本地目录。

关键点：

- 捕获格式固定是 PNG，前端文件名规则由 `formatFrameFilename()` 生成。
- 时间点支持显式覆盖，避免 HLS / 会话播放时只使用局部 `currentTime`。
- 保存策略和持久化是两层概念：
  - 持久化位置由 Web API / Mock 模式决定；
  - “保存方式”只决定是否额外下载或写本地目录。

相关代码：

- `src/components/jav-library/PlayerPage.vue:1159`
- `src/lib/curated-frames/capture.ts:9`
- `src/lib/curated-frames/save-capture.ts:47`
- `src/lib/curated-frames/settings-storage.ts:5`
- `src/lib/curated-frames/export-file.ts:1`

### 3.2 设置页里的“保存方式”

设置页定义了三种保存方式，值保存在浏览器 `localStorage`：

- `app`：只保存到应用库。
- `download`：保存到应用库，同时触发浏览器下载一张 PNG。
- `directory`：保存到应用库，同时尝试写入用户授权的本地文件夹。

目录句柄本身存放在 IndexedDB 的 `kv` store 中，而不是后端设置。

这意味着：

- 即使在 Web API 模式下，这个“目录保存”也是浏览器侧行为。
- 如果前端页面和后端不在同一台机器上，后端库与“导出到本地文件夹”会落在不同设备。
- 这套语义对单机桌面使用是合理的，但对“浏览器访问远端 Curated 后端”的理解门槛较高。

相关代码：

- `src/components/jav-library/SettingsPage.vue:1154`
- `src/components/jav-library/SettingsPage.vue:1267`
- `src/components/jav-library/SettingsPage.vue:3194`
- `src/lib/curated-frames/db.ts:153`
- `src/lib/curated-frames/settings-storage.ts:3`

### 3.3 萃取帧库页

萃取帧库页有三种浏览维度：

- 按截取时间 `timeline`
- 按演员 `actors`
- 按影片 `movies`

页面核心流程：

1. 调用 `listCuratedFramesByCapturedAtDesc()` 取出全部帧元数据。
2. 通过 `cfq` 查询参数做页内搜索。
3. 生成展示用 `imageUrl`：
   - Mock 模式：`URL.createObjectURL(imageBlob)`
   - Web API 模式：`GET /api/curated-frames/{id}/image`
4. 支持打开详情弹窗、编辑标签、删除、从帧回播、批量导出。

几个实现细节值得注意：

- 搜索范围只在萃取帧库内部，不混用资料库的 `q` 或 `tag`。
- “按演员”视图下，一个多演员帧会出现在多个演员分组中。
- “按演员”视图下的批量导出，只允许选择同一演员组中的帧，避免导出的文件名演员段混杂。
- 详情弹窗关闭时才保存标签；从标签跳转筛选和从帧回播前，也会先尝试保存标签。

相关代码：

- `src/components/jav-library/CuratedFramesLibrary.vue:343`
- `src/components/jav-library/CuratedFramesLibrary.vue:356`
- `src/components/jav-library/CuratedFramesLibrary.vue:384`
- `src/components/jav-library/CuratedFramesLibrary.vue:410`
- `src/lib/curated-frames/search.ts:16`
- `src/lib/library-query.ts:196`
- `src/lib/player-route.ts:48`

### 3.4 后端 API 与存储

后端对萃取帧提供了五类能力：

- `GET /api/curated-frames`：返回全部帧元数据，不带图像字节。
- `POST /api/curated-frames`：创建一条帧记录，图像通过 `imageBase64` 上传。
- `GET /api/curated-frames/{id}/image`：取图像字节。
- `PATCH /api/curated-frames/{id}/tags`：覆盖标签列表。
- `DELETE /api/curated-frames/{id}`：删除帧。
- `POST /api/curated-frames/export`：导出单张或多张帧。

SQLite 表结构如下：

- 表：`curated_frames`
- 字段：`id / movie_id / title / code / actors_json / position_sec / captured_at / tags_json / image_blob / created_at`
- 索引：`movie_id`、`captured_at DESC`

也就是说，当前后端模型是“元数据 + 原始 PNG BLOB 一表存储”的典型小图库方案。

相关代码：

- `backend/internal/server/server.go:239`
- `backend/internal/server/playback_curated_handlers.go:100`
- `backend/internal/storage/playback_curated.go:88`
- `backend/internal/storage/migrations/0006_playback_and_curated_frames.sql:11`

### 3.5 导出链路

导出是目前这项功能里最“产品化”的部分。

后端导出流程：

1. 接收 `ids`，去重后限制 1 到 20 条。
2. 可选接收 `actorName`，用于生成文件名中的演员段。
3. 从数据库按请求顺序取出帧与图像。
4. 输出两种格式之一：
   - WebP：把元数据写入 EXIF `UserComment`
   - PNG：把元数据写入 `iTXt` 的 `CuratedMeta`
5. 单张直接返回图片，多张打包 ZIP。

导出文件名规则为：

- `curated-{actor}-{code}-{sec}s.webp`
- `curated-{actor}-{code}-{sec}s.png`

如果重名，会追加帧 id 前缀做区分。

当前嵌入导出的元数据字段包括：

- `title`
- `code`
- `actors`
- `positionSec`
- `capturedAt`
- `frameId`
- `movieId`

注意：当前并没有把帧标签 `tags` 一起写进导出元数据。

相关代码：

- `backend/internal/server/curated_export_handler.go:57`
- `backend/internal/curatedexport/webp.go:14`
- `backend/internal/curatedexport/png_itxt.go:86`
- `backend/internal/curatedexport/filename.go:27`
- `backend/internal/curatedexport/actor.go:14`

## 4. 这个功能目前做得好的地方

### 4.1 前后端边界相对清晰

前端通过 `db.ts` 这一层把 Web API / IndexedDB 的差异吸收掉了，页面层大部分不关心后端还是本地存储。对当前项目“Mock 与 Web API 双模式并存”的阶段来说，这个设计是合适的。

### 4.2 用户价值链是闭环的

现在这项功能不只是“截一张图”，而是：

- 截帧
- 入库
- 搜索
- 分类浏览
- 打标签
- 回跳原片时间点
- 单张或批量导出

这说明产品方向已经从“工具按钮”走向“个人镜头素材库”。

### 4.3 导出考虑了二次利用场景

导出时不是只输出像素，而是带上了结构化元数据。这一点很重要，因为它使萃取帧具备继续被外部脚本、图库工具、AI 流程或整理工具消费的基础。

### 4.4 对当前使用规模而言，实现成本控制得不错

SQLite 单表 + BLOB、前端全量拉取 + 本地分组，是很典型的“先把体验做出来”的方案。对于小规模使用者，这种实现简单、稳定、维护成本低。

## 5. 我认为当前存在的主要问题

### 5.1 这套实现默认假设“帧数量不会太多”

当前很多地方都依赖“把全部帧先拉回来再算”：

- 萃取帧库页先全量 `GET /api/curated-frames`，再前端搜索和分组。
- 设置页的萃取帧统计数量，是 `listCuratedFrames()` 后取 `items.length`。
- 侧边栏数量也是再次读取全部帧来计数。

这意味着当帧数上百、上千后，问题会集中出现：

- 首屏等待变长。
- 多处重复请求同一批数据。
- 页内搜索和分组都压在前端。
- Web API 模式下图片又是单独逐张拉取，网络往返会变多。

证据：

- `src/lib/curated-frames/db.ts:64`
- `src/services/adapters/web/web-library-service.ts:135`
- `src/components/jav-library/AppSidebar.vue:132`
- `backend/internal/server/playback_curated_handlers.go:100`

### 5.2 图像存储与展示仍然偏“原图直出”

当前 `GET /api/curated-frames/{id}/image` 返回的是原始 PNG BLOB，列表页也直接拿这个接口当缩略图源。  
这有两个直接后果：

- 列表场景会用全尺寸图片做缩略展示，浪费 IO、内存和带宽。
- 后端 SQLite 会持续积累 PNG BLOB，库文件增长较快。

对一个“会不断追加截图”的功能来说，这个存储模型在中长期会吃掉不少性能预算。

证据：

- `backend/internal/server/playback_curated_handlers.go:149`
- `backend/internal/storage/playback_curated.go:150`
- `src/components/jav-library/CuratedFramesLibrary.vue:361`

### 5.3 Web API 上传路径使用 base64，不够经济

Web API 模式下，前端先把 PNG Blob 转成 base64，再放进 JSON 的 `imageBase64` 字段上传。  
这会带来：

- 体积膨胀；
- 前端额外一次编码成本；
- 后端额外一次解码成本；
- 大图时内存峰值更高。

这在功能刚做出来时可接受，但不是长期最优解。

证据：

- `src/lib/curated-frames/save-capture.ts:12`
- `src/lib/curated-frames/save-capture.ts:77`
- `backend/internal/server/playback_curated_handlers.go:176`

### 5.4 “保存方式”与“实际存储位置”语义容易让人误解

设置页文案里，“仅保存到应用库 / 保存并自动下载 / 自动保存至本地文件夹”更像在描述“帧最终保存到哪里”。  
但实际行为是：

- 应用库位置由 Web API / Mock 模式决定；
- “下载”与“目录”只是应用库之外的附加输出动作；
- 且目录句柄只存在当前浏览器。

这对作者本人和单机场景没问题，但如果未来支持远程访问、托盘后端、多端使用，就会形成明显认知偏差。

### 5.5 标签保存时机比较脆弱

详情弹窗里标签修改并不是显式保存，而是在以下时机尝试写回：

- 关闭对话框时；
- 从标签跳转筛选时；
- 从帧回播时。

这套方式的问题是：

- 用户没有“保存成功/失败”的明确反馈；
- 这些调用点没有完整错误处理分层；
- 如果网络失败，用户可能已经离开当前上下文。

特别是 `handleDialogOpenChange()`、`browseCuratedFramesByTag()`、`playFromFrame()` 都直接 `await updateCuratedFrameTags(...)`，但失败时缺少可见的补救路径。

证据：

- `src/components/jav-library/CuratedFramesLibrary.vue:510`
- `src/components/jav-library/CuratedFramesLibrary.vue:596`
- `src/components/jav-library/CuratedFramesLibrary.vue:612`

### 5.6 导出元数据不完整

导出时确实嵌入了结构化元数据，这是优点；但当前字段里不包含：

- 帧标签 `tags`
- 影片路径 / 来源文件名
- 应用版本 / 导出版本
- 可供后续兼容升级的 schema version

如果以后要把导出的帧再导回、再筛选、再和应用内数据对齐，现在的元数据还不够完整。

证据：

- `backend/internal/curatedexport/webp.go:15`
- `backend/internal/server/curated_export_handler.go:125`

### 5.7 重复帧与近重复帧没有任何治理

当前每次截帧都会直接生成一个新的 UUID 并保存。  
也就是说，连续按两次 `C`，哪怕是同一帧，也会得到两条独立记录。

这在早期是简单直接的，但以后会带来：

- 库里重复帧越来越多；
- 批量导出前需要人工清理；
- 标签维护成本被拉高。

证据：

- `src/lib/curated-frames/save-capture.ts:61`

## 6. 我的看法

我认为这项功能的方向是对的，而且已经明显超出了“附带功能”的程度。

如果继续沿当前产品路线走，萃取帧更适合被定义成：

- “带上下文信息的镜头素材库”

而不是：

- “播放器顺手截个图”

这两个定义的差别会直接影响后面的设计优先级。

如果把它当素材库，就会自然推导出下一步需求：

- 需要更好的检索和过滤。
- 需要更清晰的导出和归档语义。
- 需要更稳定的元数据。
- 需要考虑规模增长后的性能。

如果只是播放器附带截图，那当前实现已经够用。  
但从现有代码看，项目已经在做标签、回播、批量导出、按演员分组，这些都明显是“图库”思路，而不是简单截图思路。

所以我的判断是：  
这块功能值得继续投入，但下一阶段应该从“把功能补齐”切换到“把模型做稳、把规模能力补上”。

## 7. 改进建议

下面按优先级给建议。

### 7.1 P1：先把数据面从“小规模全量拉取”改成可扩展

建议新增后端查询能力，而不是一直全量读取：

- `GET /api/curated-frames?query=&actor=&movieId=&tag=&limit=&offset=`
- `GET /api/curated-frames/stats`
- `GET /api/curated-frames/tags`
- `GET /api/curated-frames/actors`

前端再按这个能力拆成：

- 统计卡、侧边栏只拿 count。
- 搜索页拿分页结果。
- 标签建议单独拉 suggestion pool，而不是依赖当前已加载全部数据。

原因：

- 这是解决后续性能问题的总开关。
- 不改这层，前端体验优化的收益会很有限。

### 7.2 P1：补缩略图模型，别再让列表吃原图

建议把“原图展示”和“列表缩略图”分开：

- 入库时生成缩略图，或首次访问时懒生成并缓存。
- API 拆成原图与缩略图两个通道。
- 列表页优先请求缩略图，详情页再请求原图。

可选实现：

- SQLite 同表增加 `thumb_blob`
- 或改成文件系统落盘 + SQLite 存路径
- 或统一抽象成资产服务，后续和封面/预览图共享基础设施

原因：

- 这项改动对感知性能提升很直接。
- 它也会倒逼存储结构从“纯 BLOB 表”走向更可维护的形态。

### 7.3 P1：把上传从 base64 JSON 改成二进制上传

建议将 `POST /api/curated-frames` 改为以下任一形式：

- `multipart/form-data`
- `application/octet-stream + metadata json`
- 先发 metadata，再发 image blob

我更倾向于 `multipart/form-data`，原因是前后端都容易落地，也更符合浏览器上传习惯。

收益：

- 降低传输体积和内存峰值。
- 减少一次 base64 编解码。
- 为以后支持更大图片、批量上传、拖拽导入打基础。

### 7.4 P1：重新定义“保存方式”的产品语义

建议把当前设置拆成两个维度，而不是一个单选：

- “帧入库位置”
  - 当前设备浏览器
  - Curated 后端库
- “额外输出动作”
  - 无
  - 浏览器下载
  - 写入本地文件夹

如果不想现在就改交互，至少应该在文案层明确：

- 不论选哪种保存方式，应用库仍会照常保存；
- “下载”和“目录”是附加输出；
- “目录”只对当前浏览器生效。

这是一个典型的“功能已经能用，但概念还没讲清”的问题。

### 7.5 P2：把标签编辑从“隐式提交”改成“可感知提交”

建议至少做两件事：

- 详情弹窗中增加明确的保存状态提示，失败时保留草稿并可重试。
- 把 `close / jump / filter` 前的标签提交收敛到统一动作里，补齐错误处理和 toast。

如果再进一步，可以考虑：

- 失焦自动保存，但带状态提示；
- 或保持显式“保存”按钮，保证语义明确。

这会显著降低“改了标签但不确定有没有成功”的不安感。

### 7.6 P2：丰富导出元数据，给未来兼容留版本

建议在导出元数据里增加：

- `tags`
- `schemaVersion`
- `exportedAt`
- `appName`
- `appVersion`
- 可选的 `sourceFilename`

如果以后考虑导回应用，还可以预留：

- `hash`
- `width`
- `height`
- `sourceContainer`

这项改动工作量不大，但回报很高，因为一旦用户开始把导出的帧作为长期资产保留，元数据完整性就很重要。

### 7.7 P2：增加重复帧治理

建议分两步做：

第一步，轻量去重：

- 对同一 `movieId + positionSec` 在很小阈值内的重复捕获给出提示。
- 或提供“连续按键仅保留最近一张”的可选策略。

第二步，增强治理：

- 生成简单视觉哈希或感知哈希；
- 对近重复帧做聚类或提示。

这不一定要立刻做，但如果萃取帧页会成为高频入口，重复治理迟早会成为刚需。

### 7.8 P3：补“素材库级”的组织能力

如果未来继续强化这块功能，我建议考虑这些能力：

- 收藏 / 星标萃取帧
- 批量打标签
- 按时间段连续截帧
- 过滤“只看有标签”
- 过滤“只看某演员 + 某影片”
- 支持导出命名模板
- 支持从萃取帧反查影片详情或演员详情

这部分不是当前必须项，但它们和现在的模型是连续的，不是另起炉灶。

## 8. 我建议的落地顺序

如果只做一轮较务实的迭代，我建议顺序是：

1. 后端增加分页 / 过滤 / count / tags 聚合接口。
2. 前端把侧边栏统计、设置页统计、萃取帧列表改成按需请求。
3. 详情弹窗标签保存补状态反馈与重试。
4. 上传改成二进制通道。
5. 增加缩略图能力。
6. 再考虑去重和高级组织能力。

原因很简单：

- 前三项解决的是“现在就会碰到”的性能与体验问题。
- 后三项解决的是“功能继续长大之后”的结构性问题。

## 9. 补充：功能演进脉络

从提交历史看，这项功能大概经历了三个阶段：

### 9.1 初始落地

提交 `c60c0bd4`

- 引入萃取帧库页、路由、播放页截帧、后端存储与 HTTP、Mock / Web 双模式。

### 9.2 导出能力补齐

提交 `c27c344b`

- 新增后端导出链路、WebP EXIF 元数据、按演员命名等能力。

### 9.3 批量管理与 PNG 导出

提交 `737141bd`

- 新增底部批量栏、按演员/影片分组勾选、PNG `iTXt` 导出、更多交互细节优化。

这条演进线也印证了前面的判断：  
它已经不是“截图功能”，而是在逐步演化成“可管理、可导出、可回播的帧素材库”。

## 10. 总结

当前萃取帧功能的优点不是“实现很炫”，而是它已经具备了明确的产品闭环和扩展方向。  
真正需要补的，不再是按钮和页面，而是以下三类基础能力：

- 可扩展的数据访问模型
- 更清晰的存储与导出语义
- 更稳定的编辑与资产化能力

如果只让我用一句话概括我的判断：

> 这块功能已经值得被当作“图库子系统”来设计，而不应该继续仅按“播放器附属截图”来维护。

## 11. 2026-09-06 复核：性能与截图 UI 优化

### 11.1 范围与结论

本次为当前源码静态审查，未运行真实视频、浏览器视觉验收或性能基准。以下区分代码可确认的问题与需要测量的优化方向，不声称已发生特定卡顿或已有速度提升。只更新建议文档，未修改功能。

当前已经实现：按下时暂存帧、短按保存 / 长按生成 GIF、可配置快捷键、提示音、成功 / 失败 HUD、aria-live、reduced-motion、逐帧控制、进度条帧标记、multipart 二进制上传、服务端分页与筛选、320×180 内缩略图、懒加载卡片、详情轮播、标签编辑、近邻时间去重检查、批量导出。应在这些基础上改进。

优先顺序：修正无谓原图读取与加载 → 保证截图画面、影片、时间和异步回执一致 → 增强轻量预览与失败恢复 → 按实测结果处理编码、大库和 GIF 调度。

### 11.2 性能：直接可确认的优化点

| 优先级 | 当前证据 | 改进建议与预期作用 |
| --- | --- | --- |
| P0 | `backend/internal/storage/playback_curated.go` 的 `GetCuratedFrameThumbnail` 同时 SELECT `thumb_blob, image_blob` 并 Scan 到 Go 内存 | 优先只取缩略图，缺失时再回退原图；也可用仅返回一个 BLOB 的 SQL 表达式。减少原图向应用层复制及内存分配。是否进一步拆表按数据库页读取实测决定。 |
| P0 | `CuratedFramesLibrary.vue` 详情 `v-for="entry in dialogNavigationEntries"` 为所有已加载条目挂载原图 `<img>`，没有原图窗口限制 | 只加载当前原图，受控预取前后各一张；其它轮播位置只保留布局占位 / 缩略图。不能仅依靠原生 lazy 属性控制轮播内请求。 |
| P1 | `QueryCuratedFrames` 先 COUNT、再取页、然后逐条 `loadCuratedFrameMotion`；前端每页 60 条 | motion 使用 LEFT JOIN 或单次批量查询；60 条完整页面当前可产生 62 次 SQL 查询，应降到固定 2–3 次。SQLite 无网络往返，但仍有语句执行与数据访问成本。 |
| P1 | 列表按 `captured_at DESC` + OFFSET；每页重新 COUNT | 先补 `(captured_at, id)` 稳定排序，避免相同时间条目排序不确定。大库再引入复合索引与游标分页，COUNT 按筛选条件缓存 / 独立请求；用 EXPLAIN QUERY PLAN 验证。 |
| P1 | `reloadFromDb` / `loadMoreRows` 没有请求代次校验；变更筛选与翻页可能并行完成 | 为每套查询生成版本，晚到的旧响应不得覆盖新筛选或追加到新结果；支持取消和可见的重试状态。 |
| P1 | 网格对已加载条目持续 v-for；深层监听 rawRows 后全量重建 URL，Mock 还会全量 revoke / create Blob URL | 优先按 id 缓存 URL，删除 / 替换时释放；大规模滚动再做虚拟网格与分组窗口化，保留选择、键盘焦点和滚动恢复。 |
| P2 | Mock 的 `listCuratedFramesPage` 每次 `getAll()`，含原始 Blob，排序过滤后切页 | 如 Mock 需要承载大量截图，升级 IndexedDB 索引 / 游标，将元数据和原图读取分开。不能把 Web 的服务端分页结论套到 Mock。 |

额外说明：原图 / 缩略图 HTTP 已有 `private, max-age=3600`，不是完全没有缓存。后续可加版本化 ETag；缓存命中不能替代首次加载时减少数据读取。

### 11.3 截图链路：准确性与连续操作

1. **统一时间来源。** `beginCuratedPress()` 捕获按下瞬间画面和绝对时间，`runCuratedCapture()` 却重新读松开时的时间作为成功提示。成功 HUD 应使用 `result.positionSec`。`captureCuratedFrameCandidate()` 的无 override 路径在 PNG 编码后读取 currentTime，也应改成绘制前冻结元数据。已有 requestVideoFrameCallback 可用于记录已呈现帧时间，但不要简单等待下一帧再截图，否则改变“按下即捕获”的语义；暂停时尤其要保留当前帧。
2. **冻结影片与播放会话。** 暂存 candidate 当前没有 movie 快照，等待 candidate 后再读取 `props.movie` 保存。应将 movieId / 必要元数据 / 会话代次与画面一起冻结，防止切片、切影片后把旧画面关联到新影片或更新新播放器的 HUD。
3. **有界保存队列。** 单帧保存没有统一并发上限，多次操作共享一个反馈状态和定时器，早发晚到的成功可能覆盖后一次错误。每次截图独立 id 与状态；开始可采用 1 个上传任务 + 小型等待队列，同时限制待处理总字节数。达到上限明确显示忙碌，不能静默丢帧或无限持有 canvas / Blob。
4. **失败可以恢复。** 候选在保存前已从 pending 清空；失败应有受限缓存和“重试 / 保存到本地”入口。重试沿用原 id，并先定义幂等规则：当前重复 id 返回 409，不能直接把所有 409 当成功；要验证相同内容或提供回执查询。
5. **区分入库与外部导出。** 当前额外目录写入失败被忽略，最终仍返回成功；目录写入还处于保存等待链路。回执应表达“已入库，导出失败”，目录写入独立排队和重试。浏览器发起下载只能报告“已发起下载”，不能保证用户磁盘已保存完成。

### 11.4 编码与清晰度：测量后决定

- `capture.ts` 每次建立源视频尺寸 canvas，`drawImage` 后 `toBlob('image/png', 0.92)`。PNG 的 quality 参数不会按 JPEG / WebP 方式降低体积，不能用调低 0.92 当作 PNG 性能优化。
- 3840×2160 的单份 RGBA 像素缓冲约 31.6 MiB，连续操作时还可能并存编码、Blob 和预览对象。这是像素大小估算，不是当前进程内存实测。
- 分段测量绘制 / PNG 编码 / 上传 / 后端缩图 / SQLite 插入。如果编码成为瓶颈，再评估冻结 ImageBitmap 后转给 Worker + OffscreenCanvas，做好资源 close / 释放、并发预算及不支持时回退。`toBlob` 回调异步不等于整条链路没有主线程成本，也不能据此断言编码一定阻塞主线程。
- 原图默认保留当前质量；缩略图可比较 JPEG / WebP 与现有 PNG 的体积、生成耗时和视觉效果。当前后端解码注册 PNG/JPEG，新增 WebP 上传必须同时补齐格式校验、解码与合约，不能只改前端 MIME。
- 上传时当前同步解码原图、CatmullRom 缩放并 PNG 编码；先测量，再决定是否后台生成缩略图。若异步化，应定义缩略图未就绪状态和占位，不要让所有请求退回下载原图。
- 浏览器截图保留的是当前播放流分辨率；HLS 若降低清晰度，PNG 也无法恢复源文件细节。可后续提供显式“从源文件提取高清帧”，显示当前截图尺寸与来源；源时间映射、VFR 和转码帧对应需单独验证，不承诺与屏幕逐像素一致。
- 后端当前限制 12 MiB 图片字节数；高细节 4K PNG 存在超限可能，需要真实样本测试。不要直接提高上限替代像素预算、格式策略和错误反馈。

### 11.5 UI：以确认画面和低干扰为主

**截图回执建议**：沿用视频优先布局，在不与播放列表、字幕和控制栏重叠的位置显示一条小浮层，包含约 96×54 缩略图、准确时间码和状态。截取后立即显示预览；只有持久化完成才显示“已保存”。动作可提供“查看”“撤销”，失败则保留“重试”。连续截图合并为“已保存 3 张 / 还有 1 张保存中”，不要堆叠 Toast。可交互预览需要明确点击隔离，不能沿用当前整个 HUD 的 `pointer-events-none`；点击不得误触视频暂停。

- 快门环目前 0.55 秒，可缩短到约 150–250 ms；慢操作超过约 150 ms 再显示保存中文字，减少瞬间状态闪烁。这些为设计起点，需实际体验调节。
- 提示音保留独立开关；明确其代表触发接收，保存结果由视觉回执确认。
- 当前短按 / 长按由 400 ms 区分。保留快捷方式，同时提供可发现的“截图 / GIF”菜单入口，适配不方便长按的用户；准备、录制、生成、失败、成功共用一个反馈区域。
- 目前 `armed` 阶段已有大块 GIF 提示，短按也会经过它。可让短按仅显示轻量接收反馈，到长按阈值才展开 GIF 录制控件。
- 精选帧可利用现有逐帧能力，加可发现的前后帧按钮和局部放大 / 100% 查看入口；高级附近帧选片作为后续能力，避免每次截图都弹出编辑器。
- 帧卡片现有近似重复徽标与 GIF 徽标都在右上角，应错位或合并；GIF 卡片分支当前统一使用 `<video>`，但后端产物为 `image/gif`，应与详情页一样按 MIME 选择 `<img>` / `<video>`，并提供加载失败静态图回退。
- 大图保持当前大预览 + 元数据侧栏，补原图加载状态、失败重试、缩放；已有左右切换、回播、标签和导出继续复用。窄屏改为预览优先、操作区可滚动。
- 使用已有主题令牌，保留键盘 / 焦点；触屏主要操作至少 44px；减少动效下不自动启动 GIF 预览。HUD role=status 与独立 aria-live 目前可能形成重复播报，应统一状态播报出口并合并连续操作通知。

### 11.6 GIF 与规模扩展

- `movie_clip_handlers.go` 每个请求直接启动 goroutine / FFmpeg，绑定运行时 context，但没有此任务专属 deadline 和可见的并发预算。增加有限工作队列、执行超时和取消语义，避免多个客户端与 HLS 播放竞争资源。初始并发值根据 CPU / 磁盘与播放体验测定。
- 前端每 500 ms 轮询；后续定时器发起的请求缺少就地错误恢复。增加退避、隐藏页面降频、任务代次校验与“状态暂不可用”反馈，避免卡在生成中。
- 当前录制 elapsed 使用媒体时间与墙钟时间的最大值，暂停 / 缓冲 / 倍速时可能与实际看过的范围不同；明确产品选择，优先保存真实媒体起止，暂停或 seek 时结束 / 取消并明确提示。
- GIF 更高画质可评估 palettegen / paletteuse；体积优先的动图预览可后续采用 MP4 / WebM，GIF 保留为导出。编码耗时和色彩改善需要样本比较，不保证调色板策略一定更快。
- 近似重复目前是按影片和相邻时间阈值分组，不是图像相似度识别。先明确“时间相近”的文案；真正视觉去重再评估感知哈希，提供并排比较，由用户选择保留，不能自动删掉相邻但不同的精彩帧。
- 原图 BLOB 是否迁移文件存储，应等万级数据、备份体积和写延迟基准后决定；迁移会影响备份、删除与孤儿清理，不适合作为本轮第一步。

### 11.7 建议交付切片与验收

1. **读取优化**：缩略图只读取必要 BLOB、详情原图窗口化、motion 批量查询。每项可独立验证与提交。
2. **截图一致性**：冻结画面 / 影片 / 时间、采用保存结果时间码、任务代次与有限队列、失败回执及幂等重试。
3. **轻量截图 UI**：缩略图回执、统一反馈区域、可见 GIF 入口、卡片 GIF MIME 修正及徽标布局。
4. **实测后的扩展**：编码 Worker、缩略图格式、虚拟列表 / 游标、GIF 调度、高清源帧。

建议基准使用合成或专门测试素材，不改真实资料库：1080p / 4K、直放 / HLS、正常播放 / 暂停 / seek / 切影片、连续 10 次操作、慢网络与保存失败；图库规模 100 / 1,000 / 10,000 条，并区分冷缓存 / 热缓存。

指标：触发到预览和持久化的 p50/p95、各阶段耗时、主线程 >50 ms 长任务、播放丢帧增量、前后端内存峰值、图片请求数 / 字节数、每页 SQL 次数。初始交互目标可设为触发回执 p95 <100 ms（仅回执，不含 PNG 完成），但必须先建立机器与素材基线。

结构性验收：打开详情只请求当前原图及受控相邻原图；缩略图成功路径不把原图 Scan 到 Go；60 条列表不逐条查 motion；晚到响应不污染新筛选 / 新影片；成功时间码与已保存记录一致；队列满、部分保存失败、外部导出失败均有明确状态。代码实施时按仓库统一构建 / 测试入口执行相关测试，实际 UI 还需暗色 / 浅色、窄屏、键盘、reduced-motion 验收。

## 12. 2026-09-06 实施记录与验证

用户在 §11 审查后要求执行全部优化。本节记录已落地内容与实测后保留的选择；§11 是实施前证据，不能作为当前实现描述。未修改真实资料库，未推送远程。

### 12.1 已交付

| 范围 | 实现与边界 |
| --- | --- |
| 缩略图 / SQL | 缩略图查询只向 Go 返回选中的一个 BLOB；历史缺失缩略图时回退原图。motion 按页批量查询，60 条列表不再产生 60 次独立 motion 查询。原图/缩略图支持 ETag、If-None-Match 与 304；ETag 在读取后计算，节省传输而非消除数据库读取。 |
| 分页 / 网格 | 0045 增加稳定排序索引，按 captured_at/id 倒序，游标续页支持 skipTotal；续页 total=-1 时沿用首屏统计。前端过滤请求有代次保护与失败重试，按 ID 去重。离屏卡片卸载、保留高度及键盘进入点，焦点所在行保留；行占位 DOM 仍随已加载行数增长，不是严格常量 DOM。 |
| 大图 / Mock | 详情仅当前及相邻各一张加载原图，其余保持缩略图。Blob URL 增量复用并释放。IndexedDB v2 事务迁移图片到独立 images store，分页只加载本页图片；筛选/统计仍扫描元数据，不宣称全流程 O(page)。 |
| 保存一致性 | 按下时冻结画面、影片快照、绝对媒体时间与重试身份。最多保留 4 个任务，像素估算接收预算 128 MiB，串行上传；这是接收预算，不是浏览器进程实际内存硬上限。相同 ID/影片/时间/原图字节重放成功，不覆盖后来编辑的标签，不同内容仍 409。 |
| 截图回执 | 即时小图、准确时间、独立任务状态；连续成功计数，旧失败优先可见，原图下载、重试、显式 JPEG 压缩、撤销入库。成功回执保留 12 秒，失败候选最多 3 分钟；单独打开的大图独立持有 URL，回执消失不破坏预览。目录导出失败单独反馈/重试；撤销入库不会删除已下载文件。 |
| 截图 UI | 可见截图、源帧、前后帧与片段格式操作；短按不展开录制面板，快门动效约 200ms，提示音仍有独立开关。回执和录制共用单一播报出口，录制计时不逐刻重复播报。移动端回执避开底部双行控制，工具栏避开标题；主要操作至少 44px。 |
| 原图检查 | FrameImageViewer 显示尺寸、加载/失败重试、100% 与适应窗口。默认启用独立预览控件，以 inline-size containment 防止原图撑宽弹窗。全屏 Dialog/DropdownMenu 支持 portalTo 定位到播放器 surface。 |
| 编码 / 上传 | 4K 优先冻结 ImageBitmap 后交 Worker PNG 编码，15 秒超时并回退已冻结 canvas；小图预览先返回。PNG 默认保留，移除无效 quality 参数。超过 12 MiB 时允许下载原始 PNG 或主动选择 JPEG 副本；后端先检查尺寸/格式再解码，最多 33,177,600 像素，无效数据不入库。 |
| 源文件帧 | POST /api/library/movies/{movieId}/frame，按绝对媒体秒数从配置库内主视频提取 PNG，20 秒期限、32 MiB 输出，与片段共享 2 个编码槽。只在 Web API 模式提供；不承诺 HLS、色彩转换、VFR 下与屏幕逐像素一致。 |
| 片段 | GIF 使用 palettegen/paletteuse；新增 MP4/WebM 格式。媒体时钟决定录制范围，暂停结束、seek 取消，旧手势不吞下一次点击。最多 8 个在途任务、2 个编码任务，排队/执行总期限 2 分钟、输出 128 MiB。支持取消 FFmpeg，静态帧保留；取消以失败终态报告，未新增 cancelled 枚举。轮询有错误退避、后台降频与代次保护。 |
| 卡片 / 相似检查 | GIF 按 MIME 用 img，视频格式用 video，失败回退静态图；徽标错位，减少动效时不自动启动动态预览。用户可发起同影片 dHash 相似检查，排除平坦图像，前 200 个已加载帧、并发 2、最多 50 对；可取消、并排比较和打开大图，不自动删除。 |
| 发布预算 | 英文/日文词典改为按需读取带内容哈希的 JSON 资源，避免编译成额外可执行 JS；保留中文首屏与按需切换。未提高任何 bundle hard budget，未提交依赖/锁文件变更。 |

后续调整（2026-09-24）：按用户要求移除萃取帧库的视觉相似检查入口、弹窗、dHash 计算模块及专用文案。上表记录 2026-09-06 的历史实施状态；当前仍保留同片时间相近提示和手动大图检查，不再生成视觉相似候选。

主要提交：`6f1b88fe`、`902f83f3`、`2eae285c`、`6e5fc8fc`、`85e18b7d`、`e5ab69e0`、`08bfa234`、`b6a3e889`、`a610ebf5`；收尾提交包含 HTTP 校验/缓存、真实源帧测试、手势状态、语言资源、大图和回执修复。提交按行为拆分，未混入工作区其它任务。

### 12.2 实测与选择

机器：Windows / Ryzen 7 9700X。视频使用 FFmpeg testsrc2，编码另外使用确定性高熵合成像素；不是实际影片或跨机器承诺。浏览器数据为小样本，p50 与单次最大值应分开理解。

| 样本 | 结果 |
| --- | --- |
| 高熵 1080p PNG，5 次 | 主路径 p50 89.2ms，最大 115.5ms；Worker p50 70.4ms，最大 88.1ms；同为 5,850,852 bytes。采样定时器最大间隔约 25.3→14.8ms。 |
| 高熵 4K PNG，5 次 | 主路径 p50 284.4ms，最大 296.6ms；Worker p50 194.3ms，最大 209.3ms；同为 20,821,624 bytes。采样定时器最大间隔约 29.3→18.7ms。该 PNG 超过上传上限，验证显式压缩/下载需求。 |
| 同一高熵 4K JPEG / WebP | JPEG 约 6,794,163 bytes / p50 86.2ms；WebP 约 6,825,780 bytes / p50 1078.7ms。小体积不代表编码更快。 |
| 低细节 testsrc2 放大到 4K | PNG p50 22.4ms、p95 31ms、164,893 bytes；WebP p50 213ms、p95 218.2ms、15,666 bytes。显示编码结果强烈依赖素材。 |
| Go 320×180 合成缩略图 | PNG 约 1.014ms / 1,849 bytes / 856,260 分配字节；JPEG85 约 0.706ms / 15,779 bytes / 33,264 分配字节。 |
| SQLite 热缓存、每页 60 条 | 100 / 1,000 / 10,000 条库：约 0.210 / 0.260 / 0.266ms，分配约 68KB。图片仅使用合成 4KiB 原图 / 1KiB 缩略图，不能代表真实大 BLOB 库、冷缓存或备份成本。 |
| 0.6 秒 320×180 合成片段 | GIF 69,132 bytes，MP4 13,209 bytes，WebM 11,827 bytes；三个产物均由真实 FFmpeg 生成。该比较不证明所有影片体积或画质关系。 |

据此保留三项选择：**默认 PNG、同步生成缩略图、SQLite 原图 BLOB**。缩略图全量改 JPEG/WebP、异步缩略图流水线和原图迁移文件存储没有无条件实施；前两项需真实高细节原图的端到端缩图/上传延迟，后一项需真实万级原图库的冷读、写入及备份测试。高级“附近多帧选片接触表”仍属 §11 标明的后续能力，本轮提供逐帧入口和 100% 检查。

未建立的指标包括：真实 HLS/VFR 对齐误差、用户视频连续十次截取的端到端 p95、播放丢帧增量、进程峰值内存及万级实际截图备份成本。不能用编码微基准代替这些验收。

### 12.3 验证记录

- 全量前端测试曾通过：218 文件 / 1,010 测试；后续新增手势与大图回归测试通过，最终相关测试组亦通过。
- `go test ./...`、`go vet ./...` 通过；最终 HTTP 缓存/重复上传/非法图片测试与真实合成源文件提帧测试单独重跑通过。
- `pnpm lint`、`pnpm test:electron`（4 文件 / 30 测试）通过；最后修改的文件再次检查。
- `pnpm build`（含类型检查）通过，未放宽预算。最后一次记录全部 JS 约 2.240MB raw / 0.740MB gzip，首屏约 365KB raw / 131KB gzip；英文/日文 JSON 是额外按需数据资源，不计作 JS，但仍计实际下载字节。
- Playwright CLI 在隔离的合成视频组件宿主页验证实际 CaptureReceipt / FrameImageViewer：1280px 浅色、375px 深色、320px 浅色，截图入库/撤销、无横向溢出，窄屏撤销按钮 44px。全屏预览位于 fullscreenElement 内，1920px 原图在 100% 模式实际宽度 1920px，可切回适应窗口；英文/日文资源在真实浏览器加载成功。
- 独立浏览器 context 构建旧 IndexedDB v1 后执行真实 v2 升级，验证图片迁移、标签编辑后图片仍在、kv 元数据保留及删除后无图片残留。
- `pnpm test:e2e`：3 通过、2 失败。失败分别为 375px 图库用例找不到 `[data-mobile-theme-toggle]`，以及设置备份选择器 `[data-settings-backup-pick]` 高约 32.39px、未达到用例要求的 44px。涉及其它正在修改的设置/移动端界面，未据此修改本任务之外的用户布局，也不报告全套 E2E 通过。
- 未运行需单独授权的 `test:display`，未声称 Safari、macOS 原生环境、全部播放器路由和真实资料库完成视觉验收。组件宿主页截图用于确认实际组件，不等同完整播放器截图。

可复查本机产物在忽略目录 `.workspace/curated-optimization/`：合成视频、浏览器基准脚本、迁移/回执/全屏查看验证脚本、桌面与移动端图片；不提交临时文件。API、操作指南、UI 规范、项目事实、架构说明和 README 入口已同步。

# 2026-04-11 萃取帧功能现状梳理与改进建议

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

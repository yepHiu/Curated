# Curated 影片信息 CSV 导出 PRD 与实施计划

日期：2026-06-21  
状态：Verified（2026-06-21）  
范围：Settings -> 影片存储、前端服务层、CSV 生成与下载

实现结果：已按前端服务层方案完成 CSV helper、`LibraryService.listMoviesForExport()`、设置页影片存储入口、三语文案与测试覆盖；本版未新增后端 CSV 端点。

## 1. 原始需求理解

用户希望在 Curated 中新增一个导出能力：把“目前库中的所有影片信息”导出为表格文件。导出入口放在设置页的“影片存储”区域；导出字段为番号、演员、标题、影片发布日期、影片片商；文件格式为 CSV。

基于当前代码和项目事实，本需求默认解释如下：

- “目前库中的所有影片”指当前资料库中的 active movies，不包含回收站 `mode=trash` 条目。
- 导出字段使用现有有效展示数据，即已合并用户覆盖值后的 `Movie` / `MovieListItemDTO` 字段：`code`、`actors`、`title`、`releaseDate`、`studio`。
- 导出是本地浏览器下载，不上传到外部服务，不写入后端磁盘。
- 第一版不提供字段自定义、筛选条件、导出历史、定时导出或 Excel `.xlsx` 格式。
- Mock 与 Web API 模式都应可使用该入口；Web API 模式应从服务层拉取完整 active movie 列表，避免只导出 UI 已渲染范围。

## 2. 背景与现状

当前 Curated 前端是 Vue 3 + TypeScript + Vite，业务数据通过 `LibraryService` 合约隔离 Web API 与 Mock adapter。设置页的“影片存储”已拆为 `SettingsLibraryPathsSection.vue`，父级 `SettingsPage.vue` 负责状态和事件编排。影片列表 DTO 已包含导出需要的全部字段：

- `src/api/types.ts` 的 `MovieListItemDTO`：`title`、`code`、`studio`、`actors`、`releaseDate`。
- `src/domain/movie/types.ts` 的 `Movie`：同样包含上述字段。
- `src/services/adapters/web/web-library-service.ts` 已有 `fetchPagedMovies()`，会以 `LIST_BATCH_SIZE = 500` 分页拉取完整 active library。

因此第一版不需要新增 Go 后端端点。更合适的实现是：在前端服务层提供“导出用完整影片列表”方法，再由前端生成 UTF-8 CSV Blob 并触发下载。

## 3. 产品目标

1. 用户可以在 `Settings -> 影片存储` 中一键导出当前 active library 的影片基础信息。
2. CSV 文件可以被 Excel、WPS、Numbers、Google Sheets 等表格工具打开。
3. 导出字段固定、顺序稳定、缺失值为空，方便二次整理。
4. 导出行为不改变资料库数据，不依赖后端文件系统写权限。
5. 大于 500 条的资料库仍能导出完整列表，而不是只导出第一页或当前页面可见项。

## 4. 非目标

- 不导出封面、缩略图、预览图、视频路径、播放进度、收藏、评分、标签、简介、运行时长、分辨率、导入时间。
- 不导出回收站影片。
- 不提供字段勾选、列顺序配置、按筛选条件导出、分页选择导出。
- 不新增后端 CSV 流式端点。
- 不新增后台任务、导出历史、导出目录设置。
- 不支持 `.xlsx`、`.json`、`.nfo` 或压缩包。

## 5. 用户故事与验收标准

### US1：从设置页导出影片基础信息

作为 Curated 用户，我希望在设置页的“影片存储”中点击一个导出按钮，将当前库中的影片基础信息保存为 CSV，方便在表格软件中查看和整理。

验收标准：

- “影片存储”卡片底部动作区显示导出入口，文案建议为“导出影片 CSV”。
- 点击后下载一个 `.csv` 文件。
- 文件包含表头：`番号,演员,标题,影片发布日期,影片片商`。
- 每一行对应一部 active movie。
- 字段映射为：番号=`movie.code`，演员=`movie.actors`，标题=`movie.title`，影片发布日期=`movie.releaseDate`，影片片商=`movie.studio`。
- 多演员使用 `; ` 连接在同一个 CSV 单元格内。
- 缺失的发布日期、演员或片商输出为空单元格。

### US2：导出完整资料库而非当前可见列表

作为拥有大量影片的用户，我希望导出的 CSV 包含所有 active movies，即使当前瀑布流或设置页没有展示这些影片。

验收标准：

- Web API 模式通过服务层分页获取所有 active movies。
- 当影片数量超过 500 时，导出仍包含所有页的数据。
- 导出不受当前资料库页的搜索、演员筛选、厂商筛选或滚动位置影响。

### US3：导出失败时有明确反馈

作为用户，如果导出时资料库加载失败，我希望看到失败提示，而不是没有任何反应。

验收标准：

- 导出过程中按钮进入 busy/disabled 状态。
- 成功后可显示 toast 或轻量成功文案，例如“已导出 N 部影片”。
- 失败时在“影片存储”卡片附近显示错误文案，并可通过再次点击重试。
- 控制台可以记录技术错误，但用户可见反馈不能只依赖控制台。

### US4：CSV 对表格软件友好且安全

作为用户，我希望 CSV 在常见表格软件中不乱码，并避免标题或演员名被当作公式执行。

验收标准：

- CSV 使用 UTF-8 with BOM。
- 行结束符使用 CRLF。
- 包含逗号、换行或双引号的字段按 CSV 规则加引号并转义双引号。
- 以 `=`、`+`、`-`、`@` 开头的文本字段做 spreadsheet formula 防护，建议前缀单引号。

## 6. CSV 规格

文件名建议：

```text
curated-movies-YYYYMMDD-HHmmss.csv
```

MIME：

```text
text/csv;charset=utf-8
```

表头固定为：

```csv
番号,演员,标题,影片发布日期,影片片商
```

示例：

```csv
番号,演员,标题,影片发布日期,影片片商
ABC-123,"Alice; Bella","Example, quoted title",2026-01-02,Studio One
XYZ-777,,Untitled,,Studio Two
```

排序：

- 第一版沿用后端 active movie listing 的默认顺序，即 `added_at DESC, id ASC`。
- 不在 CSV 层重新排序，避免导出结果与用户当前库默认顺序不一致。

## 7. 方案比较

### 方案 A：前端服务层拉全量数据并生成 CSV（推荐）

做法：

- 在 `LibraryService` 增加 `listMoviesForExport()` 或 `loadMoviesForExport()`。
- Web adapter 复用现有 `fetchPagedMovies()` 拉取完整 active movie 列表。
- Mock adapter 返回当前 mock active movies。
- 新增 `src/lib/movie-csv-export.ts` 负责 CSV 序列化、BOM、文件名。
- 设置页调用服务方法，生成 Blob 并复用现有 `triggerDownloadBlob()` 下载。

优点：

- 不新增后端端点，改动面小。
- 与现有前端服务边界一致，Mock/Web 都能走同一 UI。
- 现有 DTO 已包含所有字段，不需要额外查询。
- 便于单元测试 CSV 转义和 UI 事件。

缺点：

- 超大库会把完整列表加载到浏览器内存；但当前 Web adapter 本来已支持拉完整 active library，第一版风险可接受。

### 方案 B：新增后端 `GET /api/library/movies/export.csv`

做法：

- Go 后端查询 SQLite 并直接写 CSV 响应。
- 前端只触发 Blob 下载。

优点：

- 更适合非常大的资料库和未来流式导出。
- 可避免前端持有完整数据。

缺点：

- 新增 API、合同、handler、文档同步和后端测试，当前需求的字段不需要这层复杂度。
- Mock 模式仍需前端 CSV 实现或模拟 Blob，最终会有两套导出逻辑。

### 方案 C：直接导出当前 `libraryService.movies.value`

做法：

- 设置页直接读取已有 computed state，生成 CSV。

优点：

- 改动最小。

缺点：

- 如果首轮加载失败、尚未完成或缓存不新，导出结果不可靠。
- 不够明确地表达“导出完整库快照”的业务边界。

结论：第一版采用方案 A。

## 8. 交互设计

入口位置：

- 放在 `SettingsLibraryPathsSection.vue` 的“影片存储”卡片底部动作区，和“添加路径”“重新检测”并列。
- 图标使用 `lucide-vue-next` 的 `Download`。
- 按钮尺寸沿用现有存储动作按钮：`h-8 min-w-28 rounded-2xl px-3`。

状态：

- 默认：`导出影片 CSV`
- 导出中：`正在导出`
- 成功：toast 或卡片内轻量成功文案 `已导出 {count} 部影片`
- 失败：卡片内 destructive 文案 `导出失败，请稍后重试`

边界：

- 空库仍允许导出，生成只有表头的 CSV，并提示 `已导出 0 部影片`。
- 导出按钮 busy 时禁用，避免重复下载。
- 不因某个字段缺失阻断整份导出。

## 9. 技术设计

### 9.1 新增 CSV 工具

新增文件：

- `src/lib/movie-csv-export.ts`
- `src/lib/movie-csv-export.test.ts`

职责：

- 将 `readonly Movie[]` 转为 CSV 字符串。
- 固定输出 5 列。
- 处理 CSV escaping、CRLF、UTF-8 BOM、spreadsheet formula 防护。
- 生成稳定下载文件名。
- 创建 Blob。

建议导出函数：

```ts
export function buildMovieCsv(movies: readonly Movie[]): string
export function buildMovieCsvBlob(movies: readonly Movie[]): Blob
export function buildMovieCsvFilename(now?: Date): string
```

### 9.2 扩展服务契约

修改：

- `src/services/contracts/library-service.ts`
- `src/services/adapters/web/web-library-service.ts`
- `src/services/adapters/mock/mock-library-service.ts`
- 对应 adapter 测试

建议新增：

```ts
listMoviesForExport(): Promise<readonly Movie[]>
```

Web adapter 行为：

- 调用现有 `fetchPagedMovies(undefined)`，即 active movies。
- 成功后可同步刷新 `moviesState.value` 和 `moviesLoadedState.value`，让导出后的 UI 缓存也保持新鲜。
- 失败时抛出错误给设置页处理。

Mock adapter 行为：

- 返回当前 mock active movies，过滤 `trashedAt`。

### 9.3 设置页编排

修改：

- `src/components/jav-library/SettingsPage.vue`
- `src/components/jav-library/settings/SettingsLibraryPathsSection.vue`
- `src/components/jav-library/settings/SettingsLibraryPathsSection.test.ts`

父级新增状态：

```ts
const movieCsvExportBusy = ref(false)
const movieCsvExportError = ref("")
```

父级新增动作：

```ts
async function exportMovieLibraryCsv() {
  movieCsvExportBusy.value = true
  movieCsvExportError.value = ""
  try {
    const movies = await libraryService.listMoviesForExport()
    const blob = buildMovieCsvBlob(movies)
    triggerDownloadBlob(blob, buildMovieCsvFilename())
    pushAppToast(t("settings.movieCsvExportSuccess", { count: movies.length }))
  } catch (err) {
    console.error("[settings] export movie csv failed", err)
    movieCsvExportError.value = t("settings.movieCsvExportFailed")
  } finally {
    movieCsvExportBusy.value = false
  }
}
```

子组件新增 props/emits：

- `movieCsvExportBusy: boolean`
- `movieCsvExportError: string`
- `exportMoviesCsv: []`

### 9.4 i18n

当前仓库使用 `src/i18n/index.ts` 管理多语言文案。新增 keys：

- `settings.movieCsvExport`
- `settings.movieCsvExporting`
- `settings.movieCsvExportSuccess`
- `settings.movieCsvExportFailed`
- `settings.movieCsvExportDesc`（如需要说明文案）

至少补齐 `zh-CN`、`en`、`ja`。

## 10. 实施计划

### Task 1：实现 CSV 序列化工具

文件：

- Create: `src/lib/movie-csv-export.ts`
- Create: `src/lib/movie-csv-export.test.ts`

步骤：

- [ ] 写失败测试：表头固定、字段顺序正确、多演员用 `; ` 连接。
- [ ] 写失败测试：逗号、双引号、换行正确转义。
- [ ] 写失败测试：空字段输出为空单元格。
- [ ] 写失败测试：UTF-8 BOM 和 CRLF 存在。
- [ ] 写失败测试：以公式触发字符开头的文本被前缀单引号。
- [ ] 实现 `buildMovieCsv()`、`buildMovieCsvBlob()`、`buildMovieCsvFilename()`。
- [ ] 运行：

```powershell
pnpm test -- src/lib/movie-csv-export.test.ts
```

### Task 2：扩展 LibraryService 导出数据方法

文件：

- Modify: `src/services/contracts/library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.ts`
- Modify: `src/services/adapters/mock/mock-library-service.ts`
- Modify: `src/services/adapters/web/web-library-service.test.ts`
- Modify: `src/services/adapters/mock/mock-library-service.test.ts`

步骤：

- [ ] 在合约中新增 `listMoviesForExport(): Promise<readonly Movie[]>`。
- [ ] Web adapter 测试：超过 500 条时会分页调用 `api.listMovies({ limit: 500, offset: ... })` 并返回所有 active movies。
- [ ] Web adapter 测试：导出不传 `mode: "trash"`。
- [ ] Mock adapter 测试：返回 active mock movies，不包含 `trashedAt` 条目。
- [ ] 实现 Web/Mock adapter。
- [ ] 运行：

```powershell
pnpm test -- src/services/adapters/web/web-library-service.test.ts src/services/adapters/mock/mock-library-service.test.ts
```

### Task 3：接入设置页“影片存储”入口

文件：

- Modify: `src/components/jav-library/SettingsPage.vue`
- Modify: `src/components/jav-library/settings/SettingsLibraryPathsSection.vue`
- Modify: `src/components/jav-library/settings/SettingsLibraryPathsSection.test.ts`

步骤：

- [ ] 在 `SettingsLibraryPathsSection.test.ts` 中增加按钮渲染、busy 禁用、点击 emit、错误文案渲染测试。
- [ ] 在 `SettingsLibraryPathsSection.vue` 增加 `Download` 图标按钮和错误文案展示。
- [ ] 在 `SettingsPage.vue` 增加导出 busy/error 状态。
- [ ] 在 `SettingsPage.vue` 实现 `exportMovieLibraryCsv()`，调用 service、CSV helper、`triggerDownloadBlob()` 和 toast。
- [ ] 将 props 和 `@export-movies-csv` 事件接到 `SettingsLibraryPathsSection`。
- [ ] 运行：

```powershell
pnpm test -- src/components/jav-library/settings/SettingsLibraryPathsSection.test.ts
```

### Task 4：补齐多语言文案与边界测试

文件：

- Modify: `src/i18n/index.ts`
- Modify: `src/i18n/locales.test.ts`（如当前测试要求 key 完整性）

步骤：

- [ ] 为 `zh-CN`、`en`、`ja` 添加导出按钮、导出中、成功、失败文案。
- [ ] 如果 locale 测试有 key 完整性断言，补齐测试。
- [ ] 运行：

```powershell
pnpm test -- src/i18n/locales.test.ts
```

### Task 5：回归与文档同步

文件：

- Modify: `docs/prd/requirements.csv`
- Keep: `docs/plan/2026-06-21-library-movie-csv-export-prd-and-plan.md`
- 可选修改：`README.md`、`README.zh-CN.md`、`README.ja-JP.md`

步骤：

- [ ] 如果实现后认为该功能属于用户可见功能摘要，更新 README 的设置/资料库能力摘要。
- [ ] 因为第一版不新增后端端点，不需要更新 `CLAUDE.md` API 列表、`API.md` 或 `project-facts.mdc` 的 HTTP API 列表；如后续改为后端导出端点，再同步这些文档。
- [ ] 运行 PRD lint：

```powershell
python scripts/prd/prd_lint.py docs/prd/requirements.csv
```

### Task 6：最终验证

运行：

```powershell
pnpm test -- src/lib/movie-csv-export.test.ts src/services/adapters/web/web-library-service.test.ts src/services/adapters/mock/mock-library-service.test.ts src/components/jav-library/settings/SettingsLibraryPathsSection.test.ts src/i18n/locales.test.ts
pnpm typecheck
pnpm lint
```

预期：

- 所有测试通过。
- TypeScript 无类型错误。
- ESLint 无新增问题。
- 手工验证 Web API 模式下 `Settings -> 影片存储 -> 导出影片 CSV` 能下载 CSV，且 CSV 包含当前 active library 全量影片。

## 11. 风险与取舍

1. 大库导出会占用浏览器内存。  
   第一版复用已有全量列表拉取策略；如后续用户资料库规模明显变大，再升级为后端流式 CSV 端点。

2. Excel 兼容性。  
   通过 UTF-8 BOM、CRLF 和标准 CSV escaping 降低乱码和解析问题。

3. CSV 注入。  
   对可能被表格软件识别为公式的文本单元格加单引号前缀。

4. 数据新鲜度。  
   不直接导出当前可见列表，而是通过 service 方法主动拉取完整 active movies，降低缓存过期风险。

5. “所有影片”范围歧义。  
   第一版明确为 active library，不包含回收站。后续如需要可新增“包含回收站”选项，但不放入本次 MVP。

## 12. 后续可演进项

- 增加后端 `GET /api/library/movies/export.csv` 流式端点，适配超大库。
- 增加导出字段选择，如标签、评分、收藏、路径、运行时长、分辨率、导入时间。
- 增加按当前筛选条件导出。
- 增加 `.xlsx` 导出。
- 增加导出结果中的应用版本、导出时间、资料库统计等元信息。

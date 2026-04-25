# Curated 生产打包、配置、版本与发布计划

## 1. 文档定位

本文是 **Curated** 下一阶段生产发布工作的规划文档，重点回答以下问题：

- 如何从当前开发态的 Vue + Go 项目过渡到可分发的生产版本
- 在“单二进制分发”目标下，配置、数据库、缓存、日志应该如何处理
- 如何满足程序内必须显示版本号的既有约定
- 如何同时提供安装器版本和绿色版 zip 版本
- 如何规划打包、产物整理和发布脚本

本文描述的是 **推荐方案、约束和实施计划**，不是当前仓库已经全部完成的事实实现。

## 1.1 当前生产安装包链路梳理（2026-04-11）

本节按当前仓库脚本与代码事实梳理生产环境安装包链路；与后续规划不一致时，以本节为当前实现快照。

### 入口命令

当前 `package.json` 暴露了如下 release 脚本：

- `pnpm release:frontend`
- `pnpm release:backend`
- `pnpm release:portable`
- `pnpm release:installer`
- `pnpm release:publish`

### 1.1.1 当前生效的生产包版本规则（2026-04-12）

- 生产包版本的唯一自动化来源是 `scripts/release/version.json`，当前基线为 `1.1.0`。
- `pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish` 在未显式传入 `-Version` 时，都会自动把 `patch` 加 1。
- `pnpm release:publish` 只在入口处分配一次新版本，再把同一个版本号传给便携包、安装包与 `release/manifest/release.json`，避免一轮整机发布消耗多个 patch。
- `major` / `minor` 只允许人工调整，命令为 `pnpm release:version:set-base -- --Major <major> --Minor <minor>`；调整时必须把 `patch` 重置为 `0`。
- `package.json` 的 `version` 不再作为生产包版本来源。
- 同一次发布中，安装包文件名、便携包文件名、`release/manifest/release.json` 与 `docs/ops/2026-04-02-package-build-history.md` 必须保持一致。

当前推荐入口命令：

```powershell
pnpm release:publish
```

如需固定构建戳，可显式执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/publish.ps1 -BuildStamp <yyyyMMdd.HHmmss>
```

以下段落保留为 2026-04-12 之前的链路说明；当前生效规则以上一节 `1.1.1 当前生效的生产包版本规则` 为准。旧流程里，正式整机安装包需要显式传入 `-Version`：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/publish.ps1 -Version <version>
```

如果需要固定构建戳，可额外传入：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/publish.ps1 -Version <version> -BuildStamp <yyyyMMdd.HHmmss>
```

同一次发布中，`-Version`、安装包文件名、绿色包文件名、`release/manifest/release.json` 与 `docs/ops/2026-04-02-package-build-history.md` 台账记录必须保持一致。

### 主链路

当前完整发布入口是 `scripts/release/publish.ps1`，执行顺序如下：

```text
publish.ps1
  -> build-frontend.ps1
  -> build-backend.ps1
  -> assemble-release.ps1
  -> package-portable.ps1
  -> package-installer.ps1
  -> write release/manifest/release.json
```

1. `build-frontend.ps1`
   - 工作目录切到仓库根目录。
   - 设置 `VITE_APP_VERSION=$Version`。
   - 执行 `pnpm typecheck`。
   - 执行 `pnpm exec vite build --configLoader native`。
   - 将根目录 `dist/` 复制到 `release/frontend/`（或传入的 `-OutputDir`）。
   - 注意：当前 `src/` 内未检索到 `VITE_APP_VERSION` 消费点，前端产物虽注入该环境变量，但 UI 版本展示主要仍来自后端 `GET /api/health` 的 `version/channel`。

2. `build-backend.ps1`
   - 工作目录切到 `backend/`。
   - 默认输出 `release/backend/curated.exe`。
   - 创建并使用仓库内 `.gocache/` 作为 `GOCACHE`。这是发布脚本的当前实现，与日常测试“不要把 Go 缓存指到仓库内”的默认约定不同。
   - 执行：

```powershell
go build -tags release -ldflags "-H=windowsgui -X curated-backend/internal/version.BuildStamp=<BuildStamp>" -o <binaryPath> ./cmd/curated
```

3. `assemble-release.ps1`
   - 默认输入 `release/backend/curated.exe` 与 `release/frontend/`。
   - 默认输出 `release/Curated/`。
   - 品牌图标约定：安装包、桌面快捷方式、托盘运行时统一使用 `backend/internal/assets/curated.ico`；该 `.ico` 当前来自 `icon/curated-icon-rg-dark-pink.png` 的多尺寸派生版本。README 顶部带字标志使用 `icon/curated-title-nobg.png`，不参与 Windows 可执行图标链路。
   - 目录内容包括：
      - `curated.exe`
      - `curated.ico`
      - `frontend-dist/`
     - `third_party/`（如果 `backend/third_party/` 存在）
     - `runtime/config/`
     - `runtime/data/`
     - `runtime/cache/`
     - `runtime/logs/`
     - `runtime/config/library-config.example.cfg`
     - `README-release.txt`
     - `docs/production-packaging-and-config-strategy.md`
   - 2026-04-11 修正：脚本现在复制当前实际存在的 `docs/plan/2026-03-31-production-packaging-and-config-strategy.md`，并在发布目录中仍输出为 `docs/production-packaging-and-config-strategy.md`。

4. `package-portable.ps1`
   - 默认输入 `release/Curated/`。
   - 默认输出 `release/portable/Curated-<version>-windows-x64.zip`。
   - 使用 `Compress-Archive` 打包 `release/Curated/*`。

5. `package-installer.ps1`
   - 默认输入 `release/Curated/`。
   - 读取模板 `scripts/release/windows/Curated.iss.tpl`。
   - 生成 `release/installer/Curated.iss`，替换 `__APP_VERSION__`、`__APP_DIR__`、`__OUTPUT_DIR__`、`__SETUP_BASENAME__`。
   - 查找 `ISCC.exe`；若存在则调用 Inno Setup 生成 `release/installer/Curated-Setup-<version>.exe`。
   - 若找不到 `ISCC.exe`，脚本只生成 `.iss` 并 warning 后返回，不会生成安装器 exe。

6. `release/manifest/release.json`
   - `publish.ps1` 在最后创建或更新 manifest。
   - 当前字段包括：
      - `productName`
      - `version`
     - `buildStamp`
     - `channel`
     - `generatedAtUtc`
     - `artifacts[]`
   - `artifacts[]` 只在对应文件存在时追加：
      - portable zip：记录文件名、绝对路径、SHA256
      - installer exe：记录文件名、绝对路径、SHA256
   - 若更新品牌图标，先更新 `icon/curated-icon-rg-dark-pink.png`，再同步生成/替换 `public/Curated-icon.png`、`backend/frontend-dist/Curated-icon.png` 与 `backend/internal/assets/curated.ico`，避免 Web 图标、托盘图标、安装包快捷方式图标不一致。

### 运行态链路

当前 release 二进制的运行态关键点：

- `-tags release` 使后端默认 HTTP 地址从 `:8080` 切到 `:8081`。
- `-tags release` 使 `version.Channel` 为 `release`，健康名为 `curated`。
- Windows + release build 默认启动模式为 `tray`：启动本地 HTTP 服务、托盘图标、单实例互斥，并打开浏览器。
- 后端 `webui.FindDistDir()` 会优先从可执行文件旁查找 `frontend-dist/` 或 `dist/`，因此安装目录中的 `curated.exe + frontend-dist/` 可以直接服务前端页面。
- 前端生产构建未设置 `VITE_API_BASE_URL` 时，`src/api/http-client.ts` 默认请求 `http://127.0.0.1:8081/api`，与 release 后端默认端口一致。
- release build 的默认数据根目录为 `%LOCALAPPDATA%\Curated`，可通过 `CURATED_DATA_DIR` 覆盖；默认派生：
  - `config/library-config.cfg`
  - `data/curated.db`
  - `cache/`
  - 日志目录在托盘菜单中按配置解析，未显式配置时基于数据根目录的 `logs/`。

### 当前已观察到的台账风险

- `release/manifest/release.json` 当前记录的是 `0.0.1-master`，但 `docs/ops/2026-04-02-package-build-history.md` 当前最后一条是 `0.0.0-local`。这表示已有产物与版本台账存在不同步风险。
- 后续任何实际产出安装包、绿色包或发布清单的动作完成后，都需要立即追加 `docs/ops/2026-04-02-package-build-history.md`，不要发布后再补写不一致版本。

### 正式打包操作清单

执行整机安装包或完整发布前：

1. 读取 `docs/ops/2026-04-02-package-build-history.md`，确认最近一条有效发布记录。
2. 确认本次 `-Version`，不要直接使用 `package.json` 中 `release:*` 脚本默认的 `0.0.0-local`。
3. 确认当前 commit / branch，并记录将用于台账的 short SHA。
4. 确认 Inno Setup 是否可用：`ISCC.exe` 需存在于 PATH，或位于 `C:\Program Files (x86)\Inno Setup 6\ISCC.exe` / `C:\Program Files\Inno Setup 6\ISCC.exe`。
5. 确认 `config/library-config.cfg` 中示例配置可作为 `runtime/config/library-config.example.cfg` 随包分发；不要把本机私密代理、私有路径或临时调试配置带入示例配置。
6. 如需随包提供 FFmpeg，确认 `backend/third_party/ffmpeg/` 内存在实际运行时文件；当前仓库只有 README 时，包内也只会带 README。

推荐执行命令：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/publish.ps1 -Version <version>
```

执行完成后：

1. 检查 `release/portable/Curated-<version>-windows-x64.zip` 是否存在。
2. 检查 `release/installer/Curated-Setup-<version>.exe` 是否存在；如果只生成 `release/installer/Curated.iss`，说明本机未找到 Inno Setup 编译器。
3. 检查 `release/manifest/release.json` 中的 `version`、`artifacts[].fileName`、`sha256` 是否与实际产物一致。
4. 解压或检查 `release/Curated/`，确认至少包含：
   - `curated.exe`
   - `curated.ico`
   - `frontend-dist/index.html`
   - `runtime/config/library-config.example.cfg`
   - `README-release.txt`
5. 追加 `docs/ops/2026-04-02-package-build-history.md`，记录日期、版本、提交 / 分支、打包类型、产物路径、状态、操作人与备注。
6. 如发布态需要手动验收，运行 `release/Curated/curated.exe`，确认托盘模式启动、浏览器打开、`GET http://127.0.0.1:8081/api/health` 返回 `name=curated`、`channel=release`，并确认前端页面可加载。

## 2. 当前仓库现状

### 2.1 已有能力

当前仓库已经具备以下基础能力：

- 前端：`Vue 3 + TypeScript + Vite`
- 后端：`Go + SQLite`
- 后端已有完整 HTTP API
- 后端已有 SQLite migration 和数据持久化能力
- 后端已有日志系统，支持控制台输出和按天滚动文件日志
- 后端已有 `library-config.cfg` 的读取、合并和原子写回
- 后端已经区分了开发态与发布态的数据目录逻辑

### 2.2 已有的发布态数据目录机制

当前发布态路径逻辑已经在以下代码中存在：

- [datapaths_release.go](C:/Users/wujiahui/code/jav-lib/jav-shadcn/backend/internal/config/datapaths_release.go)

当前行为：

- `release` build tag 下，默认数据根目录为 Windows 的 `%LOCALAPPDATA%\Curated`
- 可通过环境变量 `CURATED_DATA_DIR` 覆盖默认数据根目录

当前发布态已经隐含使用如下布局：

- `config/library-config.cfg`
- `data/curated.db`
- `cache/`

这说明仓库已经具备“程序本体与可写数据分离”的基本方向。

### 2.3 已有的版本机制

当前版本机制已经在后端存在：

- [version.go](C:/Users/wujiahui/code/jav-lib/jav-shadcn/backend/internal/version/version.go)
- [channel_dev.go](C:/Users/wujiahui/code/jav-lib/jav-shadcn/backend/internal/version/channel_dev.go)
- [channel_release.go](C:/Users/wujiahui/code/jav-lib/jav-shadcn/backend/internal/version/channel_release.go)

当前能力：

- 支持 `BuildStamp`
- 支持区分 `dev` / `release` channel
- 健康检查接口 `GET /api/health` 已返回：
  - `version`
  - `channel`

也就是说，生产发布方案必须兼容并强化这一套机制，而不是另起一套版本规则。

## 3. 生产发布的核心原则

### 3.1 程序文件与可写数据分离

生产版必须将以下内容分开：

- 程序本体
  - `curated.exe`
  - 内嵌前端静态资源
- 可写数据
  - 配置文件
  - SQLite 数据库
  - 缓存目录
  - 日志目录

推荐目标不是“所有内容都封进一个可执行文件内部再原地修改”，而是：

**单二进制分发 + 外置数据目录**

### 3.2 升级不覆盖用户数据

生产版升级应满足：

- 替换程序文件即可升级
- 不覆盖用户配置
- 不清空数据库
- 不删除缓存和日志

### 3.3 首次启动自动初始化

生产版应尽量做到：

- 首次启动自动创建数据目录
- 首次启动自动创建默认配置文件
- 数据库不存在时自动初始化
- 即使还没有配置任何资料库路径，也能先正常启动

### 3.4 版本信息必须稳定可见

由于项目已有“程序里必须显示版本”的约定，生产方案必须满足：

- 程序 UI 中必须能显示版本信息
- 后端 API 仍必须返回版本信息
- release 产物中的版本号必须是可追踪、可复现、可区分渠道的
- 安装器版本、绿色版 zip 版本、程序运行时显示版本必须保持一致

## 4. 版本策略

### 4.1 统一版本来源

建议采用“双层版本”：

- **产品版本**
  - 例如：`0.1.0`
  - 来自发布输入参数或 CI
- **构建戳**
  - 例如：`20260331.153000`
  - 注入后端 `BuildStamp`

推荐 release 展示形式：

- UI 展示：`0.1.0 (20260331.153000, release)`
- API 字段继续保留：
  - `version = 20260331.153000`
  - `channel = release`

如果后续需要，也可以增加前端展示用的产品版本字段，但不应破坏现有后端 `version + channel` 语义。

### 4.2 当前约定下的兼容要求

由于当前 `GET /api/health` 已返回 `version` 与 `channel`，建议保持以下兼容规则：

- `version` 继续表示构建标识
- `channel` 继续表示构建通道
- UI 在显示版本时，应至少展示：
  - 版本号
  - release / dev 通道

### 4.3 release 构建要求

正式产物必须满足：

- 使用 `-tags release`
- 使用 `-ldflags` 注入 `BuildStamp`
- 发布脚本统一接收一个产品版本号
- 所有分发产物命名都包含该版本号

推荐命名：

- 安装器：`Curated-Setup-0.1.0.exe`
- 绿色包：`Curated-0.1.0-windows-x64.zip`
- 运行时 channel：`release`

## 5. 推荐的数据目录结构

### 5.1 Windows 默认目录

推荐默认用户数据根目录：

`%LOCALAPPDATA%\Curated`

推荐布局：

```text
%LOCALAPPDATA%\Curated\
  config\
    app.json
    library-config.cfg
  data\
    curated.db
  cache\
  logs\
```

推荐安装目录：

```text
C:\Program Files\Curated\curated.exe
```

说明：

- 安装目录只放程序本体
- 用户数据目录放所有可写内容

### 5.2 自定义数据目录

继续支持：

- `CURATED_DATA_DIR`

当该环境变量存在时，以下目录都从该路径派生：

- `config/`
- `data/`
- `cache/`
- `logs/`

## 6. 配置文件策略

### 6.1 配置分层

建议将配置拆成两层：

#### A. 启动级配置

文件建议：

- `config/app.json`

职责：

- 决定程序如何启动
- 决定运行时环境与全局策略

建议字段：

- `httpAddr`
- `enableFileLog`
- `defaultLogDir`
- 未来可能增加的服务模式或托盘相关开关

特点：

- 更偏程序运行参数
- 修改后通常需要重启

#### B. 业务设置

文件建议继续使用：

- `config/library-config.cfg`

职责：

- 存储设置页驱动的业务配置

当前适合保留在这里的字段：

- `organizeLibrary`
- `autoLibraryWatch`
- `metadataMovieProvider`
- `metadataMovieProviderChain`
- `metadataMovieScrapeMode`
- `proxy`
- `logDir`
- `logFilePrefix`
- `logMaxAgeDays`
- `logLevel`

### 6.2 配置优先级

建议采用如下优先级：

1. 命令行参数
2. 环境变量
3. `config/app.json`
4. `config/library-config.cfg`
5. 代码默认值

### 6.3 首次启动行为

首次启动建议自动完成：

- 创建 `config/`
- 创建 `data/`
- 创建 `cache/`
- 创建 `logs/`
- 若 `library-config.cfg` 不存在，则创建默认模板
- 若数据库不存在，则自动初始化

## 7. 日志策略

### 7.1 日志必须外置

日志不应写到程序安装目录，推荐始终写入：

- `%LOCALAPPDATA%\Curated\logs`

原因：

- 安装目录通常不适合写入
- 日志需要轮转和清理
- 升级时不应影响历史日志
- 用户和开发者都需要可定位、可打包的日志

### 7.2 当前已具备的日志能力

当前后端日志系统已支持：

- 控制台输出
- 文件输出
- 按天滚动
- 保留天数限制

这些能力足以作为第一版生产日志方案。

### 7.3 推荐默认策略

建议生产环境默认：

- `logLevel = info`
- `logFilePrefix = curated`
- `logMaxAgeDays = 7` 或 `14`
- 同时保留控制台输出和文件输出

### 7.4 日志安全

日志应避免记录：

- 代理密码
- 明文认证口令
- 无必要的敏感隐私路径信息

## 8. 单二进制发布形态

### 8.1 推荐目标

最终推荐发布形态：

- 一个 `curated.exe`
- 前端生产资源嵌入 Go 二进制
- Go 同时提供 API 和前端页面
- 所有可写数据进入用户数据目录

### 8.2 当前尚未完成的部分

当前仓库仍需补齐：

- 前端 `dist/` 的 Go `embed`
- 后端对前端静态资源的托管
- SPA fallback
- 生产首次启动初始化流程
- 统一构建脚本

## 9. 分发方式规划

项目正式发布时，要求同时提供两种分发方式。

### 9.1 分发方式 A：安装器

目标：

- 面向普通用户
- 安装到系统推荐目录
- 创建开始菜单或桌面快捷方式

推荐行为：

- 安装目录：`C:\Program Files\Curated`
- 数据目录：`%LOCALAPPDATA%\Curated`
- 卸载程序时不默认删除用户数据
- 卸载时可提供“是否删除用户数据”的可选项

安装器至少应包含：

- `curated.exe`
- 版本信息
- 安装目录规则
- 快捷方式规则
- 卸载器

推荐安装器产物命名：

- `Curated-Setup-{version}.exe`

### 9.2 分发方式 B：绿色版 zip

目标：

- 面向高级用户、测试用户、便携场景
- 无需安装，解压即用

推荐行为：

- zip 内包含 `curated.exe`
- 首次运行时仍默认把可写数据放到 `%LOCALAPPDATA%\Curated`
- 如需便携模式，可后续再设计明确的“便携数据目录开关”，但不应在第一阶段混入

绿色包产物命名建议：

- `Curated-{version}-windows-x64.zip`

### 9.3 两种分发方式的共通要求

两种分发方式都必须满足：

- 使用同一份 release 二进制
- 使用同一套版本号与构建戳
- 程序内显示的版本信息一致
- 默认数据目录策略一致

## 10. release 产物规划

建议统一产物目录结构：

```text
release/
  Curated/
    curated.exe
  installer/
    Curated-Setup-0.1.0.exe
  portable/
    Curated-0.1.0-windows-x64.zip
  manifest/
    release.json
```

建议 `release.json` 至少记录：

- 产品版本号
- BuildStamp
- channel
- 构建时间
- 产物文件名
- 哈希值

## 11. 打包与发布脚本规划

### 11.1 脚本目标

脚本需要覆盖以下工作：

1. 构建前端生产资源
2. 构建 Go release 二进制
3. 注入版本号与 BuildStamp
4. 整理 release 目录
5. 生成绿色版 zip
6. 构建安装器
7. 生成发布清单

### 11.2 推荐脚本拆分

建议在仓库中新增 `scripts/release/` 目录，并拆分为以下脚本：

- `scripts/release/build-frontend.ps1`
  - 执行前端生产构建
- `scripts/release/build-backend.ps1`
  - 使用 `-tags release`
  - 注入 `BuildStamp`
  - 产出 `curated.exe`
- `scripts/release/assemble-release.ps1`
  - 整理 `release/Curated`
  - 输出清单信息
- `scripts/release/package-portable.ps1`
  - 生成绿色版 zip
- `scripts/release/package-installer.ps1`
  - 基于安装器模板生成 setup 包
- `scripts/release/publish.ps1`
  - 串联全部步骤

### 11.3 推荐脚本输入参数

建议所有发布脚本围绕以下参数工作：

- `-Version`
  - 产品版本号，例如 `0.1.0`
- `-BuildStamp`
  - 构建戳，例如 `20260331.153000`
- `-Channel`
  - 对 release 脚本固定为 `release`
- `-OutputDir`
  - 发布输出目录

### 11.4 推荐主脚本行为

建议主脚本行为如下：

```text
publish.ps1
  -> build-frontend.ps1
  -> build-backend.ps1
  -> assemble-release.ps1
  -> package-portable.ps1
  -> package-installer.ps1
  -> generate manifest / checksums
```

### 11.5 版本注入要求

脚本必须确保以下信息统一：

- Go `BuildStamp`
- release channel
- 安装器文件名
- zip 文件名
- 发布清单

如果后续前端 UI 需要直接显示产品版本号，脚本也应负责将该版本号注入前端构建环境。

## 12. 安装器实现建议

当前计划阶段建议先明确目标，不强绑具体工具。

Windows 安装器建议满足：

- 支持安装到默认目录
- 支持创建快捷方式
- 支持卸载
- 支持版本升级覆盖安装
- 不误删用户数据目录

后续可在实现阶段比较如下工具：

- Inno Setup
- NSIS
- WiX

当前建议优先选择：

- **Inno Setup**

原因：

- Windows 分发成熟
- 脚本化简单
- 适合单 exe 安装
- 易于与 PowerShell 发布脚本集成

## 13. 分阶段实施计划

### 阶段 1：确认版本与产物规则

目标：

- 确认 release 版本命名规则
- 确认 UI 版本展示规则
- 确认安装器与绿色包命名规则

产出：

- 本文档评审通过
- 明确 `Version` 与 `BuildStamp` 的输入来源

### 阶段 2：确认配置与数据目录策略

目标：

- 确认数据目录结构
- 确认 `app.json` 与 `library-config.cfg` 的边界
- 确认首次启动初始化内容

### 阶段 3：打通 release 构建链路

目标：

- 前端可稳定生成生产资源
- Go 可用 `-tags release` 构建
- release 二进制带正确 `BuildStamp`

### 阶段 4：实现 release 打包脚本

目标：

- 一条命令生成 release 目录
- 一条命令生成绿色版 zip
- 一条命令生成安装器

### 阶段 5：实现发布产物与安装器

目标：

- 安装器可安装、升级、卸载
- 绿色版 zip 可解压即用
- 程序内版本显示正确

### 阶段 6：补齐文档与发布说明

目标：

- README 增加 release 使用说明
- 补充故障排查说明
- 补充数据目录与日志位置说明

## 14. 近期建议落地顺序

结合当前项目阶段，建议按如下顺序推进：

1. 先确认本文档中的版本策略和产物命名规则
2. 设计最小 `app.json` 草案
3. 打通前端生产构建与 Go release 构建
4. 实现 release 目录整理脚本
5. 实现绿色版 zip 打包
6. 实现安装器打包
7. 最后补齐 UI 版本显示和发布文档

## 15. 推荐结论

对于 Curated，推荐的正式生产方案是：

- **单二进制分发**
- **外置数据目录**
- **程序内稳定显示版本**
- **同时提供安装器和绿色版 zip**
- **用统一脚本完成构建、打包和发布**

一句话概括：

**程序负责分发，数据目录负责持久化，版本信息贯穿构建、产物和运行时显示。**

## 16. 预览包可行性补充（2026-04-16）

### 16.1 当前状态结论

当前仓库对“预览包”是**部分具备基础，但还没有完整支持**。

已经具备的基础：
- `-tags release` 构建默认把可写数据放到 **`%LOCALAPPDATA%\Curated`**，即：
  - `config/library-config.cfg`
  - `data/curated.db`
  - `cache/`
  - `logs/`
- 这意味着：**只要还是 release 语义的构建，天然就能复用正式版已经存在的数据库、配置和缓存目录**。

当前缺失的部分：
- 发布脚本当前只有 **`release`** 通道，没有 **`preview`** 通道。
- 安装器模板当前固定写死：
  - `AppId={{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}`
  - `AppName=Curated`
  - `DefaultDirName={autopf}\Curated`
  - `AppMutex=Local\Curated.Tray.Singleton`
- 这表示当前如果直接“再打一份预览安装包”，它不会是一个真正独立的 preview 安装身份，而更像同一个正式版产品的另一份安装产物。

### 16.2 是否能支持

**能支持。**

但要先明确一个边界：你说的“预览包”其实包含两个不同层面的诉求：

1. **安装身份**
   - 是覆盖正式版安装，还是作为 `Curated Preview` 并存安装。
2. **数据身份**
   - 是继续共用 **`%LOCALAPPDATA%\Curated`**，还是使用独立的 preview 数据目录。

### 16.3 三种可选方案

#### 方案 A：预览包覆盖正式版安装，共用正式版数据

特点：
- 改动最小。
- 安装器仍沿用同一个 `AppId`、安装目录和快捷方式。
- 数据继续使用 **`%LOCALAPPDATA%\Curated`**。

优点：
- 实现成本最低。
- 升级/回滚路径简单。

缺点：
- **不能与正式版并存安装。**
- 用户体验上更像“候选正式版”，不是真正独立的 preview。

#### 方案 B：预览包独立安装，共用正式版数据

特点：
- 安装器使用独立身份，例如：
  - `AppId=Curated Preview`
  - `AppName=Curated Preview`
  - `DefaultDirName={autopf}\Curated Preview`
- 但运行时**仍然指向**正式版同一份数据根：
  - **`%LOCALAPPDATA%\Curated`**

优点：
- **可以和正式版并存安装。**
- 预览包可以直接读取现有数据库、配置、缓存，满足“无缝接上正式版数据”的目标。
- 最符合“预览包是整机包，但又不重新建一套库”的诉求。

缺点：
- 必须认真处理**并发访问同一份 SQLite / 配置**的问题。
- 预览包和正式版不能被允许同时运行并同时写同一份数据。

补充约束：
- 如果 preview 与 release 共用同一份数据目录，建议它们也共用同一把全局互斥锁，或者显式做“跨通道单实例”控制。
- 否则两个安装实例同时打开同一个 `curated.db`，风险不可接受。

#### 方案 C：预览包独立安装，独立数据目录

特点：
- 安装身份独立。
- 数据目录也独立，例如 **`%LOCALAPPDATA%\CuratedPreview`**。

优点：
- 最安全，互不影响。
- 不存在共用 SQLite 的运行时竞争。

缺点：
- **不满足“无缝复用正式版数据库/配置”的核心诉求。**

### 16.4 推荐方案

**推荐方案 B：独立安装身份，共用正式版数据目录。**

这是最符合“预览包是一个可安装的整机包，同时又能直接接上正式版已有数据库和配置”的方案。

但必须附带两个硬约束：
- **硬约束 1：preview 与 release 不能同时运行。**
- **硬约束 2：preview 默认不自动接管正式版的自启动、系统集成和默认打开行为。**

### 16.5 为支持方案 B 需要补的实现点

1. **新增打包通道/风味（flavor）**
   - 当前只有 `release`。
   - 需要新增一个例如 `preview` 的打包风味，但它的数据目录策略要继续复用 release 的用户数据根。

2. **脚本参数化**
   - `build-backend.ps1`
   - `assemble-release.ps1`
   - `package-installer.ps1`
   - `publish.ps1`
   - 这些脚本都需要支持 `-Channel` 或 `-Flavor`，不能再把 `release` 写死。

3. **安装器模板参数化**
   - `AppId`
   - `AppName`
   - `DefaultDirName`
   - `DefaultGroupName`
   - 输出文件名
   - 让 preview 包拥有独立安装身份。

4. **运行时版本/渠道显示**
   - 当前 `version.Channel` 只有 `dev` / `release`。
   - 需要新增可显示的 preview 标识，至少让健康检查和 UI 能看出这是 preview 包。

5. **开发态功能开关**
   - 不能直接把 preview 做成 dev build。
   - 正确做法应是：**仍然是 production packaging / release-style runtime**，但通过显式的 feature flags 打开部分开发态能力。

6. **共享数据下的互斥策略**
   - 如果 preview 共用正式版数据目录，建议复用正式版 mutex，或单独增加一个“共享数据根互斥”。
   - 目标是防止 preview 和 release 双开。

7. **自启动与系统集成策略**
   - preview 不建议默认写正式版相同的开机自启项。
   - 是否允许 preview 修改正式版的系统自启配置，需要单独定规则。

### 16.6 一句话结论

**支持，但不能只靠“再打一份 release 安装包”解决。**

如果要满足：
- 可安装到本机
- 保留 preview 独立身份
- 无缝复用正式版数据库与配置

则需要把当前发布体系从“单一 release 通道”扩展为“release / preview 两种 packaging flavor + 共享数据策略 + 跨安装实例互斥策略”。

## 17. 打包历史补充：记录相对上一包的 Git 变更内容（2026-04-16）

### 17.1 目标

在 `docs/ops/2026-04-02-package-build-history.md` 的表格中新增一列：

- `变更内容`

用于记录**本次打包相对于上一条打包记录**之间的 Git 提交历史，直接来源于 `git log --oneline`，不做智能归类、不做自然语言总结。

### 17.2 适用范围

本规则适用于所有会自动写入打包历史的打包类型：

- `release:publish`
- `release:installer`
- `release:portable`

也就是说，不区分打包类型；只要脚本调用 `Add-PackageBuildHistoryEntry` 追加台账，就要同时写入 `变更内容`。

### 17.3 对比基线定义

`变更内容` 的生成范围定义为：

- **上界**：当前写入打包历史时的 Git `HEAD`
- **下界**：打包历史文件中**上一条记录**的 `提交 / 分支` 列里记录的 commit short SHA

等价于执行：

```powershell
git log --oneline <previous-commit>..<current-commit>
```

注意点：

- 这里的“上一条记录”指打包历史文件里时间上紧邻当前追加位置的上一行有效记录。
- **不是**“上一条同类型打包记录”。
- 这样可以保证在“所有打包类型都写历史”的规则下，对比基线始终唯一且稳定。

### 17.4 写入内容格式

`变更内容` 列直接写 `git log --oneline` 的结果，保留原始提交标题，使用 Markdown 单元格安全编码后写入：

- 多条提交之间使用 `<br>` 换行
- 不做 `feat/fix/revert/docs` 分类
- 不额外改写成“新增 / 修改 / 回退”中文摘要

示例：

```text
abc1234 feat(player): add external player settings<br>
def5678 fix(release): keep installer close-app behavior<br>
9876abc docs: update release packaging notes
```

### 17.5 特殊情况规则

#### A. 同一提交重复打包

如果上一条记录和当前记录指向同一个 commit，则写：

```text
无代码差异（同一提交重复打包）
```

#### B. 历史中没有上一条记录

如果当前要写入的是第一条打包记录，没有可对比的上一条记录，则写：

```text
首条打包记录，无上一包可比对
```

#### C. 无法解析上一条记录中的 commit

如果上一条记录里的 `提交 / 分支` 列格式异常，或 short SHA 无法在当前仓库解析到真实 commit，则写：

```text
无法解析上一条打包记录对应 commit
```

#### D. 范围内没有新提交

如果 `git log --oneline <prev>..<current>` 返回空，等价处理为：

```text
无代码差异（同一提交重复打包）
```

### 17.6 文档表头调整

`docs/ops/2026-04-02-package-build-history.md` 的历史表格表头需要从：

```text
| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 备注 |
```

调整为：

```text
| 日期 | 版本 | 提交 / 分支 | 打包类型 | 产物路径 | 状态 | 操作人 | 变更内容 | 备注 |
```

同时需要把已有历史记录补一个占位值，避免 Markdown 表格列数不一致。历史旧记录的 `变更内容` 可统一补为：

```text
历史记录补齐前未采集
```

### 17.7 脚本实现边界

实现应集中在 `scripts/release/release-common.ps1` 中，围绕 `Add-PackageBuildHistoryEntry` 补齐，避免把同一套逻辑分散到：

- `publish.ps1`
- `package-installer.ps1`
- `package-portable.ps1`

推荐新增两个辅助函数：

- `Get-LastPackageHistoryCommit`
  - 从历史文件中解析上一条记录的 short SHA
- `Get-PackageHistoryChangeSummary`
  - 基于 `git log --oneline <prev>..<head>` 生成要写入 `变更内容` 的文本

然后由 `Add-PackageBuildHistoryEntry` 在最终拼接 Markdown 行时统一写入。

### 17.8 不做的事情

本次规则明确**不做**以下能力：

- 不按 `feat/fix/revert` 自动归类
- 不生成中文“新增了什么 / 修改了什么 / 回退了什么”的总结句
- 不按“上一条同类型打包记录”作为比较基线
- 不跨版本号或 manifest 再做二次推断

目标只保留一个：

**在打包台账中稳定记录“上一包到当前包之间的原始 Git 提交标题列表”。**

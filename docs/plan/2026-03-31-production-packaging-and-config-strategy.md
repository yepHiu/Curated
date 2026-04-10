# Curated 生产打包、配置、版本与发布计划

## 1. 文档定位

本文是 **Curated** 下一阶段生产发布工作的规划文档，重点回答以下问题：

- 如何从当前开发态的 Vue + Go 项目过渡到可分发的生产版本
- 在“单二进制分发”目标下，配置、数据库、缓存、日志应该如何处理
- 如何满足程序内必须显示版本号的既有约定
- 如何同时提供安装器版本和绿色版 zip 版本
- 如何规划打包、产物整理和发布脚本

本文描述的是 **推荐方案、约束和实施计划**，不是当前仓库已经全部完成的事实实现。

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
- `extendedLibraryImport`
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

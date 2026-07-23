# Curated 容器化部署可行性调研

> 调研日期：2026-07-18
> 调研范围：当前 `master` 分支（调研时 HEAD `6fb3752b`）的 Vue 前端、Go 后端、SQLite、FFmpeg、Intel 核显、桌面能力、数据目录、开发环境与发布脚本。
> 本文区分“当前已经实现”“容器化目标方案”“迁移前待补能力”，不把建议误写成现状。

## 1. 结论摘要

**结论：可行，且目标已经明确为“Linux 主机上的单节点、单实例 Curated 服务”。Intel 核显转码应从原计划中的可选 Phase 4 前移为 Linux beta 的核心能力；容器化开发环境则作为独立工作流建设，不能通过进入生产容器开发来代替。**

### 1.1 本轮确认的目标基线

后续方案按以下前提设计：

1. **宿主机是 Linux**：Docker Engine / Compose 或兼容 OCI 运行时负责 Curated 生命周期，Electron 和 Windows 托盘不进入服务器部署链。
2. **宿主机有 Intel 核显**：希望把 `/dev/dri` 中的 DRM render node 提供给 Curated，优先用于 H.264 HLS 转码；实际支持的解码 / 编码格式由 CPU 代际、宿主机内核驱动、容器用户态驱动和 FFmpeg 构建共同决定，不能只凭“Intel 核显”四个字静态假设。
3. **长期可能在 Linux 容器内开发**：需要可复现的前端 + Go + FFmpeg 开发容器、热更新、缓存卷、测试命令和 iGPU 调试能力；生产镜像继续保持最小、不可变、无编译工具。
4. **部署仍是单实例**：SQLite、上传会话、任务和 HLS session 不做多副本共享。

因此本文把工作拆为三条互相配合但不混用的链路：

| 链路 | 目标 | 镜像特征 |
|---|---|---|
| 生产运行 | Linux 主机长期托管 Curated | 最小运行时、非 root、只含 Curated + 前端 + FFmpeg / Intel 用户态运行库 |
| GPU 验证 | 证明 Intel 核显在目标主机与目标镜像中真实可用 | 可带 `vainfo`、FFmpeg 诊断和主动编码测试；验证完成后可独立为 debug target |
| 容器化开发 | 在 Linux 主机容器内写代码、跑 Vite / Go / 测试 | Node、pnpm、Go、Git、调试工具、命名缓存卷、源码 bind mount，可映射同一 iGPU |

当前项目已经拥有容器化最关键的基础：

- Vue 前端可构建为纯静态文件。
- Go 后端可同时托管 `/api` 与前端静态文件，运行时不需要 Node.js 或 Vite。
- 主数据库是单文件 SQLite，并已有自动 migration。
- release 构建支持通过 `CURATED_DATA_DIR` 把配置、数据库、缓存和日志放到程序目录之外。
- 后端使用纯 Go SQLite 驱动，不依赖系统 SQLite 或 CGO。
- HTTP 服务已处理 `SIGTERM` 上下文取消，并有 5 秒 HTTP graceful shutdown。
- 当前代码已经用 build tag / stub 隔离大部分 Windows 专属能力。

本次在 Windows 开发机上进行了实际交叉编译验证：

| 目标 | 构建参数 | 结果 |
|---|---|---|
| Linux x86-64 | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags release` | 成功，二进制约 34.9 MB |
| Linux ARM64 | `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags release` | 成功，二进制约 32.8 MB |

这证明后端并非只是“代码看起来跨平台”，而是已经能生成常见服务器与 NAS 架构的 Linux release 二进制。

不过，当前仓库还没有 `Dockerfile`、Compose 文件、容器 CI 或容器运行模式。要达到可公开交付的容器版本，至少还要解决以下五类问题：

1. **跨系统路径迁移**：现有 Windows 数据库会保存 `D:\...` 一类绝对路径；Linux 容器需要 `/media/...`。当前 API 只能修改资料库标题，不能事务性替换资料库根路径及其派生资源路径。
2. **运行时能力识别**：托盘、Electron 目录选择、打开文件管理器、本机播放器、开机启动和 Windows `.exe` 自动更新在无桌面的 Linux 容器里无意义；目前部分功能只会执行失败，而不会统一显示为“不受支持”。
3. **远程访问安全**：PIN 默认关闭，CORS 当前会回显任意 `Origin` 并允许凭据，认证 cookie 未设置 `Secure`。因此当前后端不应不经保护直接暴露到公网。
4. **Linux Intel 媒体链路**：FFmpeg 软件转码可用，但当前硬件编码 profile 只为 Windows 和 macOS 生成；Linux 即使映射 `/dev/dri`，也仍只会回退到 `libx264`。这不是容器权限配置就能解决的问题，还要修改播放 profile、配置契约和能力探测。
5. **开发态数据隔离**：当前 `CURATED_DATA_DIR` 只在 `release` build 生效；普通 `go run` 仍使用仓库相对路径和根目录 `config/library-config.cfg`。若要把开发环境放进容器，需要补统一的数据根 / 设置文件覆盖机制。

综合判断：

| 目标场景 | 可行性 | 结论 |
|---|---:|---|
| 新建一个可信 LAN 内的单实例容器，使用新数据库 | 高 | 可直接进入 PoC |
| NAS / 家庭服务器长期运行，媒体目录挂载到容器 | 较高 | 补齐镜像、权限、备份、安全与降级能力后可发布 beta |
| 从现有 Windows 安装版携带完整数据库迁移 | 中 | 必须先实现或提供受控的路径迁移工具 |
| 依赖 HLS 软件转码 | 中高 | 功能可用，但需要评估 CPU、并发和散热 |
| Linux 主机 Intel 核显转码 | 中高（需开发） | 设备透传本身成熟；当前代码缺 Linux QSV/VAAPI profile、驱动探测和设备配置，完成后适合作为 beta 核心能力 |
| Linux 容器内开发 | 高（需工程化） | Vite 已监听 `0.0.0.0`，Go 可原生运行；需补 dev image、Compose、缓存卷、非 release 数据根与 GPU 调试入口 |
| 直接公网开放端口 | 低 | 当前安全边界不满足，必须放在 TLS 反向代理 / 身份代理后并加固应用侧 |
| 多副本、滚动扩容、共享同一数据库 | 低 | SQLite、内存任务、上传会话和 HLS 会话决定了当前只能单实例 |

## 2. 当前架构与容器映射

### 2.1 当前已经实现的运行链路

生产态的核心并不是 Electron 本身，而是：

```text
浏览器 / Electron BrowserWindow
          │
          │ HTTP，同源 /api
          ▼
Go HTTP 服务
  ├─ 托管 frontend-dist/
  ├─ REST API + SSE
  ├─ SQLite
  ├─ 扫描 / 刮削 / 目录监听
  └─ FFmpeg / ffprobe
```

Electron 只是桌面生命周期和目录选择壳；业务 API 仍走 Go HTTP 服务。因此容器版不需要把 Electron 放进容器，只需保留浏览器 Web 入口。

### 2.2 推荐的 Linux 主机目标架构

```text
浏览器 / 手机 / 平板
          │
          │ HTTPS（推荐）
          ▼
可选反向代理 / 身份代理
  Caddy / Nginx / Traefik / Tailscale
          │
          │ HTTP :8081
          ▼
Linux 宿主机
  ├─ 内核 Intel i915 / xe 驱动
  ├─ /dev/dri/renderD*（DRM render node）
  ├─ /srv/curated/data（本地持久数据）
  └─ /mnt/media（宿主机媒体挂载）
          │
          ▼
Curated 单容器、单实例
  ├─ /app/curated
  ├─ /app/frontend-dist/
  ├─ /dev/dri/renderD*（设备映射，不要求 privileged）
  ├─ Intel 用户态媒体驱动 + FFmpeg
  ├─ /data（读写、本地持久卷）
  │    ├─ config/
  │    ├─ data/curated.db
  │    ├─ cache/
  │    └─ logs/
  └─ /media（通常读写、宿主机媒体挂载）
       └─ library-a / library-b ...
```

这里推荐继续让 Go 服务直接托管静态前端，而不是再加入 Nginx 前端容器，原因是：

- 当前同源 `/api`、SSE、视频 Range、HLS 和大文件上传已经围绕单一 HTTP 服务设计。
- 少一个容器和一层内部代理，能减少 Range、流式上传、SSE buffering 与超时配置错误。
- 反向代理只在需要 TLS、域名、SSO 或公网入口时放在 Curated 外部。

### 2.3 Linux 宿主机与容器的职责边界

| 层 | 负责内容 | 不负责内容 |
|---|---|---|
| Linux 宿主机 | 内核、Intel `i915` / `xe` 驱动、DRM 设备、磁盘 / NAS 挂载、Docker 生命周期、主机监控 | 不运行 Curated Node / Go 开发依赖，不直接打开 Curated SQLite |
| Curated 生产容器 | Go 服务、静态前端、FFmpeg、Intel 用户态媒体驱动、SQLite 访问、转码 session | 不加载内核模块、不管理宿主机驱动、不获得 Docker socket |
| 反向代理 | TLS、域名、可选外层身份认证、SSE / Range / 上传代理 | 不拆分 Curated 前端和后端状态 |
| 开发容器 | 编译、热更新、测试、GPU 命令诊断 | 不复用生产 `/data`，不作为正式服务长期运行 |

Intel 核显能否使用的完整依赖链是：

```text
CPU / iGPU 代际
  -> Linux 内核 i915 / xe 驱动
  -> /dev/dri/renderD* 设备存在
  -> 容器用户拥有 render node 权限
  -> 容器内 Intel VA-API / oneVPL 用户态驱动正确
  -> 容器内 FFmpeg 编译启用了目标 hwaccel / encoder
  -> Curated 生成匹配该设备和 encoder 的参数
  -> 使用真实片源主动测试成功
```

其中任意一层失败都必须可诊断，并安全回退到软件转码；不能只检查 `/dev/dri` 存在就报告“硬件加速可用”。

## 3. 组件级可行性矩阵

| 能力 | 当前容器可用性 | 说明 / 迁移要求 |
|---|---|---|
| Vue 页面 | 可用 | 构建阶段必须显式设置 `VITE_USE_WEB_API=true`，不能依赖被 Git 忽略的本机 `.env`。 |
| Go HTTP API | 可用 | release 默认监听 `:8081`，Linux 默认运行模式是 `http`。 |
| 前端静态托管 | 可用 | 将 `dist/` 复制为二进制同目录下的 `/app/frontend-dist/`。 |
| SQLite | 可用 | 纯 Go 驱动、单连接写入；数据库应放本地 `/data` 持久卷，不应直接放 SMB/NFS。 |
| 数据库 migration | 可用 | 启动时自动执行；升级前仍需备份数据库。 |
| 资料库扫描 | 可用 | 容器内保存的资料库路径必须是 `/media/...` 形式的容器路径。 |
| 元数据刮削 / 资源下载 | 可用 | 需要 DNS、CA 证书和出站网络；代理地址中的 `127.0.0.1` 在容器中指容器自己。 |
| 本地封面 / 预览图 | 可用 | 缓存目录和媒体目录权限必须正确；迁移旧库时需处理数据库中的绝对路径。 |
| 视频 Range 直传 | 可用 | 同源服务天然适合；若加反向代理，必须保留 Range 头且避免不必要缓存。 |
| HLS remux / 软件转码 | 可用 | 运行镜像需包含 `ffmpeg` 与 `ffprobe`；转码时 CPU 和缓存 I/O 可能较高。 |
| Linux Intel 核显设备映射 | 容器层可行 | 映射 DRM render node 并补 render/video 附加组即可，不需要 `privileged`；设备名和组 GID 必须从目标主机发现。 |
| Linux Intel 硬件转码 | 代码层当前不可用，目标可行 | 代码只在 Windows 添加 NVENC/QSV/AMF，在 macOS 添加 VideoToolbox；Linux 只生成 `libx264`。需新增 QSV/VAAPI profile、主动探测和软件回退。 |
| SSE 任务事件 | 可用 | 反向代理需关闭响应 buffering，并提高长连接超时。 |
| 浏览器导入 / 断点上传 | 可用 | `/media` 必须可写；代理需允许大请求体并关闭请求 buffering。上传会话状态在内存中，容器重启会丢会话状态。 |
| 目录监听 | 条件可用 | 本地 bind mount 通常可用；SMB/NFS/FUSE 的 inotify 事件可能不可靠，应支持关闭监听并使用周期 / 手动扫描。 |
| 存储在线检测 | 基础可用 | Linux 只做路径存在性和可读性探测，没有 Windows 卷序列号级身份能力。 |
| PIN 锁 | 条件可用 | 后端 PIN + cookie 可工作，但还不足以单独承担公网身份认证。 |
| Electron 目录选择 | 不可用 | 普通浏览器无法获得服务器文件系统绝对路径；容器版需要用户手工输入 `/media/...` 或使用服务端目录浏览器。 |
| 打开文件管理器 / 定位影片 | 实质不可用 | 当前 Linux 路径会尝试 `xdg-open`，在无桌面的容器里会失败；UI 应按 runtime capability 隐藏。 |
| 后端启动本机播放器 | 实质不可用 | 会尝试在容器内启动 `mpv` 等进程；应在容器模式禁用。浏览器协议模板仍可作为客户端本机能力单独保留。 |
| 托盘 / 开机启动 | 不适用 | Docker restart policy 取代托盘常驻与 Windows Run 注册表。 |
| Windows 安装器自动更新 | 不适用且有误导风险 | 当前服务按 GitHub Release 选择 `.exe`，Linux release 仍可能显示下载 / 安装入口；容器版应改为“镜像更新由容器平台负责”。 |
| Connected Clients | 可用但需代理适配 | 反向代理后 `RemoteAddr` 会变成代理地址；若要显示真实客户端，需要受信任代理与转发头策略。 |
| Linux 容器化开发 | 基础可行 | Vite 已监听 `0.0.0.0`；需新增 dev image / Compose、命名缓存卷、开发数据卷和非 release `CURATED_DATA_DIR` 支持。 |

## 4. 镜像设计建议

### 4.1 使用多阶段构建

建议单独新增容器构建链，不复用现有 Windows release 组装流程。现有 `scripts/release/release_lib/build_steps.py` 会注入 `-H=windowsgui`、输出 `curated.exe`、组装 Electron runtime 和 Inno Setup 产物，它适合 Windows 桌面包，不适合 Linux 镜像。

推荐三阶段：

1. **frontend-builder**
   - Node 24 或满足 Vite 8 要求的受支持 Node 版本。
   - Corepack / pnpm 10。
   - `pnpm install --frozen-lockfile`。
   - 显式执行 `VITE_USE_WEB_API=true pnpm build`。
   - 可注入 `VITE_APP_VERSION`，但不要写死外部 API 地址，保持同源 `/api`。

2. **backend-builder**
   - `golang:1.25.4`。
   - `CGO_ENABLED=0`。
   - `go build -tags release`。
   - 用 `-ldflags` 注入 `BuildStamp` 和容器镜像版本。
   - 同时构建 `linux/amd64` 与 `linux/arm64`。

3. **runtime**
   - Debian slim 作为第一版比 Alpine / distroless 更稳妥，主要原因是 FFmpeg 包与运行诊断更简单。
   - 安装 `ca-certificates`、`ffmpeg`、`tzdata`、Intel 对应代际的 VA-API / oneVPL 用户态运行库，以及在采用 HTTP healthcheck 时所需的极小 HTTP 客户端。
   - 复制 Go 二进制到 `/app/curated`。
   - 复制前端产物到 `/app/frontend-dist/`。
   - `WORKDIR /app`，让现有静态文件和 FFmpeg 邻接查找逻辑保持可预测。
   - 设置 `CURATED_DATA_DIR=/data`。
   - 入口固定使用 `-mode http`，避免未来默认模式变化影响容器。

### 4.2 不建议的镜像设计

- 不要在运行镜像中安装 Node、pnpm、Vite 或 Electron。
- 不要把 SQLite、配置和缓存烘焙进镜像层。
- 不要把用户媒体 `COPY` 进镜像。
- 不要直接复制当前本机 `config/library-config.cfg` 作为公共默认配置；该文件可能含代理凭据、Windows 路径或本机命令。
- 不要把 Windows FFmpeg 二进制复制到 Linux 镜像；直接使用目标平台 FFmpeg 包或明确的 Linux 静态发行物。
- 不要把 Windows 安装包版本治理与容器镜像更新强行绑成同一套运行时行为。版本号可以统一，更新机制应分开。

### 4.3 Intel 运行镜像策略

Intel 媒体驱动不能只写成一个永远不变的 apt 包名。不同 Linux 发行版、CPU 代际和 FFmpeg 版本可能使用不同的 Intel media driver / oneVPL 组合，因此建议把镜像治理拆成：

1. **生产 target**：只保留已经在目标 Linux 主机验证过的运行库，不携带编译器和大批诊断工具。
2. **GPU debug target**：在同一 Dockerfile 中额外包含 `vainfo`、`ffmpeg` 完整诊断、`lspci` / DRM 查看工具，供部署前和驱动升级后检查。
3. **构建时门禁**：CI 至少检查 `ffmpeg -hwaccels` 和 `ffmpeg -encoders` 中包含预期的 `qsv` / `vaapi` 能力；这只证明 FFmpeg 构建支持，不等于宿主机设备已可用。
4. **运行时门禁**：目标主机上实际打开 render node，并执行短时编码测试；成功后才向 Curated 报告硬件 profile 可用。

推荐优先顺序不是在文档中永久写死“只用 QSV”或“只用 VAAPI”，而是：

```text
auto
  -> 主动验证 QSV H.264 编码，成功则优先
  -> 否则主动验证 VAAPI H.264 编码，成功则使用
  -> 否则标记硬件不可用并回退 libx264
```

理由：现有设置与 profile 已有 `qsv` 语义，扩展 Linux QSV 的改动较集中；VAAPI 则是 Linux DRM 原生、覆盖面较好的后备路径。最终选择必须以目标镜像中的实际 FFmpeg 和目标机器的主动测试为准。

Intel GPU runtime 不应通过浮动 `latest` 镜像隐式升级。正式发布至少固定：

- 基础发行版版本。
- FFmpeg 主版本或完整包版本。
- Intel 用户态媒体驱动版本。
- Curated 镜像 digest。

因为同一份 Curated Go 代码在 FFmpeg / 驱动变化后可能出现 encoder 名称仍存在、实际设备初始化却失败的情况。

## 5. 运行配置、存储与权限

### 5.1 推荐卷布局

| 容器路径 | 权限 | 用途 | 备份策略 |
|---|---|---|---|
| `/data` | 读写 | 配置、SQLite、缓存、日志 | 配置与数据库必须备份；缓存和日志可选 |
| `/media` | 通常读写 | 资料库、导入目标、NFO、封面及文件整理 | 媒体由用户现有备份策略管理 |
| `/tmp` | 临时读写 | FFmpeg 或系统临时文件 | 不备份，可使用 tmpfs |

当前 Curated 不只是“只读媒体索引器”：

- `organizeLibrary` 可能移动 / 重命名视频。
- 刮削可能在媒体旁写 NFO 和资源。
- 浏览器导入会把文件写入资料库根目录。
- 删除 / 恢复与上传 staging 会修改资料库。

因此不能简单把 `/media` 设为只读后仍承诺全部功能可用。如果未来要支持严格只读部署，应新增明确的 `readOnlyLibrary` 运行模式，在服务端统一关闭所有写文件 API，而不是只依赖 Docker mount 报权限错误。

### 5.2 SQLite 卷约束

- `/data/data/curated.db` 推荐位于宿主机本地 ext4、xfs、btrfs、ZFS dataset 或 Docker local volume。
- 不建议把 SQLite 文件直接放到 SMB / NFS / WebDAV / 云盘同步目录。
- 当前数据库只允许一个打开连接，能降低 SQLite busy 风险，但也意味着单实例是明确边界。
- 同一数据库禁止同时被 Windows 桌面实例和容器实例打开。
- 容器升级前先停止写入并备份数据库；至少保留最近一次可回滚镜像对应的数据库副本。

### 5.3 用户与文件权限

正式镜像不应长期以 root 写媒体目录。建议：

- 镜像内默认非 root 用户，或在 Compose 用 `user: "${PUID}:${PGID}"` 与宿主机媒体目录所有者对齐。
- 启动前验证 `/data` 可创建文件、SQLite 可写、所有资料库根可读；启用整理 / 导入时还需验证可写。
- Intel render node 的权限通常来自宿主机 `render` 附加组；如需要访问 `card*`，可能还需要 `video` 组。Compose 应使用宿主机实际数字 GID 做 `group_add`，不能假设所有发行版的 `render` GID 相同。
- `cap_drop: [ALL]`、`no-new-privileges:true`。
- Intel 核显只需映射必要的 DRM render node；不需要 privileged、host PID、host IPC 或 Docker socket。

### 5.4 概念性 Compose 形态

以下仅用于说明目标接口；正式落地时应由实际 `Dockerfile`、镜像名和 healthcheck 能力验证后再提交：

```yaml
services:
  curated:
    image: ghcr.io/<owner>/curated:<version>
    restart: unless-stopped
    user: "${PUID:-1000}:${PGID:-1000}"
    group_add:
      - "${RENDER_GID}" # 宿主机 render 组的数字 GID
      # 只有实际需要 card* 节点时再增加 VIDEO_GID
    environment:
      CURATED_DATA_DIR: /data
      TZ: Asia/Shanghai
    command:
      - -mode
      - http
      - -config
      - /data/config/app.json
    ports:
      - "8081:8081" # 仅 LAN；接本机反向代理时改绑 127.0.0.1
    devices:
      - "/dev/dri/renderD128:/dev/dri/renderD128"
    volumes:
      - /srv/curated/data:/data
      - /mnt/media:/media
    stop_grace_period: 30s
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
```

建议的 `/data/config/app.json` 最小基线：

```json
{
  "httpAddr": ":8081",
  "databasePath": "/data/data/curated.db",
  "cacheDir": "/data/cache",
  "libraryPaths": [],
  "autoScanIntervalSeconds": 0,
  "libraryWatchEnabled": false,
  "player": {
    "hardwareDecode": false,
    "hardwareEncoder": "software",
    "nativePlayerEnabled": false,
    "streamPushEnabled": true,
    "ffmpegCommand": "ffmpeg"
  }
}
```

上面的 `app.json` 是当前代码可以接受的软件回退基线。完成 Intel 支持后，建议扩展现有 player 配置，而不是另造一套互不兼容的 GPU 配置：

```json
{
  "player": {
    "hardwareDecode": true,
    "hardwareEncoder": "auto",
    "hardwareDevice": "/dev/dri/renderD128",
    "hardwareFallback": true,
    "maxConcurrentTranscodes": 1,
    "nativePlayerEnabled": false,
    "streamPushEnabled": true,
    "ffmpegCommand": "ffmpeg"
  }
}
```

其中 `hardwareDevice`、`hardwareFallback` 和 `maxConcurrentTranscodes` 是**建议新增、当前尚未实现**的字段。在实现前不能直接写进 `app.json`，因为主配置启用了未知字段拒绝策略。`hardwareEncoder` 还应增加 `vaapi` 合法值，并同步后端 normalize、HTTP DTO、前端类型、设置页选项与测试。

说明：

- `libraryWatchEnabled=false` 是 NAS / 网络挂载的保守默认值；本地文件系统验证可靠后可开启。
- 资料库根应通过 UI 手工填写 `/media/...`，或在首次初始化配置中写入容器路径。
- 应把 `organizeLibrary` 的容器默认值单独确认。对首次 beta，建议默认关闭，待用户确认挂载与权限无误后再开启。

### 5.5 Linux 宿主机部署前检查

在写 Compose 前先在目标 Linux 主机记录以下事实，避免把驱动、设备名和 GID 猜进镜像：

```bash
uname -a
cat /etc/os-release
lspci -nn | grep -Ei 'vga|display'
ls -l /dev/dri
getent group render
getent group video
```

然后在宿主机或 GPU debug 容器中验证：

```bash
vainfo --display drm --device /dev/dri/renderD128
ffmpeg -hide_banner -hwaccels
ffmpeg -hide_banner -encoders | grep -E 'qsv|vaapi'
ffmpeg -hide_banner -decoders | grep -E 'qsv|vaapi'
```

检查结果应记录到部署台账：

- CPU / iGPU 型号和代际。
- 内核版本以及使用 `i915` 还是 `xe`。
- render node 路径。
- render / video 数字 GID。
- FFmpeg 版本与构建参数。
- 可用 H.264 / HEVC / AV1 decode、encode 能力。

不同 Intel 代际的 8-bit / 10-bit、HEVC、AV1 能力不同。Curated 不应仅根据设备厂商字符串推断支持矩阵，应以 FFmpeg 主动测试结果为准。

## 6. Windows 现有数据迁移

### 6.1 两种迁移策略

#### 策略 A：新数据库重新扫描

优点：

- 最简单，避免 Windows 路径进入 Linux。
- 可以快速验证容器本体。

缺点：

- 会丢失或难以自然恢复用户评分、收藏、播放进度、评论、PIN 会话、推荐状态、萃取帧等数据库状态。
- 即使影片 ID 扫描后碰巧稳定，也不能把它当作完整迁移保证。

适合 PoC，不适合作为已有用户的正式迁移方案。

#### 策略 B：复制数据目录并事务性改写路径

这是正式迁移推荐方案，但当前尚缺受支持工具。路径映射示例：

```text
D:\Media\JAV  ->  /media/jav
E:\Curated    ->  /media/curated
```

至少需要审计 / 迁移以下 SQLite 路径字段：

- `library_paths.path`
- `movies.location`
- `media_assets.local_path`
- `actors.avatar_local_path`
- `library_path_storage_bindings.root_path` 及原 Windows 卷身份字段
- `scan_items.path`（历史任务，可选择清理而非迁移）
- `app_update_status.downloaded_file_path`（应清空，不应映射 Windows installer）

同时要审计 `library-config.cfg` 中的：

- `logDir`
- `player.nativePlayerCommand`
- `player.ffmpegCommand`
- `proxy.url`，尤其是 `127.0.0.1` / `localhost`
- `launchAtLogin`
- 任何手工保存的 Windows 绝对路径

### 6.2 为什么当前“存储重绑”接口不够

现有 `storage-binding/rebind` 只重建资料库路径对应的存储设备身份，不会修改 `library_paths.path`，也不会级联更新影片、资源和演员缓存路径。现有 `PATCH /api/library/paths/{id}` 只修改显示标题。

因此，正式迁移前建议实现一个独立、可回滚的路径迁移能力，二选一：

1. `curated migrate-path --from <old-root> --to <new-root> --dry-run` CLI；或
2. 只对管理员开放的事务性资料库根迁移 API。

推荐 CLI，因为它可以在服务离线时执行，避免边扫描边改路径。应具备：

- `--dry-run` 输出命中行数和示例。
- 要求旧根和新根都是规范化绝对路径。
- 单事务更新所有相关表。
- 更新前自动生成数据库备份或明确要求备份存在。
- Windows `\` 与 Linux `/` 的分隔符转换。
- 清理旧存储 binding，并在新路径上重新探测。
- 迁移后校验每个 `movies.location` 都位于某个新资料库根下。
- 可生成迁移报告，便于回滚与人工抽查。

### 6.3 正式迁移顺序

1. 停止 Windows Curated，避免 SQLite 和媒体目录继续变化。
2. 备份完整用户数据目录和 `library-config.cfg`。
3. 记录每个 Windows 资料库根到容器根的映射。
4. 在宿主机挂载媒体，并从容器内验证 `/media/...` 可读 / 可写。
5. 复制数据库到新的 `/data/data/curated.db`，不要让两个实例共享同一文件。
6. 离线运行路径迁移 dry-run，人工检查命中范围。
7. 执行迁移并清理 Windows installer 状态、Windows 卷 binding 和桌面专属配置。
8. 启动容器，只做健康检查和只读浏览验证。
9. 抽查海报、预览图、直传播放、HLS、萃取帧和用户状态。
10. 最后才开启自动监听、整理、导入与删除能力。

## 7. 安全边界

### 7.1 当前风险

容器发布端口后，Curated 从“本机桌面后端”变成持续监听局域网或远程网络的服务。当前实现需要注意：

- PIN 默认关闭；关闭时所有业务 `/api/*` 都无需认证。
- PIN setup / unlock 是公开入口；如果在首次设置前暴露到不可信网络，存在被他人抢先初始化的风险。
- PIN 是隐私锁，不是完整账号、角色和审计系统。
- `lanRequiresPin` 当前只被保存和返回，认证中间件并未用它建立一条独立的“远程请求必须认证”策略；真正的保护条件仍是全局 `pin_enabled`。
- CORS 当前对任意请求 Origin 原样设置 `Access-Control-Allow-Origin`，并设置 `Access-Control-Allow-Credentials: true`。
- `curated_auth` 是 HttpOnly + SameSite=Lax，但未设置 `Secure`，应用本身也不终止 TLS。
- 当前未见专门的登录速率限制、反向代理可信 IP 配置或 CSRF token。
- 反向代理后客户端 IP 默认会被识别为代理的 `RemoteAddr`。

### 7.2 分阶段安全要求

**LAN alpha 最低要求：**

- 仅在受信任家庭 LAN 发布端口，或只绑定 `127.0.0.1` 后通过 Tailscale / 反向代理访问。
- 首次启动后立即初始化 PIN。
- 不做路由器公网端口转发。
- `/data` 权限只授予容器运行用户。

**可远程访问 beta 最低要求：**

- TLS 反向代理。
- 推荐外层 SSO / Basic Auth / OIDC 身份代理，PIN 作为应用内第二层保护。
- CORS 改为同源默认或显式 allowlist。
- 在 HTTPS / 可信代理场景为 auth cookie 设置 `Secure`。
- 登录 / PIN unlock 速率限制。
- 定义可信反向代理和 `X-Forwarded-For` / `Forwarded` 解析边界。
- 为首次初始化增加一次性 bootstrap secret、仅 loopback 初始化或文件式预配置方案。

**公网正式服务：**

- 在完成上述加固并做独立安全评审前，不建议支持。

## 8. 反向代理与流媒体注意事项

如果加入 Caddy、Nginx 或 Traefik，需要针对 Curated 的实际流量做配置，而不能只套普通 CRUD 应用模板：

- `/api/events`：关闭响应 buffering，允许长连接，避免 heartbeat 被缓存。
- `/api/import/movies` 与 `/api/import/movies/uploads/*`：允许大请求体，关闭请求 buffering，增加读写超时。
- `/api/library/movies/*/stream`：保留 `Range` / `If-Range` / `Content-Range` 行为，不做整段代理缓存。
- `/api/playback/sessions/*/hls/*`：允许连续小分片请求，避免过度缓存陈旧 playlist。
- 浏览器认证依赖 cookie，跨域拆分前后端会显著增加 CORS / SameSite 复杂度；推荐同域同源。
- Curated 当前以根路径和 `/api` 为约定，第一版不要部署到复杂的 URL 子路径前缀。

## 9. 单实例、备份与升级

### 9.1 为什么只能单实例

除 SQLite 外，以下状态也是进程内的：

- 当前任务管理器状态。
- Connected Clients 列表。
- 断点上传 session 状态。
- 活跃与最近 HLS session。
- SSE 事件订阅。

因此：

- `replicas` 必须为 1。
- 不要让两个容器共享同一 `/data`。
- 第一阶段不做 Kubernetes Deployment 横向扩容。
- 更新采用 stop → backup → replace image → start，而不是两个版本同时滚动。

### 9.2 备份建议

第一版至少提供一份运维文档和脚本，覆盖：

- 停止容器或进入应用静默写入窗口。
- 备份 `/data/config/`。
- 通过 SQLite backup API、`VACUUM INTO` 或停机复制备份 `/data/data/curated.db`。
- 可选备份 `/data/cache/`；通常可重建。
- 不备份日志或只做短期保留。
- 升级失败时恢复旧镜像和对应数据库备份。

不要在数据库持续写入时把普通文件复制当成唯一可靠备份策略。

### 9.3 更新机制

容器版的更新模型应是：

```text
镜像 registry 发布新 tag / digest
  -> 用户或 Watchtower / NAS UI 拉取
  -> 停止旧容器
  -> 备份数据库
  -> 启动新镜像并执行 migration
```

应用内 Windows installer 下载 / 执行入口在容器 runtime 应报告 `unsupported`，不能下载 `.exe` 到 Linux 数据卷后再尝试执行。

## 10. Linux Intel 核显转码方案

### 10.1 当前代码事实

当前已经具备：

- FFmpeg / ffprobe 媒体探测。
- 浏览器友好媒体 direct play。
- 满足条件时 HLS remux：`-c:v copy -c:a copy`。
- HLS 软件转码：`libx264 -preset veryfast -crf 17`。
- profile 顺序尝试：一个 profile 启动失败后会清理 session 目录并尝试下一个 profile。
- session 诊断已能记录最终成功的 `ProfileName`，并有 active / recent session API。

但 Intel Linux 支持仍有四个明确缺口：

1. 配置和前端接受 `qsv`，但 `buildTranscodeProfiles` 只在 Windows 生成 `h264_qsv`；Linux 选择 `qsv` 最终仍只有 `libx264`。
2. `vaapi` 尚未进入后端 normalize、前端类型或设置页选项。
3. `hardwareDecode=true` 当前把共享 `inputPrefix` 改为 `-hwaccel auto`，该前缀随后被 remux、硬件转码和软件 fallback 共用。不同 profile 对 frame format / device 的要求不同，不应共享这一套输入参数。
4. 当前没有检查 render node、文件权限、FFmpeg encoder、Intel 用户态驱动或实际设备初始化。

因此，“Compose 增加 `/dev/dri`”只是宿主机接入步骤，不等于 Curated 已经支持 Intel 转码。

### 10.2 推荐的后端能力模型

建议新增独立的 playback capability provider，不要让设置页直接猜运行环境。它负责：

- 解析最终使用的 FFmpeg 路径和版本。
- 枚举 `ffmpeg -hwaccels`、`-encoders`、`-decoders`。
- 枚举 Linux DRM render node，默认候选为 `/dev/dri/renderD*`，但不写死只有 `renderD128`。
- 检查当前进程对设备节点的打开权限。
- 在受控超时内执行 QSV / VAAPI 短时主动编码测试。
- 缓存最近一次探测结果，并允许从设置页人工重新检测。
- 将失败分成稳定的错误码，而不是只返回 FFmpeg stderr 文本。

建议增加受 PIN / 管理认证保护的接口：

```text
GET  /api/system/capabilities
POST /api/system/capabilities/playback/test
```

响应至少包含：

```json
{
  "runtime": "container",
  "os": "linux",
  "arch": "amd64",
  "ffmpeg": {
    "available": true,
    "version": "...",
    "path": "ffmpeg"
  },
  "videoAcceleration": {
    "configuredDevice": "/dev/dri/renderD128",
    "deviceAccessible": true,
    "qsv": { "available": true, "testedAt": "...", "errorCode": "" },
    "vaapi": { "available": true, "testedAt": "...", "errorCode": "" },
    "selectedEncoder": "qsv",
    "softwareFallback": true
  }
}
```

不要把完整宿主机文件系统、任意命令行或未脱敏的 stderr 暴露给普通客户端。详细诊断进入后端日志；API 只返回受控字段和短消息。

建议的错误分类：

- `GPU_DEVICE_NOT_FOUND`
- `GPU_DEVICE_PERMISSION_DENIED`
- `FFMPEG_NOT_FOUND`
- `FFMPEG_QSV_NOT_BUILT`
- `FFMPEG_VAAPI_NOT_BUILT`
- `INTEL_DRIVER_INIT_FAILED`
- `GPU_ENCODER_TEST_FAILED`
- `GPU_CODEC_UNSUPPORTED`
- `GPU_BUSY`

### 10.3 配置契约调整

保留现有配置以降低迁移成本，但补充以下能力：

| 字段 | 状态 | 建议语义 |
|---|---|---|
| `hardwareDecode` | 已有 | 是否允许对应硬件 profile 使用硬件解码；不再作为“是否生成硬件 encoder”的唯一开关 |
| `hardwareEncoder` | 已有，需扩展 | `auto / qsv / vaapi / software / ...`；Linux Intel 默认 `auto` |
| `hardwareDevice` | 新增 | Linux DRM render node，例如 `/dev/dri/renderD128`；空值时自动探测 |
| `hardwareFallback` | 新增 | 硬件启动失败时是否允许软件回退，容器默认 `true` |
| `maxConcurrentTranscodes` | 新增 | 限制需要重新编码的并发数；初始默认 `1`，remux 是否计入应单独定义 |

需要同步修改：

- `backend/internal/config.Config` 和 `library-config.cfg` merge / write。
- `config.NormalizeHardwareEncoderPreference`。
- playback manager `Config`。
- HTTP contracts 与 `src/api/types.ts`。
- `playback-settings-normalize.ts`。
- 设置页 encoder 选项和运行时 capability 展示。
- Mock adapter、文档和中英文文案。
- 配置、profile、回退和 API 单元测试。

UI 不应在 Linux 容器里继续展示 AMF / VideoToolbox 等不相关选项作为普通推荐。更好的呈现是：

- `自动（推荐）`
- `Intel Quick Sync（已检测 / 不可用）`
- `VA-API（已检测 / 不可用）`
- `软件 libx264`

其他平台选项只在对应 runtime capability 中显示。

### 10.4 按 profile 独立生成 FFmpeg 参数

当前共享 `inputPrefix` 的结构应拆开。推荐 profile 顺序：

```text
direct play（不启动 FFmpeg）
  -> remux_copy（不启用 hwaccel）
  -> h264_qsv_linux（仅 capability 测试成功时加入）
  -> h264_vaapi（仅 capability 测试成功时加入）
  -> libx264（无 hwaccel 的可靠软件回退）
```

实现边界：

- **remux**：不需要硬件解码，不能因为 `hardwareDecode=true` 自动加入 `-hwaccel auto`。
- **QSV**：显式绑定选定 render node，使用 QSV 能接受的像素格式和设备初始化方式；参数需按镜像内 FFmpeg 主版本验证。
- **VAAPI**：显式绑定 render node，正确处理 `format=nv12` / `hwupload` 或硬件 frame 流；不能直接复用软件 `-pix_fmt yuv420p` 参数块。
- **libx264**：不带任何 QSV / VAAPI 设备参数，保证驱动损坏或权限变化时仍可独立工作。
- **音频**：当前继续转 AAC 双声道；视频硬件后端不应改变 HLS 音频兼容目标。
- **seek**：保留现有快速 seek + 精确 seek 语义，但要分别验证 QSV / VAAPI 在 seek 后首片段和时间轴上的行为。
- **HLS readiness**：继续复用当前 playlist、首分片、第二分片检查，硬件 profile 只有通过同一 readiness 标准才算成功。

不建议在计划阶段永久固化一条未经目标机器验证的 FFmpeg 命令。实现时应把 QSV / VAAPI 参数生成器独立成纯函数，以目标镜像的 FFmpeg 版本和真实片源测试确定参数，并用单元测试固定最终 argv。

### 10.5 回退、熔断与并发

现有顺序回退循环可以复用，但需要加强：

- capability 已判定不可用的 profile 不进入候选列表，避免每次播放都等待超时。
- 运行中首次硬件失败时记录 profile attempt、稳定错误码和 stderr 摘要，然后尝试另一个硬件后端或软件。
- 连续多次相同驱动初始化失败后，对该 profile 做进程内短期熔断；用户手工重新检测或冷却到期后恢复。
- `lastSuccessfulProfile` 只能在同一设备和 capability revision 下复用；设备、FFmpeg、配置变化后清空。
- recent playback session 增加 `attempts[]`，记录尝试顺序、耗时和失败分类，便于判断“最终用了软件”是配置选择还是硬件故障。
- 增加全局 transcode semaphore，初始只允许一个真正的视频重编码 session；direct play 不占用，纯 remux 可单独定义更高上限。
- 容器收到 SIGTERM 时停止 FFmpeg，释放 semaphore，并在 `stop_grace_period` 内完成清理。

`hardwareFallback=false` 只适合调试或严格要求 GPU 的环境：硬件不可用时直接返回清晰错误，不能静默使用 CPU。生产家庭环境默认应为 `true`。

### 10.6 Intel 主机主动测试

能力列表只能说明 FFmpeg 编译时支持某个名字，主动测试才说明整条设备链可用。建议分两层：

1. **容器外预检**：用 `vainfo` 确认宿主机内核和用户态驱动基本正常。
2. **容器内测试**：在映射设备、指定非 root 用户和附加组后，运行 FFmpeg 生成的短视频或短时真实样本编码，确认输出非空、进程成功并使用预期 encoder。

测试动作必须：

- 最长数秒，受 context timeout 控制。
- 输出到 `/data/cache/diagnostics` 或临时目录，完成后删除。
- 不使用用户私人影片作为默认诊断素材；可用 FFmpeg test source。
- 同时测试设备初始化和 H.264 encode，不只运行 `ffmpeg -encoders`。
- 把完整命令与 stderr 写入受控日志，并对路径 / 凭据脱敏。

驱动或镜像升级后应重新运行。启动时可以做轻量被动探测，主动编码测试可在首次设置、用户点击“检测硬件加速”或 capability cache 失效时执行，避免每次容器重启都无条件启动 GPU 工作负载。

### 10.7 性能验收矩阵

至少准备以下合法测试素材：

| 样本 | 期望路径 | 重点指标 |
|---|---|---|
| 1080p H.264 + AAC MP4 | direct play | 不启动 FFmpeg、Range seek 正常 |
| 1080p H.264 + AAC MKV | remux | 使用 `remux_copy`、低 CPU、首帧和 seek 时间 |
| 1080p HEVC 8-bit | Intel transcode | QSV / VAAPI H.264 输出、GPU busy、CPU、实时速度 |
| 1080p HEVC 10-bit | Intel transcode 或明确不支持 | 按 iGPU 代际验证，不允许错误宣称 |
| 4K HEVC 样本 | 能力测试，不作为所有设备硬门槛 | 温度、GPU/CPU、实时速度、内存、首帧 |
| 不支持的 codec / profile | 软件回退或清晰失败 | 回退耗时、错误分类、UI 提示 |

每个样本记录：

- descriptor decision 与最终 `transcodeProfile`。
- 首个 playlist、首分片、可播放首帧耗时。
- seek 后恢复耗时。
- FFmpeg 实时速度。
- 宿主机 CPU、容器 CPU / 内存。
- `intel_gpu_top` 或等价工具显示的 Video / VideoEnhance 利用率。
- 连续播放 30 分钟的稳定性、温度和 session 清理。

性能门槛应在知道具体 Intel CPU 型号后确定。计划阶段可以先以“单路 1080p HEVC → H.264 必须持续高于实时速度，且确实产生 GPU video engine 利用率”为 beta 基线，而不是预先承诺任意 Intel 核显都能做 4K 多路转码。

### 10.8 驱动异常时的运维行为

需要覆盖以下真实故障：

- `/dev/dri` 未映射。
- 映射了错误 render node。
- 宿主机 render GID 变化。
- 容器内缺 Intel 用户态驱动。
- FFmpeg 更新后 QSV / VAAPI 参数不兼容。
- iGPU 被 BIOS 禁用或宿主机内核没有创建设备。
- iGPU 支持解码但不支持目标编码格式。
- GPU reset、设备忙或长时间转码失败。

这些故障不能导致整个 Curated 启动失败。默认行为是：应用健康、direct play / 扫描可用、设置页明确显示硬件不可用、HLS 在允许时回退软件；只有用户选择 `hardwareFallback=false` 时，相关转码请求才快速失败。

## 11. Linux 容器化开发环境

### 11.1 生产容器与开发容器必须分开

后续“在 Linux 主机的容器内工作”应解释为使用专门的 dev container，而不是 `docker exec` 进入生产容器安装 Go、Node 或修改源码。

| 维度 | 生产容器 | 开发容器 |
|---|---|---|
| 源码 | 不挂载，镜像内只有构建产物 | 仓库 bind mount 到 `/workspace` |
| 工具链 | 无 Node / Go 编译器 | Go 1.25.4、Node、pnpm、Git、FFmpeg、调试工具 |
| 数据 | `/srv/curated/data` 正式数据 | 独立 named volume，禁止复用生产 SQLite |
| 媒体 | 正式 `/mnt/media`，按功能读写 | 优先测试素材；挂正式媒体时默认只读 |
| GPU | 只映射运行所需 render node | 映射同一 render node 并附带 `vainfo` / FFmpeg 诊断 |
| 进程 | 固定 Curated server entrypoint | 开发者在两个终端分别运行 Go 与 Vite，或使用可替换的开发编排 |
| 安全 | 最小权限、无 Docker socket | 仍不默认挂 Docker socket；需要构建镜像时从宿主机执行 Docker |

这样可以保证生产镜像仍然可复现、可回滚，开发工具升级也不会污染正式服务。

### 11.2 建议新增的开发资产

```text
.devcontainer/
  devcontainer.json          # 可选：VS Code /兼容编辑器入口
  Dockerfile                 # 或复用根 Dockerfile 的 dev target
compose.yaml                 # 生产服务
compose.dev.yaml             # 开发覆盖
docker/
  app.json                   # 生产配置示例
  app.dev.json               # 开发配置示例
  entrypoint.sh              # 仅确有初始化需求时添加
```

如果希望减少重复，更推荐一个多 target Dockerfile：

- `frontend-builder`
- `backend-builder`
- `runtime`
- `gpu-debug`
- `dev`

`dev` target 应包含：

- Go 1.25.4 和 `gopls` 等必要工具，版本固定。
- 满足 Vite 8 的 Node 版本、Corepack 和 pnpm 10。
- FFmpeg、ffprobe、`vainfo` 及 Intel 用户态运行库。
- Git、CA、curl 和最小 shell 工具。
- 与宿主机用户 UID/GID 对齐的非 root 用户。

不建议无边界地装入所有全局 npm / Go 工具；工具清单和版本应进 Dockerfile，避免开发容器变成不可复现的个人工作站。

### 11.3 开发 Compose 基线

概念形态：

```yaml
services:
  curated-dev:
    build:
      context: .
      target: dev
    working_dir: /workspace
    user: "${PUID:-1000}:${PGID:-1000}"
    group_add:
      - "${RENDER_GID}"
    environment:
      CURATED_DATA_DIR: /data
      VITE_USE_WEB_API: "true"
      TZ: Asia/Shanghai
    command: ["sleep", "infinity"]
    ports:
      - "5173:5173"
      - "8080:8080"
    devices:
      - "/dev/dri/renderD128:/dev/dri/renderD128"
    volumes:
      - .:/workspace
      - curated-dev-node-modules:/workspace/node_modules
      - curated-dev-pnpm-store:/pnpm/store
      - curated-dev-go-build:/home/dev/.cache/go-build
      - curated-dev-go-mod:/go/pkg/mod
      - curated-dev-data:/data
      - /mnt/media-test:/media:ro

volumes:
  curated-dev-node-modules:
  curated-dev-pnpm-store:
  curated-dev-go-build:
  curated-dev-go-mod:
  curated-dev-data:
```

说明：

- Go build cache、Go module cache、pnpm store 和 `node_modules` 使用命名卷，不写进仓库；符合现有构建测试规则对仓库内缓存的限制。
- 测试素材优先使用独立 `/mnt/media-test`。如果必须调试写入、整理或导入，再显式改为受控读写挂载。
- Linux 宿主机上的原生 bind mount 通常能提供可靠的 Vite 文件监听，不需要默认启用轮询。
- Vite 已监听 `0.0.0.0:5173`。浏览器通过 Linux 主机 IP 访问时，前端可继续走 Vite 的同源 `/api` proxy 到同容器 `127.0.0.1:8080`。
- 不建议一开始引入复杂 supervisor。开发者可在两个 dev container terminal 中分别启动前后端；若后续需要一键启动，再增加明确的 dev script。

### 11.4 开发态数据目录缺口

当前 `datapaths_dev.go` 的 `curatedDataRoot()` 固定返回空字符串，所以 `CURATED_DATA_DIR=/data` 在普通 `go run` 下没有效果；`DefaultLibrarySettingsPath()` 仍会回到仓库根 `config/library-config.cfg`。

建议调整为：

- 非 release 构建在**显式设置** `CURATED_DATA_DIR` 时也使用该路径。
- 未设置时继续保留当前 repo-relative 默认行为，不破坏 Windows 本地开发。
- 或新增更细的 `CURATED_LIBRARY_SETTINGS_PATH`，但应避免数据库、缓存、日志和业务设置各有一套互相冲突的覆盖方式。

推荐优先让 `CURATED_DATA_DIR` 跨 build tag 保持一致：

```text
dev，无环境变量   -> 当前 backend/runtime + repo config 行为
dev，有环境变量   -> /data/config、/data/data、/data/cache、/data/logs
release            -> 保持当前 /data 或 OS 用户数据目录行为
```

这样 dev container 可以完全隔离本机 `config/library-config.cfg`、开发 SQLite 和日志。

### 11.5 容器内标准开发流程

首次进入 dev container：

```bash
pnpm install --frozen-lockfile
```

终端 1：

```bash
cd /workspace/backend
go run ./cmd/curated -mode http -config /data/config/app.dev.json
```

终端 2：

```bash
cd /workspace
pnpm dev
```

检查与测试继续遵循仓库统一命令：

```bash
pnpm typecheck
pnpm lint
pnpm test
pnpm build

cd /workspace/backend
go test ./...
```

GPU 调试在同一个 dev container 中执行 `vainfo`、FFmpeg capability 和主动编码测试，确保开发时看到的用户态驱动与生产 runtime target 同源。生产镜像最终仍要单独 build 并做 smoke test，不能因为 dev container 可用就跳过生产镜像验证。

### 11.6 开发数据与密钥安全

- dev container 使用独立 SQLite，绝不挂生产 `/srv/curated/data`。
- 开发代理地址、PIN、GitHub token 等通过未跟踪的 `.env.dev.local`、Docker secret 或宿主机环境注入，不写进镜像和 Compose 示例。
- 不把宿主机 Docker socket 默认挂入开发容器；确需容器内构建时，单独评估 rootless BuildKit / 远程 builder，而不是直接给开发容器宿主机 root 等价权限。
- 调试真实媒体时优先只读；需要测试删除 / 整理时使用副本。
- `.devcontainer` 只描述环境，不改变仓库规定的 pnpm / Go 工作目录和测试命令。

## 12. 建议实施阶段

### Phase 0：目标 Linux 主机盘点与 Intel GPU 透传 spike

目标：在改业务代码前确认目标硬件、驱动、设备和镜像 FFmpeg 组合成立。

- 记录发行版、内核、CPU / iGPU 型号、`i915` / `xe`、DRM 节点和组 GID。
- 在宿主机运行 `vainfo`，确认驱动基本正常。
- 构建临时 `gpu-debug` target，映射 render node 并以目标非 root 用户运行。
- 验证容器内 `vainfo`、FFmpeg hwaccel / encoder 列表和短时 H.264 主动编码。
- 记录 QSV、VAAPI 哪一条链在该机器上真实成功，以及所需用户态包版本。
- 用一份真实但可公开 / 合法的 HEVC 测试素材完成一次手工 FFmpeg 转码。

退出标准：至少 QSV 或 VAAPI 有一条在目标主机与目标容器中完成 H.264 编码；如果都失败，先解决宿主机 / 驱动 / 镜像问题，不进入 Curated profile 开发。

工作量粗估：0.5–1.5 人日，受 Linux 主机驱动现状影响。

### Phase 1：基础生产容器 PoC

目标：证明不依赖 Electron 的 Curated server 镜像能长期启动。

- 新增多 target `Dockerfile`、`.dockerignore` 与生产 Compose。
- 显式以 `VITE_USE_WEB_API=true` 构建前端。
- 构建 `CGO_ENABLED=0` 的 Linux release 后端。
- runtime 安装 CA、FFmpeg、时区和 Phase 0 确认的 Intel 用户态运行库。
- `/data`、`/media`、render node 和附加组挂载。
- 固定 `-mode http`，健康检查 `GET /api/health`。
- 使用全新数据库验证持久化、扫描、direct play、remux 和软件 fallback。
- 首次只验证 `linux/amd64`；Intel 核显目标通常是该架构，ARM64 镜像不与 Intel GPU 目标强绑定。

工作量粗估：1–2 人日，不含 registry CI。

### Phase 2：Curated Intel QSV / VAAPI 核心实现

目标：Curated 自己能够探测、选择、使用并诊断 Intel 核显，而不是依赖手工 FFmpeg 命令。

- 新增 playback capability provider 与受保护的检测 API。
- 扩展 player 配置、DTO、前端类型和设置页；加入 `vaapi`、设备、软件回退和并发限制。
- 重构 profile 参数生成，移除跨 remux / hardware / software 共用的 `-hwaccel auto` 前缀。
- 实现 Linux QSV 与 VAAPI H.264 profile，保留 `libx264` 独立回退。
- 复用现有 profile 启动 fallback，增加 attempts、错误分类、熔断和 capability revision。
- 增加全局转码并发控制。
- 用 Phase 0 的目标主机执行单元、集成、seek、HLS readiness 和长时播放测试。
- Settings 显示实际设备、FFmpeg 版本、已验证后端和当前 session profile。

退出标准：单路目标样本持续高于实时速度，GPU video engine 有明确利用率，硬件故障可快速回退软件且不影响应用健康。

工作量粗估：4–8 人日；FFmpeg 参数与 Intel 代际兼容问题可能扩大测试时间。

### Phase 3：Linux 容器化开发环境

目标：日常前后端开发、测试和 Intel GPU 调试可在 Linux dev container 内完成。

- 新增 `dev` target、`compose.dev.yaml` 和可选 `.devcontainer/devcontainer.json`。
- 让显式 `CURATED_DATA_DIR` 在非 release 构建中也生效。
- 为 pnpm store、`node_modules`、Go build cache、Go module cache 和开发数据建立命名卷。
- 保持源码 bind mount，暴露 5173 / 8080，验证 Vite HMR。
- 映射 render node，确保 dev / gpu-debug / runtime 使用同一类用户态驱动基线。
- 固化容器内 `pnpm typecheck / lint / test / build` 与 `go test ./...`。
- 验证开发容器不接触生产 SQLite，真实媒体默认只读。

工作量粗估：2–4 人日，取决于编辑器集成和 UID/GID 兼容范围。

### Phase 4：可信 LAN beta

目标：让 Linux 主机作为家庭服务器稳定运行 Curated。

- 定义 `container/server` runtime capability。
- 容器模式隐藏 / 禁用 reveal、后端 native player、launch-at-login、Windows installer 更新。
- 明确资料库路径是容器路径，并改进设置页提示。
- 验证目标媒体文件系统上的扫描、写入、删除、导入和 fsnotify；不可靠时启用周期扫描。
- 增加 SQLite 备份 / 恢复和镜像回滚流程。
- 默认要求 PIN，限制监听 / 发布范围。
- 验证 SSE、Range、HLS、长上传、Intel 转码和正常 SIGTERM 停机。
- 连续运行和转码压力测试，记录温度、CPU、GPU、内存和 HLS session 清理。

工作量粗估：3–6 人日。

### Phase 5：Windows 数据正式迁移

目标：现有桌面用户可保留数据库状态迁入 Linux 容器。

- 实现路径迁移 CLI / 事务工具与 dry-run。
- 覆盖全部路径字段及 Windows 配置清理。
- 增加迁移前备份、迁移报告和回滚流程。
- 使用真实副本验证海报、预览、播放、萃取帧和用户数据。
- 禁止桌面实例与容器实例同时使用同一数据副本。

工作量粗估：3–5 人日；多资料库、多盘符和历史脏数据需要额外测试。

### Phase 6：镜像发布与远程访问加固

目标：可以在 GHCR 等 registry 发布可追踪 beta / stable 镜像。

- CI 构建目标架构 manifest；Intel GPU 集成测试仍在自托管 Linux runner 上执行，不能由普通无 GPU runner 替代。
- 镜像 tag 同时提供产品版本和不可变 commit SHA / digest。
- 固定 FFmpeg / Intel 用户态驱动，生成 SBOM、漏洞扫描和镜像签名。
- CORS allowlist、Secure cookie、可信代理、登录速率限制。
- 提供 Caddy / Nginx 示例并验证 SSE、上传、Range。
- 明确数据库升级与镜像回滚兼容策略。

工作量粗估：3–6 人日，不含独立安全审计。

## 13. 上线验收清单

### 13.1 构建与镜像

- [ ] 干净构建上下文中前端确定使用 Web API，不依赖开发机 `.env`。
- [ ] `linux/amd64` 与目标架构二进制可启动。
- [ ] 镜像内有可执行的 `ffmpeg` 和 `ffprobe`。
- [ ] `/app/frontend-dist/index.html` 存在，任意前端路由可 SPA fallback。
- [ ] 镜像不含 Node、Electron、源媒体、本机配置和凭据。
- [ ] 版本、commit 和构建时间可由 `/api/health` 追踪。

### 13.2 运行与存储

- [ ] `CURATED_DATA_DIR=/data` 时数据库、配置、缓存和日志都落在持久卷。
- [ ] 非 root 用户可写 `/data` 与需要写入的资料库。
- [ ] `/data` 为空时可首次启动并自动 migration。
- [ ] 重建容器后配置、数据库和用户状态仍存在。
- [ ] 同一 `/data` 不会被第二实例并发打开。
- [ ] SIGTERM 后服务停止，无残留 FFmpeg 子进程，数据库可再次打开。

### 13.3 业务功能

- [ ] 添加 `/media/...` 资料库并完成扫描。
- [ ] 封面、头像、预览图可显示。
- [ ] MP4 Range 直传可 seek。
- [ ] MKV 等目标样本能 remux 或软件 HLS 播放。
- [ ] SSE 任务进度可持续接收。
- [ ] 大文件断点上传可完成；重启导致的 session 丢失行为有清晰提示。
- [ ] 导入、整理、删除 / 恢复只作用于预期挂载目录。
- [ ] 网络存储在关闭 fsnotify 时仍可通过手动 / 周期扫描工作。

### 13.4 Intel 核显

- [ ] 目标 Linux 主机记录了 CPU / iGPU、内核、驱动、DRM 节点和 render/video GID。
- [ ] 非 root 生产用户能打开选定 render node，无需 privileged。
- [ ] 容器内 `vainfo` 能针对 render node 返回能力。
- [ ] 容器内 FFmpeg 构建包含计划使用的 QSV / VAAPI hwaccel、decoder 和 H.264 encoder。
- [ ] Curated 主动测试真实通过，而不是只根据 encoder 列表报告可用。
- [ ] 1080p HEVC 目标样本最终使用 `h264_qsv` 或 `h264_vaapi`，GPU video engine 有可观测利用率。
- [ ] remux profile 不因开启硬件加速而加入不必要的 hwaccel 参数。
- [ ] libx264 profile 不携带 QSV / VAAPI 设备参数。
- [ ] 设备未映射、组权限错误、驱动缺失和不支持 codec 时均产生稳定错误码并按设置回退。
- [ ] 最大并发转码限制有效，超限行为明确，不会把主机拖入失控负载。
- [ ] seek、首分片、长时间播放、容器停止和 FFmpeg 子进程清理通过验收。

### 13.5 容器化开发

- [ ] dev container 使用独立数据卷，无法误开生产 SQLite。
- [ ] pnpm / Go 缓存均在命名卷或容器用户目录，不写入仓库。
- [ ] Vite 从另一台 LAN 客户端通过 Linux 主机 `5173` 可访问，HMR 正常。
- [ ] Go API 在 `8080` 可用，Vite `/api` proxy、SSE、上传和播放调试正常。
- [ ] 显式 `CURATED_DATA_DIR=/data` 在 `go run` 下生效，设置写回不污染仓库根 `config/`。
- [ ] dev container 中可执行完整前端检查和 `go test ./...`。
- [ ] dev / gpu-debug / runtime 的 FFmpeg 与 Intel 驱动基线可追踪，避免“开发可用、生产不可用”。
- [ ] 默认不挂 Docker socket，真实生产媒体默认不以读写方式进入开发容器。

### 13.6 安全与运维

- [ ] 未配置 PIN 的实例不会暴露到不可信网络。
- [ ] 远程入口使用 HTTPS，并有外层身份控制或明确的受信任网络边界。
- [ ] CORS、cookie、可信代理与速率限制达到选定发布级别。
- [ ] Windows updater、reveal、后端 native player 在容器模式不显示为可用。
- [ ] 数据库有可恢复的备份，并实际演练过恢复。
- [ ] 反向代理下 SSE、Range、HLS 和上传均通过验收。

## 14. Go / No-Go 建议

### 建议 Go

- 立项做单容器 PoC。
- 首发定位为 Linux 主机上的单节点、单用户 / 家庭、可信 LAN server mode。
- 保持 Go 同时托管前端和 API 的单容器形态。
- 先做目标 Linux 主机 Intel GPU 透传 spike，再实现 Curated QSV / VAAPI 能力；Intel 核显不再放到不确定的远期可选项。
- 先支持新数据库、direct play、remux、Intel 单路硬件转码和软件回退，再处理完整 Windows 数据迁移。
- 建立独立 dev container，使后续日常开发能在 Linux 容器内完成，同时保持生产镜像最小化。
- 生产目标优先 `linux/amd64`；已能编译的 `linux/arm64` 可继续纳入 CI，但它不是本次 Intel 核显主线的发布门槛。

### 条件 Go

- 携带现有 Windows 数据迁移：路径迁移工具与备份 / 回滚完成后再开放。
- NAS 网络挂载：针对目标协议验证扫描与 fsnotify；不可靠时默认使用周期扫描。
- Intel HLS：必须在目标 CPU 代际、目标 Linux 内核、目标 FFmpeg / 驱动镜像组合上主动测试。
- 4K、多路并发、HEVC 10-bit 或 AV1：按具体 iGPU 能力单独承诺，不能由 1080p H.264 测试外推。

### 当前 No-Go

- 未启用认证和 TLS 就直接公网开放。
- 多副本共享 SQLite。
- 只映射 `/dev/dri`、没有 Curated profile 和主动测试就宣称 Linux GPU 硬件转码已支持。
- 在没有路径迁移工具的情况下直接复制 Windows SQLite 并让用户手工修库。
- 在容器里继续提供 Windows `.exe` 自动安装或服务器端桌面操作入口。
- 进入生产容器安装编译器、修改源码或把生产 `/data` 当开发数据库。

## 15. 最终判断

Curated 容器化不是一次架构重写。核心运行形态已经非常接近容器友好的单体服务：静态 SPA、Go HTTP、SQLite、外置数据目录和 FFmpeg。针对已经明确的 Linux 主机目标，推荐同时建设三个清晰边界：**不可变的生产 Server runtime、可主动验证 Intel 核显的 GPU capability 层、与生产数据完全隔离的 dev container**。

Intel 核显链路的关键不是 Compose 中多写一行 `/dev/dri`，而是把宿主机驱动、容器用户态驱动、设备权限、FFmpeg 构建、Curated profile、真实编码测试和软件回退连成一条可诊断链。现有 playback manager 已有 profile 顺序回退和 session readiness 检查，这是很好的复用基础；需要重构的是 profile 参数和能力选择，而不是推翻播放器架构。

如果目标是先在这台 Linux 主机上运行，建议执行顺序是：**主机 / GPU spike → 基础生产镜像 → Curated Intel profile → dev container → LAN beta → Windows 数据迁移与镜像发布**。这样最早暴露硬件与驱动风险，也能让后续开发逐步迁入同一台 Linux 主机，而不污染正式服务。

## 16. 目标 Linux 主机调试容器准备

### 16.1 是否现在就需要创建容器

需要目标主机上的真实容器测试，但不建议一开始就创建正式 Curated 容器。推荐按以下顺序准备：

1. 先采集宿主机发行版、内核、CPU / iGPU、DRM 设备、设备组和 Docker 版本信息。
2. 确认宿主机能够直接访问 Intel 核显；如果宿主机上的 `vainfo` 已经失败，先修复内核或宿主机驱动，不进入容器排查。
3. 创建一个不挂生产数据的临时 `gpu-debug` 容器，只验证设备透传、用户态驱动、FFmpeg QSV / VAAPI 和一次真实硬件编码。
4. `gpu-debug` 验证通过后，再据此固定 Curated 的 runtime / dev 镜像依赖和 Compose 权限模型。

因此，当前最有价值的准备不是先启动一个长期运行、拥有大量权限的通用容器，而是准备宿主机信息和一个可以随时重建、用途单一的 GPU 验证容器。

### 16.2 远程协作方式

仅仅在目标主机上启动容器，并不会自动让 Codex 获得该主机或容器的访问能力。可选协作方式按复杂度从低到高排列如下：

- 用户在目标主机运行本节命令，把输出返回到当前任务；适合完成第一轮主机与 GPU 盘点。
- 在目标 Linux 主机克隆本仓库，并将 Codex 工作区 / 终端连接到该工作目录；适合后续反复构建、修改和验证。
- 使用 SSH 连接目标主机；应预先创建受限的非 root 开发用户，使用密钥和主机别名，不要在对话中发送密码或私钥。该用户只有确有需要时才加入 `docker` 组，因为 Docker daemon 访问通常近似宿主机 root 权限。

无论采用哪种方式，调试容器都不需要 SSH server；调试入口应使用宿主机上的 `docker compose exec gpu-debug bash` 或等价命令。

### 16.3 第一轮宿主机预检

在创建镜像前，先在目标 Linux 主机执行并保存以下输出：

```bash
uname -a
cat /etc/os-release
lscpu
lspci -nn | grep -Ei 'vga|display'
ls -l /dev/dri
getent group render
getent group video
docker version
docker compose version
```

如果宿主机已经安装 `vainfo` 和 FFmpeg，再执行：

```bash
vainfo --display drm --device /dev/dri/renderD128
ffmpeg -hide_banner -hwaccels
ffmpeg -hide_banner -encoders | grep -E 'qsv|vaapi'
ffmpeg -hide_banner -decoders | grep -E 'qsv|vaapi'
```

注意事项：

- `/dev/dri/renderD128` 是常见 render node，但不是应写死的唯一值；必须先查看实际的 `/dev/dri/renderD*`。
- 新一代 Intel 平台通常使用 `iHD` / Intel media driver，老平台可能需要 i965；镜像包不能在不知道 CPU 代际和 Linux 发行版时提前写死。
- `ffmpeg -encoders` 中出现 `h264_qsv` 或 `h264_vaapi` 只代表 FFmpeg 编译时包含对应接口，不代表当前设备、驱动和权限链路真的可用。
- 如果系统使用 `xe` 而不是 `i915`，需要记录实际内核驱动和内核版本，不能套用只针对 `i915` 的排障结论。

第一轮需要回传的最小信息是上述命令输出；如果命令不存在，也应保留 `command not found` 信息，这有助于确定宿主机应补包还是只在调试镜像内补包。

### 16.4 临时调试目录与 Compose 基线

建议在目标主机建立独立目录，不与生产目录和生产数据卷混用：

```text
curated-gpu-debug/
├── Dockerfile.gpu-debug
├── compose.gpu-debug.yaml
└── samples/
```

`samples/` 只放可公开或合法使用的短测试素材。初始 Compose 可以采用以下安全基线；实际 render node 和 GID 由宿主机预检结果填写：

```yaml
services:
  gpu-debug:
    build:
      context: .
      dockerfile: Dockerfile.gpu-debug
    user: "${PUID:-1000}:${PGID:-1000}"
    group_add:
      - "${RENDER_GID}"
    devices:
      - "${DRI_RENDER_NODE:-/dev/dri/renderD128}:${DRI_RENDER_NODE:-/dev/dri/renderD128}"
    volumes:
      - ./samples:/samples:ro
    tmpfs:
      - /work:mode=1777,size=2g
    command: ["sleep", "infinity"]
    stdin_open: true
    tty: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
```

目标主机可以按实际设备生成未提交版本的 `.env`：

```bash
id -u
id -g
stat -c '%g' /dev/dri/renderD128
```

对应变量示例：

```dotenv
PUID=1000
PGID=1000
RENDER_GID=993
DRI_RENDER_NODE=/dev/dri/renderD128
```

这里应填数字 GID，而不是假定所有发行版中的 `render` 组都使用同一编号。多数转码只需要 render node；只有验证结果表明确实需要访问 `/dev/dri/card*` 时，才增加该设备和宿主机 `video` 组的数字 GID。

### 16.5 `gpu-debug` 镜像内容

调试镜像选择与目标生产镜像相同系列的 Debian / Ubuntu 基础镜像，并至少包含：

- CA certificates；
- `ffmpeg` 与 `ffprobe`；
- `vainfo`；
- 与目标 Intel 代际匹配的 VA-API / oneVPL 用户态运行库；
- `bash`、`procps`、`pciutils` 等最小诊断工具；
- 可选的 `intel-gpu-tools`，用于在宿主机或具备所需设备权限的诊断环境中观察 video engine 利用率。

宿主机负责内核 DRM 驱动和设备节点，容器负责 FFmpeg 与 Intel 用户态媒体驱动。容器不需要安装内核模块，也不应尝试在容器里加载或替换宿主机内核驱动。

具体基础镜像 tag、Intel 驱动包名和 FFmpeg 来源，应在收到发行版、CPU 代际和 `vainfo` 输出后确定。调试镜像确定可用后应固定版本或 digest，避免同一 Dockerfile 随软件源更新后产生不可追踪的 GPU 行为差异。

### 16.6 两阶段权限验证

为了快速区分驱动问题与非 root 权限问题，临时容器按两轮验证：

1. **诊断轮**：临时覆盖为容器 root，但仍只映射选定 render node，不使用 `privileged`。验证 `vainfo`、FFmpeg 能力和一次短编码。若这一轮失败，优先排查设备、内核、用户态驱动或 FFmpeg 构建。
2. **正式权限轮**：恢复目标非 root UID / GID 和 `group_add: RENDER_GID`，重复完全相同的测试。若 root 成功、非 root 失败，问题集中在设备节点和附加组权限。

诊断轮可使用 Compose override 或一次性命令设置 `user: "0:0"`，测试结束后立即恢复；不应为了绕过权限问题把 `privileged: true` 留在配置中。

### 16.7 容器内验收步骤

容器启动后至少执行：

```bash
id
ls -l /dev/dri
vainfo --display drm --device /dev/dri/renderD128
ffmpeg -hide_banner -version
ffmpeg -hide_banner -hwaccels
ffmpeg -hide_banner -encoders | grep -E 'qsv|vaapi'
ffmpeg -hide_banner -decoders | grep -E 'qsv|vaapi'
```

然后使用 `samples/` 中的短视频分别执行候选 QSV / VAAPI 硬件编码，并检查：

- FFmpeg 返回码为 0，日志表明实际初始化了预期硬件设备和 encoder；
- 生成文件可被 `ffprobe` 读取，codec、分辨率、帧数和时长符合预期；
- 编码期间 Intel video engine 有利用率，而不是悄悄使用 `libx264`；
- root 与目标非 root 用户得到一致结果；
- 不支持的 codec / pixel format 能产生可识别错误，后续 Curated 可据此回退软件编码。

最终 FFmpeg 主动测试命令应根据容器中的实际 FFmpeg 版本、设备初始化方式、输入编码和 Intel 代际生成，不能仅凭通用模板宣告通过。至少应覆盖 Curated 首期真正需要的一个典型链路，例如 1080p HEVC 输入转 H.264 输出；AV1、10-bit、HDR 和多路并发另行验收。

### 16.8 不应授予或挂载的内容

第一轮 `gpu-debug` 明确禁止：

- `privileged: true`；
- 挂载 `/var/run/docker.sock`；
- 挂载生产 SQLite、生产 `/data` 或现有 Windows 数据副本；
- 把正式媒体库以读写方式挂载；需要真实素材时优先使用独立副本或只读挂载；
- 使用 host PID / network namespace，除非后续存在独立且可解释的诊断需求；
- 把 SSH 私钥、GitHub token、代理密码或其他凭据写进 Dockerfile、镜像层或 Compose 文件；
- 在未来生产容器中继续保留编译器、shell 调试套件和 GPU 诊断工具。

GPU 调试通过后，生产 runtime、开发容器和 `gpu-debug` 应继续保持三个不同 target：生产 runtime 最小化，开发容器承载源码和工具链，`gpu-debug` 专门复现驱动与 FFmpeg 能力。三者共享经验证的 FFmpeg / Intel 用户态驱动基线，而不是共享生产数据或不必要的宿主机权限。

### 16.9 本阶段退出标准

满足以下条件即可开始编写目标主机专用 Dockerfile 和 Curated GPU profile：

- 已记录目标发行版、内核、CPU / iGPU、DRM 驱动、render node 和组 GID；
- 宿主机与容器内 `vainfo` 均可识别 Intel 媒体能力；
- `gpu-debug` 在非 privileged、非 root 模式下能够打开选定 render node；
- QSV 或 VAAPI 至少一条路径完成一次真实 H.264 硬件编码，并有输出校验和 GPU video engine 证据；
- 已记录成功路径所需的基础镜像、FFmpeg 和 Intel 用户态驱动版本；
- 测试没有接触生产数据库，也没有给容器 Docker socket 或等价宿主机管理权限。

### 16.10 目标主机首次盘点结果（2026-07-18）

目标主机首次终端信息已经确认：

| 项目 | 实际结果 | 当前判断 |
|---|---|---|
| 主机 | `jiahui-nas` | 单台目标 Linux 主机 |
| 发行版 | Debian 12（bookworm） | 可选择 Debian 12 系列作为首轮调试镜像基线 |
| 内核 | `6.18.18-trim`，定制 NAS 内核 | 后续需记录 DRM 驱动为 `i915` 还是 `xe`，不能完全按 Debian 官方内核假设 |
| 架构 | `x86_64` | 与 Curated 首期 `linux/amd64` 目标一致 |
| CPU | Intel Core i7-10710U，6 核 12 线程 | Comet Lake-U 平台 |
| iGPU | Intel Comet Lake UHD Graphics，PCI ID `8086:9bca` | 具备验证 QSV / VAAPI 的硬件基础；具体 codec 仍需主动测试 |
| DRM | `card0` 与 `renderD128` 均存在 | 宿主机内核至少已经创建 Intel DRM 设备节点 |
| 设备组 | `render` GID 105，`video` GID 44 | Compose 的 `group_add` 应使用实际数字 GID 105；首轮只映射 `renderD128` |
| Docker | Engine CLI 28.5.2，Compose v2.40.3 | 工具已安装，但当前用户无法连接 daemon |
| FFmpeg | 列出 `qsv`、`vaapi`、`drm` 等 hwaccel，以及多种 QSV / VAAPI encoder | 只证明 FFmpeg build 包含接口，尚未证明硬件会话可初始化 |
| `vainfo` | `Failed to open the given device!` | 结合设备权限，优先怀疑当前 SSH 用户不在 `render` 组；修复权限后才能判断驱动 |
| SSH 用户 | `jiahui.wu` 的 `/home/jiahui.wu` 不存在 | 在容器化开发前必须修复有效 home，供 SSH、Git、Docker 配置和开发缓存使用 |

当前用户执行 `docker version` 时连接 `/var/run/docker.sock` 返回 `permission denied`，因此现在还不能由该账户构建或启动调试容器。首次 `getent group render` 与 `getent group video` 的成员列表为空，但是否存在目录服务、主组或其他补充组情况仍以 `id` 输出为最终依据。

下一轮先执行以下只读检查：

```bash
id
getent passwd jiahui.wu
getent group docker
ls -ld /home /home/jiahui.wu
ls -l /var/run/docker.sock
lspci -k -s 00:02.0
readlink -f /sys/class/drm/renderD128/device/driver
test -r /dev/dri/renderD128 && echo render-readable || echo render-not-readable
test -w /dev/dri/renderD128 && echo render-writable || echo render-not-writable
```

如果该主机的管理方式允许通过标准 Debian 用户管理命令修改账户，并且 `jiahui.wu` 是受信任的容器开发账号，可以由管理员处理：

```bash
sudo usermod -aG render,video jiahui.wu
sudo usermod -aG docker jiahui.wu
```

加入 `docker` 组意味着该用户通常可以借助 Docker 获得近似宿主机 root 的能力，不应给普通或不受信任账号授予该组。若 NAS 系统要求通过管理界面维护用户和组，应使用管理界面，避免系统升级覆盖手工修改。

home 目录应由管理员按 `getent passwd jiahui.wu` 中记录的 home 和该用户实际主组创建；不要在尚未确认用户主组时直接假定组名也叫 `jiahui.wu`。修复组成员关系和 home 后需要完全退出 SSH 并重新登录，再检查：

```bash
id
docker version
vainfo --display drm --device /dev/dri/renderD128
```

只有重登录后的 `id` 包含预期的 `render` 和 `docker` 组、Docker client/server 均能返回版本、宿主机 `vainfo` 能打开 `renderD128` 后，才进入 `gpu-debug` 镜像构建。`video` 组对仅访问 render node 的首轮验证通常不是必需项，但可以为后续确需访问 `card0` 的诊断保留。

FFmpeg 当前列出了 `av1_qsv` / `av1_vaapi`，这不能推导 Comet Lake UHD 实际支持 AV1 硬件编解码。首轮应聚焦 H.264 与 HEVC 输入能力、H.264 输出能力；AV1、VP9、10-bit 和多路并发都必须根据主动测试单独记录。

## 17. fnOS FPK 交付方案

### 17.1 FPK 的定位

FPK 是飞牛 fnOS 的应用安装包格式。官方 `fnpack` 工具可以创建应用项目，并将项目打包为可在 fnOS 安装的 `.fpk` 文件；它既支持 Native 应用，也正式支持 Docker 应用模板。

FPK 不是 Docker 镜像格式，也不是 Debian 的 `deb` 包。一个典型 FPK 包含：

- `manifest`：应用标识、版本、平台、服务端口、启动 / 停止控制和桌面入口；
- `config/privilege`：FPK 本地脚本或原生进程的运行用户；官方明确说明它不负责指定容器内进程身份；
- `config/resource`：共享目录、Docker Compose 项目等 fnOS 资源声明；
- `cmd/`：安装、升级、配置、卸载和 `start` / `stop` / `status` 生命周期脚本；
- `wizard/`：安装、升级、卸载和配置向导；
- `app/ui/` 与图标：fnOS 桌面入口；
- `app/docker/docker-compose.yaml`：Docker 模板中的 Compose 项目。

安装后，fnOS 会在 `/var/apps/{appname}` 建立应用入口，并把应用文件、配置、持久化数据、临时数据和应用 home 分别映射到 `target`、`etc`、`var`、`tmp` 和 `home`。应用应使用 `TRIM_APPDEST`、`TRIM_PKGETC`、`TRIM_PKGVAR`、`TRIM_PKGTMP`、`TRIM_PKGHOME` 等变量，不能硬编码底层 `/vol{n}` 路径。

官方资料：

- [fnpack：创建与构建 `.fpk`](https://developer.fnnas.com/docs/cli/fnpack/)
- [fnOS 应用框架与安装目录](https://developer.fnnas.com/docs/core-concepts/framework/)
- [Docker 应用完整案例](https://developer.fnnas.com/docs/examples/docker/)
- [Manifest 字段](https://developer.fnnas.com/docs/core-concepts/manifest/)
- [应用资源与 Docker 项目](https://developer.fnnas.com/docs/core-concepts/resource/)
- [fnOS 环境变量](https://developer.fnnas.com/docs/core-concepts/environment-variables/)

### 17.2 三种交付形态比较

| 方案 | 运行形态 | 优点 | 主要代价 | 对 Curated 的建议 |
|---|---|---|---|---|
| 手工 Docker Compose | 用户手工拉镜像并运行 Compose | 最适合早期 GPU spike，排障透明 | fnOS 应用中心无完整安装 / 升级 / 桌面集成 | 保留为开发、诊断和紧急恢复入口 |
| Native FPK | FPK 直接携带 Go、前端和可能的 FFmpeg / 驱动用户态文件 | 无容器层，fnOS 生命周期直接管理进程 | FFmpeg / Intel 库打包、动态库兼容、进程权限和升级回滚更复杂 | 不作为首选发布路径 |
| Docker FPK | FPK 声明 Compose，Curated 在容器中运行 | 同时获得镜像可重复性与 fnOS 应用中心集成 | 仍需解决镜像分发、GPU 映射、数据授权和 Compose 兼容 | **推荐的正式交付方向** |

FPK 因此不替代现有容器化工作，而是成为容器镜像外层的 fnOS 安装与管理控制面。

### 17.3 Curated 推荐结构

首期建议新增一个独立 FPK 工程目录，避免把 fnOS 元数据混进通用 Docker runtime：

```text
deploy/fnos/curated/
├── app/
│   ├── docker/
│   │   └── docker-compose.yaml
│   └── ui/
│       ├── config
│       └── images/
├── cmd/
│   ├── main
│   ├── install_init
│   ├── install_callback
│   ├── upgrade_init
│   ├── upgrade_callback
│   ├── uninstall_init
│   └── uninstall_callback
├── config/
│   ├── privilege
│   └── resource
├── wizard/
│   ├── install
│   ├── upgrade
│   ├── uninstall
│   └── config
├── manifest
├── ICON.PNG
└── ICON_256.PNG
```

FPK 骨架可由官方工具创建：

```bash
fnpack create curated --template docker
```

首期 `manifest` 原则：

- `appname=curated`；
- `source=thirdparty`；
- `platform=x86`，因为当前目标主机和 Intel GPU 主线为 `linux/amd64`，不应在 ARM 镜像尚未等价验证时声明 `all`；
- `service_port` 使用安装时确认的端口，首期可从 8080 起；
- `checkport=true`；
- `ctl_stop=true`，让应用中心展示启动、停止和状态；
- `disable_authorization_path=false`，允许用户授权现有媒体目录；
- `os_min_version` 只填写已经实际安装验证过的 fnOS 最低版本。

### 17.4 数据、媒体和临时目录

推荐容器路径映射：

```text
fnOS TRIM_PKGVAR            -> Curated /data
fnOS TRIM_PKGTMP            -> Curated 转码临时目录
fnOS TRIM_DATA_ACCESSIBLE_PATHS -> 用户在 fnOS 授权的现有媒体目录
fnOS data-share             -> 可选的 Curated 导入、导出或备份共享目录
```

其中：

- SQLite、配置和必须跨重启 / 升级保留的状态进入 `${TRIM_PKGVAR}`；
- HLS 分片、转码中间文件等可重建内容进入 `${TRIM_PKGTMP}`；
- 用户现有媒体库不复制进 FPK，也不放入应用私有数据目录；通过 fnOS 授权路径传给容器；
- 只有确实需要让用户在文件管理器看到的导入、导出或备份目录才声明 `data-share`；
- 卸载默认不应静默删除用户媒体；是否保留或删除 Curated 私有数据应由卸载向导明确询问；
- 升级前对 SQLite 做一致性备份，并在镜像启动失败时保留回滚路径。

`TRIM_DATA_ACCESSIBLE_PATHS` 是以冒号分隔的宿主机授权路径列表。将多个动态路径安全转换为 Compose bind mounts 需要在 FPK 安装 / 配置流程中生成受控的环境或 override 文件，并验证每个路径，不能把未经检查的向导输入直接拼进 Shell 或 YAML。

### 17.5 Intel GPU 在 Docker FPK 中的处理

FPK 不会自动让 Curated 获得 Intel 核显能力。Compose 仍需显式映射：

```yaml
devices:
  - "/dev/dri/renderD128:/dev/dri/renderD128"
group_add:
  - "105"
```

其中 `renderD128` 与 GID `105` 来自当前目标主机盘点，只适用于该主机的首轮 PoC。要制作可分发 FPK，应在安装检查中：

1. 枚举 `/dev/dri/renderD*`；
2. 取得目标 render node 的实际数字 GID；
3. 验证设备可用；
4. 生成 Compose 使用的受控配置；
5. 在容器启动后执行 `vainfo` 和 FFmpeg 主动编码健康检查。

不能使用 `privileged: true` 代替检测与权限处理。`config/privilege` 只管理 FPK 本地脚本 / Native 进程，不会替代容器中的 `user`、`group_add` 和 `devices`。

FPK 安装也不会修复宿主机 Intel 用户态驱动或内核问题。当前宿主机 `vainfo` 尚未通过，因此 GPU spike 仍是 FPK PoC 的前置条件。

### 17.6 镜像交付

Docker FPK 可以采用两种镜像交付方式：

1. **Registry 拉取**：Compose 引用固定版本和 digest 的 Curated 镜像。FPK 小、升级清晰，是 beta 首选；缺点是安装时需要访问 registry。
2. **FPK 携带镜像归档**：安装时执行受控的 `docker load`。适合离线环境，但 FPK 很大，升级和磁盘空间管理更复杂；不建议作为首个 PoC。

首期建议发布 `linux/amd64` 镜像到选定 registry，并在 Compose 中固定不可变 digest。FPK 的 `version`、镜像产品版本和镜像 digest 应可追踪。不要使用浮动 `latest` 作为稳定发布依据。

### 17.7 Web 入口与认证

官方 Docker FPK 支持两种入口：

- 通过 `service_port` 暴露宿主机端口并注册 fnOS 桌面入口；
- 通过 fnOS 统一网关，把请求转发到容器创建的 Unix Socket。

Curated 当前 Go HTTP 服务按 TCP 端口运行，因此首个 FPK PoC 使用端口入口更直接。统一网关需要 Curated 新增或适配 Unix Socket 监听，并完整验证 SPA、API、SSE、Range、HLS、上传和 WebSocket（若后续使用），不能只改入口配置。

fnOS 桌面入口不应被视为 Curated 业务认证的替代品。Curated 的 PIN / 会话、可信代理、cookie、CORS 和 LAN 暴露边界仍需按发布范围实现和测试。

### 17.8 开发、调试与发布边界

- 日常源码开发仍使用独立 dev container 或目标 Linux 工作区，不把编译器塞进生产 FPK / runtime 镜像。
- `gpu-debug` 继续作为最小硬件链路诊断工具；FPK 只在该链路通过后接入相同 FFmpeg / Intel 驱动基线。
- 手工 Compose 是 FPK 失败时的重要诊断入口，但不能与 FPK 管理的同名容器、端口和 SQLite 同时运行。
- FPK 生命周期脚本必须幂等；`status` 应检查最能代表 Curated 可用性的容器健康状态，而不只检查容器是否存在。
- FPK 升级应停止旧容器、备份数据、更新 Compose / 镜像并运行数据迁移；失败时保留旧镜像 digest 和数据恢复点。

### 17.9 推荐实施顺序

1. 修复目标主机 SSH 用户 home、Docker daemon 权限和 render node 权限。
2. 在宿主机完成 `vainfo`，再用独立 `gpu-debug` 容器完成 QSV / VAAPI 主动编码。
3. 构建 Curated `linux/amd64` runtime 镜像，先用手工 Compose 验证 `/data`、媒体授权路径、端口和 GPU。
4. 使用 `fnpack create curated --template docker` 创建 FPK 骨架。
5. 接入 `manifest`、`config/resource`、Compose、桌面入口和 `status`，生成首个内部测试 `.fpk`。
6. 从 fnOS 应用中心安装，验证安装、启动、停止、重启、升级、卸载保留数据、端口冲突和镜像拉取失败。
7. 验证 FPK 容器内的真实 Intel 转码与软件回退，确认 FPK 管理路径和手工 Compose 结果一致。
8. 固定 FPK 版本、镜像 digest、最低 fnOS 版本、备份 / 回滚流程后，再考虑对外分发。

### 17.10 决策

将 Curated 改为通过 FPK 安装是可行且推荐的 fnOS 专用交付方向，但推荐形式是 **Docker FPK**：

```text
Curated FPK
├── fnOS 应用中心元数据、安装向导、权限与资源声明
├── 桌面入口、启动 / 停止 / 状态与升级脚本
└── Docker Compose
    └── Curated runtime 镜像
        ├── Go server + SPA
        ├── FFmpeg + Intel 用户态媒体驱动
        ├── /data -> TRIM_PKGVAR
        ├── media -> fnOS 授权路径
        └── /dev/dri/renderD* -> Intel iGPU
```

这条路线可以把“用户手工管理 Docker”变成“用户从 fnOS 应用中心安装和管理 Curated”，同时保留容器提供的可重复运行环境。它不取消宿主机 GPU 预检、Curated Linux GPU profile、数据迁移和安全加固工作，只是把最终交付体验变得更符合 NAS 平台。

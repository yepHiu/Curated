# Agent 构建 / 编译 / 测试范式（Curated）

本文档约束 **Agent 与本机协作者** 在本仓库内执行安装、开发、构建、测试时的**默认做法**，避免不同会话各用一套命令（例如在错误目录跑 `go test`、或随意改锁文件而不提交说明）。

**优先级**：若与对话中的临时指令冲突，以本文档为准；若与 `package.json` / `go.mod` 实际脚本冲突，以仓库脚本为准并应更新本文档。

---

## 1. 全局约定

| 项 | 约定 |
|----|------|
| 包管理器 | **首选 `pnpm`**（以 `pnpm-lock.yaml` 为准）；**允许** `npm`、`npx`（见下节等价命令） |
| 前端工作目录 | **仓库根目录**（含 `package.json`、`vite.config.ts`） |
| 后端工作目录 | **`backend/`**（Go 模块 `curated-backend`） |
| Go 构建/模块缓存 | **默认使用** `go env GOCACHE`、`go env GOMODCACHE`（一般在用户目录）；**禁止**为跑测试而把缓存指到仓库内路径（见 §5.1） |
| 联调端口 | 前端 Vite 常见 **5173**；后端开发 HTTP 默认 **`127.0.0.1:8080`**（`vite` 仍提供 `/api` 代理；本地 loopback Web API 开发态默认直连，避免大上传经过代理）；release 默认 **`127.0.0.1:8081`** 仍走同源 `/api` |
| 环境变量 | 前端见根目录 `.env` / `.env.example`；联调常用 `VITE_USE_WEB_API=true` |

**禁止（除非用户明确要求）**

- 在 **`backend/`** 外目录执行 **`go run ./cmd/curated`**（路径会错）。
- 将 **`GOCACHE`**、**`GOMODCACHE`**、**`GOTMPDIR`** 指到**本仓库目录或其子目录**（见 §5.1；发布脚本另有约定时以脚本为准）。
- 跳过锁文件（**`pnpm-lock.yaml`** / 若使用 npm 则 **`package-lock.json`**）随意升级依赖并提交，且不在 PR 中说明。

### 1.1 `npm` / `npx` 与 `pnpm` 等价（任选其一）

| 场景 | pnpm（首选） | npm / npx |
|------|----------------|-----------|
| 安装依赖 | `pnpm install` | `npm install` |
| 开发服务器 | `pnpm dev` | `npm run dev` |
| 类型检查 | `pnpm typecheck` | `npm run typecheck` |
| Lint | `pnpm lint` | `npm run lint` |
| 测试 | `pnpm test` | `npm run test` |
| 构建 | `pnpm build` | `npm run build` |
| 单文件 Vitest | `pnpm test <file>` | `npm run test -- <file>` |

若用 **`npm`** 管理依赖并需提交锁文件，应维护 **`package-lock.json`**；**不要**在同一 PR 里混改 `pnpm-lock.yaml` 与 `package-lock.json` 且无说明，以免团队混乱。

---

## 2. 一次性安装

在**仓库根目录**执行：

```bash
pnpm install
```

（或使用 `npm install`；见 §1.1。）

后端无单独 Node 依赖；Go 依赖由模块在构建/测试时解析。

---

## 3. 开发环境启动前后端

先完成 **§2 一次性安装**（`pnpm install`）。联调需要**两个终端**，顺序不限，但需**同时保持运行**。

| 角色 | 工作目录 | 默认地址 | 说明 |
|------|-----------|----------|------|
| 后端 HTTP API | `backend/` | `http://127.0.0.1:8080` | 默认仅监听 loopback；`vite` 仍把 `/api` 代理到此端口；release `127.0.0.1:8081` 静态托管使用同源 `/api`。非 loopback 监听必须显式 `lanEnabled: true` 且已初始化 PIN。 |
| 前端 Vite | 仓库根目录 | `http://127.0.0.1:5173` | 浏览器访问此地址 |

**终端 1 — 后端**

```bash
cd backend
go run ./cmd/curated
```

看到服务监听 **8080**（或你在配置里改的 `httpAddr`）即就绪。可访问 **`GET http://127.0.0.1:8080/api/health`** 确认后端已响应。

**终端 2 — 前端**

```bash
pnpm dev
```

（或 `npm run dev`；见 §1.1。）

浏览器打开 Vite 提示的本地地址（一般为 **5173**）。

**与后端真实联调（非 Mock）**

1. 在仓库根目录 `.env` 中设置 **`VITE_USE_WEB_API=true`**（可参考 `.env.example`）。
2. 确保后端已启动在默认 **`http://127.0.0.1:8080`**（本地 loopback Web API 开发态会直连该端口；非 loopback / fallback 仍由 **`vite.config.ts`** 里的 `/api` 代理转发）。
3. 改 `.env` 后需**重启** `pnpm dev` 才会生效。

**端口被占用时**：先结束占用 **5173** / **8080** 的旧进程，再重新启动前后端。

---

## 4. 前端：类型检查 /  Lint / 测试 / 构建

均在**仓库根目录**执行：

| 目的 | 命令 |
|------|------|
| 类型检查 | `pnpm typecheck` |
| ESLint | `pnpm lint` |
| 单元测试（Vitest） | `pnpm test` |
| Electron 单元测试 | `pnpm test:electron` |
| 轻量运行时 e2e（Chromium） | `pnpm test:e2e` |
| 生产构建 | `pnpm build`（内部含 `typecheck` + `vite build`） |

`pnpm build` 执行分级体积监管，配置集中在 `bundle-policy.json`。正常增长放行；超过关注线、单次突增或累计增长只提醒；仅超过绝对严重上限才失败。取消逐个具名 chunk 的硬预算与“chunk 缺失即失败”规则，单块超过 1 MB 或 HLS 进入首屏只提示检查懒加载。

| 指标 | 关注线（不阻断） | 严重上限（阻断） |
|---|---:|---:|
| 首屏静态 JS raw / gzip | 900 / 300 kB | 1500 / 500 kB |
| 全部 JS raw / gzip | 5000 / 1600 kB | 8000 / 2500 kB |
| 全部 CSS raw | 500 kB | 1500 kB |
| 前端产物总量 raw | 100 MB | 150 MB |

阈值严格使用 `>`，等于阈值不升级等级。单位为十进制字节。首屏包括入口的静态依赖闭包，动态导入不计入；总量包含字体、图片、JSON、public 复制文件等，排除报告自身。gzip 为逐 JS 文件压缩估算，不代表真实网络传输；范围不含 Electron、Go、FFmpeg 或最终安装包。

增长提醒采用双基线：

- 最近主分支 **frontend-quality 作业成功**的 Web 构建：增加量同时超过 **20%** 和绝对噪声门槛时提醒。
- 版本控制中的人工确认基线 `bundle-baseline.json`：增加量同时超过 **50%** 和两倍绝对噪声门槛时提醒，防止每次小幅增长不断累计。
- 各指标噪声门槛依次为首屏 raw 100 kB / gzip 30 kB、总 JS raw 300 kB / gzip 100 kB、CSS 100 kB、全部产物 10 MB。缩小不报警。基线不自动抬高绝对上限。

`dist/bundle-analysis.json` 提供完整机器报告（指标、策略、对比快照、分块及大模块、资源排行），`dist/bundle-report.md` 提供可读摘要。报告在严重超限报错前写出。GitHub Actions 自动写入作业 Summary、显示 warning/error，并上传报告保留 90 天。只有主分支 push 且前端质量作业成功才保存下一次比较缓存；PR 只读。缓存可能因首次运行、过期或驱逐不可用，报告会注明，绝对上限及人工基线仍执行；Web/Mock 不跨模式比较。没有同模式人工基线时明确注明未进行累计比较。

人工更新基线：先在 Web API 模式运行 `pnpm build`，审阅报告及新增功能合理性，再运行 `pnpm bundle:baseline` 并提交 `bundle-baseline.json`。该命令拒绝 Mock、损坏的报告和绝对严重超限，不改变 `bundle-policy.json`；读取的是最近一次本机 `dist` 报告，必须确保重新构建后再运行。不要把该命令加入每次构建或 CI，以免掩盖累计增长。调整策略阈值需在修改中说明原因，不能为了消除提醒而自动上调。

新增监管测试：`pnpm test src/lib/bundle-policy.test.ts`。该测试覆盖真实 Vite 构建，包括提醒放行、超限失败后保留报告，以及 public/CSS 统计和动态导入排除。

**单测文件**（示例）：

```bash
pnpm test path/to/file.test.ts
```

`pnpm test:e2e` 会自动在独立的 4173（Mock）与 4174（Web API stub）端口启动 Vite，覆盖 Mock 导航不访问后端、锁定启动不请求受保护资源、解锁后只 hydrate 一次等日常运行时流程。它不依赖本机 5173/8080 开发服务。`pnpm test:display` 是独立的跨浏览器/多 viewport 长耗时套件，仍需按 UI 规范获得明确同意后才能运行，不纳入日常 CI。

---

## 5. 后端：测试与编译

在 **`backend/`** 目录执行：

| 目的 | 命令 |
|------|------|
| 全量测试 | `go test ./...` |
| 静态检查 | `go vet ./...` |
| 单包测试 | `go test ./internal/storage/...`（示例） |

备份维护命令同样必须从 `backend/` 运行。`backup-create` / `backup-verify` 可在不替换运行数据的情况下执行；`backup-restore` 必须先完全退出 Curated，并在成功 preflight 后显式传 `-confirm-restore`：

```powershell
go run ./cmd/curated -maintenance backup-create -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-verify -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-preflight -backup-path C:\Backups\curated.curated-backup
go run ./cmd/curated -maintenance backup-restore -backup-path C:\Backups\curated.curated-backup -confirm-restore
```

自定义主配置需同时传 `-config <path>`。恢复会获取 `<databasePath>.runtime.lock`；仍有 Curated 进程持锁时必须失败，不能通过删除锁文件绕过。

路径迁移同样是离线维护操作。必须先完全退出 Curated，先运行只读 plan，确认 `canApply=true`、影响数、样例、目标状态和冲突，再为 apply 提供一个尚不存在的备份包路径与显式确认：

```powershell
go run ./cmd/curated -maintenance path-migrate-plan -path-from D:\Media -path-to E:\Media
go run ./cmd/curated -maintenance path-migrate-apply -path-from D:\Media -path-to E:\Media -backup-path D:\Backups\before-path-migration.curated-backup -confirm-path-migration
```

Windows→Unix 等当前操作系统无法检查目标的场景，只有在人工确认目标布局后才可显式增加 `-allow-missing-paths`；该参数只放行 missing/unchecked，不放行目标类型错误、I/O error 或冲突。apply 会自动创建并验证迁移前备份，随后在同一事务内更新白名单路径、清除旧存储 binding、执行完整性检查并写审计。禁止删除 `.runtime.lock`、跳过 plan、复用已有备份目标或直接用 SQL 字符串替换代替维护命令。

从仓库根目录也可：

```bash
cd backend && go test ./...
```

**运行二进制（开发）**：见第 3 节 `go run ./cmd/curated`。

**Windows 开发构建辅助**（可选，见 `package.json`）：在仓库根目录 `pnpm backend:build:dev`，产出以脚本与 `workspace-quick-reference` 为准。

### 5.1 Go 缓存与测试产物（避免在仓库里「长」一大坨）

`go test`、`go build` 会写**构建缓存**（`GOCACHE`）和**模块缓存**（`GOMODCACHE`）。若把这两项指到仓库内（例如自建 `.tmp-go-modcache`、`.gocache` 再设环境变量），会产生数万文件、拖慢 Git 与 IDE，且易被误提交。

**Agent / 日常约定：**

1. **不要设置** `GOCACHE`、`GOMODCACHE`、`GOTMPDIR` 指向本仓库路径。直接执行 `cd backend && go test ./...` 即可，让 Go 使用默认位置（Windows 常见为 `%LocalAppData%\go-build` 与用户 `go` 目录下的 `pkg/mod`）。
2. 若曾误设，可删掉仓库内误生成的目录后，在新终端执行 **`go env -u GOCACHE`**、**`go env -u GOMODCACHE`**（若此前用 `go env -w` 写过），或改为本机**仓库外**路径。
3. **覆盖率 / 测试输出**：不要默认把 `-coverprofile=coverage.out`、`-memprofile` 等写到仓库根或未忽略路径；需要时写到系统临时目录、或 `backend/` 下明确路径并确保在 **`.gitignore`** 中。
4. 仓库内 **`.gocache/`**、**`.tmp-go/`**、**`.tmp-go-modcache/`** 已在 **`.gitignore`** 中作为兜底；仍应避免主动把缓存指到此处。当前 release Python 脚本也已改为使用系统临时目录承载 Go 构建缓存，不再以仓库内缓存目录作为例外。

---

## 6. 变更后推荐检查顺序（PR 前）

在仓库根目录依次执行（可按需跳过已确认步骤）；使用 npm 时见 §1.1 将 `pnpm`/`pnpm vitest` 换成 `npm run` / `npx vitest`。

1. `pnpm typecheck`
2. `pnpm lint`
3. `pnpm test`
4. `pnpm test:electron`
5. `pnpm test:e2e`
6. `cd backend && go test ./...`
7. `cd backend && go vet ./...`

全绿后再进行 `pnpm build`（若本次改动涉及前端发布构建）。

GitHub Actions 的 `.github/workflows/ci.yml` 在 pull request 与 `master` push 上执行以上质量门禁，并额外运行生产依赖 high 漏洞审计、前端/Electron 构建和发布脚本测试。display-scaling 套件保持人工选择，不在该工作流中运行。

CI 的生产前端构建步骤显式设置 `VITE_USE_WEB_API=true`，与 Windows 生产包保持一致；仓库检出不依赖开发机未跟踪的 `.env`。Mock 模式继续由独立的运行时 e2e 服务覆盖。复现生产构建时，在 PowerShell 中先执行 `$env:VITE_USE_WEB_API = 'true'`，再运行 `pnpm build`；Web/Mock 报告必须按模式隔离，不可通过交换基线绕过差异。

---

## 7. 与 Cursor 规则的关系

- 日常端口、代理、库配置：**`.cursor/rules/workspace-quick-reference.mdc`**
- 目录与 API 概览：**`.cursor/rules/project-facts.mdc`**
- 本文档：**只解决「命令与目录」的统一范式**；产品行为细节以上述规则与 `CLAUDE.md` 为准。

---

## 8. 修订

修改 `package.json` 脚本或默认端口时，请同步更新本文档与 `AGENTS.md` / `workspace-quick-reference.mdc` 中相关描述。

## 9. 生产包版本号

- 生产包版本的唯一自动化来源是 `scripts/release/version.json`，当前基线为 `1.5.1`。
- `pnpm release:*` 当前统一调用 `python scripts/release/release_cli.py`。
- `pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish` 在未显式传入 `-Version` 时，都会自动执行 `patch + 1`。
- `major` / `minor` 只允许人工通过 `pnpm release:version:set-base -- --Major <major> --Minor <minor>` 调整，并在调整时把 `patch` 重置为 `0`。
- `pnpm release:publish` 是整机发布推荐入口，它只分配一次版本号，再复用到便携包、安装包、manifest 与 `docs/ops/package-build-history.csv` 打包台账。
- 发布打包会把 FFmpeg 运行时放入 `resources/app/third_party/ffmpeg/bin/`：优先使用 `backend/third_party/ffmpeg/bin/`，否则从 Scoop 或 PATH 发现真实二进制；`scoop/shims` 下的 shim 不会被复制，找不到真实运行时时打包失败。
- 未得到用户明确要求时，禁止删除已经打出的生产包产物；`release/installer/*.exe` 与 `release/portable/*.zip` 都必须保留。准备重新打包、同版本重打、清理 release 目录或整理产物时，也不能主动删除既有 installer / portable 包。

## Production pinyin dictionary asset

`vite.pinyin-data.ts` extracts the five pure dictionary literals from the locked `pinyin-pro` ESM source into a content-hashed JSON asset. The optional pinyin chunk fetches it from the same origin before initializing its existing synchronous matching exports. Deploy the complete `dist` directory, including JSON assets. A changed upstream dictionary format fails the build; a missing runtime asset rejects the import explicitly. This reduces executable JavaScript, not total dictionary data. Existing JS budgets stay unchanged. `src/lib/pinyin-data-build.test.ts` checks extraction and rejects executable literals; real production-browser matching was checked during the media Beta integration.

## Desktop 独立构建（2026-09-25）

根目录 `pnpm build:electron` 只构建 main/preload 和本地连接页，`pnpm dev:electron` 不再启动后端。Server 需从 backend 目录独立启动并提供 Web UI。Node 需支持现有 `--configLoader native` TypeScript 配置加载；本机验证使用已安装的 Node 25。

## 三种组件包构建（2026-09-25，Windows 安装验收未完成）

`python scripts/release/release_cli.py publish --variant all` 在一次版本分配内构建 Full、Server、Desktop 三种安装包；`--variant server|desktop|full` 可单独选择。该命令只构建本地产物，不上传 Release。输出位于新的 `release/components-<version>-<stamp>/`，不覆盖已有包。Desktop 纯客户端不需要 Go/FFmpeg；Server 包含 Web UI 与 FFmpeg，不需要 Electron。完整包默认串行安装两个独立组件，复用组件安装身份和卸载入口。

Windows 组件按当前用户安装至 `%LOCALAPPDATA%/Programs/Curated/Server|Desktop`；Server 数据仍在原数据根（默认 `%LOCALAPPDATA%/Curated`），独立于程序目录。Server 就绪后写入本机地址提示，Desktop 仅在无连接记录/无输入时建议该地址，不替代身份检查。自启继续为 Server 的用户登录托盘模式，不是系统服务。

跨平台构建 Go 显式使用 windows/amd64；打包 Desktop 时需要 Windows Electron runtime，可用 `CURATED_ELECTRON_RUNTIME_DIR` 指向它。没有 Inno Setup 时只输出 `.iss` 和 `scripts-only` manifest，不算已生成 EXE。已组装的 payload 可通过 `package-components --version X.Y.Z --components-dir <payloads> --output-dir <installer> --variant all` 生成/编译安装器。旧 `package-installer` / `package-portable` 是历史单包低层入口，不能用于发布当前客户端拆分版本。

Server 更新只接受精确命名的 Server 安装包，旧缓存中的整包或 Desktop 包会被拒绝；Desktop 本地连接页提供官方下载入口，不由远端执行客户端更新。当前仍读取既有官方 Release 接口，新三包**不得直接挂到旧客户端的 latest feed**；新旧 feed 隔离发布流程尚未落地。manifest 已标注这一限制，`publish` 不执行外部发布。

当前安装器会阻止直接覆盖旧一体包，并给出备份/迁移提示；自动迁移、Windows 实机安装/升级/卸载、真实两机 SSDP 仍待完成。不要将本实现当作已通过生产发布验收。

## 跨平台 CI 打包（2026-09-26）

新增手动 `Build installers` 工作流（`.github/workflows/package.yml`），Windows x64 EXE、macOS 15+ arm64/x64 PKG 各支持 Full/Server/Desktop，默认九包；显式版本，核验 manifest/校验和后上传 Actions artifacts，不发布 Release。macOS app 仅 ad-hoc 签名、安装器未签名未公证；Server 以 curated-server 命令独立运行，尚无 macOS 托盘/服务。详见 docs/guide.md「GitHub Actions 安装包构建」。云端首次运行及真实安装验收待完成。

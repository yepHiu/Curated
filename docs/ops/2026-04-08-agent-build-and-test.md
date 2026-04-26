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
| 联调端口 | 前端 Vite 常见 **5173**；后端开发 HTTP 常见 **8080**（`vite` 将 `/api` 代理到该端口） |
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
| 单文件 Vitest | `pnpm test -- <file>` | `npm run test -- <file>` |

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
| 后端 HTTP API | `backend/` | `http://127.0.0.1:8080` | `vite` 把前端的 `/api` 代理到此端口 |
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
2. 确保后端已启动且端口与 **`vite.config.ts`** 里 `/api` 代理目标一致（默认 **`http://localhost:8080`**）。
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
| 生产构建 | `pnpm build`（内部含 `typecheck` + `vite build`） |

**单测文件**（示例）：

```bash
pnpm test -- path/to/file.test.ts
```

---

## 5. 后端：测试与编译

在 **`backend/`** 目录执行：

| 目的 | 命令 |
|------|------|
| 全量测试 | `go test ./...` |
| 单包测试 | `go test ./internal/storage/...`（示例） |

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
4. `cd backend && go test ./...`

全绿后再进行 `pnpm build`（若本次改动涉及前端发布构建）。

---

## 7. 与 Cursor 规则的关系

- 日常端口、代理、库配置：**`.cursor/rules/workspace-quick-reference.mdc`**
- 目录与 API 概览：**`.cursor/rules/project-facts.mdc`**
- 本文档：**只解决「命令与目录」的统一范式**；产品行为细节以上述规则与 `CLAUDE.md` 为准。

---

## 8. 修订

修改 `package.json` 脚本或默认端口时，请同步更新本文档与 `AGENTS.md` / `workspace-quick-reference.mdc` 中相关描述。

## 9. 生产包版本号

- 生产包版本的唯一自动化来源是 `scripts/release/version.json`，当前基线为 `1.3.2`。
- `pnpm release:*` 当前统一调用 `python scripts/release/release_cli.py`。
- `pnpm release:portable`、`pnpm release:installer`、`pnpm release:publish` 在未显式传入 `-Version` 时，都会自动执行 `patch + 1`。
- `major` / `minor` 只允许人工通过 `pnpm release:version:set-base -- --Major <major> --Minor <minor>` 调整，并在调整时把 `patch` 重置为 `0`。
- `pnpm release:publish` 是整机发布推荐入口，它只分配一次版本号，再复用到便携包、安装包、manifest 与 `docs/ops/package-build-history.csv` 打包台账。
- 发布打包会把 FFmpeg 运行时放入 `third_party/ffmpeg/bin/`：优先使用 `backend/third_party/ffmpeg/bin/`，否则从 Scoop 或 PATH 发现真实二进制；`scoop/shims` 下的 shim 不会被复制，找不到真实运行时时打包失败。
- 未得到用户明确要求时，禁止删除已经打出的生产包产物；`release/installer/*.exe` 与 `release/portable/*.zip` 都必须保留。准备重新打包、同版本重打、清理 release 目录或整理产物时，也不能主动删除既有 installer / portable 包。

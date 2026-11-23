# CI/CD 瘦身评估（2026-11-02）

status: planned

## 1. 现状实测（GitHub Actions）

### 1.1 日常 CI（`.github/workflows/ci.yml`）

触发：`pull_request`、`push` 到 `master`、以及 CD `workflow_call`。

样本：成功 run `39058653995`（2026-10-17 master push）。

| Job | 墙钟 | 主要耗时步骤 |
|-----|------|--------------|
| Frontend, Electron, and release scripts | ~4.5 min | Vitest 前端单元测试 ~177s；typecheck ~21s；lint ~12s；`pnpm build` ~25s |
| Go test and vet | ~2.3 min | `go test ./...` ~102s；`go vet` ~11s |
| Runtime browser e2e | ~1.3 min | Playwright Chromium 安装 ~21s；e2e ~38s |

**关键路径 ≈ 4.5 min**，几乎完全由「单体 frontend-quality job」决定；backend / e2e 已并行，不拉长墙钟。

### 1.2 发布 CD（`.github/workflows/cd-release.yml`）

样本：成功 run `39055037119`（Server 1.8.0 / Desktop 0.2.7）。

| Job | 墙钟 |
|-----|------|
| prepare | ~14s |
| quality（复用完整 CI） | ~4.5 min |
| Build Windows x64 | ~16.5 min |
| Build Mac Desktop | ~5 min（与 Windows 并行，不主导） |
| publish | ~3 min |

**关键路径 ≈ 25 min**：prepare → quality → Windows → publish。Windows 打包/冒烟占大头；完整 CI 复跑在 Windows 前硬性挡约 4.5 min。

### 1.3 已观察到的浪费信号

- **无 path filter**：纯 `docs/*` 提交仍跑完整三 job。近期两次 docs push 都因非文档门禁失败（e2e / frontend），浪费 runner 分钟且污染 master 状态。
- **typecheck 重复**：独立 `pnpm typecheck` 后再跑 `pnpm build`（脚本内已含 `pnpm typecheck && vite build`）。
- **frontend-quality 单体串行**：单元测试 ~3 min 与后续 build/Electron/release 脚本无法并行。
- **CD 必复跑完整 CI**：即便同 commit 刚在 master 绿过，仍再跑一遍 quality 才允许 Windows 开始。

## 2. 瘦身空间判断

有空间。优先级应按「节省分钟 × 实施风险」排序；**不要**先砍 Windows 安装器/原生引擎冒烟——那是发布安全门禁，不是日常 CI 噪音。

### 高收益 / 低风险（建议先做）

1. **Path filter / 变更感知跳过**
   - 仅 `docs/**`、`*.md`、规则文案等：跳过 frontend / backend / e2e，或只跑极轻量「docs/lint」占位 job。
   - 仅 `backend/**`：跳过 Playwright e2e 与 Electron 测试（保留 Go）。
   - 仅 `src/**` 前端：可跳过 Go（接受「跨边界漏检」风险，或保留 Go 作为便宜保险——当前仅 ~2 min 且不在关键路径）。
   - 预期：文档类 push/PR 从 ~4.5 min × 3 job 降到接近 0；减少 docs 提交被 flaky e2e 误伤。

2. **去掉重复 typecheck**
   - CI 删除独立 Typecheck 步骤，保留 `pnpm build` 内的 typecheck；或改 `pnpm build` 为纯 vite build、只保留一次显式 typecheck。
   - 预期节省 ~20s，实现几乎零风险。

3. **拆分 frontend-quality 并行**
   - Job A：typecheck + lint + Vitest（关键路径）
   - Job B：Electron 测试 + Electron main build + Python release 脚本测试
   - Job C（可选）：生产 `pnpm build` + bundle 报告（仍可独立失败）
   - 预期墙钟从 ~4.5 min 降到约 **3.0–3.3 min**（受 Vitest 主导）。

### 中收益 / 中风险

4. **Vitest 本身瘦身或分片**
   - 当前前端单元测试是 CI 最大单项（~3 min）。可审计慢用例、减少重复 fixture、或 `shard` 到 2 个并行 job。
   - 分片会增加并发分钟数，但缩短墙钟；适合「等 CI」体验优先。

5. **Playwright 浏览器缓存**
   - 缓存 Chromium 安装目录，e2e job 可再省 ~20s；对总墙钟帮助有限（e2e 已非关键路径），但降低分钟与 flaky 安装面。

6. **CD quality 条件化**
   - 若 immutable tag 指向的 commit 已有成功的同 SHA CI（master/PR），允许跳过 `quality` 或只跑差异子集，直接进入 Windows。
   - 可省发布路径 ~4.5 min；需严格校验 SHA、workflow 定义未变、结论为 success。失败/过期则强制全量。

### 低收益或应保留

7. **Windows CD（~16.5 min）**：含 Inno、FFmpeg、MSYS2、native engine 准备、installer/lifecycle smoke。这是发版门禁；可做缓存优化（FFmpeg/choco、native downloads 已有部分 cache），不宜删冒烟。
8. **`pnpm audit --prod`**：~1s，保留。
9. **Go vet / Go test**：不在关键路径；backend 改动时必须保留。
10. **退役 workflow（`cd-legacy.yml`）**：几乎不耗资源；可删以减噪音，非性能问题。
11. **display-scaling / 付费云调用**：本来就不在日常 CI，无需再砍。

## 3. 建议实施批次

| 批次 | 内容 | 目标 |
|------|------|------|
| A | path filter + 去重复 typecheck | 文档/无关变更不再烧满额 CI；小幅缩短每次运行 |
| B | 拆分 frontend-quality 并行 | 日常墙钟目标 ≤ ~3.5 min |
| C | Vitest 慢用例审计 / 可选 shard | 视 A/B 后是否仍觉得 Vitest 过重 |
| D | CD quality 同 SHA 跳过（可选） | 发布总时长目标再降 ~4–5 min |

## 4. 明确不建议的砍法

- 为省时间默认跳过 Windows installer smoke 或 native engine 准备。
- 用 `paths-ignore: docs` 的同时让 docs 失败仍显示为「required check 缺失」却不处理分支保护——需配套 required checks / 占位成功 job。
- 把 e2e 完全移出 PR：近期 docs 失败已证明它会拦无关提交，但也说明它在抓运行时回归；更合适的是「有前端/壳层变更才跑」，而不是永久关闭。

## 5. 开放决策

1. path filter 粒度：粗（docs-only vs 代码）还是细（frontend / backend / electron / scripts / release）？
2. 日常 CI 优化目标是 **墙钟** 还是 **Actions 分钟账单**？（分片改善前者、可能增加后者）
3. CD 是否接受「同 SHA 已绿则跳过 quality」？

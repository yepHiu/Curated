# Desktop Icon Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the new pink-on-dark Curated icon to the canonical desktop/tray icon source and sync every derived desktop/web asset to it.

**Architecture:** Treat `icon/curated-appicon.png` as the single source image for desktop-facing icon assets. Regenerate the public PNG copies and the embedded Windows `.ico`, then update project memory/docs so future icon updates continue from the same source file.

**Tech Stack:** PowerShell, Python Pillow, Go embed asset, Vite static assets, Markdown docs

---

### 2026-09-26 居中资源同步记录

- 发现 `public/Curated-icon.png` 的粉色图案向下偏移 15 px，Windows ICO 也继承了偏移；`src/icon/curated-appicon.png` 仍是不同尺寸的旧版资源。
- 统一以已经居中的 `icon/curated-appicon.png` 为源，覆盖 `src/icon/` 与 `public/` 应用图标及本地 `dist/`、`backend/frontend-dist/` 中的副本，重新生成 ICO 的全部 9 个尺寸。
- 透明标志副本同步自 `icon/curated-mark.png`；横向字标保持原有组合排版。
- 删除未使用的模板 favicon 及本地构建目录中的对应副本，更新入口图标 URL 的版本参数，避免浏览器沿用旧图标缓存。
- 验收以粉色图案外接框为准：源图上下留白 79 / 79 px、左右 78 / 79 px；栅格尺寸奇偶差允许最多半像素的中心偏差。维护规则见 `icon/README.md`。

### 2026-09-26 macOS Dock 视觉尺寸修正

- 原源图的深色底外接框占满 322 × 322 画布，直接传给 `app.dock.setIcon` 时视觉尺寸偏大。
- 新增 `public/Curated-icon-macos.png`，把源图原尺寸放在 384 × 384 透明画布中央，四边留 31 px，主体宽度占 83.85%。
- Electron 仅为 macOS Dock 选择此派生图，缺失时回退现有图标；托盘与 Web / Windows 资源继续使用原图。完整退出并重新启动 Electron 后生效。
- 生成方法见 `icon/README.md`；验证画布尺寸、透明外接框与原图像素一致性，并运行 Electron 测试及主进程编译。尚无 macOS 安装包构建流程，此次修正运行时 Dock 图标。

### Task 1: Record the canonical icon source

**Files:**
- Create: `docs/plan/2026-04-13-desktop-icon-sync-plan.md`
- Modify: `.cursor/rules/workspace-quick-reference.mdc`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `docs/reference/2026-03-20-project-memory.md`
- Modify: `docs/plan/2026-03-31-production-packaging-and-config-strategy.md`

- [ ] **Step 1: Update the repo memory files to point to the new source image**

State explicitly that:
- README wordmark still uses `icon/curated-wordmark.png`
- Desktop/tray/app icon source is now `icon/curated-appicon.png`
- Derived files remain `public/Curated-icon.png`, `backend/frontend-dist/Curated-icon.png`, and `backend/internal/assets/curated.ico`

- [ ] **Step 2: Save the implementation note**

Keep this plan in `docs/plan/` so the icon swap has a discoverable record alongside other packaging/tray docs.

### Task 2: Sync the binary icon assets

**Files:**
- Modify: `public/Curated-icon.png`
- Modify: `backend/frontend-dist/Curated-icon.png`
- Modify: `backend/internal/assets/curated.ico`
- Add: `icon/curated-appicon.png`

- [ ] **Step 1: Copy the canonical PNG into the two shipped PNG targets**

Run:

```powershell
Copy-Item icon\curated-appicon.png public\Curated-icon.png -Force
Copy-Item icon\curated-appicon.png backend\frontend-dist\Curated-icon.png -Force
```

Expected: both PNG targets have the same bytes as the canonical source.

- [ ] **Step 2: Regenerate the Windows `.ico` from the canonical PNG**

Run:

```powershell
python -c "from PIL import Image; img = Image.open(r'icon/curated-appicon.png').convert('RGBA'); img.save(r'backend/internal/assets/curated.ico', sizes=[(16,16),(20,20),(24,24),(32,32),(40,40),(48,48),(64,64),(128,128),(256,256)])"
```

Expected: `backend/internal/assets/curated.ico` is rewritten as a multi-size icon suitable for the embedded tray asset and packaged desktop shortcuts.

### Task 3: Verify the icon chain

**Files:**
- Verify: `index.html`
- Verify: `backend/internal/assets/tray_icon.go`

- [ ] **Step 1: Verify the frontend still references the PNG icon path**

Run:

```powershell
Get-Content index.html
```

Expected: `<link rel="icon" type="image/png" href="/Curated-icon.png" />`

- [ ] **Step 2: Verify the backend still embeds the `.ico` asset**

Run:

```powershell
Get-Content backend\internal\assets\tray_icon.go
```

Expected: `//go:embed curated.ico`

### Task 4: Run repository verification

**Files:**
- Verify: frontend workspace
- Verify: `backend/`

- [ ] **Step 1: Run the frontend production build**

Run:

```powershell
pnpm build
```

Expected: exit code `0`

- [ ] **Step 2: Run backend tests to verify the embed and tray packages still compile**

Run:

```powershell
cd backend
go test ./...
```

Expected: exit code `0`

- [ ] **Step 3: Inspect the working tree before completion**

Run:

```powershell
git status --short
```

Expected: only the intended icon/doc files are modified or added.

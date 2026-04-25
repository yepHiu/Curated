# Desktop Icon Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the new pink-on-dark Curated icon to the canonical desktop/tray icon source and sync every derived desktop/web asset to it.

**Architecture:** Treat `icon/curated-icon-rg-dark-pink.png` as the single source image for desktop-facing icon assets. Regenerate the public PNG copies and the embedded Windows `.ico`, then update project memory/docs so future icon updates continue from the same source file.

**Tech Stack:** PowerShell, Python Pillow, Go embed asset, Vite static assets, Markdown docs

---

### Task 1: Record the canonical icon source

**Files:**
- Create: `docs/plan/2026-04-13-desktop-icon-sync-plan.md`
- Modify: `.cursor/rules/workspace-quick-reference.mdc`
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `docs/reference/2026-03-20-project-memory.md`
- Modify: `docs/plan/2026-03-31-production-packaging-and-config-strategy.md`

- [ ] **Step 1: Update the repo memory files to point to the new source image**

State explicitly that:
- README wordmark still uses `icon/curated-title-nobg.png`
- Desktop/tray/app icon source is now `icon/curated-icon-rg-dark-pink.png`
- Derived files remain `public/Curated-icon.png`, `backend/frontend-dist/Curated-icon.png`, and `backend/internal/assets/curated.ico`

- [ ] **Step 2: Save the implementation note**

Keep this plan in `docs/plan/` so the icon swap has a discoverable record alongside other packaging/tray docs.

### Task 2: Sync the binary icon assets

**Files:**
- Modify: `public/Curated-icon.png`
- Modify: `backend/frontend-dist/Curated-icon.png`
- Modify: `backend/internal/assets/curated.ico`
- Add: `icon/curated-icon-rg-dark-pink.png`

- [ ] **Step 1: Copy the canonical PNG into the two shipped PNG targets**

Run:

```powershell
Copy-Item icon\curated-icon-rg-dark-pink.png public\Curated-icon.png -Force
Copy-Item icon\curated-icon-rg-dark-pink.png backend\frontend-dist\Curated-icon.png -Force
```

Expected: both PNG targets have the same bytes as the canonical source.

- [ ] **Step 2: Regenerate the Windows `.ico` from the canonical PNG**

Run:

```powershell
python -c "from PIL import Image; img = Image.open(r'icon/curated-icon-rg-dark-pink.png').convert('RGBA'); img.save(r'backend/internal/assets/curated.ico', sizes=[(16,16),(20,20),(24,24),(32,32),(40,40),(48,48),(64,64),(128,128),(256,256)])"
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

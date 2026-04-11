# Curated README And API Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a dedicated root-level `API.md`, redesign the README into a polished GitHub-style English homepage, add complete Chinese and Japanese README translations, and write the new maintenance rules into project memory.

**Architecture:** Public-facing docs move to a root-level structure: `README.md` remains the English primary entry, `README.zh-CN.md` and `README.ja-JP.md` mirror its structure as full translations, and `API.md` becomes the single public API reference. Agent-facing memory files record that API changes must update `API.md` instead of rebuilding API tables inside the README.

**Tech Stack:** Markdown documentation, existing repository rules under `.cursor/rules/`, root README language variants.

---

### Task 1: Create The Public API Reference

**Files:**
- Create: `API.md`
- Modify: `README.md`

- [ ] **Step 1: Extract the current API material from `README.md`**

Source sections to migrate:
- `## HTTP API (summary)`
- the “DTOs and error codes” note
- the “Scrape stability additions” note

Keep these runtime references available for the API document:
- `backend/internal/server/server.go`
- `backend/internal/contracts/contracts.go`
- `src/api/types.ts`

- [ ] **Step 2: Write `API.md` as the single public API reference**

Required sections:
- `# Curated API Reference`
- `## Overview`
- `## Base URLs`
- `## Conventions`
- `## Health`
- `## Movies`
- `## Playback`
- `## Actors`
- `## Settings`
- `## Curated Frames`
- `## Scans And Tasks`
- `## Type References`

Each endpoint group should contain:
- method
- path
- purpose
- important query/body notes where needed

- [ ] **Step 3: Remove the full API table from `README.md`**

Replace it with a short section like:

```md
## API

Curated exposes a Go HTTP API for library, playback, actor, settings, and curated-frame workflows.

See [API.md](API.md) for the full endpoint reference.
```

- [ ] **Step 4: Verify API migration coverage**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "^## HTTP API|^# Curated API Reference|API.md" README.md API.md -S
```

Expected:
- `README.md` no longer contains the old `## HTTP API (summary)` section
- `API.md` contains the new API reference title
- `README.md` links to `API.md`

### Task 2: Redesign The English README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite the README structure into GitHub-style project documentation**

Required top-level sections:
- branding and language switch
- overview
- highlights
- quick start
- features
- configuration
- API
- repository layout
- release / packaging
- documentation
- notes

Required language switch block:

```md
English | [简体中文](README.zh-CN.md) | [日本語](README.ja-JP.md)
```

- [ ] **Step 2: Add GitHub-style technology badges near the top**

Use stable badge labels only. Prefer stack badges such as:
- Vue 3
- TypeScript
- Vite
- Go
- SQLite
- Tailwind CSS v4
- shadcn-vue
- Windows

- [ ] **Step 3: Keep the existing branded logo**

Retain:

```md
<p align="center">
  <img src="icon/curated-title-nobg.png" alt="Curated" width="520" />
</p>
```

- [ ] **Step 4: Verify the English README structure**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "^# |^## " README.md -n
```

Expected:
- the section order matches the redesign
- the README contains a language switch and API link section

### Task 3: Add Full Chinese And Japanese README Variants

**Files:**
- Create: `README.zh-CN.md`
- Create: `README.ja-JP.md`
- Modify: `README.md`

- [ ] **Step 1: Create `README.zh-CN.md` as a full translation**

Requirements:
- same section order as `README.md`
- same project claims and commands
- natural Simplified Chinese phrasing
- language switch at the top linking back to English and Japanese

- [ ] **Step 2: Create `README.ja-JP.md` as a full translation**

Requirements:
- same section order as `README.md`
- same project claims and commands
- natural Japanese phrasing
- language switch at the top linking back to English and Chinese

- [ ] **Step 3: Verify cross-links between language variants**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "README.zh-CN.md|README.ja-JP.md|README.md" README.md README.zh-CN.md README.ja-JP.md -S
```

Expected:
- all three documents link to the other two
- no broken filename references appear in the language switch lines

### Task 4: Update Project Memory And Agent Rules

**Files:**
- Modify: `.cursor/rules/project-facts.mdc`
- Modify: `.cursor/rules/workspace-quick-reference.mdc`
- Modify: `docs/2026-03-20-project-memory.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Record the new public documentation structure**

Required facts to add:
- `README.md` is the English primary README
- `README.zh-CN.md` and `README.ja-JP.md` are full translations
- root `API.md` is the single public API reference

- [ ] **Step 2: Record the maintenance rule**

Required rule:
- future API changes must update `API.md`
- README should keep only API overview + link, not the full API table
- user-facing README changes should be reflected across all language variants when relevant

- [ ] **Step 3: Verify memory references**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "API.md|README.zh-CN.md|README.ja-JP.md|single public API reference|完整翻译|full translation" .cursor/rules docs/2026-03-20-project-memory.md CLAUDE.md -S
```

Expected:
- the new documentation convention is present in the agent-facing files

### Task 5: Final Consistency Review

**Files:**
- Modify if needed: `README.md`
- Modify if needed: `README.zh-CN.md`
- Modify if needed: `README.ja-JP.md`
- Modify if needed: `API.md`

- [ ] **Step 1: Review for duplicated API tables**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "^## HTTP API|\\| Method \\| Path \\| Purpose \\|" README.md README.zh-CN.md README.ja-JP.md API.md -S
```

Expected:
- the full API table exists only in `API.md`

- [ ] **Step 2: Review for language-switch and doc links**

Run:

```powershell
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; rg -n "API.md|README.zh-CN.md|README.ja-JP.md|architecture-and-implementation.html|2026-03-24-frontend-ui-spec.md" README.md README.zh-CN.md README.ja-JP.md -S
```

Expected:
- language links are correct
- important documentation links still resolve by filename

- [ ] **Step 3: Commit**

```bash
git add README.md README.zh-CN.md README.ja-JP.md API.md .cursor/rules/project-facts.mdc .cursor/rules/workspace-quick-reference.mdc docs/2026-03-20-project-memory.md CLAUDE.md docs/plan/2026-04-12-readme-and-api-docs-redesign.md docs/plan/2026-04-12-readme-and-api-docs-implementation-plan.md
git commit -m "docs: redesign README and split API reference"
```

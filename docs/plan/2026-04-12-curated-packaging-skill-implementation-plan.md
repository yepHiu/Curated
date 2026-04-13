# Curated Packaging Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a repo-local `curated-packaging` skill that responds to natural-language packaging requests, previews the predicted release version and actions first, then executes the requested packaging workflow.

**Architecture:** Keep the skill private to this repository under `.cursor/skills/curated-packaging/`. Use one small shared intent/preview module to make preview logic deterministic and testable, then expose that behavior through two PowerShell scripts: one for preview and one for execution. Reuse the existing release/versioning scripts instead of duplicating version rules.

**Tech Stack:** Markdown skill files, PowerShell, Node.js ESM helper modules, Vitest, existing `scripts/release/*.ps1` packaging commands.

---

### Task 1: Scaffold The Repo-Local Skill

**Files:**
- Create: `.cursor/skills/curated-packaging/SKILL.md`
- Create: `.cursor/skills/curated-packaging/agents/openai.yaml`
- Create: `.cursor/skills/curated-packaging/scripts/preview-package.ps1`
- Create: `.cursor/skills/curated-packaging/scripts/execute-package.ps1`

- [ ] **Step 1: Write the failing test for the skill trigger surface**

Create the test file with the first failing case:

```ts
import { describe, expect, it } from "vitest"

import { detectPackagingIntent } from "@/lib/skills/curated-packaging-intent"

describe("curated packaging intent detection", () => {
  it("maps natural-language publish requests to publish mode", () => {
    expect(detectPackagingIntent("打生产包")).toEqual({
      mode: "publish",
      baseChange: null,
    })
  })
})
```

Save it to:

`src/lib/skills/curated-packaging-intent.test.ts`

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-intent.test.ts --configLoader native --pool threads
```

Expected: FAIL because `@/lib/skills/curated-packaging-intent` does not exist yet.

- [ ] **Step 3: Create the repo-local skill folder and metadata**

Create these files with minimal valid contents:

`.cursor/skills/curated-packaging/SKILL.md`

```md
---
name: curated-packaging
description: Use when the user asks in natural language to preview or run Curated packaging workflows in this repository, including requests like 打生产包, 打整机包, 打安装包, 只打安装包, 打便携包, 只打便携包, 预览这次打包版本, or requests to bump major/minor before packaging.
---

# Curated Packaging

Use this skill only inside the Curated repository.

## Workflow

1. Detect the request mode from natural language.
2. Preview the action first.
3. Tell the user:
   - detected mode
   - current base version
   - predicted version
   - commands to be executed
   - expected artifacts
4. Execute only after showing the preview.

## Modes

- `publish`
- `installer`
- `portable`
- `preview`
- `set-base`

## Version Rules

- Read `scripts/release/version.json` as the only automatic product-version source.
- Respect the current `major.minor.patch` rules already implemented in `scripts/release/`.
- Do not invent a parallel versioning system.
- `publish` must allocate one version once and reuse it for all artifacts.

## Scripts

- Use `scripts/preview-package.ps1` for previews.
- Use `scripts/execute-package.ps1` for execution.
```

`.cursor/skills/curated-packaging/agents/openai.yaml`

```yaml
display_name: Curated Packaging
short_description: Preview and run Curated packaging workflows inside this repository.
default_prompt: Preview or run Curated packaging commands from natural-language requests while respecting the repository release version rules.
```

Create empty script placeholders:

`.cursor/skills/curated-packaging/scripts/preview-package.ps1`

```powershell
[CmdletBinding()]
param()

throw "Not implemented"
```

`.cursor/skills/curated-packaging/scripts/execute-package.ps1`

```powershell
[CmdletBinding()]
param()

throw "Not implemented"
```

No `.gitignore` change is required because `.cursor/skills/` is already a normal tracked path in this repository.

- [ ] **Step 4: Run a quick existence check**

Run:

```powershell
Get-ChildItem .\.cursor\skills\curated-packaging -Recurse
```

Expected: `SKILL.md`, `agents/openai.yaml`, and the two script files exist.

- [ ] **Step 5: Commit**

```bash
git add .cursor/skills/curated-packaging/SKILL.md .cursor/skills/curated-packaging/agents/openai.yaml .cursor/skills/curated-packaging/scripts/preview-package.ps1 .cursor/skills/curated-packaging/scripts/execute-package.ps1 src/lib/skills/curated-packaging-intent.test.ts
git commit -m "feat: scaffold curated packaging skill"
```

### Task 2: Implement Natural-Language Intent Detection

**Files:**
- Create: `src/lib/skills/curated-packaging-intent.ts`
- Modify: `src/lib/skills/curated-packaging-intent.test.ts`

- [ ] **Step 1: Expand the failing test cases**

Replace the test file contents with:

```ts
import { describe, expect, it } from "vitest"

import { detectPackagingIntent } from "@/lib/skills/curated-packaging-intent"

describe("curated packaging intent detection", () => {
  it("maps publish requests to publish mode", () => {
    expect(detectPackagingIntent("打生产包")).toEqual({
      mode: "publish",
      baseChange: null,
    })
  })

  it("maps installer-only requests to installer mode", () => {
    expect(detectPackagingIntent("只打安装包")).toEqual({
      mode: "installer",
      baseChange: null,
    })
  })

  it("maps portable-only requests to portable mode", () => {
    expect(detectPackagingIntent("只打便携包")).toEqual({
      mode: "portable",
      baseChange: null,
    })
  })

  it("maps preview requests to preview mode", () => {
    expect(detectPackagingIntent("预览这次打包版本")).toEqual({
      mode: "preview",
      baseChange: null,
    })
  })

  it("captures base-change requests before packaging", () => {
    expect(detectPackagingIntent("把 minor 升到 2 再打生产包")).toEqual({
      mode: "publish",
      baseChange: {
        major: null,
        minor: 2,
      },
    })
  })
})
```

- [ ] **Step 2: Run the test to verify it fails correctly**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-intent.test.ts --configLoader native --pool threads
```

Expected: FAIL because `detectPackagingIntent` is not implemented.

- [ ] **Step 3: Write the minimal implementation**

Create `src/lib/skills/curated-packaging-intent.ts`:

```ts
export type PackagingMode = "publish" | "installer" | "portable" | "preview" | "set-base"

export interface PackagingBaseChange {
  major: number | null
  minor: number | null
}

export interface PackagingIntent {
  mode: PackagingMode
  baseChange: PackagingBaseChange | null
}

function parseBaseChange(input: string): PackagingBaseChange | null {
  const majorMatch = input.match(/major\s*升到\s*(\d+)/i)
  const minorMatch = input.match(/minor\s*升到\s*(\d+)/i)

  if (!majorMatch && !minorMatch) {
    return null
  }

  return {
    major: majorMatch ? Number(majorMatch[1]) : null,
    minor: minorMatch ? Number(minorMatch[1]) : null,
  }
}

export function detectPackagingIntent(input: string): PackagingIntent {
  const text = input.trim()
  const baseChange = parseBaseChange(text)

  if (text.includes("预览")) {
    return { mode: "preview", baseChange }
  }

  if (text.includes("安装包")) {
    return { mode: "installer", baseChange }
  }

  if (text.includes("便携包")) {
    return { mode: "portable", baseChange }
  }

  if (text.includes("生产包") || text.includes("整机包")) {
    return { mode: "publish", baseChange }
  }

  if (baseChange) {
    return { mode: "set-base", baseChange }
  }

  return { mode: "preview", baseChange: null }
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-intent.test.ts --configLoader native --pool threads
```

Expected: PASS with 5 tests passing.

- [ ] **Step 5: Commit**

```bash
git add src/lib/skills/curated-packaging-intent.ts src/lib/skills/curated-packaging-intent.test.ts
git commit -m "feat: detect curated packaging intents"
```

### Task 3: Build Deterministic Preview Logic

**Files:**
- Create: `src/lib/skills/curated-packaging-preview.ts`
- Create: `src/lib/skills/curated-packaging-preview.test.ts`
- Modify: `.cursor/skills/curated-packaging/scripts/preview-package.ps1`

- [ ] **Step 1: Write the failing preview tests**

Create `src/lib/skills/curated-packaging-preview.test.ts`:

```ts
import { describe, expect, it } from "vitest"

import { buildPackagingPreview } from "@/lib/skills/curated-packaging-preview"

describe("curated packaging preview", () => {
  it("predicts the next patch version for installer mode", () => {
    expect(
      buildPackagingPreview({
        mode: "installer",
        currentBaseVersion: "1.1.0",
        baseChange: null,
      }),
    ).toEqual({
      mode: "installer",
      currentBaseVersion: "1.1.0",
      predictedVersion: "1.1.1",
      willBumpPatch: true,
      baseVersionAfterChange: "1.1.0",
    })
  })

  it("predicts the post-base-change version for publish mode", () => {
    expect(
      buildPackagingPreview({
        mode: "publish",
        currentBaseVersion: "1.1.0",
        baseChange: {
          major: null,
          minor: 2,
        },
      }),
    ).toEqual({
      mode: "publish",
      currentBaseVersion: "1.1.0",
      predictedVersion: "1.2.1",
      willBumpPatch: true,
      baseVersionAfterChange: "1.2.0",
    })
  })
})
```

- [ ] **Step 2: Run the preview test to verify it fails**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-preview.test.ts --configLoader native --pool threads
```

Expected: FAIL because `buildPackagingPreview` does not exist.

- [ ] **Step 3: Write the minimal preview helper**

Create `src/lib/skills/curated-packaging-preview.ts`:

```ts
import type { PackagingBaseChange, PackagingMode } from "@/lib/skills/curated-packaging-intent"

export interface PackagingPreviewInput {
  mode: PackagingMode
  currentBaseVersion: string
  baseChange: PackagingBaseChange | null
}

export interface PackagingPreview {
  mode: PackagingMode
  currentBaseVersion: string
  predictedVersion: string
  willBumpPatch: boolean
  baseVersionAfterChange: string
}

function splitVersion(version: string) {
  const [major, minor, patch] = version.split(".").map(Number)
  return { major, minor, patch }
}

export function buildPackagingPreview(input: PackagingPreviewInput): PackagingPreview {
  const current = splitVersion(input.currentBaseVersion)
  const nextBase = {
    major: input.baseChange?.major ?? current.major,
    minor: input.baseChange?.minor ?? current.minor,
    patch: input.baseChange ? 0 : current.patch,
  }

  const willBumpPatch = input.mode !== "preview" && input.mode !== "set-base"
  const predictedPatch = willBumpPatch ? nextBase.patch + 1 : nextBase.patch
  const baseVersionAfterChange = `${nextBase.major}.${nextBase.minor}.${nextBase.patch}`

  return {
    mode: input.mode,
    currentBaseVersion: input.currentBaseVersion,
    predictedVersion: `${nextBase.major}.${nextBase.minor}.${predictedPatch}`,
    willBumpPatch,
    baseVersionAfterChange,
  }
}
```

- [ ] **Step 4: Wire the PowerShell preview script to the helper output shape**

Replace `.cursor/skills/curated-packaging/scripts/preview-package.ps1` with:

```powershell
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [string]$Mode,

  [Parameter(Mandatory = $true)]
  [string]$CurrentBaseVersion,

  [string]$RequestedMajor,

  [string]$RequestedMinor
)

$majorValue = if ([string]::IsNullOrWhiteSpace($RequestedMajor)) { $null } else { [int]$RequestedMajor }
$minorValue = if ([string]::IsNullOrWhiteSpace($RequestedMinor)) { $null } else { [int]$RequestedMinor }

$currentParts = $CurrentBaseVersion.Split(".")
$major = if ($null -ne $majorValue) { $majorValue } else { [int]$currentParts[0] }
$minor = if ($null -ne $minorValue) { $minorValue } else { [int]$currentParts[1] }
$basePatch = if ($null -ne $majorValue -or $null -ne $minorValue) { 0 } else { [int]$currentParts[2] }
$willBumpPatch = $Mode -ne "preview" -and $Mode -ne "set-base"
$predictedPatch = if ($willBumpPatch) { $basePatch + 1 } else { $basePatch }

$result = [ordered]@{
  mode = $Mode
  currentBaseVersion = $CurrentBaseVersion
  baseVersionAfterChange = "$major.$minor.$basePatch"
  predictedVersion = "$major.$minor.$predictedPatch"
  willBumpPatch = $willBumpPatch
}

$result | ConvertTo-Json -Depth 5
```

- [ ] **Step 5: Run the preview tests to verify they pass**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-preview.test.ts --configLoader native --pool threads
```

Expected: PASS with 2 tests passing.

- [ ] **Step 6: Commit**

```bash
git add src/lib/skills/curated-packaging-preview.ts src/lib/skills/curated-packaging-preview.test.ts .cursor/skills/curated-packaging/scripts/preview-package.ps1
git commit -m "feat: preview curated packaging actions"
```

### Task 4: Implement The Execute Script

**Files:**
- Modify: `.cursor/skills/curated-packaging/scripts/execute-package.ps1`

- [ ] **Step 1: Write a smoke-check script expectation**

Document the expected commands inside the execution script as the initial failing target:

```powershell
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("publish", "installer", "portable", "set-base")]
  [string]$Mode,

  [int]$Major,

  [int]$Minor
)

throw "execute-package.ps1 has not been implemented yet"
```

- [ ] **Step 2: Run the script to verify it fails**

Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\.cursor\skills\curated-packaging\scripts\execute-package.ps1 -Mode portable
```

Expected: FAIL with `execute-package.ps1 has not been implemented yet`.

- [ ] **Step 3: Write the minimal execution router**

Replace the file contents with:

```powershell
[CmdletBinding()]
param(
  [Parameter(Mandatory = $true)]
  [ValidateSet("publish", "installer", "portable", "set-base")]
  [string]$Mode,

  [int]$Major,

  [int]$Minor
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..\..\..\..")).Path

Push-Location $repoRoot
try {
  switch ($Mode) {
    "publish" {
      & pnpm release:publish
      break
    }
    "installer" {
      & pnpm release:installer
      break
    }
    "portable" {
      & pnpm release:portable
      break
    }
    "set-base" {
      if ($Major -lt 0 -or $Minor -lt 0) {
        throw "Mode set-base requires -Major and -Minor."
      }
      & pnpm release:version:set-base -- --Major $Major --Minor $Minor
      break
    }
  }

  if ($LASTEXITCODE -ne 0) {
    throw "Packaging command failed with exit code $LASTEXITCODE"
  }
}
finally {
  Pop-Location
}
```

- [ ] **Step 4: Run a safe smoke check**

Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\.cursor\skills\curated-packaging\scripts\execute-package.ps1 -Mode set-base -Major 1 -Minor 1
```

Expected: PASS and print the existing `release:version:set-base` output.

- [ ] **Step 5: Commit**

```bash
git add .cursor/skills/curated-packaging/scripts/execute-package.ps1
git commit -m "feat: execute curated packaging workflows"
```

### Task 5: Finalize The Skill Instructions And Verify End-To-End

**Files:**
- Modify: `.cursor/skills/curated-packaging/SKILL.md`
- Modify: `.cursor/skills/curated-packaging/agents/openai.yaml`
- Modify: `docs/plan/2026-04-12-curated-packaging-skill-design.md`

- [ ] **Step 1: Expand the skill instructions to include the final workflow**

Update `.cursor/skills/curated-packaging/SKILL.md` so it explicitly says:

```md
## Request Handling

- Map natural-language packaging requests to one of: `publish`, `installer`, `portable`, `preview`, `set-base`.
- Always preview before execution.
- Preview must include:
  - detected mode
  - current base version
  - predicted version
  - commands to run
  - expected artifacts
  - whether patch will bump
- If the request includes a base change, preview the base change first and then the packaging result.

## Execution

- Use `scripts/preview-package.ps1` for the preview output.
- Use `scripts/execute-package.ps1` for execution.
- Reuse repository packaging commands rather than duplicating logic.
```

Keep the frontmatter description aligned with the final trigger surface.

- [ ] **Step 2: Run targeted verification commands**

Run:

```bash
pnpm exec vitest run src/lib/skills/curated-packaging-intent.test.ts src/lib/skills/curated-packaging-preview.test.ts --configLoader native --pool threads
```

Expected: PASS with all tests green.

Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\.cursor\skills\curated-packaging\scripts\preview-package.ps1 -Mode publish -CurrentBaseVersion 1.1.0 -RequestedMinor 2
```

Expected JSON includes:

```json
{
  "mode": "publish",
  "baseVersionAfterChange": "1.2.0",
  "predictedVersion": "1.2.1",
  "willBumpPatch": true
}
```

- [ ] **Step 3: Self-review the skill against the design doc**

Check:

- the skill is repo-local only
- the skill supports natural-language packaging requests
- the skill previews before execution
- the skill respects `scripts/release/version.json`
- the skill does not allocate two versions for `publish`

If any of those are missing, update the skill or scripts before moving on.

- [ ] **Step 4: Commit**

```bash
git add .cursor/skills/curated-packaging/SKILL.md .cursor/skills/curated-packaging/agents/openai.yaml .cursor/skills/curated-packaging/scripts/preview-package.ps1 .cursor/skills/curated-packaging/scripts/execute-package.ps1 docs/plan/2026-04-12-curated-packaging-skill-design.md
git commit -m "feat: add curated packaging skill"
```

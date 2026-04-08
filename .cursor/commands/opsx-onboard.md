---
name: /opsx:onboard
id: opsx-onboard
category: Workflow
description: Onboard this repo to the OpenSpec + opsx workflow (CLI, config, Cursor)
---

Onboard **Curated** (`jav-shadcn`) to the **OpenSpec** spec-driven workflow and Cursor slash commands (`/opsx:*`).

**Input**: Optional. User may specify a focus (e.g. "only CLI", "fill context") or leave empty for the full checklist.

---

**Steps**

1. **Verify OpenSpec CLI**

   From the **repository root** (where `openspec/config.yaml` lives):

   ```bash
   openspec --version
   openspec list
   ```

   If the command is missing, install globally (example): `npm i -g @fission-ai/openspec` — use the package name that matches the user’s environment.

2. **Sync instruction files (safe to repeat)**

   ```bash
   openspec update .
   ```

   Use `openspec update . --force` only if the user explicitly wants to overwrite local instruction copies.

   **Note:** `openspec update` may sync Cursor commands from the OpenSpec schema and **remove** custom slash-command files not in the schema. If you maintain extra commands (e.g. this file), re-add them or pin workflow; keep them under version control.

3. **First-time OpenSpec in this clone (only if `openspec/` is incomplete)**

   ```bash
   openspec init . --tools cursor
   ```

   Skip or adapt if the repo already has `openspec/config.yaml` and `.cursor/commands/opsx-*.md` and the user does not want scaffolding refreshed.

4. **Project context for AI artifacts**

   Open `openspec/config.yaml`. If `context:` is empty, suggest filling it with a short pointer to in-repo sources (do **not** duplicate long rules):

   - `AGENTS.md` — agent entry and rule index  
   - `.cursor/rules/workspace-quick-reference.mdc`, `project-facts.mdc` — daily commands and API facts  
   - Product: **Curated**; stack: Vue 3 + Vite + Go backend under `backend/`

   Keep `context` concise (stack, where specs live, commit conventions if relevant).

5. **Workflow commands available in this repo**

   | Command | Purpose |
   |---------|---------|
   | `/opsx:explore` | Think through a problem before proposing |
   | `/opsx:propose` | Create a change under `openspec/changes/<name>/` |
   | `/opsx:apply` | Implement from `tasks.md` |
   | `/opsx:archive` | Archive a completed change |

6. **Sanity check**

   ```bash
   openspec validate
   ```

   If the schema or paths differ, follow the CLI error output.

---

**Output**

- Summarize what was verified or what the user should run locally.
- If anything is ambiguous (e.g. `openspec init` vs `update`), ask before overwriting files.

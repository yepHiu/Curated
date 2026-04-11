# Curated README And API Docs Redesign

## 1. Goal

Restructure the repository's public-facing documentation so it reads like a polished GitHub open source project instead of an internal engineering note set.

This redesign must:

- move the HTTP API reference out of `README.md`
- introduce a dedicated root-level `API.md` as the single public API reference
- keep `README.md` as the English primary entrypoint
- add full translated README variants for Simplified Chinese and Japanese
- make future API and README maintenance rules explicit in project memory and agent-facing docs

## 2. Approved Structure

The approved documentation structure is:

- `README.md`
  - English primary README
  - public GitHub homepage entry
- `README.zh-CN.md`
  - full Simplified Chinese translation of the README
- `README.ja-JP.md`
  - full Japanese translation of the README
- `API.md`
  - English-only public HTTP API reference

The README files will remain at the repository root so language switching and API discovery are obvious on GitHub.

## 3. README Responsibilities

The README files are responsible for:

- project identity and visual presentation
- concise product positioning
- stack and architecture overview
- quick start and development entrypoints
- feature highlights
- configuration summary
- release / packaging summary
- links to `API.md` and deeper docs

The README files are not responsible for carrying the full HTTP API table anymore.

Instead, each README will include a short API section that links to `API.md`.

## 4. API.md Responsibilities

`API.md` becomes the single public API reference document for the repository.

It will contain:

- API overview
- base paths and runtime expectations
- conventions such as pagination, async tasks, and DTO references
- grouped endpoint sections
- endpoint purposes and important request/response notes
- pointers to backend and frontend type sources

The document should be organized by capability group instead of being a flat dump from the router table.

Recommended groups:

- Health
- Movies
- Playback
- Actors
- Settings
- Curated Frames
- Scans and Tasks

## 5. README Presentation Style

The new README style should be closer to polished GitHub open source projects.

### 5.1 Top Section

The top section should include:

- the existing branded logo image from `icon/curated-title-nobg.png`
- a concise one-line product description
- language switch links:
  - English
  - 简体中文
  - 日本語
- small technology badges in a GitHub-friendly style

Recommended badges:

- Vue 3
- TypeScript
- Vite
- Go
- SQLite
- Tailwind CSS v4
- shadcn-vue
- Windows

Optional product-state badges may also be added if they stay stable and factual.

### 5.2 Content Flow

The README files should follow one consistent structure across all languages:

1. Branding and language switch
2. Project overview
3. Highlights
4. Quick start
5. Features
6. Configuration
7. API link section
8. Repository layout
9. Release / packaging summary
10. Documentation links
11. Notes / roadmap boundary

All three README files should preserve the same section order and roughly the same content scope.

## 6. Translation Rule

`README.zh-CN.md` and `README.ja-JP.md` are full translations, not abridged summaries.

This means:

- same structural sections as English
- same main technical claims
- same linked documentation set
- same command examples unless a language-specific explanation is necessary

Natural translation is allowed, but the documents must not drift in scope or omit important sections.

## 7. Maintenance Rule

The following maintenance rule is approved:

- API changes must update `API.md`
- public onboarding or feature-summary changes must update all three README files when relevant
- `README.md` must not become the full API reference again

This rule must be written into project memory so future agents treat it as a standing convention.

## 8. Project Memory Updates

The implementation must update the following memory / instruction surfaces:

- `.cursor/rules/project-facts.mdc`
  - state that root `API.md` is the single public API reference
  - state that README no longer carries the full API table
- `.cursor/rules/workspace-quick-reference.mdc`
  - state that API changes require `API.md` updates
  - state that README changes with user-facing impact must be reflected in all language variants
- `docs/2026-03-20-project-memory.md`
  - record the public documentation structure:
    - `README.md`
    - `README.zh-CN.md`
    - `README.ja-JP.md`
    - `API.md`
- `CLAUDE.md`
  - add a short maintenance note so future code/documentation work does not move API details back into README

## 9. Implementation Outline

Once implementation starts, the work should proceed in this order:

1. create `API.md` by moving and restructuring the existing API content from `README.md`
2. redesign `README.md` into the new English GitHub-style layout
3. create `README.zh-CN.md` as a full Chinese translation
4. create `README.ja-JP.md` as a full Japanese translation
5. update project memory and agent-facing docs with the maintenance rule
6. verify links, paths, and section consistency

## 10. Non-Goals

This redesign does not attempt to:

- redesign the product UI
- add screenshots that do not yet exist
- create multilingual API references in this phase
- rewrite deep architecture docs outside the README / API boundary

## 11. Approval Gate

Implementation should begin only after this design document is reviewed and approved.

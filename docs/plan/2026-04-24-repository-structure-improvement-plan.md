# Repository Structure Improvement Plan

> Scope: repository directory layout review for `Curated` (`jav-shadcn` repo). This document focuses on structure, discoverability, and maintenance cost. It does not require immediate code moves unless explicitly scheduled.

## 1. Current Snapshot

Based on the current workspace state on `2026-04-24`:

- Top-level directories currently visible in the working tree: `24`
- Top-level directories with tracked repository files: `9`
- Recursive directory count in the current workspace: `5256`
- Recursive file count in the current workspace: `35394`

Important interpretation:

- The recursive counts are heavily inflated by local caches and generated assets, so they are **not** a good measure of repository architecture quality by themselves.
- From a tracked-files perspective, the repository is mainly concentrated in:
  - `docs/` (`380` tracked files)
  - `src/` (`284` tracked files)
  - `backend/` (`230` tracked files)
  - `.cursor/` (`31` tracked files)
  - `scripts/` (`17` tracked files)

Additional notable distribution:

- `docs/film-scanner/` alone contains `291` tracked files, which means a large share of the repository's tracked documentation area is currently occupied by a runnable subproject / experiment rather than product docs.
- `src/components/` contains `135` files and `src/lib/` contains `90` files, which suggests the frontend is starting to spread business concerns across multiple technical-layer directories.
- `backend/internal/storage/` contains `51` files, making it one of the densest backend areas.

## 2. High-Level Assessment

### What is already reasonable

- The repo still has a clear primary split between `src/` and `backend/`. That is good and should stay.
- `scripts/` already has intent-based grouping (`dev/`, `prd/`, `release/`), which is the right direction.
- Frontend shared UI is separated into `src/components/ui/`, which is a good long-term boundary and should remain.
- Backend keeps Go implementation under `backend/internal/`, which is idiomatic and maintainable if grouping continues to stay disciplined.

### What is currently weak

1. The **root directory is too noisy** for daily navigation.
2. `docs/` is carrying too many different responsibilities.
3. Frontend code is increasingly organized by **technical layer**, not by **feature**.
4. Backend `internal/` package count is growing, but package grouping is still only partly domain-oriented.
5. Local caches and runtime artifacts are being created inside the repo, even though project docs already say this should be avoided.

## 3. Main Issues

### Issue A: Root directory mixes source, runtime, cache, and workflow state

Current root contains source directories together with local-only directories such as:

- `.codex-temp/`
- `.gocache/`
- `.pnpm-store/`
- `.tmp/`
- `dist/`
- `log/`
- `output/`
- `release/`
- `videos_test/`
- `openspec/`
- `config/`

Even if most of them are ignored by Git, they still reduce scanability for humans and agents. A root that requires filtering before understanding is more expensive to work in.

Exception noted after review:

- Root `videos_test/` is treated as a fixed local test-fixture directory and should remain in place. It is a root-noise example for navigation cost, but it is **not** a relocation target in this plan.

### Issue B: `docs/` is overloaded

`docs/` currently contains all of the following:

- long-lived architecture/product docs
- review and audit docs
- implementation plans
- release notes
- PRD material
- runnable experiment/tool content in `docs/film-scanner/`
- standalone HTML prototypes mixed into `docs/plan/`

This is the biggest structural problem in the tracked tree. The directory name says "documentation", but part of the content is actually tool code, experiments, and disposable prototypes.

### Issue C: Frontend business logic is split across too many layers

The current frontend layout is still workable, but feature code is spread across:

- `src/views/`
- `src/components/jav-library/`
- `src/composables/`
- `src/lib/`
- `src/api/`
- `src/services/`
- `src/domain/`

That organization is fine in early and mid growth, but it gets expensive once features like homepage recommendations, actor profiles, player pipeline, history, curated frames, and settings all evolve in parallel.

### Issue D: Backend package naming is mostly good, but grouping can become clearer

The backend already has real domains such as:

- `library`
- `playback`
- `scanner`
- `scraper`
- `desktop`
- `server`
- `storage`

But it also has many small infrastructure or cross-cutting packages at the same level:

- `assets`
- `browserheaders`
- `config`
- `executil`
- `logging`
- `proxyenv`
- `shellopen`
- `version`
- `webui`

Flattening all of these into one `internal/` namespace makes the package list longer and less self-explanatory over time.

### Issue E: Repo-local cache policy is not enforced strongly enough

The current workspace contains repo-local cache or runtime areas such as:

- root `.gocache/`
- `backend/.gocache/`
- `backend/.tmp-go/`
- `backend/runtime/`

`backend/runtime/` is intentional for dev binaries and runtime data, but repo-local Go build caches are specifically the kind of thing your docs already warn against. This is not only cosmetic; it also inflates file counts and makes tooling slower.

## 4. Recommended Direction

I recommend a **conservative, staged reorganization** rather than a one-shot repo rewrite.

Why this is the right approach:

- The core top-level layout is not fundamentally wrong.
- The real problems are concentration, naming, and misplaced content.
- A phased cleanup gives most of the benefit without creating a high-risk move-everything migration.

## 5. Target Structure Principles

### Root-level principles

The root should answer three questions within a few seconds:

1. Where is product code?
2. Where is documentation?
3. Where do local artifacts go?

Recommended root intent:

- Keep as primary tracked roots:
  - `src/`
  - `backend/`
  - `docs/`
  - `scripts/`
  - `public/`
  - `icon/`
- Keep config files at root only when they are truly repo-wide (`package.json`, `vite.config.ts`, `components.json`, `README*`, `API.md`, rules docs).
- Move experiments and internal tooling out of `docs/` into a purpose-named area such as:
  - `tools/`
  - or `experiments/`
- Consolidate local-only folders under one obvious ignored namespace if possible:
  - `.local/`
  - or `.workspace/`

### Docs principles

`docs/` should only contain documents and lightweight reference assets, not runnable side projects.

Recommended substructure:

- `docs/architecture/`
- `docs/guides/`
- `docs/plan/`
- `docs/prd/`
- `docs/reviews/`
- `docs/release-notes/`
- `docs/reference/`

This does not need a big-bang rename. It can be done incrementally.

### Frontend principles

Keep shared infrastructure shared, but move feature code closer together.

Recommended end state:

- `src/features/library/`
- `src/features/actors/`
- `src/features/player/`
- `src/features/history/`
- `src/features/home/`
- `src/features/curated-frames/`
- `src/features/settings/`
- `src/shared/ui/` or keep `src/components/ui/`
- `src/shared/lib/` or keep `src/lib/` for cross-feature utilities
- `src/app/` for router, shell, providers, bootstrapping

This should be done feature-by-feature, not by moving the entire frontend at once.

### Backend principles

Do not rewrite idiomatic Go package boundaries just for aesthetic consistency. Group only where the grouping improves comprehension.

Recommended direction:

- keep domain packages domain-oriented
- introduce one clearer home for platform/infrastructure concerns
- avoid leaving every small utility package directly under `internal/`

Example direction:

```text
backend/internal/
  app/
  domain/
    library/
    playback/
    scanner/
    scraper/
    desktop/
    appupdate/
  platform/
    config/
    logging/
    assets/
    executil/
    proxyenv/
    shellopen/
    version/
    webui/
  storage/
  server/
  contracts/
```

This is a direction, not a forced exact layout. If `storage/` or `server/` remain top-level, that is still acceptable.

## 6. Concrete Improvement Plan

### Phase 1: Root cleanup and directory policy

Priority: highest  
Risk: low  
Goal: make the repository understandable at a glance

Actions:

1. Define a root-directory policy in one short doc section:
   - tracked source/documentation directories
   - allowed runtime directories
   - local-only ignored directories
   - explicit fixed exceptions such as root `videos_test/`
2. Stop generating Go caches inside the repo:
   - remove repo-local cache generation paths
   - align tooling with the existing build/test policy doc
3. Audit ignored local folders and consolidate where practical:
   - prefer one ignored workspace area over many unrelated root folders
4. Decide whether `config/` should remain repo-root local-only or move under a clearer ignored local namespace.

Expected outcome:

- fewer misleading root folders
- cleaner `Get-ChildItem` / file explorer view
- lower accidental confusion for agents and contributors

### Phase 2: Fix `docs/` overload

Priority: highest  
Risk: low to medium  
Goal: make `docs/` mean documentation again

Actions:

1. Move `docs/film-scanner/` to a non-docs area:
   - recommended: `tools/film-scanner/`
   - alternative: `experiments/film-scanner/`
2. Move one-off prototype HTML files out of `docs/plan/`:
   - recommended: `docs/prototypes/`
   - or `experiments/ui-prototypes/`
3. Introduce docs taxonomy:
   - architecture
   - guides
   - plan
   - prd
   - reviews
   - release notes
4. Add a small README or index doc inside `docs/` describing where new docs belong.

Expected outcome:

- `docs/` becomes easier to search
- fewer "where should this document go?" decisions
- reduced mixing of durable docs and temporary artifacts

### Phase 3: Frontend feature-oriented consolidation

Priority: medium  
Risk: medium  
Goal: reduce cross-directory hopping when changing one product feature

Actions:

1. Freeze the current shared layers:
   - keep `src/components/ui/`
   - keep global app shell/router/bootstrap separate
2. Create `src/features/` for new or actively evolving product areas.
3. Migrate one feature at a time, starting with the most scattered areas:
   - `actors`
   - `player`
   - `curated-frames`
   - `home`
4. For each migrated feature, co-locate:
   - feature page/view
   - feature components
   - feature composables
   - feature mappers/types when feature-specific
5. Keep only truly cross-feature utilities in `src/lib/` and `src/services/`.

Expected outcome:

- faster feature work
- lower coupling between unrelated UI areas
- fewer oversized "misc" shared utility directories

### Phase 4: Backend package taxonomy refinement

Priority: medium  
Risk: medium  
Goal: improve backend discoverability without destabilizing package imports

Actions:

1. Identify packages that are clearly:
   - domain
   - transport
   - platform/infrastructure
   - persistence
2. Group infrastructure packages more explicitly.
3. Avoid moving stable high-traffic packages unless the grouping meaningfully improves navigation.
4. Split only when a directory is large **and** semantically mixed.

Expected outcome:

- clearer mental model for backend contributors
- better package scanability in `backend/internal/`
- lower risk than a full package tree rewrite

## 7. What I Would Not Change Right Now

These areas look acceptable and should not be changed just to "look cleaner":

- Do not merge frontend and backend into one hybrid app directory.
- Do not remove `backend/` as the boundary for Go code.
- Do not dissolve `src/components/ui/`; shared UI primitives still deserve a stable home.
- Do not move root `videos_test/`; treat it as an approved fixed-location test fixture directory.
- Do not optimize around total recursive directory count; that number is distorted by caches and runtime artifacts.
- Do not aggressively rename Go packages that already map cleanly to business capabilities.

## 8. Suggested Execution Order

Recommended order:

1. Root cleanup policy
2. `docs/` cleanup
3. Frontend feature-based migration for one pilot feature
4. Backend taxonomy refinement only after the first three are stable

This order gives visible gains early and avoids large-scale churn before naming and ownership rules are clear.

## 9. Decision Options

### Option A: Conservative cleanup only

Do:

- root cleanup
- docs cleanup
- no major source moves

Best when:

- you want immediate clarity with minimal risk

### Option B: Balanced cleanup plus selective source reorg

Do:

- root cleanup
- docs cleanup
- frontend `src/features/` pilot migration
- light backend grouping improvements

Best when:

- you want noticeable long-term payoff without a rewrite

### Option C: Full structural refactor

Do:

- root cleanup
- docs cleanup
- broad frontend move
- broad backend regrouping

Best when:

- you are willing to pay a temporary slowdown for a deeper reset

Recommended choice: **Option B**

It gives the best ratio of clarity to migration risk for the current repository shape.

## 10. Summary

The repository does **not** have a fundamentally broken top-level architecture. The biggest issues are:

1. root noise from local/generated directories
2. `docs/` being overloaded, especially by `docs/film-scanner/`
3. frontend feature code spreading across technical layers
4. backend `internal/` taxonomy needing clearer grouping over time

So the right move is not "rebuild the whole directory tree". The right move is:

- clean the root
- make `docs/` honest
- start a gradual feature-based frontend layout
- refine backend grouping only where it actually improves comprehension

# Curated Frames Tag Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a visible top tag filter area to the curated frames page so users can see existing frame tags and click one to filter matching frames.

**Architecture:** Keep the filter state in the curated frames route query as `cft`, separate from the existing full-text search query `cfq`. Use a small pure helper to build tag facets from loaded rows in Mock mode, and reuse the existing backend `/api/curated-frames/tags` facet endpoint in Web API mode.

**Tech Stack:** Vue 3, TypeScript, vue-router, Vitest, existing shadcn-vue `Badge` and `Button` UI components.

---

### Task 1: Add Curated Frame Tag Query Helpers

**Files:**
- Modify: `src/lib/library-query.ts`
- Test: `src/lib/library-query.test.ts`

- [x] **Step 1: Write the failing test**

Add a test proving `cft` is independent from `cfq`, can be read, set, and cleared:

```ts
it("reads and merges curated frame tag filter separately from curated search", () => {
  expect(getCuratedFrameTagQuery({ cft: "pose", cfq: "abc" })).toBe("pose")
  expect(getCuratedFrameTagQuery({})).toBe("")

  const merged = mergeCuratedFramesQuery(
    { cfq: "abc", cft: "old" },
    { cft: "new" },
  )
  expect(merged).toEqual({ cfq: "abc", cft: "new" })

  const cleared = mergeCuratedFramesQuery(merged, { cft: "" })
  expect(cleared).toEqual({ cfq: "abc" })
})
```

- [x] **Step 2: Run the focused test and verify RED**

Run: `pnpm test -- src/lib/library-query.test.ts`

Expected: FAIL because `getCuratedFrameTagQuery` is not exported and `mergeCuratedFramesQuery` does not handle `cft`.

- [x] **Step 3: Implement helper support**

Update `src/lib/library-query.ts`:

```ts
export const getCuratedFrameTagQuery = (query: LocationQuery) =>
  typeof query.cft === "string" ? query.cft : ""

export const mergeCuratedFramesQuery = (
  sourceQuery: LocationQuery,
  patch: Partial<{ cfq: string | undefined; cft: string | undefined }>,
) => {
  const nextQuery: LocationQuery = { ...sourceQuery }
  const apply = (key: "cfq" | "cft", value: string | undefined) => {
    const t = value?.trim()
    if (t) {
      nextQuery[key] = t
    } else {
      delete nextQuery[key]
    }
  }
  if (hasOwnKey(patch, "cfq")) apply("cfq", patch.cfq)
  if (hasOwnKey(patch, "cft")) apply("cft", patch.cft)
  return nextQuery
}
```

- [x] **Step 4: Run focused test and verify GREEN**

Run: `pnpm test -- src/lib/library-query.test.ts`

Expected: PASS.

### Task 2: Add Tag Facet Helper

**Files:**
- Create: `src/lib/curated-frames/tag-facets.ts`
- Create: `src/lib/curated-frames/tag-facets.test.ts`
- Modify: `src/lib/curated-frames/db.ts`

- [x] **Step 1: Write the failing test**

Create `src/lib/curated-frames/tag-facets.test.ts`:

```ts
import { describe, expect, it } from "vitest"
import { buildCuratedFrameTagFacets, visibleCuratedFrameTagFacets } from "./tag-facets"

describe("curated frame tag facets", () => {
  it("counts trimmed tags and sorts by count then locale", () => {
    const facets = buildCuratedFrameTagFacets(
      [
        { tags: [" pose ", "制服"] },
        { tags: ["pose", "构图"] },
        { tags: ["构图", ""] },
      ],
      "zh-CN",
    )

    expect(facets).toEqual([
      { name: "构图", count: 2 },
      { name: "pose", count: 2 },
      { name: "制服", count: 1 },
    ])
  })

  it("returns a preview list until expanded", () => {
    const facets = Array.from({ length: 18 }, (_, index) => ({
      name: `tag-${index + 1}`,
      count: 1,
    }))

    expect(visibleCuratedFrameTagFacets(facets, 16, false)).toHaveLength(16)
    expect(visibleCuratedFrameTagFacets(facets, 16, true)).toHaveLength(18)
  })
})
```

- [x] **Step 2: Run the focused test and verify RED**

Run: `pnpm test -- src/lib/curated-frames/tag-facets.test.ts`

Expected: FAIL because the helper file does not exist yet.

- [x] **Step 3: Implement helper and DB facet function**

Create `src/lib/curated-frames/tag-facets.ts`:

```ts
import type { CuratedFrameFacetItemDTO } from "@/api/types"

export function buildCuratedFrameTagFacets(
  rows: ReadonlyArray<{ tags: readonly string[] }>,
  locale: string,
): CuratedFrameFacetItemDTO[] {
  const counts = new Map<string, number>()
  for (const row of rows) {
    for (const raw of row.tags) {
      const tag = raw.trim()
      if (!tag) continue
      counts.set(tag, (counts.get(tag) ?? 0) + 1)
    }
  }
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, locale, { numeric: true }))
}

export function visibleCuratedFrameTagFacets<T>(
  facets: readonly T[],
  limit: number,
  expanded: boolean,
): T[] {
  return expanded ? [...facets] : facets.slice(0, limit)
}
```

Add `listCuratedFrameTagFacets(locale = "zh-CN")` to `src/lib/curated-frames/db.ts`, calling `api.listCuratedFrameTags()` in Web API mode and `buildCuratedFrameTagFacets` from all local rows in Mock mode.

- [x] **Step 4: Run focused test and verify GREEN**

Run: `pnpm test -- src/lib/curated-frames/tag-facets.test.ts`

Expected: PASS.

### Task 3: Wire The Top Tag Filter Into The Curated Frames Page

**Files:**
- Modify: `src/components/jav-library/CuratedFramesLibrary.vue`
- Modify: `src/locales/zh-CN.json`

- [x] **Step 1: Update page state and data loading**

In `CuratedFramesLibrary.vue`:

- Import `getCuratedFrameTagQuery`.
- Import `listCuratedFrameTagFacets`.
- Import `visibleCuratedFrameTagFacets`.
- Add refs for `curatedTagFacets`, `tagFiltersExpanded`, and constants for the preview limit.
- Pass `tag: currentCuratedTagFilter()` into `listCuratedFramesPage()` in both reload and load-more calls.
- Include `route.query.cft` in the reload watcher.
- Add handlers to toggle a tag and clear the tag filter through `mergeCuratedFramesQuery`.

- [x] **Step 2: Add top filter UI**

In the template, insert a filter section between the tab/action toolbar and scrollable content:

- `全部` button clears `cft`.
- Tag chip buttons set or unset `cft`.
- Each chip displays name and count.
- `更多标签` / `收起标签` controls the preview limit.
- Empty tag state shows a light hint.

- [x] **Step 3: Add locale strings**

Add strings under `curated` in `src/locales/zh-CN.json`:

```json
"tagFilterTitle": "标签筛选",
"tagFilterAll": "全部",
"tagFilterEmpty": "还没有萃取帧标签，给帧打标签后可在这里筛选。",
"tagFilterShowMore": "更多标签（还有 {count} 个）",
"tagFilterShowLess": "收起标签",
"ariaFilterFrameTag": "按萃取帧标签筛选：{tag}，{count} 张",
"ariaClearFrameTagFilter": "清除萃取帧标签筛选"
```

- [x] **Step 4: Run relevant tests**

Run:

```powershell
pnpm test -- src/lib/library-query.test.ts src/lib/curated-frames/tag-facets.test.ts
pnpm typecheck
```

Expected: tests and typecheck pass, or report the exact failure.

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

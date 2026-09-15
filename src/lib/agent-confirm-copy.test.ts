import { describe, expect, it } from "vitest"
import { agentConfirmCopy } from "@/lib/agent-confirm-copy"

const t = (key: string, values?: Record<string, unknown>) => {
  if (!values) return key
  return `${key}:${JSON.stringify(values)}`
}

describe("agentConfirmCopy", () => {
  it("describes a new bookmark in product language instead of empty before/after rows", () => {
    const copy = agentConfirmCopy(
      {
        name: "create_saved_view",
        arguments: { name: "未看完", filters: { schemaVersion: 1 } },
        changes: [
          { path: "savedView.name", before: "", after: "未看完" },
          {
            path: "savedView.filters",
            before: null,
            after: { schemaVersion: 1, playState: "unwatched", actor: "Ada" },
          },
        ],
      },
      t,
    )
    expect(copy.title).toBe("agentWindow.confirmTitleCreateView")
    expect(copy.applyLabel).toBe("agentWindow.confirmApplyCreateView")
    expect(copy.showRawChanges).toBe(false)
    expect(copy.paragraphs[0]).toBe('agentWindow.confirmCreateViewLead:{"name":"未看完"}')
    expect(copy.paragraphs[1]).toContain("agentWindow.confirmCreateViewFilters")
    expect(copy.paragraphs[1]).toContain("library.savedViewPlay.unwatched")
    expect(copy.paragraphs[1]).toContain("Ada")
    expect(copy.paragraphs.join(" ")).not.toContain("（空）")
    expect(copy.paragraphs.join(" ")).not.toContain("confirmBefore")
  })

  it("describes a comic display title change as a display-field confirmation", () => {
    const copy = agentConfirmCopy(
      {
        name: "update_comic_title",
        arguments: { comicId: "c1", title: "展示标题" },
        changes: [{ path: "display.userTitle", before: "旧标题", after: "展示标题" }],
      },
      t,
    )
    expect(copy.title).toBe("agentWindow.confirmTitleDisplay")
    expect(copy.paragraphs).toEqual(['agentWindow.confirmDisplayTitle:{"after":"展示标题"}'])
    expect(copy.showRawChanges).toBe(false)
  })

  it("describes a new comment without a current-vs-next grid", () => {
    const copy = agentConfirmCopy(
      {
        name: "save_movie_comment",
        arguments: { movieId: "m1", body: "值得再看" },
        changes: [{ path: "comment.body", before: "", after: "值得再看" }],
      },
      t,
    )
    expect(copy.title).toBe("agentWindow.confirmTitleComment")
    expect(copy.paragraphs).toEqual(['agentWindow.confirmCommentCreate:{"body":"值得再看"}'])
    expect(copy.showRawChanges).toBe(false)
  })
})

import { describe, expect, it } from "vitest"
import {
  clearCuratedFrameExportSelection,
  reconcileCuratedFrameActorExportSelection,
  toggleCuratedFrameExportSelection,
  type CuratedFrameExportSelectionState,
} from "@/lib/curated-frames/selection"

const emptyState: CuratedFrameExportSelectionState = {
  selectedFrameIds: [],
  exportSelectionBucket: "none",
  namedActorForExport: null,
}

describe("curated frame export selection", () => {
  it("clears selected ids and export bucket metadata", () => {
    expect(clearCuratedFrameExportSelection()).toEqual(emptyState)
  })

  it("adds and removes timeline selections without actor bucket changes", () => {
    const added = toggleCuratedFrameExportSelection(emptyState, {
      id: "frame-a",
      mainTab: "timeline",
      max: 20,
      anonymousActorLabel: "No actor",
    })

    expect(added.error).toBeNull()
    expect(added.state).toEqual({
      selectedFrameIds: ["frame-a"],
      exportSelectionBucket: "none",
      namedActorForExport: null,
    })

    const removed = toggleCuratedFrameExportSelection(added.state, {
      id: "frame-a",
      mainTab: "timeline",
      max: 20,
      anonymousActorLabel: "No actor",
    })

    expect(removed.state).toEqual(emptyState)
  })

  it("blocks new selections once the export cap is reached", () => {
    const selectedFrameIds = Array.from({ length: 20 }, (_, index) => `frame-${index}`)
    const result = toggleCuratedFrameExportSelection(
      {
        selectedFrameIds,
        exportSelectionBucket: "none",
        namedActorForExport: null,
      },
      {
        id: "frame-extra",
        mainTab: "timeline",
        max: 20,
        anonymousActorLabel: "No actor",
      },
    )

    expect(result.error).toBe("max")
    expect(result.state.selectedFrameIds).toEqual(selectedFrameIds)
  })

  it("keeps actor tab selections inside one named actor bucket", () => {
    const first = toggleCuratedFrameExportSelection(emptyState, {
      id: "frame-a",
      mainTab: "actors",
      sectionActor: "Alice",
      max: 20,
      anonymousActorLabel: "No actor",
    })
    const second = toggleCuratedFrameExportSelection(first.state, {
      id: "frame-b",
      mainTab: "actors",
      sectionActor: "Alice",
      max: 20,
      anonymousActorLabel: "No actor",
    })
    const mixed = toggleCuratedFrameExportSelection(second.state, {
      id: "frame-c",
      mainTab: "actors",
      sectionActor: "Bob",
      max: 20,
      anonymousActorLabel: "No actor",
    })

    expect(second.error).toBeNull()
    expect(second.state).toEqual({
      selectedFrameIds: ["frame-a", "frame-b"],
      exportSelectionBucket: "named",
      namedActorForExport: "Alice",
    })
    expect(mixed.error).toBe("mixed-actor")
    expect(mixed.state).toEqual(second.state)
  })

  it("keeps anonymous actor selections separate from named actor selections", () => {
    const first = toggleCuratedFrameExportSelection(emptyState, {
      id: "frame-a",
      mainTab: "actors",
      sectionActor: "No actor",
      max: 20,
      anonymousActorLabel: "No actor",
    })
    const mixed = toggleCuratedFrameExportSelection(first.state, {
      id: "frame-b",
      mainTab: "actors",
      sectionActor: "Alice",
      max: 20,
      anonymousActorLabel: "No actor",
    })

    expect(first.state).toEqual({
      selectedFrameIds: ["frame-a"],
      exportSelectionBucket: "anonymous",
      namedActorForExport: null,
    })
    expect(mixed.error).toBe("mixed-actor")
    expect(mixed.state).toEqual(first.state)
  })

  it("reconciles actor export bucket from remaining selected ids", () => {
    expect(
      reconcileCuratedFrameActorExportSelection({
        selectedFrameIds: [],
        actorGroups: [["Alice", ["a"]]],
        currentNamedActorForExport: "Alice",
        anonymousActorLabel: "No actor",
      }),
    ).toEqual({
      exportSelectionBucket: "none",
      namedActorForExport: null,
    })
    expect(
      reconcileCuratedFrameActorExportSelection({
        selectedFrameIds: ["a"],
        actorGroups: [["Alice", ["a", "b"]]],
        currentNamedActorForExport: null,
        anonymousActorLabel: "No actor",
      }),
    ).toEqual({
      exportSelectionBucket: "named",
      namedActorForExport: "Alice",
    })
    expect(
      reconcileCuratedFrameActorExportSelection({
        selectedFrameIds: ["x"],
        actorGroups: [
          ["Alice", ["x"]],
          ["Bob", ["x"]],
        ],
        currentNamedActorForExport: "Bob",
        anonymousActorLabel: "No actor",
      }),
    ).toEqual({
      exportSelectionBucket: "named",
      namedActorForExport: "Bob",
    })
    expect(
      reconcileCuratedFrameActorExportSelection({
        selectedFrameIds: ["z"],
        actorGroups: [["No actor", ["z"]]],
        currentNamedActorForExport: null,
        anonymousActorLabel: "No actor",
      }),
    ).toEqual({
      exportSelectionBucket: "anonymous",
      namedActorForExport: null,
    })
  })
})

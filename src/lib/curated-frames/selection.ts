export type CuratedFrameExportSelectionBucket = "none" | "anonymous" | "named"
export type CuratedFrameMainTab = "timeline" | "actors" | "movies"
export type CuratedFrameExportSelectionError = "max" | "mixed-actor"

export type CuratedFrameExportSelectionState = {
  selectedFrameIds: string[]
  exportSelectionBucket: CuratedFrameExportSelectionBucket
  namedActorForExport: string | null
}

export type ToggleCuratedFrameExportSelectionInput = {
  id: string
  mainTab: CuratedFrameMainTab
  sectionActor?: string
  max: number
  anonymousActorLabel: string
}

export type ToggleCuratedFrameExportSelectionResult = {
  state: CuratedFrameExportSelectionState
  error: CuratedFrameExportSelectionError | null
}

export function clearCuratedFrameExportSelection(): CuratedFrameExportSelectionState {
  return {
    selectedFrameIds: [],
    exportSelectionBucket: "none",
    namedActorForExport: null,
  }
}

export function toggleCuratedFrameExportSelection(
  current: CuratedFrameExportSelectionState,
  input: ToggleCuratedFrameExportSelectionInput,
): ToggleCuratedFrameExportSelectionResult {
  const idx = current.selectedFrameIds.indexOf(input.id)
  if (idx >= 0) {
    const selectedFrameIds = current.selectedFrameIds.filter((id) => id !== input.id)
    return {
      state:
        selectedFrameIds.length === 0
          ? clearCuratedFrameExportSelection()
          : {
              ...current,
              selectedFrameIds,
            },
      error: null,
    }
  }

  if (current.selectedFrameIds.length >= input.max) {
    return { state: current, error: "max" }
  }

  let exportSelectionBucket = current.exportSelectionBucket
  let namedActorForExport = current.namedActorForExport

  if (input.mainTab === "actors" && input.sectionActor !== undefined) {
    const anonymous = input.sectionActor === input.anonymousActorLabel
    if (exportSelectionBucket === "none") {
      exportSelectionBucket = anonymous ? "anonymous" : "named"
      namedActorForExport = anonymous ? null : input.sectionActor
    } else if (exportSelectionBucket === "anonymous") {
      if (!anonymous) {
        return { state: current, error: "mixed-actor" }
      }
    } else if (namedActorForExport !== input.sectionActor || anonymous) {
      return { state: current, error: "mixed-actor" }
    }
  }

  return {
    state: {
      selectedFrameIds: [...current.selectedFrameIds, input.id],
      exportSelectionBucket,
      namedActorForExport,
    },
    error: null,
  }
}

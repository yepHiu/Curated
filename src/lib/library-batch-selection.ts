export type LibraryBatchToggleInput = {
  selectedIds: ReadonlySet<string>
  orderedIds: readonly string[]
  movieId: string
  shiftKey: boolean
  anchorId: string | null
  maxCount: number
}

export type LibraryBatchToggleResult = {
  selectedIds: Set<string>
  anchorId: string | null
  truncated: boolean
}

function capSelection(
  selected: Set<string>,
  orderedIds: readonly string[],
  maxCount: number,
  anchorId: string | null,
): LibraryBatchToggleResult {
  if (selected.size <= maxCount) {
    return { selectedIds: selected, anchorId, truncated: false }
  }
  const capped = new Set<string>()
  for (const id of orderedIds) {
    if (!selected.has(id)) continue
    capped.add(id)
    if (capped.size >= maxCount) break
  }
  return { selectedIds: capped, anchorId, truncated: true }
}

export function applyLibraryBatchToggle(input: LibraryBatchToggleInput): LibraryBatchToggleResult {
  const movieId = input.movieId.trim()
  if (!movieId) {
    return {
      selectedIds: new Set(input.selectedIds),
      anchorId: input.anchorId,
      truncated: false,
    }
  }

  const next = new Set(input.selectedIds)
  const anchorId = input.anchorId?.trim() || null

  if (input.shiftKey && anchorId) {
    const start = input.orderedIds.indexOf(anchorId)
    const end = input.orderedIds.indexOf(movieId)
    if (start >= 0 && end >= 0) {
      const from = Math.min(start, end)
      const to = Math.max(start, end)
      for (let index = from; index <= to; index++) {
        const id = input.orderedIds[index]
        if (id) next.add(id)
      }
      return capSelection(next, input.orderedIds, input.maxCount, anchorId)
    }
  }

  if (next.has(movieId)) {
    next.delete(movieId)
  } else {
    next.add(movieId)
  }
  return capSelection(next, input.orderedIds, input.maxCount, movieId)
}

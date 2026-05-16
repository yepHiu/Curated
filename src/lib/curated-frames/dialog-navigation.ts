export type CuratedFrameDialogMainTab = "timeline" | "actors" | "movies"
export type CuratedFrameDialogDirection = "previous" | "next"

export type CuratedFrameDialogGridItem = {
  row: {
    id: string
  }
}

export type CuratedFrameDialogNavigationEntry<T extends CuratedFrameDialogGridItem> = {
  item: T
  sectionActor: string | null
}

export function buildCuratedFrameDialogNavigationEntries<T extends CuratedFrameDialogGridItem>({
  mainTab,
  listItems,
  actorGroups,
  movieGroups,
}: {
  mainTab: CuratedFrameDialogMainTab
  listItems: readonly T[]
  actorGroups: readonly (readonly [actor: string, items: readonly T[]])[]
  movieGroups: readonly { items: readonly T[] }[]
}): CuratedFrameDialogNavigationEntry<T>[] {
  if (mainTab === "actors") {
    return actorGroups.flatMap(([actor, items]) =>
      items.map((item) => ({ item, sectionActor: actor })),
    )
  }

  if (mainTab === "movies") {
    return movieGroups.flatMap((group) =>
      group.items.map((item) => ({ item, sectionActor: null })),
    )
  }

  return listItems.map((item) => ({ item, sectionActor: null }))
}

export function findAdjacentCuratedFrameDialogEntry<T extends CuratedFrameDialogGridItem>(
  entries: readonly CuratedFrameDialogNavigationEntry<T>[],
  currentFrameId: string | undefined,
  currentSectionActor: string | null,
  direction: CuratedFrameDialogDirection,
): CuratedFrameDialogNavigationEntry<T> | null {
  const id = currentFrameId?.trim()
  if (!id) return null

  const currentIndex = entries.findIndex(
    (entry) => entry.item.row.id === id && entry.sectionActor === currentSectionActor,
  )
  if (currentIndex < 0) return null

  const nextIndex = direction === "previous" ? currentIndex - 1 : currentIndex + 1
  return entries[nextIndex] ?? null
}

export type DeleteCuratedFramesBatchResult =
  | { ok: true; deletedIds: string[] }
  | { ok: false; deletedIds: string[]; failedId: string; error: unknown }

export async function deleteCuratedFramesBatch(
  ids: readonly string[],
  remove: (id: string) => Promise<void>,
): Promise<DeleteCuratedFramesBatchResult> {
  const uniqueIds = [...new Set(ids.map((id) => id.trim()).filter(Boolean))]
  const deletedIds: string[] = []

  for (const id of uniqueIds) {
    try {
      await remove(id)
      deletedIds.push(id)
    } catch (error) {
      return { ok: false, deletedIds, failedId: id, error }
    }
  }

  return { ok: true, deletedIds }
}

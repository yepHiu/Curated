export type CuratedFrameTagSaveStatus = "idle" | "dirty" | "saving" | "error"

export type CommitCuratedFrameTagsParams = {
  frameId: string
  tags: string[]
  lastSavedTags: string[]
  update: (frameId: string, tags: string[]) => Promise<void>
}

export type CommitCuratedFrameTagsResult =
  | { ok: true; status: "idle"; lastSavedTags: string[] }
  | { ok: false; status: "error"; lastSavedTags: string[]; error: unknown }

export type ShouldCommitCuratedFrameTagDraftParams = {
  tags: string[]
  lastSavedTags: string[]
  saveInFlight: boolean
}

function sameTags(a: string[], b: string[]) {
  if (a.length !== b.length) {
    return false
  }
  return a.every((tag, index) => tag === b[index])
}

export function shouldCommitCuratedFrameTagDraft(
  params: ShouldCommitCuratedFrameTagDraftParams,
): boolean {
  return !params.saveInFlight && !sameTags(params.tags, params.lastSavedTags)
}

export function shouldShowCuratedFrameTagRetry(status: CuratedFrameTagSaveStatus): boolean {
  return status === "error"
}

export async function commitCuratedFrameTags(
  params: CommitCuratedFrameTagsParams,
): Promise<CommitCuratedFrameTagsResult> {
  const nextTags = [...params.tags]
  const lastSavedTags = [...params.lastSavedTags]
  if (sameTags(nextTags, lastSavedTags)) {
    return { ok: true, status: "idle", lastSavedTags }
  }

  try {
    await params.update(params.frameId, nextTags)
    return { ok: true, status: "idle", lastSavedTags: nextTags }
  } catch (error) {
    return { ok: false, status: "error", lastSavedTags, error }
  }
}

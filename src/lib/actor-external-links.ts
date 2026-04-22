export const MAX_ACTOR_EXTERNAL_LINKS = 1

export function normalizeActorExternalLinkDraft(raw: string): string {
  return raw.trim()
}

export function isValidActorExternalLink(raw: string): boolean {
  try {
    const parsed = new URL(raw)
    return parsed.protocol === "http:" || parsed.protocol === "https:"
  } catch {
    return false
  }
}

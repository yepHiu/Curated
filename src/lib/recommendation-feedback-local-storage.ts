import type { RecommendationFeedbackDTO } from "@/api/types"
import { normalizeActorIdentity } from "@/lib/actor-identity"

export const RECOMMENDATION_FEEDBACK_STORAGE_KEY = "curated-homepage-recommendation-feedback-v1"

interface StoredRecommendationFeedback {
  schemaVersion: 1
  items: RecommendationFeedbackDTO[]
}

const actions = new Set(["not_interested", "snooze", "less"])
const targetTypes = new Set(["movie", "actor", "studio", "tag"])

function normalizedFeedbackTarget(item: RecommendationFeedbackDTO): string {
  return item.targetType === "actor"
    ? normalizeActorIdentity(item.targetValue)
    : item.targetValue.trim().toLocaleLowerCase()
}

function isFeedback(value: unknown): value is RecommendationFeedbackDTO {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return false
  const item = value as Partial<RecommendationFeedbackDTO>
  return (
    typeof item.id === "string" &&
    actions.has(String(item.action)) &&
    targetTypes.has(String(item.targetType)) &&
    typeof item.targetValue === "string" &&
    typeof item.sourceMovieId === "string" &&
    (item.expiresAt === undefined || typeof item.expiresAt === "string") &&
    typeof item.createdAt === "string" &&
    typeof item.updatedAt === "string"
  )
}

export function loadLocalRecommendationFeedback(
  storage: Pick<Storage, "getItem"> | undefined = globalThis.localStorage,
  now = Date.now(),
): RecommendationFeedbackDTO[] {
  if (!storage) return []
  try {
    const parsed = JSON.parse(storage.getItem(RECOMMENDATION_FEEDBACK_STORAGE_KEY) ?? "null") as unknown
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      Array.isArray(parsed) ||
      (parsed as { schemaVersion?: unknown }).schemaVersion !== 1 ||
      !Array.isArray((parsed as { items?: unknown }).items)
    ) {
      return []
    }
    const items = (parsed as StoredRecommendationFeedback).items
    if (items.length > 500 || !items.every(isFeedback)) return []
    const keys = new Set<string>()
    const ids = new Set<string>()
    const active: RecommendationFeedbackDTO[] = []
    for (const item of items) {
      const key = `${item.action}\u0000${item.targetType}\u0000${normalizedFeedbackTarget(item)}`
      if (!item.id.trim() || !item.targetValue.trim() || ids.has(item.id) || keys.has(key)) return []
      ids.add(item.id)
      keys.add(key)
      if (!item.expiresAt || Date.parse(item.expiresAt) > now) active.push({ ...item })
    }
    return active.sort((left, right) => right.createdAt.localeCompare(left.createdAt) || left.id.localeCompare(right.id))
  } catch {
    return []
  }
}

export function saveLocalRecommendationFeedback(
  items: readonly RecommendationFeedbackDTO[],
  storage: Pick<Storage, "setItem"> | undefined = globalThis.localStorage,
): void {
  if (!storage) return
  try {
    storage.setItem(
      RECOMMENDATION_FEEDBACK_STORAGE_KEY,
      JSON.stringify({ schemaVersion: 1, items: items.map((item) => ({ ...item })) } satisfies StoredRecommendationFeedback),
    )
  } catch {
    // Private mode and quota failures keep the current in-memory feedback usable.
  }
}

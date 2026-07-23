import type { ActorMergeAuditDTO } from "@/api/types"
import { isActorMergeAuditDTO } from "@/api/guards"
import { normalizeActorIdentity } from "@/lib/actor-identity"

export const ACTOR_MERGE_STORAGE_KEY = "curated-actor-merges-v1"
const ACTOR_MERGE_SCHEMA_VERSION = 1

export interface LocalActorMergeState {
  aliases: Record<string, LocalActorAlias>
  audits: ActorMergeAuditDTO[]
}

export interface LocalActorAlias {
  alias: string
  canonicalName: string
}

interface StoredActorMergeState extends LocalActorMergeState {
  schemaVersion: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function isActorAliasRecord(value: unknown): value is Record<string, LocalActorAlias> {
  return (
    isRecord(value) &&
    Object.values(value).every(
      (item) =>
        isRecord(item) &&
        typeof item.alias === "string" &&
        typeof item.canonicalName === "string",
    )
  )
}

function isStoredActorMergeState(value: unknown): value is StoredActorMergeState {
  return (
    isRecord(value) &&
    value.schemaVersion === ACTOR_MERGE_SCHEMA_VERSION &&
    isActorAliasRecord(value.aliases) &&
    Array.isArray(value.audits) &&
    value.audits.every(isActorMergeAuditDTO)
  )
}

export function emptyLocalActorMergeState(): LocalActorMergeState {
  return { aliases: {}, audits: [] }
}

export function loadLocalActorMergeState(
  storage: Pick<Storage, "getItem"> | undefined,
): LocalActorMergeState {
  if (!storage) return emptyLocalActorMergeState()
  try {
    const raw = storage.getItem(ACTOR_MERGE_STORAGE_KEY)
    if (!raw) return emptyLocalActorMergeState()
    const parsed: unknown = JSON.parse(raw)
    if (!isStoredActorMergeState(parsed)) return emptyLocalActorMergeState()
    return {
      aliases: Object.fromEntries(
        Object.values(parsed.aliases).map((alias) => [
          normalizeActorIdentity(alias.alias),
          { ...alias },
        ]),
      ),
      audits: parsed.audits.map((audit) => ({ ...audit })),
    }
  } catch {
    return emptyLocalActorMergeState()
  }
}

export function saveLocalActorMergeState(
  state: LocalActorMergeState,
  storage: Pick<Storage, "setItem"> | undefined,
): void {
  if (!storage) return
  const payload: StoredActorMergeState = {
    schemaVersion: ACTOR_MERGE_SCHEMA_VERSION,
    aliases: Object.fromEntries(
      Object.values(state.aliases).map((alias) => [
        normalizeActorIdentity(alias.alias),
        { ...alias },
      ]),
    ),
    audits: state.audits.map((audit) => ({
      ...audit,
      summary: {
        ...audit.summary,
        userTags: [...audit.summary.userTags],
        externalLinks: [...audit.summary.externalLinks],
        aliases: [...audit.summary.aliases],
        profileDecisions: { ...audit.summary.profileDecisions },
      },
    })),
  }
  storage.setItem(ACTOR_MERGE_STORAGE_KEY, JSON.stringify(payload))
}

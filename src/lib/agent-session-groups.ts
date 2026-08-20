import type { AIChatSessionDTO } from "@/api/types"

export const AGENT_SESSION_GROUP_KEYS = ["today", "yesterday", "previous7Days", "older"] as const

export type AgentSessionGroupKey = (typeof AGENT_SESSION_GROUP_KEYS)[number]

export interface AgentSessionGroup {
  key: AgentSessionGroupKey
  items: AIChatSessionDTO[]
}

const DAY_MS = 24 * 60 * 60 * 1000

function startOfLocalDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

export function agentSessionGroupKey(updatedAt: string, now = new Date()): AgentSessionGroupKey {
  const timestamp = Date.parse(updatedAt)
  if (Number.isNaN(timestamp)) return "older"
  const diffDays = Math.round((startOfLocalDay(now) - startOfLocalDay(new Date(timestamp))) / DAY_MS)
  if (diffDays <= 0) return "today"
  if (diffDays === 1) return "yesterday"
  if (diffDays < 7) return "previous7Days"
  return "older"
}

/** 按本地日历把会话分成 Today / Yesterday / 近 7 天 / 更早；空组不返回。 */
export function groupAgentSessions(sessions: AIChatSessionDTO[], now = new Date()): AgentSessionGroup[] {
  const buckets: Record<AgentSessionGroupKey, AIChatSessionDTO[]> = {
    today: [],
    yesterday: [],
    previous7Days: [],
    older: [],
  }
  for (const session of sessions) {
    buckets[agentSessionGroupKey(session.updatedAt, now)].push(session)
  }
  return AGENT_SESSION_GROUP_KEYS.flatMap((key) => {
    const items = buckets[key]
    return items.length > 0 ? [{ key, items }] : []
  })
}

export const MESSAGE_LEVELS = ["silent", "notify", "needs-you", "now"] as const
export type MessageLevel = (typeof MESSAGE_LEVELS)[number]

export const MESSAGE_CENTER_SLOTS = ["none", "recent", "needs-you", "now"] as const
export type MessageCenterSlot = (typeof MESSAGE_CENTER_SLOTS)[number]

export interface MessagePolicy {
  id: string
  level: MessageLevel
  toast: boolean
  center: MessageCenterSlot
  badge: "none" | "needs-you"
  group?: string
}

export const MESSAGE_POLICIES = {
  "MSG-0001": { id: "MSG-0001", level: "now", toast: false, center: "now", badge: "none" },
  "MSG-0002": { id: "MSG-0002", level: "now", toast: false, center: "now", badge: "none" },
  "MSG-0003": { id: "MSG-0003", level: "now", toast: false, center: "now", badge: "none" },
  "MSG-0010": {
    id: "MSG-0010",
    level: "needs-you",
    toast: true,
    center: "needs-you",
    badge: "needs-you",
    group: "storage-offline",
  },
  "MSG-0011": { id: "MSG-0011", level: "needs-you", toast: true, center: "needs-you", badge: "needs-you" },
  "MSG-0012": { id: "MSG-0012", level: "needs-you", toast: true, center: "needs-you", badge: "needs-you" },
  "MSG-0013": { id: "MSG-0013", level: "needs-you", toast: true, center: "needs-you", badge: "needs-you" },
  "MSG-0020": { id: "MSG-0020", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0021": { id: "MSG-0021", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0022": { id: "MSG-0022", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0023": { id: "MSG-0023", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0024": {
    id: "MSG-0024",
    level: "notify",
    toast: true,
    center: "recent",
    badge: "none",
    group: "app-update",
  },
  "MSG-0025": { id: "MSG-0025", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0026": { id: "MSG-0026", level: "notify", toast: true, center: "recent", badge: "none" },
  "MSG-0027": {
    id: "MSG-0027",
    level: "notify",
    toast: true,
    center: "recent",
    badge: "none",
    group: "library-watch-scan",
  },
  "MSG-0028": {
    id: "MSG-0028",
    level: "notify",
    toast: true,
    center: "recent",
    badge: "none",
    group: "library-watch-scrape",
  },
  "MSG-0040": { id: "MSG-0040", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0041": { id: "MSG-0041", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0042": { id: "MSG-0042", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0043": { id: "MSG-0043", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0044": { id: "MSG-0044", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0045": { id: "MSG-0045", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0046": { id: "MSG-0046", level: "silent", toast: true, center: "none", badge: "none" },
  "MSG-0047": { id: "MSG-0047", level: "now", toast: false, center: "now", badge: "none" },
} as const satisfies Record<string, MessagePolicy>

export type MessagePolicyId = keyof typeof MESSAGE_POLICIES

export function isMessagePolicyId(value: string): value is MessagePolicyId {
  return Object.prototype.hasOwnProperty.call(MESSAGE_POLICIES, value)
}

export function getMessagePolicy(id: MessagePolicyId): MessagePolicy {
  return MESSAGE_POLICIES[id]
}

export function shouldPersistToMessageCenter(id: MessagePolicyId): boolean {
  const policy = MESSAGE_POLICIES[id]
  return policy.center === "recent" || policy.center === "needs-you"
}

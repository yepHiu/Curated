import { computed, ref } from "vue"

export type NotificationType = "scan" | "scrape" | "update" | "error" | "system"
export type NotificationSeverity = "info" | "success" | "warning" | "error"

export interface NotificationSource {
  taskId?: string
  movieId?: string
  route?: string
}

export interface AppNotification {
  id: string
  type: NotificationType
  severity: NotificationSeverity
  title: string
  message: string
  timestamp: number
  read: boolean
  source?: NotificationSource
}

const STORAGE_KEY = "curated-notification-center-v1"
const MAX_NOTIFICATIONS = 200
const RETENTION_MS = 7 * 24 * 60 * 60 * 1000
const NOTIFICATION_TYPES = ["scan", "scrape", "update", "error", "system"] as const
const NOTIFICATION_SEVERITIES = ["info", "success", "warning", "error"] as const

let nextSeq = 0

function uid(): string {
  nextSeq += 1
  return `${Date.now().toString(36)}-${nextSeq.toString(36)}`
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function isNotificationType(value: unknown): value is NotificationType {
  return typeof value === "string" && NOTIFICATION_TYPES.includes(value as NotificationType)
}

function isNotificationSeverity(value: unknown): value is NotificationSeverity {
  return (
    typeof value === "string" &&
    NOTIFICATION_SEVERITIES.includes(value as NotificationSeverity)
  )
}

function sanitizeSource(value: unknown): NotificationSource | undefined {
  if (!isRecord(value)) return undefined
  const source: NotificationSource = {}
  if (typeof value.taskId === "string" && value.taskId.trim()) {
    source.taskId = value.taskId
  }
  if (typeof value.movieId === "string" && value.movieId.trim()) {
    source.movieId = value.movieId
  }
  if (typeof value.route === "string" && value.route.trim()) {
    source.route = value.route
  }
  return Object.keys(source).length > 0 ? source : undefined
}

function normalizeNotification(value: unknown): AppNotification | null {
  if (!isRecord(value)) return null
  const timestamp = value.timestamp
  if (
    typeof value.id !== "string" ||
    !value.id.trim() ||
    !isNotificationType(value.type) ||
    !isNotificationSeverity(value.severity) ||
    typeof value.title !== "string" ||
    typeof value.message !== "string" ||
    typeof timestamp !== "number" ||
    !Number.isFinite(timestamp) ||
    typeof value.read !== "boolean"
  ) {
    return null
  }

  const source = sanitizeSource(value.source)
  return {
    id: value.id,
    type: value.type,
    severity: value.severity,
    title: value.title,
    message: value.message,
    timestamp,
    read: value.read,
    ...(source ? { source } : {}),
  }
}

function load(): AppNotification[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return []
    return arr
      .map((item) => normalizeNotification(item))
      .filter((item): item is AppNotification => item !== null)
  } catch {
    return []
  }
}

function persist(notifications: AppNotification[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(notifications))
  } catch {
    /* quota exceeded — oldest notifications already trimmed */
  }
}

function cleanup(notifications: AppNotification[]): AppNotification[] {
  const cutoff = Date.now() - RETENTION_MS
  return notifications
    .filter((n) => n.timestamp > cutoff)
    .sort((a, b) => b.timestamp - a.timestamp)
    .slice(0, MAX_NOTIFICATIONS)
}

const notifications = ref<AppNotification[]>(cleanup(load()))
const centerOpen = ref(false)

const unreadNotifications = computed(() =>
  notifications.value.filter((n) => !n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const readNotifications = computed(() =>
  notifications.value.filter((n) => n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const unreadCount = computed(() => unreadNotifications.value.length)

function flush() {
  notifications.value = cleanup(notifications.value)
  persist(notifications.value)
}

function notificationDedupKey(
  notif: Pick<AppNotification, "type" | "severity" | "title" | "message" | "source">,
): string {
  if (notif.source?.taskId) {
    return `task:${notif.source.taskId}`
  }
  if (notif.source?.movieId || notif.source?.route) {
    return [
      "source",
      notif.type,
      notif.title,
      notif.source.movieId ?? "",
      notif.source.route ?? "",
    ].join("\u001f")
  }
  return ["content", notif.type, notif.severity, notif.title, notif.message].join("\u001f")
}

function addNotification(notif: Omit<AppNotification, "id" | "read" | "timestamp">): string {
  const source = sanitizeSource(notif.source)
  const normalized = {
    ...notif,
    ...(source ? { source } : { source: undefined }),
  }
  const key = notificationDedupKey(normalized)
  const existingIndex = notifications.value.findIndex((n) => notificationDedupKey(n) === key)
  if (existingIndex >= 0) {
    const existing = notifications.value[existingIndex]
    const updated: AppNotification = {
      ...existing,
      ...normalized,
      id: existing.id,
      read: centerOpen.value ? true : existing.read,
      timestamp: Date.now(),
    }
    notifications.value = [
      updated,
      ...notifications.value.filter((_, index) => index !== existingIndex),
    ]
    flush()
    return existing.id
  }

  const id = uid()
  const entry: AppNotification = {
    ...normalized,
    id,
    read: centerOpen.value,
    timestamp: Date.now(),
  }
  notifications.value = [entry, ...notifications.value]
  flush()
  return id
}

function markAllRead() {
  if (!notifications.value.some((n) => !n.read)) return
  notifications.value = notifications.value.map((n) => (n.read ? n : { ...n, read: true }))
  flush()
}

function dismissOne(id: string) {
  notifications.value = notifications.value.filter((n) => n.id !== id)
  flush()
}

function clearAll() {
  notifications.value = []
  flush()
}

function setCenterOpen(open: boolean) {
  centerOpen.value = open
  if (open) {
    markAllRead()
  }
}

export function useNotificationCenter() {
  return {
    notifications,
    unreadNotifications,
    readNotifications,
    unreadCount,
    addNotification,
    markAllRead,
    dismissOne,
    clearAll,
    centerOpen,
    setCenterOpen,
  }
}

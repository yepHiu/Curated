import { computed, ref } from "vue"
import {
  getMessagePolicy,
  isMessagePolicyId,
  shouldPersistToMessageCenter,
  type MessagePolicyId,
} from "@/lib/message-policy"

export type NotificationType = "scan" | "scrape" | "storage" | "update" | "error" | "system"
export type NotificationSeverity = "info" | "success" | "warning" | "error"
export type NotificationLevel = "notify" | "needs-you"

export interface NotificationSource {
  taskId?: string
  movieId?: string
  libraryPathId?: string
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
  level: NotificationLevel
  resolved: boolean
  messageId?: MessagePolicyId
  group?: string
  source?: NotificationSource
}

const STORAGE_KEY = "curated-notification-center-v1"
const SUPPRESS_KEY = "curated-message-center-suppressed-v1"
const MAX_NOTIFICATIONS = 200
const RETENTION_MS = 7 * 24 * 60 * 60 * 1000
const NOTIFICATION_TYPES = ["scan", "scrape", "storage", "update", "error", "system"] as const
const NOTIFICATION_SEVERITIES = ["info", "success", "warning", "error"] as const
const NOTIFICATION_LEVELS = ["notify", "needs-you"] as const

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

function isNotificationLevel(value: unknown): value is NotificationLevel {
  return typeof value === "string" && NOTIFICATION_LEVELS.includes(value as NotificationLevel)
}

function inferLevel(value: Record<string, unknown>): NotificationLevel {
  if (isNotificationLevel(value.level)) {
    return value.level
  }
  if (value.type === "storage" && (value.severity === "warning" || value.severity === "error")) {
    return "needs-you"
  }
  if (value.type === "error") {
    return "needs-you"
  }
  return "notify"
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
  if (typeof value.libraryPathId === "string" && value.libraryPathId.trim()) {
    source.libraryPathId = value.libraryPathId
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
  const messageId =
    typeof value.messageId === "string" && isMessagePolicyId(value.messageId)
      ? value.messageId
      : undefined
  const group = typeof value.group === "string" && value.group.trim() ? value.group.trim() : undefined
  return {
    id: value.id,
    type: value.type,
    severity: value.severity,
    title: value.title,
    message: value.message,
    timestamp,
    read: value.read,
    level: inferLevel(value),
    resolved: value.resolved === true,
    ...(messageId ? { messageId } : {}),
    ...(group ? { group } : {}),
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

function persist(list: AppNotification[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
  } catch {
    /* quota exceeded — oldest notifications already trimmed */
  }
}

function cleanup(list: AppNotification[]): AppNotification[] {
  const cutoff = Date.now() - RETENTION_MS
  return list
    .filter((n) => n.timestamp > cutoff)
    .sort((a, b) => b.timestamp - a.timestamp)
    .slice(0, MAX_NOTIFICATIONS)
}

function readSuppressed(): Set<string> {
  try {
    const raw = sessionStorage.getItem(SUPPRESS_KEY)
    if (!raw) return new Set()
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return new Set()
    return new Set(arr.filter((item): item is string => typeof item === "string" && item.length > 0))
  } catch {
    return new Set()
  }
}

function writeSuppressed(ids: Set<string>) {
  try {
    sessionStorage.setItem(SUPPRESS_KEY, JSON.stringify([...ids].slice(-200)))
  } catch {
    /* private mode */
  }
}

function suppressKeyFor(
  notif: Pick<AppNotification, "messageId" | "type" | "group" | "source">,
): string {
  return [
    notif.messageId ?? notif.type,
    notif.group ?? "",
    notif.source?.libraryPathId ?? "",
    notif.source?.taskId ?? "",
    notif.source?.route ?? "",
  ].join("\u001f")
}

const notifications = ref<AppNotification[]>(cleanup(load()))
const centerOpen = ref(false)
const suppressedKeys = readSuppressed()

const needsYouNotifications = computed(() =>
  notifications.value
    .filter((n) => n.level === "needs-you" && !n.resolved)
    .sort((a, b) => b.timestamp - a.timestamp),
)

const recentNotifications = computed(() =>
  notifications.value
    .filter((n) => n.level === "notify" || n.resolved)
    .sort((a, b) => b.timestamp - a.timestamp),
)

const unreadNotifications = computed(() =>
  recentNotifications.value.filter((n) => !n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const readNotifications = computed(() =>
  recentNotifications.value.filter((n) => n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const unreadCount = computed(() => needsYouNotifications.value.length)

function flush() {
  notifications.value = cleanup(notifications.value)
  persist(notifications.value)
}

function notificationDedupKey(
  notif: Pick<AppNotification, "type" | "severity" | "title" | "message" | "source" | "group" | "messageId">,
): string {
  if (notif.group) {
    return `group:${notif.group}`
  }
  if (notif.source?.taskId) {
    return `task:${notif.source.taskId}`
  }
  if (notif.source?.movieId || notif.source?.libraryPathId || notif.source?.route) {
    return [
      "source",
      notif.messageId ?? notif.type,
      notif.title,
      notif.source.movieId ?? "",
      notif.source.libraryPathId ?? "",
      notif.source.route ?? "",
    ].join("\u001f")
  }
  return ["content", notif.type, notif.severity, notif.title, notif.message].join("\u001f")
}

export interface AddNotificationInput {
  type: NotificationType
  severity: NotificationSeverity
  title: string
  message: string
  source?: NotificationSource
  messageId?: MessagePolicyId
  group?: string
  level?: NotificationLevel
}

function addNotification(notif: AddNotificationInput): string | null {
  const messageId = notif.messageId
  if (messageId && !shouldPersistToMessageCenter(messageId)) {
    return null
  }
  const policy = messageId ? getMessagePolicy(messageId) : undefined
  const level: NotificationLevel =
    notif.level ?? (policy?.level === "needs-you" ? "needs-you" : "notify")
  const group = notif.group?.trim() || policy?.group
  const source = sanitizeSource(notif.source)
  const normalized = {
    type: notif.type,
    severity: notif.severity,
    title: notif.title,
    message: notif.message,
    level,
    resolved: false,
    ...(messageId ? { messageId } : {}),
    ...(group ? { group } : {}),
    ...(source ? { source } : { source: undefined }),
  }
  const suppressKey = suppressKeyFor(normalized)
  if (level === "needs-you" && suppressedKeys.has(suppressKey)) {
    return null
  }

  const key = notificationDedupKey(normalized)
  const existingIndex = notifications.value.findIndex((n) => notificationDedupKey(n) === key)
  if (existingIndex >= 0) {
    const existing = notifications.value[existingIndex]
    const updated: AppNotification = {
      ...existing,
      ...normalized,
      id: existing.id,
      read: level === "needs-you" ? false : centerOpen.value ? true : existing.read,
      resolved: false,
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
    read: level === "needs-you" ? false : centerOpen.value,
    timestamp: Date.now(),
  }
  notifications.value = [entry, ...notifications.value]
  flush()
  return id
}

function markNotifyRead() {
  if (!notifications.value.some((n) => n.level === "notify" && !n.read)) return
  notifications.value = notifications.value.map((n) =>
    n.level === "notify" && !n.read ? { ...n, read: true } : n,
  )
  flush()
}

function markAllRead() {
  markNotifyRead()
}

function dismissOne(id: string) {
  const target = notifications.value.find((n) => n.id === id)
  if (target?.level === "needs-you") {
    suppressedKeys.add(suppressKeyFor(target))
    writeSuppressed(suppressedKeys)
  }
  notifications.value = notifications.value.filter((n) => n.id !== id)
  flush()
}

function clearAll() {
  notifications.value = []
  flush()
}

function resolveMatching(predicate: (notification: AppNotification) => boolean) {
  let changed = false
  notifications.value = notifications.value.map((n) => {
    if (n.level !== "needs-you" || n.resolved || !predicate(n)) {
      return n
    }
    changed = true
    return { ...n, resolved: true, read: true }
  })
  if (changed) {
    flush()
  }
}

function setCenterOpen(open: boolean) {
  centerOpen.value = open
  if (open) {
    markNotifyRead()
  }
}

export function useNotificationCenter() {
  return {
    notifications,
    unreadNotifications,
    readNotifications,
    needsYouNotifications,
    recentNotifications,
    unreadCount,
    addNotification,
    markAllRead,
    dismissOne,
    clearAll,
    resolveMatching,
    centerOpen,
    setCenterOpen,
  }
}

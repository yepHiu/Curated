import { computed, ref, watch } from "vue"

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

let nextSeq = 0

function uid(): string {
  nextSeq += 1
  return `${Date.now().toString(36)}-${nextSeq.toString(36)}`
}

function load(): AppNotification[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const arr = JSON.parse(raw) as unknown
    if (!Array.isArray(arr)) return []
    return arr.filter(
      (x): x is AppNotification =>
        typeof x === "object" &&
        x !== null &&
        typeof (x as AppNotification).id === "string" &&
        typeof (x as AppNotification).timestamp === "number",
    )
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
  return notifications.filter((n) => n.timestamp > cutoff).slice(-MAX_NOTIFICATIONS)
}

const notifications = ref<AppNotification[]>(cleanup(load()))

const unreadNotifications = computed(() =>
  notifications.value.filter((n) => !n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const readNotifications = computed(() =>
  notifications.value.filter((n) => n.read).sort((a, b) => b.timestamp - a.timestamp),
)

const unreadCount = computed(() => unreadNotifications.value.length)

let dirty = false

watch(
  notifications,
  () => {
    dirty = true
  },
  { deep: true },
)

function flush() {
  if (!dirty) return
  dirty = false
  persist(cleanup(notifications.value))
}

function addNotification(notif: Omit<AppNotification, "id" | "read" | "timestamp">): string {
  const id = uid()
  const entry: AppNotification = {
    ...notif,
    id,
    read: false,
    timestamp: Date.now(),
  }
  notifications.value = [entry, ...notifications.value]
  flush()
  return id
}

function markAllRead() {
  let changed = false
  for (const n of notifications.value) {
    if (!n.read) {
      n.read = true
      changed = true
    }
  }
  if (changed) {
    notifications.value = [...notifications.value]
    flush()
  }
}

function dismissOne(id: string) {
  notifications.value = notifications.value.filter((n) => n.id !== id)
  flush()
}

function clearAll() {
  notifications.value = []
  flush()
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
  }
}

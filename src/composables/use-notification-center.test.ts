import { afterEach, describe, expect, it, vi } from "vitest"

const STORAGE_KEY = "curated-notification-center-v1"

async function freshNotificationCenter(options?: { preserveStorage?: boolean }) {
  vi.resetModules()
  if (!options?.preserveStorage) {
    localStorage.clear()
    sessionStorage.clear()
  }
  const { useNotificationCenter } = await import("@/composables/use-notification-center")
  return useNotificationCenter()
}

function notificationInput(index: number) {
  return {
    type: "system" as const,
    severity: "info" as const,
    title: `Notification ${index}`,
    message: `message-${index}`,
  }
}

afterEach(() => {
  vi.useRealTimers()
  localStorage.clear()
  sessionStorage.clear()
})

describe("useNotificationCenter", () => {
  it("trims the in-memory queue to the newest 200 notifications", async () => {
    vi.useFakeTimers()
    const center = await freshNotificationCenter()
    const start = new Date("2026-05-02T00:00:00.000Z").getTime()

    for (let i = 0; i < 205; i += 1) {
      vi.setSystemTime(start + i * 1000)
      center.addNotification(notificationInput(i))
    }

    expect(center.notifications.value).toHaveLength(200)
    expect(center.notifications.value.some((n) => n.message === "message-204")).toBe(true)
    expect(center.notifications.value.some((n) => n.message === "message-0")).toBe(false)
  })

  it("does not count notify items toward the badge", async () => {
    const center = await freshNotificationCenter()
    center.addNotification(notificationInput(1))
    expect(center.unreadCount.value).toBe(0)
    expect(center.unreadNotifications.value).toHaveLength(1)
  })

  it("treats notify items added while the center is open as already read", async () => {
    const center = await freshNotificationCenter()

    center.addNotification(notificationInput(1))
    expect(center.unreadNotifications.value).toHaveLength(1)

    center.setCenterOpen(true)
    expect(center.unreadNotifications.value).toHaveLength(0)

    center.addNotification(notificationInput(2))

    expect(center.unreadNotifications.value).toHaveLength(0)
    expect(center.readNotifications.value.map((n) => n.message)).toContain("message-2")
  })

  it("keeps needs-you items in the badge while the center is open", async () => {
    const center = await freshNotificationCenter()
    center.addNotification({
      ...notificationInput(1),
      level: "needs-you",
      messageId: "MSG-0010",
    })
    expect(center.unreadCount.value).toBe(1)

    center.setCenterOpen(true)
    expect(center.unreadCount.value).toBe(1)
    expect(center.needsYouNotifications.value).toHaveLength(1)

    center.addNotification({
      ...notificationInput(2),
      level: "needs-you",
      messageId: "MSG-0011",
      source: { taskId: "import-1" },
    })
    expect(center.unreadCount.value).toBe(2)
  })

  it("deduplicates notifications from the same task source", async () => {
    vi.useFakeTimers()
    const center = await freshNotificationCenter()
    const start = new Date("2026-05-02T00:00:00.000Z").getTime()

    vi.setSystemTime(start)
    const firstId = center.addNotification({
      ...notificationInput(1),
      source: { taskId: "task-1" },
    })

    vi.setSystemTime(start + 1000)
    const secondId = center.addNotification({
      type: "scan",
      severity: "warning",
      title: "Updated task",
      message: "latest task state",
      source: { taskId: "task-1" },
    })

    expect(secondId).toBe(firstId)
    expect(center.notifications.value).toHaveLength(1)
    expect(center.notifications.value[0]).toMatchObject({
      id: firstId,
      type: "scan",
      severity: "warning",
      title: "Updated task",
      message: "latest task state",
      source: { taskId: "task-1" },
      read: false,
    })
  })

  it("collapses grouped watch events into one recent row", async () => {
    const center = await freshNotificationCenter()
    center.addNotification({
      ...notificationInput(1),
      messageId: "MSG-0027",
      group: "library-watch-scan",
    })
    center.addNotification({
      type: "scan",
      severity: "success",
      title: "Scan digest",
      message: "latest watch scan",
      messageId: "MSG-0027",
      group: "library-watch-scan",
    })
    expect(center.notifications.value).toHaveLength(1)
    expect(center.notifications.value[0].message).toBe("latest watch scan")
    expect(center.unreadCount.value).toBe(0)
  })

  it("resolves needs-you storage alerts when the matching path is online", async () => {
    const center = await freshNotificationCenter()
    center.addNotification({
      ...notificationInput(1),
      type: "storage",
      severity: "warning",
      messageId: "MSG-0010",
      source: { libraryPathId: "library-b", route: "/settings?section=library" },
    })
    expect(center.unreadCount.value).toBe(1)
    center.resolveMatching((item) => item.source?.libraryPathId === "library-b")
    expect(center.unreadCount.value).toBe(0)
    expect(center.needsYouNotifications.value).toHaveLength(0)
  })

  it("drops malformed persisted notifications during load", async () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify([
        {
          id: "valid-1",
          type: "error",
          severity: "error",
          title: "Valid error",
          message: "still useful",
          timestamp: Date.now(),
          read: false,
          source: { taskId: "task-1", movieId: 123, route: "/settings" },
        },
        {
          id: "invalid-type",
          type: "unknown",
          severity: "info",
          title: "Invalid",
          message: "bad type",
          timestamp: Date.now(),
          read: false,
        },
        {
          id: "invalid-read",
          type: "system",
          severity: "info",
          title: "Invalid",
          message: "bad read",
          timestamp: Date.now(),
          read: "false",
        },
      ]),
    )

    const center = await freshNotificationCenter({ preserveStorage: true })

    expect(center.notifications.value).toHaveLength(1)
    expect(center.notifications.value[0]).toMatchObject({
      id: "valid-1",
      type: "error",
      severity: "error",
      title: "Valid error",
      message: "still useful",
      level: "needs-you",
      source: { taskId: "task-1", route: "/settings" },
    })
    expect(center.notifications.value[0].source).not.toHaveProperty("movieId")
  })
})

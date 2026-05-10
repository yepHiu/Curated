import { mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"

const notificationState = vi.hoisted(() => ({
  read: [] as TestNotification[],
  unread: [] as TestNotification[],
  markAllRead: vi.fn(),
  dismissOne: vi.fn(),
  clearAll: vi.fn(),
  setCenterOpen: vi.fn(),
}))

const routerPush = vi.hoisted(() => vi.fn())

type TestNotification = {
  id: string
  type: "scan" | "scrape" | "update" | "error" | "system"
  severity: "info" | "success" | "warning" | "error"
  title: string
  message: string
  timestamp: number
  read: boolean
  source?: { route?: string }
}

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n == null ? key : `${key} ${params.n}`,
  }),
}))

vi.mock("vue-router", () => ({
  useRouter: () => ({
    push: routerPush,
  }),
}))

vi.mock("@/composables/use-notification-center", async () => {
  const { computed } = await vi.importActual<typeof import("vue")>("vue")
  return {
    useNotificationCenter: () => ({
      unreadNotifications: computed(() => notificationState.unread),
      readNotifications: computed(() => notificationState.read),
      unreadCount: computed(() => notificationState.unread.length),
      markAllRead: notificationState.markAllRead,
      dismissOne: notificationState.dismissOne,
      clearAll: notificationState.clearAll,
      setCenterOpen: notificationState.setCenterOpen,
    }),
  }
})

function makeNotification(
  index: number,
  overrides?: Partial<Pick<TestNotification, "type" | "severity" | "read" | "title" | "message">>,
): TestNotification {
  return {
    id: `${overrides?.read === false ? "unread" : "read"}-${index}`,
    type: overrides?.type ?? "system",
    severity: overrides?.severity ?? "info",
    title: overrides?.title ?? `Read ${index}`,
    message: overrides?.message ?? `read-message-${index}`,
    timestamp: Date.now() - index * 1000,
    read: overrides?.read ?? true,
  }
}

async function mountCenter() {
  const { default: NotificationCenter } = await import("./NotificationCenter.vue")
  return mount(NotificationCenter, {
    global: {
      stubs: {
        ArrowLeft: true,
        ArrowRight: true,
        Bell: true,
        Check: true,
        ChevronDown: true,
        Button: { template: "<button v-bind=\"$attrs\"><slot /></button>" },
        Popover: { template: "<div><slot /></div>" },
        PopoverTrigger: { template: "<div><slot /></div>" },
        PopoverContent: {
          template: "<section data-test=\"popover-content\" v-bind=\"$attrs\"><slot /></section>",
        },
        ScrollArea: { template: "<div v-bind=\"$attrs\"><slot /></div>" },
        Collapsible: { template: "<div><slot /></div>" },
        CollapsibleTrigger: { template: "<button v-bind=\"$attrs\"><slot /></button>" },
        CollapsibleContent: { template: "<div><slot /></div>" },
      },
    },
  })
}

beforeEach(() => {
  notificationState.read = []
  notificationState.unread = []
  notificationState.markAllRead.mockClear()
  notificationState.dismissOne.mockClear()
  notificationState.clearAll.mockClear()
  notificationState.setCenterOpen.mockClear()
  routerPush.mockClear()
})

describe("NotificationCenter", () => {
  it("bounds the popover surface and caps the read preview", async () => {
    notificationState.read = Array.from({ length: 8 }, (_, index) => makeNotification(index))

    const wrapper = await mountCenter()

    const contentClass = wrapper.get("[data-test='popover-content']").attributes("class") ?? ""
    expect(contentClass).toContain("max-h-")
    expect(contentClass).toContain("overflow-hidden")
    expect(wrapper.text()).toContain("Read 0")
    expect(wrapper.text()).toContain("Read 4")
    expect(wrapper.text()).not.toContain("Read 5")
  })

  it("caps the unread badge label at 99+", async () => {
    notificationState.unread = Array.from({ length: 105 }, (_, index) =>
      makeNotification(index, {
        read: false,
        title: `Unread ${index}`,
        message: `unread-message-${index}`,
      }),
    )

    const wrapper = await mountCenter()

    expect(wrapper.text()).toContain("99+")
  })

  it("filters the main list to notifications that need attention", async () => {
    notificationState.unread = [
      makeNotification(1, {
        read: false,
        severity: "info",
        title: "Informational",
        message: "FYI",
      }),
      makeNotification(2, {
        read: false,
        severity: "warning",
        title: "Warning item",
        message: "Needs review",
      }),
      makeNotification(3, {
        read: false,
        severity: "error",
        title: "Error item",
        message: "Needs action",
      }),
    ]

    const wrapper = await mountCenter()
    expect(wrapper.text()).toContain("Informational")
    expect(wrapper.text()).toContain("Warning item")
    expect(wrapper.text()).toContain("Error item")

    await wrapper.get('[data-test="notification-filter-attention"]').trigger("click")

    expect(wrapper.text()).not.toContain("Informational")
    expect(wrapper.text()).toContain("Warning item")
    expect(wrapper.text()).toContain("Error item")
  })

  it("navigates source-backed rows without dismissing them", async () => {
    notificationState.unread = [
      {
        ...makeNotification(1, {
          read: false,
          type: "update",
          title: "Update available",
          message: "Curated v1.4.3 is ready",
        }),
        source: { route: "/settings?section=about" },
      },
    ]

    const wrapper = await mountCenter()
    await wrapper.get('[data-test="notification-row-action"]').trigger("click")

    expect(routerPush).toHaveBeenCalledWith("/settings?section=about")
    expect(notificationState.dismissOne).not.toHaveBeenCalled()
  })

  it("dismisses a row only through the explicit dismiss action", async () => {
    notificationState.unread = [
      {
        ...makeNotification(1, {
          read: false,
          type: "update",
          title: "Update available",
          message: "Curated v1.4.3 is ready",
        }),
        source: { route: "/settings?section=about" },
      },
    ]

    const wrapper = await mountCenter()
    await wrapper.get('[data-test="notification-row-dismiss"]').trigger("click")

    expect(notificationState.dismissOne).toHaveBeenCalledWith("unread-1")
    expect(routerPush).not.toHaveBeenCalled()
  })
})

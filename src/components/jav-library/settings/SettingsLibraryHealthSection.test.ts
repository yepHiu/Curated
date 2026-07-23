import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsLibraryHealthSection from "./SettingsLibraryHealthSection.vue"

const serviceMock = vi.hoisted(() => ({
  scanLibraryHealth: vi.fn(),
  startLibraryHealthRepair: vi.fn(),
  getLibraryHealthRepair: vi.fn(),
  startLibraryHealthAction: vi.fn(),
  getTaskStatus: vi.fn(),
}))

const toastMock = vi.hoisted(() => vi.fn())

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: "en" },
  }),
}))

vi.mock("@/services/library-service", () => ({
  useLibraryService: () => serviceMock,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: toastMock,
}))

const report = {
  scannedAt: "2026-07-20T10:00:00Z",
  status: "attention" as const,
  database: {
    quickCheckOk: true,
    quickCheckMessages: ["ok"],
    foreignKeyOk: true,
    foreignKeyCount: 0,
  },
  storageStatuses: [],
  summary: {
    totalFindings: 2,
    criticalFindings: 0,
    warningFindings: 1,
    infoFindings: 1,
    skippedOfflineFiles: 0,
    categoryCounts: { metadata_missing: 1, movie_poster_missing: 1 },
  },
  findings: [
    {
      id: "health-missing",
      category: "metadata_missing",
      severity: "warning" as const,
      entityType: "movie",
      entityId: "movie-1",
      label: "TEST-001",
      path: "D:\\Movies\\TEST-001.mp4",
      message: "movie still has placeholder metadata",
      repairActions: ["rescrape_metadata"],
    },
    {
      id: "health-poster",
      category: "movie_poster_missing",
      severity: "info" as const,
      entityType: "movie",
      entityId: "movie-1",
      label: "TEST-001",
      message: "movie has no poster",
      repairActions: ["rescrape_metadata"],
    },
  ],
  truncated: false,
}

const completedRepair = {
  repairId: "repair-1",
  taskId: "task-1",
  action: "rescrape_metadata" as const,
  categories: ["metadata_missing"],
  status: "completed" as const,
  totalItems: 1,
  completedItems: 1,
  succeededItems: 1,
  failedItems: 0,
  createdAt: "2026-07-20T10:01:00Z",
  finishedAt: "2026-07-20T10:01:02Z",
  items: [
    {
      ordinal: 0,
      findingId: "health-missing",
      category: "metadata_missing",
      movieId: "movie-1",
      label: "TEST-001",
      status: "succeeded" as const,
      childTaskId: "scrape-1",
    },
  ],
}

describe("SettingsLibraryHealthSection", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    serviceMock.scanLibraryHealth.mockResolvedValue(report)
    serviceMock.startLibraryHealthRepair.mockResolvedValue({
      ...completedRepair,
      status: "pending",
      completedItems: 0,
      succeededItems: 0,
      finishedAt: undefined,
    })
    serviceMock.getLibraryHealthRepair.mockResolvedValue(completedRepair)
    serviceMock.startLibraryHealthAction.mockResolvedValue({
      taskId: "cleanup-1",
      type: "library.health.cleanup",
      status: "running",
      createdAt: "2026-07-20T10:02:00Z",
      startedAt: "2026-07-20T10:02:00Z",
      progress: 0,
      message: "cleaning",
    })
    serviceMock.getTaskStatus.mockResolvedValue({
      taskId: "cleanup-1",
      type: "library.health.cleanup",
      status: "completed",
      createdAt: "2026-07-20T10:02:00Z",
      startedAt: "2026-07-20T10:02:00Z",
      finishedAt: "2026-07-20T10:02:01Z",
      progress: 100,
      message: "cleanup completed",
    })
  })

  it("runs a read-only scan and renders categorized findings", async () => {
    const wrapper = mount(SettingsLibraryHealthSection, {
      props: { supported: true },
      global: { stubs: { teleport: true } },
    })
    const block = wrapper.get('[data-settings-maintenance-block="health"]')
    expect(block.classes()).toContain("p-4")

    await wrapper.get("[data-library-health-scan]").trigger("click")
    await flushPromises()

    expect(serviceMock.scanLibraryHealth).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain("settings.libraryHealthCategoryMetadataMissing")
    expect(wrapper.text()).toContain("TEST-001")
    expect(wrapper.get("[data-library-health-repair-missing]").attributes("disabled")).toBeUndefined()
  })

  it("requires a visible confirmation before starting a bounded metadata repair", async () => {
    const wrapper = mount(SettingsLibraryHealthSection, {
      props: { supported: true },
      attachTo: document.body,
    })
    await wrapper.get("[data-library-health-scan]").trigger("click")
    await flushPromises()
    await wrapper.get("[data-library-health-repair-missing]").trigger("click")
    await flushPromises()

    expect(serviceMock.startLibraryHealthRepair).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain("settings.libraryHealthRepairConfirmTitle")

    const confirm = document.querySelector<HTMLButtonElement>("[data-library-health-repair-confirm]")
    expect(confirm).not.toBeNull()
    confirm?.click()
    await flushPromises()

    expect(serviceMock.startLibraryHealthRepair).toHaveBeenCalledWith({
      action: "rescrape_metadata",
      categories: ["metadata_missing"],
      limit: 1,
      confirm: true,
    })
    expect(serviceMock.getLibraryHealthRepair).toHaveBeenCalledWith("repair-1")
    expect(toastMock).toHaveBeenCalled()
    wrapper.unmount()
  })

  it("disables filesystem diagnostics outside Web API mode", () => {
    const wrapper = mount(SettingsLibraryHealthSection, { props: { supported: false } })
    expect(wrapper.get("[data-library-health-scan]").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("settings.libraryHealthWebRequired")
  })

  it("requires confirmation and exact finding ids for destructive cleanup", async () => {
    serviceMock.scanLibraryHealth.mockResolvedValue({
      ...report,
      summary: {
        ...report.summary,
        totalFindings: 1,
        warningFindings: 1,
        infoFindings: 0,
        categoryCounts: { orphan_user_state: 1 },
      },
      findings: [{
        id: "health-orphan",
        category: "orphan_user_state",
        severity: "warning" as const,
        entityType: "playback_progress",
        entityId: "missing-movie",
        label: "playback_progress",
        message: "user state references a missing movie",
        repairActions: ["cleanup_orphan_state"],
      }],
    })
    const wrapper = mount(SettingsLibraryHealthSection, {
      props: { supported: true },
      attachTo: document.body,
    })
    await wrapper.get("[data-library-health-scan]").trigger("click")
    await flushPromises()
    await wrapper.get("[data-library-health-cleanup-orphans]").trigger("click")
    await flushPromises()

    expect(serviceMock.startLibraryHealthAction).not.toHaveBeenCalled()
    const confirm = document.querySelector<HTMLButtonElement>("[data-library-health-cleanup-confirm]")
    expect(confirm).not.toBeNull()
    confirm?.click()
    await flushPromises()

    expect(serviceMock.startLibraryHealthAction).toHaveBeenCalledWith({
      action: "cleanup_orphan_state",
      findingIds: ["health-orphan"],
      confirm: true,
    })
    expect(serviceMock.getTaskStatus).toHaveBeenCalledWith("cleanup-1")
    wrapper.unmount()
  })
})

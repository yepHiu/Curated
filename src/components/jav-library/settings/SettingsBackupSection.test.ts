import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsBackupSection from "./SettingsBackupSection.vue"

const serviceMock = vi.hoisted(() => ({
  createBackup: vi.fn(),
  verifyBackup: vi.fn(),
  preflightBackupRestore: vi.fn(),
}))

const pickerMock = vi.hoisted(() => vi.fn())
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

vi.mock("@/lib/pick-directory", () => ({
  pickLibraryDirectory: pickerMock,
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: toastMock,
}))

const manifest = {
  format: "curated-backup",
  formatVersion: 1,
  createdAt: "2026-07-20T02:00:00Z",
  appVersion: "1.4.11",
  appChannel: "dev",
  scope: {
    databaseIncluded: true,
    libraryConfigIncluded: true,
    userAssetsIncluded: false,
    mediaFilesIncluded: false,
  },
  schemaMigrations: ["0001_init.sql"],
  files: [{ kind: "database", path: "database/curated.db", sizeBytes: 1024, sha256: "a".repeat(64) }],
}

const verification = {
  valid: true,
  checkedAt: "2026-07-20T02:01:00Z",
  manifest,
  databaseIntegrity: { quickCheck: "ok", foreignKeyViolations: 0 },
  errors: [],
  warnings: [],
}

describe("SettingsBackupSection", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    serviceMock.createBackup.mockResolvedValue(manifest)
    serviceMock.verifyBackup.mockResolvedValue(verification)
    serviceMock.preflightBackupRestore.mockResolvedValue({
      canRestore: true,
      checkedAt: "2026-07-20T02:02:00Z",
      verification,
      targetDatabase: "D:\\Curated\\curated.db",
      targetDatabaseExists: true,
      targetConfig: "D:\\Curated\\library-config.cfg",
      targetConfigExists: true,
      requiredBytes: 2048,
      availableBytes: 4096,
      availableBytesKnown: true,
      unsupportedMigrations: [],
      errors: [],
      warnings: ["media files are not included"],
    })
  })

  it("creates a package and immediately verifies it", async () => {
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    const block = wrapper.get('[data-settings-maintenance-block="backup"]')
    expect(block.classes()).toContain("p-4")

    await wrapper.get("[data-settings-backup-path]").setValue("D:\\Backups\\curated")
    await wrapper.get("[data-settings-backup-create]").trigger("click")
    await flushPromises()

    expect(serviceMock.createBackup).toHaveBeenCalledWith(
      "D:\\Backups\\curated.curated-backup",
    )
    expect(serviceMock.verifyBackup).toHaveBeenCalledWith(
      "D:\\Backups\\curated.curated-backup",
    )
    expect(wrapper.text()).toContain("settings.backupValid")
    expect(toastMock).toHaveBeenCalled()
  })

  it("uses the native directory outcome to build a timestamped package path", async () => {
    pickerMock.mockResolvedValueOnce({ status: "ok", path: "D:\\Backups" })
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    await wrapper.get("[data-settings-backup-pick]").trigger("click")
    await flushPromises()

    const value = (wrapper.get("[data-settings-backup-path]").element as HTMLInputElement).value
    expect(value).toMatch(/^D:\\Backups\\curated-\d{8}-\d{6}Z\.curated-backup$/)
  })

  it("runs restore preflight without exposing an online restore action", async () => {
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    await wrapper.get("[data-settings-backup-path]").setValue(
      "D:\\Backups\\curated.curated-backup",
    )
    await wrapper.get("[data-settings-backup-preflight]").trigger("click")
    await flushPromises()

    expect(serviceMock.preflightBackupRestore).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain("settings.backupPreflightReady")
    expect(wrapper.find("[data-settings-backup-restore]").exists()).toBe(false)
  })

  it("disables filesystem actions outside Web API mode", () => {
    const wrapper = mount(SettingsBackupSection, { props: { supported: false } })
    expect(wrapper.get("[data-settings-backup-create]").attributes("disabled")).toBeDefined()
    expect(wrapper.get("[data-settings-backup-verify]").attributes("disabled")).toBeDefined()
    expect(wrapper.get("[data-settings-backup-preflight]").attributes("disabled")).toBeDefined()
  })
})

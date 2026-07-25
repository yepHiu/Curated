import { flushPromises, mount } from "@vue/test-utils"
import { beforeEach, describe, expect, it, vi } from "vitest"
import SettingsBackupSection from "./SettingsBackupSection.vue"

const serviceMock = vi.hoisted(() => ({
  backupDirectory: { value: "" },
  createBackup: vi.fn(),
  setBackupDirectory: vi.fn(),
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
    serviceMock.backupDirectory.value = ""
    serviceMock.createBackup.mockResolvedValue(manifest)
    serviceMock.setBackupDirectory.mockImplementation(async (directory: string) => {
      serviceMock.backupDirectory.value = directory
    })
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

    await wrapper.get("[data-settings-backup-directory]").setValue("D:\\Backups")
    await wrapper.get("[data-settings-backup-create]").trigger("click")
    await flushPromises()

    expect(serviceMock.createBackup).toHaveBeenCalledWith(
      expect.stringMatching(/^D:\\Backups\\curated-\d{8}-\d{6}Z\.curated-backup$/),
    )
    const createdPath = serviceMock.createBackup.mock.calls[0]?.[0]
    expect(serviceMock.setBackupDirectory).toHaveBeenCalledWith("D:\\Backups")
    expect(serviceMock.verifyBackup).toHaveBeenCalledWith(createdPath)
    expect((wrapper.get("[data-settings-backup-path]").element as HTMLInputElement).value).toBe(
      createdPath,
    )
    expect(wrapper.text()).toContain("settings.backupValid")
    expect(toastMock).toHaveBeenCalled()
  })

  it("prefills the remembered backup directory", () => {
    serviceMock.backupDirectory.value = "D:\\Remembered"
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })

    expect(
      (wrapper.get("[data-settings-backup-directory]").element as HTMLInputElement).value,
    ).toBe("D:\\Remembered")
  })

  it("does not persist the directory when backup creation fails", async () => {
    serviceMock.createBackup.mockRejectedValueOnce(new Error("create failed"))
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    await wrapper.get("[data-settings-backup-directory]").setValue("D:\\Backups")
    await wrapper.get("[data-settings-backup-create]").trigger("click")
    await flushPromises()

    expect(serviceMock.setBackupDirectory).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("create failed")
  })

  it("keeps verifying a created backup when saving the directory preference fails", async () => {
    serviceMock.setBackupDirectory.mockRejectedValueOnce(new Error("settings unavailable"))
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    await wrapper.get("[data-settings-backup-directory]").setValue("D:\\Backups")
    await wrapper.get("[data-settings-backup-create]").trigger("click")
    await flushPromises()

    expect(serviceMock.verifyBackup).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain("settings.backupDirectorySaveFailed")
    expect(wrapper.text()).toContain("settings.backupValid")
    expect(
      (wrapper.get("[data-settings-backup-directory]").element as HTMLInputElement).value,
    ).toBe("D:\\Backups")
  })

  it("keeps the native directory outcome as the backup destination", async () => {
    pickerMock.mockResolvedValueOnce({ status: "ok", path: "D:\\Backups" })
    const wrapper = mount(SettingsBackupSection, { props: { supported: true } })
    await wrapper.get("[data-settings-backup-pick]").trigger("click")
    await flushPromises()

    const value = (wrapper.get("[data-settings-backup-directory]").element as HTMLInputElement).value
    expect(value).toBe("D:\\Backups")
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

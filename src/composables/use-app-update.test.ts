import { flushPromises } from "@vue/test-utils"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const apiMocks = vi.hoisted(() => ({
  getAppUpdateStatus: vi.fn(),
  checkAppUpdateNow: vi.fn(),
  getSettings: vi.fn(),
  downloadAppUpdateInstaller: vi.fn(),
  installAppUpdate: vi.fn(),
  clearDownloadedAppUpdateInstaller: vi.fn(),
}))

const notificationMocks = vi.hoisted(() => ({
  addNotification: vi.fn(),
}))

vi.mock("@/api/endpoints", () => ({
  api: apiMocks,
}))

vi.mock("@/composables/use-notification-center", () => ({
  useNotificationCenter: () => ({
    addNotification: notificationMocks.addNotification,
  }),
}))

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string, params?: Record<string, unknown>) =>
        params?.version ? `${key}:${params.version}` : key,
    },
  },
}))

async function loadUseAppUpdate() {
  const { useAppUpdate } = await import("./use-app-update")
  return useAppUpdate()
}

beforeEach(() => {
  vi.resetModules()
  vi.unstubAllEnvs()
  vi.useRealTimers()
  delete window.javLibrary
  apiMocks.getAppUpdateStatus.mockReset()
  apiMocks.getAppUpdateStatus.mockResolvedValue({
    supported: true, localUpdateAllowed: true, status: "up-to-date",
  })
  apiMocks.checkAppUpdateNow.mockReset()
  apiMocks.getSettings.mockReset()
  apiMocks.downloadAppUpdateInstaller.mockReset()
  apiMocks.installAppUpdate.mockReset()
  apiMocks.clearDownloadedAppUpdateInstaller.mockReset()
  notificationMocks.addNotification.mockReset()
})

afterEach(() => {
  delete window.javLibrary
  vi.unstubAllEnvs()
  vi.useRealTimers()
})

describe("useAppUpdate", () => {
  it("immediately blocks server updates while remote simulation is enabled", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    vi.stubEnv("VITE_API_BASE_URL", "http://localhost:8080/api")
    const { setDevRemoteSimulation } = await import("@/lib/dev-remote-simulation")
    const state = await loadUseAppUpdate()
    await state.refreshStatus()
    expect(state.localUpdateAllowed.value).toBe(true)
    setDevRemoteSimulation(true)
    expect(state.localUpdateAllowed.value).toBe(false)
    await state.downloadInstaller()
    await state.installUpdate()
    await state.clearDownloadedInstaller()
    expect(apiMocks.downloadAppUpdateInstaller).not.toHaveBeenCalled()
    expect(apiMocks.installAppUpdate).not.toHaveBeenCalled()
    expect(apiMocks.clearDownloadedAppUpdateInstaller).not.toHaveBeenCalled()
    setDevRemoteSimulation(false)
    await state.refreshStatus()
    expect(state.localUpdateAllowed.value).toBe(true)
  })

  it("reports unsupported when Web API is disabled", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "false")

    const state = await loadUseAppUpdate()

    expect(state.useWebApi).toBe(false)
    expect(state.status.value).toBe("unsupported")
    expect(state.loaded.value).toBe(true)
    expect(state.summary.value?.supported).toBe(false)
    expect(apiMocks.getAppUpdateStatus).not.toHaveBeenCalled()
  })

  it("loads update status on demand", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      installedVersion: "1.0.0",
      latestVersion: "1.1.0",
    })

    const state = await loadUseAppUpdate()
    state.ensureLoaded()

    expect(state.status.value).toBe("checking")
    expect(state.loading.value).toBe(true)
    await flushPromises()

    expect(apiMocks.getAppUpdateStatus).toHaveBeenCalledTimes(1)
    expect(state.loaded.value).toBe(true)
    expect(state.status.value).toBe("update-available")
    expect(state.hasUpdateBadge.value).toBe(true)
    expect(state.summary.value?.latestVersion).toBe("1.1.0")
  })

  it("runs a silent manual check without toggling loading", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.checkAppUpdateNow.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "up-to-date",
      hasUpdate: false,
      installedVersion: "1.0.0",
      latestVersion: "1.0.0",
    })

    const state = await loadUseAppUpdate()
    expect(state.loading.value).toBe(false)

    const resultPromise = state.checkNowSilent()
    expect(state.loading.value).toBe(false)

    const result = await resultPromise
    expect(apiMocks.checkAppUpdateNow).toHaveBeenCalledTimes(1)
    expect(result?.status).toBe("up-to-date")
    expect(state.loading.value).toBe(false)
  })

  it("keeps cached release notes when a silent manual check fails", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      installedVersion: "1.0.0",
      latestVersion: "1.1.0",
      releaseName: "v1.1.0",
      releaseUrl: "https://example.com/releases/v1.1.0",
      releaseNotesSnippet: "Bug fixes",
    })
    apiMocks.checkAppUpdateNow.mockRejectedValueOnce(new Error("offline"))

    const state = await loadUseAppUpdate()
    state.ensureLoaded()
    await flushPromises()

    expect(state.summary.value?.releaseNotesSnippet).toBe("Bug fixes")

    const result = await state.checkNowSilent()

    expect(result?.status).toBe("error")
    expect(state.summary.value?.releaseName).toBe("v1.1.0")
    expect(state.summary.value?.releaseNotesSnippet).toBe("Bug fixes")
  })

  it("runs a manual update check and stores errors", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.checkAppUpdateNow.mockRejectedValueOnce(new Error("release lookup failed"))

    const state = await loadUseAppUpdate()
    const result = await state.checkNow()

    expect(apiMocks.checkAppUpdateNow).toHaveBeenCalledTimes(1)
    expect(result?.status).toBe("error")
    expect(state.status.value).toBe("error")
    expect(state.loaded.value).toBe(true)
    expect(state.errorMessage.value).toBe("release lookup failed")
  })

  it("downloads the installer and updates artifact state", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.downloadAppUpdateInstaller.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      latestVersion: "1.4.5",
      artifactStatus: "verified",
      downloadedVersion: "1.4.5",
      downloadedFileName: "Curated-Setup-1.4.5.exe",
      downloadedBytes: 20,
      totalBytes: 20,
      downloadProgress: 100,
      installReady: true,
    })

    const state = await loadUseAppUpdate()
    const result = await state.downloadInstaller()

    expect(apiMocks.downloadAppUpdateInstaller).toHaveBeenCalledTimes(1)
    expect(result?.artifactStatus).toBe("verified")
    expect(state.summary.value?.artifactStatus).toBe("verified")
    expect(state.summary.value?.installReady).toBe(true)
    expect(state.installing.value).toBe(false)
  })

  it("starts installer with explicit mode", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.installAppUpdate.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      artifactStatus: "install-launched",
      installReady: false,
    })

    const state = await loadUseAppUpdate()
    const result = await state.installUpdate("silent")

    expect(apiMocks.installAppUpdate).toHaveBeenCalledWith({ mode: "silent" })
    expect(result?.artifactStatus).toBe("install-launched")
    expect(state.summary.value?.artifactStatus).toBe("install-launched")
    expect(state.installing.value).toBe(false)
  })

  it("schedules only one silent auto check for multiple consumers", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "up-to-date",
      hasUpdate: false,
      installedVersion: "1.0.0",
      latestVersion: "1.0.0",
    })

    const first = await loadUseAppUpdate()
    const second = await loadUseAppUpdate()
    expect(first.loaded.value).toBe(false)
    expect(second.loaded.value).toBe(false)

    await vi.advanceTimersByTimeAsync(12_000)
    await flushPromises()

    expect(apiMocks.getAppUpdateStatus).toHaveBeenCalledTimes(1)
    expect(first.status.value).toBe("up-to-date")
    expect(second.status.value).toBe("up-to-date")
    expect(first.loading.value).toBe(false)
  })

  it("records a notification when a silent auto check finds an update", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      installedVersion: "1.0.0",
      latestVersion: "1.1.0",
    })

    await loadUseAppUpdate()
    await vi.advanceTimersByTimeAsync(12_000)
    await flushPromises()

    expect(notificationMocks.addNotification).toHaveBeenCalledWith({
      type: "update",
      severity: "warning",
      title: "notificationCenter.titles.updateAvailable",
      message: "settings.appUpdateToastAvailable:1.1.0",
      source: { route: "/settings?section=about" },
      messageId: "MSG-0024",
    })
  })

  it("auto-downloads a verified installer after the scheduled update check when enabled", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      installedVersion: "1.0.0",
      latestVersion: "1.1.0",
      installerDownloadUrl: "https://example.com/Curated-Setup-1.1.0.exe",
      installerSha256: "A".repeat(64),
    })
    apiMocks.getSettings.mockResolvedValueOnce({
      autoDownloadUpdates: true,
    })
    apiMocks.downloadAppUpdateInstaller.mockResolvedValueOnce({
      supported: true,
      localUpdateAllowed: true,
      status: "update-available",
      hasUpdate: true,
      latestVersion: "1.1.0",
      artifactStatus: "verified",
      installReady: true,
    })

    const state = await loadUseAppUpdate()
    await vi.advanceTimersByTimeAsync(12_000)
    await flushPromises()

    expect(apiMocks.getSettings).toHaveBeenCalledTimes(1)
    expect(apiMocks.downloadAppUpdateInstaller).toHaveBeenCalledTimes(1)
    expect(state.summary.value?.artifactStatus).toBe("verified")
    expect(state.summary.value?.installReady).toBe(true)
  })
})


describe("remote update isolation", () => {
  it.each([false, undefined])("blocks all installer mutations when server permission is %s", async (allowed) => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    apiMocks.getAppUpdateStatus.mockResolvedValue({ supported: true, status: "update-available", localUpdateAllowed: allowed, installReady: true, artifactStatus: "verified" })
    const state = await loadUseAppUpdate()
    await state.downloadInstaller()
    await state.installUpdate()
    await state.clearDownloadedInstaller()
    expect(state.localUpdateAllowed.value).toBe(false)
    expect(apiMocks.downloadAppUpdateInstaller).not.toHaveBeenCalled()
    expect(apiMocks.installAppUpdate).not.toHaveBeenCalled()
    expect(apiMocks.clearDownloadedAppUpdateInstaller).not.toHaveBeenCalled()
  })

  it.each(["remote", "old-bridge", "failed-bridge", "standalone", "remote-api"])("does not mutate Server with a %s target even if it reports local permission", async (kind) => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    if (kind === "remote-api") vi.stubEnv("VITE_API_BASE_URL", "https://nas.example/api")
    window.javLibrary = kind === "old-bridge" ? {} : {
      getDesktopInfo: kind === "failed-bridge" ? vi.fn().mockRejectedValue(new Error("offline")) : vi.fn().mockResolvedValue({
        version: "0.1.0", development: false, distribution: kind === "standalone" ? "desktop" : "legacy",
        serverOrigin: kind === "remote" ? "https://nas.example" : "http://localhost:8080",
      }),
    }
    const state = await loadUseAppUpdate()
    await state.downloadInstaller()
    await state.installUpdate()
    await state.clearDownloadedInstaller()
    expect(apiMocks.downloadAppUpdateInstaller).not.toHaveBeenCalled()
    expect(apiMocks.installAppUpdate).not.toHaveBeenCalled()
    expect(apiMocks.clearDownloadedAppUpdateInstaller).not.toHaveBeenCalled()
  })

  it("does not auto-download or announce a Server update on a remote Desktop", async () => {
    vi.useFakeTimers()
    vi.stubEnv("VITE_USE_WEB_API", "true")
    window.javLibrary = { getDesktopInfo: vi.fn().mockResolvedValue({ distribution: "legacy", serverOrigin: "http://192.168.1.20:8081" }) }
    apiMocks.getAppUpdateStatus.mockResolvedValue({ supported: true, status: "update-available", localUpdateAllowed: true, hasUpdate: true, latestVersion: "2.0.0", installerDownloadUrl: "https://example.com/full.exe", installerSha256: "a".repeat(64) })
    apiMocks.getSettings.mockResolvedValue({ autoDownloadUpdates: true })
    const state = await loadUseAppUpdate()
    await vi.advanceTimersByTimeAsync(12000)
    expect(state.hasUpdateBadge.value).toBe(false)
    expect(apiMocks.getSettings).not.toHaveBeenCalled()
    expect(apiMocks.downloadAppUpdateInstaller).not.toHaveBeenCalled()
    expect(notificationMocks.addNotification).not.toHaveBeenCalled()
  })

  it("rechecks permission before installing an already verified artifact", async () => {
    vi.stubEnv("VITE_USE_WEB_API", "true")
    const state = await loadUseAppUpdate()
    state.ensureLoaded()
    await flushPromises()
    expect(state.localUpdateAllowed.value).toBe(true)
    apiMocks.getAppUpdateStatus.mockResolvedValue({ supported: true, status: "update-available", localUpdateAllowed: false, installReady: true, artifactStatus: "verified" })
    await state.installUpdate()
    expect(apiMocks.installAppUpdate).not.toHaveBeenCalled()
  })
})

import { flushPromises, mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { beforeEach, describe, expect, it, vi } from "vitest"

import SettingsAppUpdateSection from "./SettingsAppUpdateSection.vue"

const checkNowMock = vi.hoisted(() => vi.fn())
const checkNowSilentMock = vi.hoisted(() => vi.fn())
const downloadInstallerMock = vi.hoisted(() => vi.fn())
const installUpdateMock = vi.hoisted(() => vi.fn())
const clearDownloadedInstallerMock = vi.hoisted(() => vi.fn())
const ensureLoadedMock = vi.hoisted(() => vi.fn())
const pushAppToastMock = vi.hoisted(() => vi.fn())
const appUpdateSummaryRef = vi.hoisted(() => ({ current: null as { value: AppUpdateSummary } | null }))
const appUpdateSummarySeed = vi.hoisted(() => ({ current: null as AppUpdateSummary | null }))

type AppUpdateSummary = {
  supported: boolean
  status: "unsupported" | "up-to-date" | "update-available" | "error"
  installedVersion?: string
  latestVersion?: string
  hasUpdate?: boolean
  publishedAt?: string
  checkedAt?: string
  releaseName?: string
  releaseUrl?: string
  installerDownloadUrl?: string
  installerSha256?: string
  artifactStatus?: string
  downloadedVersion?: string
  downloadedFileName?: string
  downloadedBytes?: number
  totalBytes?: number
  downloadProgress?: number
  installReady?: boolean
  releaseNotesSnippet?: string
}

function createAppUpdateSummary(overrides: Partial<AppUpdateSummary> = {}): AppUpdateSummary {
  return {
    supported: true,
    status: "update-available",
    installedVersion: "0.0.0",
    latestVersion: "1.2.8",
    hasUpdate: true,
    publishedAt: "2026-04-19T12:00:00Z",
    checkedAt: "2026-04-19T13:00:00Z",
    releaseName: "v1.2.8",
    releaseUrl: "https://github.com/yepHiu/Curated/releases/tag/v1.2.8",
    installerDownloadUrl: "https://github.com/yepHiu/Curated/releases/download/v1.2.8/Curated-Setup-1.2.8.exe",
    installerSha256: "ABCDEF",
    releaseNotesSnippet: "Bug fixes",
    ...overrides,
  }
}

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: ref("zh-CN"),
    t: (key: string, params?: Record<string, string>) => {
      const messages: Record<string, string> = {
        "settings.appUpdateSectionTitle": "版本信息与更新",
        "settings.appUpdateAvailableTitle": "发现可用更新",
        "settings.appUpdateAvailableBody": "可前往 Release 页面下载安装。",
        "settings.appUpdateCheckAction": "检查更新",
        "settings.appUpdateDownloadAction": "打开 Release 页面",
        "settings.appUpdateDownloadInstallerAction": "下载最新安装包",
        "settings.appUpdateDownloadAndInstallAction": "下载并安装",
        "settings.appUpdateDownloadingAction": "正在下载…",
        "settings.appUpdateInstallReadyAction": "立即安装",
        "settings.appUpdateInstallingAction": "正在启动…",
        "settings.appUpdateOpenReleaseAction": "打开 Release 页面",
        "settings.aboutVersionLabel": "Curated Server 版本号",
        "settings.aboutInstallerVersionLabel": "安装包版本号",
        "settings.appUpdateReleaseNotesExpand": "展开全文",
        "settings.appUpdateReleaseNotesCollapse": "收起全文",
      }

      if (params) {
        return `${messages[key] ?? key}:${Object.values(params).join("|")}`
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock("@/composables/use-app-update", () => ({
  useAppUpdate: () => {
    const summary = ref(appUpdateSummarySeed.current ?? createAppUpdateSummary())
    appUpdateSummaryRef.current = summary
    return {
      summary: computed(() => summary.value),
      status: computed(() => "update-available"),
      loading: ref(false),
      downloading: ref(false),
      installing: ref(false),
      hasUpdateBadge: computed(() => true),
      ensureLoaded: ensureLoadedMock,
      checkNow: checkNowMock,
      checkNowSilent: checkNowSilentMock,
      downloadInstaller: downloadInstallerMock,
      installUpdate: installUpdateMock,
      clearDownloadedInstaller: clearDownloadedInstallerMock,
    }
  },
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: pushAppToastMock,
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button v-bind=\"$attrs\"><slot /></button>" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

describe("SettingsAppUpdateSection", () => {
  beforeEach(() => {
    checkNowMock.mockReset()
    checkNowSilentMock.mockReset()
    downloadInstallerMock.mockReset()
    installUpdateMock.mockReset()
    clearDownloadedInstallerMock.mockReset()
    ensureLoadedMock.mockReset()
    pushAppToastMock.mockReset()
    appUpdateSummarySeed.current = null
    if (appUpdateSummaryRef.current) {
      appUpdateSummaryRef.current.value = createAppUpdateSummary()
    }
  })

  it("renders backend build version plus a merged installer version summary", async () => {
    const wrapper = mount(SettingsAppUpdateSection, {
      props: {
        backendVersionDisplay: "20260419.102030-dev",
      },
    })
    const text = wrapper.text()

    expect(text).toContain("Curated Server 版本号")
    expect(text).toContain("20260419.102030-dev")
    expect(text).toContain("安装包版本号")
    expect(text).toContain("0.0.0")
    expect(text).toContain("1.2.8")
    expect(text).toContain("发现可用更新")
    expect(text).toContain("可前往 Release 页面下载安装。")
    expect(text).not.toContain("发现可安装的新版本")
    expect(text).not.toContain("settings.appUpdateCurrentVersionLabel")
    expect(text).not.toContain("settings.appUpdateLatestVersionLabel")
    expect(text).not.toContain("settings.appUpdatePublishedAtLabel")
    expect(text).not.toContain("settings.appUpdateCheckedAtLabel")

    const download = wrapper.get("[data-app-update-download-installer]")
    expect(download.text()).toContain("下载并安装")

    await download.trigger("click")
    expect(downloadInstallerMock).toHaveBeenCalledTimes(1)

    const release = wrapper.get("[data-app-update-release]")
    expect(release.attributes("href")).toBe(
      "https://github.com/yepHiu/Curated/releases/tag/v1.2.8",
    )

    const notes = wrapper.get("[data-app-update-release-notes]")
    expect(notes.classes()).toContain("line-clamp-3")

    await wrapper.get("[data-app-update-release-notes-toggle]").trigger("click")
    await flushPromises()

    const notesExpanded = wrapper.get("[data-app-update-release-notes]")
    expect(notesExpanded.classes()).not.toContain("line-clamp-3")
    expect(notesExpanded.classes()).toContain("overflow-y-auto")
  })

  it("launches the installer when the verified update is ready", async () => {
    appUpdateSummarySeed.current = createAppUpdateSummary({
      artifactStatus: "verified",
      downloadedVersion: "1.2.8",
      downloadedFileName: "Curated-Setup-1.2.8.exe",
      installReady: true,
    })

    const wrapper = mount(SettingsAppUpdateSection)
    const install = wrapper.get("[data-app-update-install]")

    expect(install.text()).toContain("立即安装")

    await install.trigger("click")
    expect(installUpdateMock).toHaveBeenCalledWith("interactive")
  })

  it("refreshes release notes on first expand so legacy short cached notes can show the full body", async () => {
    const legacySnippet = "Bug fixes"
    const fullNotes = "Bug fixes\n\n- Fixed updater download naming\n- Added full release notes"
    appUpdateSummarySeed.current = createAppUpdateSummary({
      releaseNotesSnippet: legacySnippet,
    })
    checkNowSilentMock.mockImplementationOnce(async () => {
      if (appUpdateSummaryRef.current) {
        appUpdateSummaryRef.current.value = createAppUpdateSummary({
          releaseNotesSnippet: fullNotes,
        })
      }
      return appUpdateSummaryRef.current?.value
    })

    const wrapper = mount(SettingsAppUpdateSection)

    expect(wrapper.get("[data-app-update-release-notes]").text()).toBe(legacySnippet)

    await wrapper.get("[data-app-update-release-notes-toggle]").trigger("click")
    await flushPromises()

    const notesExpanded = wrapper.get("[data-app-update-release-notes]")
    expect(checkNowSilentMock).toHaveBeenCalledTimes(1)
    expect(notesExpanded.text()).toContain("Added full release notes")
  })

  it("adds a route source when manual update checks find an update", async () => {
    checkNowMock.mockResolvedValueOnce({
      supported: true,
      status: "update-available",
      hasUpdate: true,
      latestVersion: "1.3.0",
    })

    const wrapper = mount(SettingsAppUpdateSection)

    await wrapper.get("[data-app-update-check]").trigger("click")
    await flushPromises()

    expect(pushAppToastMock).toHaveBeenCalledWith(
      "settings.appUpdateToastAvailable:1.3.0",
      expect.objectContaining({
        variant: "warning",
        notification: expect.objectContaining({
          type: "update",
          title: "notificationCenter.titles.updateAvailable",
          source: { route: "/settings?section=about" },
        }),
      }),
    )
  })
})

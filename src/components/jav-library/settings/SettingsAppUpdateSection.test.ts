import { mount } from "@vue/test-utils"
import { computed, ref } from "vue"
import { describe, expect, it, vi } from "vitest"

import SettingsAppUpdateSection from "./SettingsAppUpdateSection.vue"

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
        "settings.aboutVersionLabel": "版本号",
        "settings.aboutInstallerVersionLabel": "安装包版本号",
        "settings.appUpdateReleaseNotesExpand": "展开摘要",
        "settings.appUpdateReleaseNotesCollapse": "收起摘要",
      }

      if (params) {
        return `${messages[key] ?? key}:${Object.values(params).join("|")}`
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock("@/composables/use-app-update", () => ({
  useAppUpdate: () => ({
    summary: computed(() => ({
      supported: true,
      status: "update-available",
      installedVersion: "0.0.0",
      latestVersion: "1.2.8",
      hasUpdate: true,
      publishedAt: "2026-04-19T12:00:00Z",
      checkedAt: "2026-04-19T13:00:00Z",
      releaseName: "v1.2.8",
      releaseUrl: "https://github.com/yepHiu/Curated/releases/tag/v1.2.8",
      releaseNotesSnippet: "Bug fixes",
    })),
    status: computed(() => "update-available"),
    loading: ref(false),
    hasUpdateBadge: computed(() => true),
    ensureLoaded: vi.fn(),
    checkNow: vi.fn(),
  }),
}))

vi.mock("@/composables/use-app-toast", () => ({
  pushAppToast: vi.fn(),
}))

vi.mock("@/components/ui/button", () => ({
  Button: { name: "Button", template: "<button><slot /></button>" },
}))

vi.mock("@/components/ui/badge", () => ({
  Badge: { name: "Badge", template: "<span><slot /></span>" },
}))

describe("SettingsAppUpdateSection", () => {
  it("renders backend build version plus a merged installer version summary", async () => {
    const wrapper = mount(SettingsAppUpdateSection, {
      props: {
        backendVersionDisplay: "20260419.102030-dev",
      },
    })
    const text = wrapper.text()

    expect(text).toContain("版本号")
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

    const notes = wrapper.get("[data-app-update-release-notes]")
    expect(notes.classes()).toContain("line-clamp-1")

    await wrapper.get("[data-app-update-release-notes-toggle]").trigger("click")

    expect(notes.classes()).not.toContain("line-clamp-1")
  })
})

import { describe, expect, it } from "vitest"
import en from "@/locales/en.json"
import ja from "@/locales/ja.json"
import zhCN from "@/locales/zh-CN.json"

const locales = {
  en,
  ja,
  "zh-CN": zhCN,
} satisfies Record<string, Record<string, unknown>>

const requiredLocaleKeys = [
  "curated.tagFilterTitle",
  "curated.tagFilterAll",
  "curated.tagFilterEmpty",
  "curated.tagFilterNoMatches",
  "curated.tagFilterShowMore",
  "curated.tagFilterShowLess",
  "curated.ariaFilterFrameTag",
  "curated.ariaClearFrameTagFilter",
  "settings.curatedExportFormatSaving",
  "scan.statusLabel",
  "scan.completed",
  "scan.finished",
  "scan.scanning",
  "scan.close",
  "scan.processed",
  "scan.newItems",
  "scan.updated",
  "scan.skipped",
  "toasts.libraryWatchScanDoneWithChanges",
  "toasts.libraryWatchScanDoneNoChanges",
  "scanTask.fetchFailed",
  "rating.ariaLabel",
  "rating.score",
  "movie.expandSummary",
  "movie.collapseSummary",
  "preview.title",
  "preview.instructions",
  "preview.close",
  "preview.previous",
  "preview.next",
  "preview.imageOf",
  "settings.appUpdateDownloadInstallerAction",
  "settings.navSecurity",
  "settings.securityTitle",
  "settings.securityDesc",
  "settings.securitySetupTitle",
  "settings.securitySetupHint",
  "settings.securityEnabledHint",
  "settings.securityPinPlaceholder",
  "settings.securityConfirmPinPlaceholder",
  "settings.securityCurrentPinPlaceholder",
  "settings.securityNewPinPlaceholder",
  "settings.securityConfirmNewPinPlaceholder",
  "settings.securitySaving",
  "settings.securityEnablePin",
  "settings.securityChangePin",
  "settings.securitySetupInvalid",
  "settings.securitySetupSaved",
  "settings.securityPinChanged",
  "settings.securitySettingsSaved",
  "settings.securitySaveFailed",
  "settings.securityPinEnabled",
  "settings.securityLockNow",
  "settings.securityLockNowHint",
  "settings.securityLockedNow",
  "settings.securitySessionTitle",
  "settings.securitySessionHint",
  "settings.securitySession15",
  "settings.securitySession60",
  "settings.securitySession240",
  "settings.securitySession1440",
  "settings.securityLanRequiresPin",
  "settings.securityLanRequiresPinHint",
  "settings.securityLockOnRestart",
  "settings.securityLockOnRestartHint",
  "settings.securityTrustDeviceTitle",
  "settings.securityTrustDeviceHint",
  "settings.securityLanPolicyTitle",
  "settings.securityLanPolicyHint",
  "lock.title",
  "lock.pinLabel",
  "lock.pinTooShort",
  "lock.unlockFailed",
  "lock.trustForever",
  "lock.unlocking",
  "lock.unlock",
  "lock.forgotPin",
  "lock.forgotPinHint",
  "player.feedbackPlay",
  "player.feedbackPause",
  "player.hideStats",
  "player.showStats",
]

function readLocaleKey(messages: Record<string, unknown>, key: string): unknown {
  let cursor: unknown = messages
  for (const segment of key.split(".")) {
    if (!cursor || typeof cursor !== "object" || Array.isArray(cursor)) {
      return undefined
    }
    cursor = (cursor as Record<string, unknown>)[segment]
  }
  return cursor
}

describe("locale key parity", () => {
  it.each(Object.entries(locales))("%s has curated tag filter and saving keys", (_locale, messages) => {
    const missing = requiredLocaleKeys.filter((key) => {
      const value = readLocaleKey(messages, key)
      return typeof value !== "string" || value.trim() === ""
    })

    expect(missing).toEqual([])
  })
})

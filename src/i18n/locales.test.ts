import { describe, expect, it, vi } from "vitest"
import en from "@/locales/en.json"
import ja from "@/locales/ja.json"
import zhCN from "@/locales/zh-CN.json"
import { ensureLocaleMessages, i18n } from "@/i18n"

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
  "curated.tagFilterSelectedCount",
  "curated.tagFilterSelectedList",
  "curated.tagFilterShowMore",
  "curated.tagFilterShowLess",
  "curated.ariaFilterFrameTag",
  "curated.ariaClearFrameTagFilter",
  "settings.movieCsvExport",
  "settings.movieCsvExporting",
  "settings.movieCsvExportSuccess",
  "settings.movieCsvExportFailed",
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
  "toasts.manualMovieScrapeStarted",
  "toasts.manualMovieScrapeDone",
  "toasts.manualMovieScrapeFailed",
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
  "settings.securityDisablePin",
  "settings.securityDisablePinTitle",
  "settings.securityDisablePinHint",
  "settings.securityDisablePinConfirm",
  "settings.securityDisablePinSaved",
  "settings.lanAccessTitle",
  "settings.lanAccessSwitch",
  "settings.lanAccessRestartHint",
  "settings.lanAccessMockHint",
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
  "settings.securityLockOnRestart",
  "settings.securityLockOnRestartHint",
  "settings.securityTrustedSessionsTitle",
  "settings.securityTrustedSessionsHint",
  "settings.securityTrustedSessionsRefresh",
  "settings.securityTrustedSessionsEmpty",
  "settings.securityTrustedSessionsCurrent",
  "settings.securityTrustedSessionsTrusted",
  "settings.securityTrustedSessionsUnknownClient",
  "settings.securityTrustedSessionsUnknownIp",
  "settings.securityTrustedSessionsLastSeen",
  "settings.securityTrustedSessionsRevoke",
  "settings.securityTrustedSessionsRevokeOthers",
  "settings.securityTrustedSessionsConfirmTitle",
  "settings.securityTrustedSessionsConfirmOne",
  "settings.securityTrustedSessionsConfirmOthers",
  "settings.securityTrustedSessionsRevoked",
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
  "detailPanel.ariaMetadataProvider",
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

function collectLocaleStrings(
  value: unknown,
  path: string[] = [],
): Array<{ path: string; value: string }> {
  if (typeof value === "string") {
    return [{ path: path.join("."), value }]
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return []
  }
  return Object.entries(value).flatMap(([key, nested]) =>
    collectLocaleStrings(nested, [...path, key]),
  )
}

describe("locale key parity", () => {
  it.each(["en", "ja"] as const)("loads %s messages on demand", async (locale) => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(locales[locale]), { status:200, headers:{'Content-Type':'application/json'} }))
    try { await ensureLocaleMessages(locale) } finally { fetchMock.mockRestore() }
    const messages = i18n.global.getLocaleMessage(locale) as Record<string, unknown>
    expect(Object.keys(messages).length).toBeGreaterThan(0)
    expect(messages.common).toBeTypeOf("object")
  })

  it("uses the concise Chinese movie library sidebar label", () => {
    expect(readLocaleKey(zhCN, "nav.library")).toBe("影片")
  })

  it.each([
    ["zh-CN", zhCN],
    ["ja", ja],
  ] as const)("%s photo library copy does not contain replacement question marks", (_locale, messages) => {
    const photoCopy = [
      ...collectLocaleStrings(readLocaleKey(messages, "photos"), ["photos"]),
      ...collectLocaleStrings(readLocaleKey(messages, "settings"), ["settings"]).filter(({ path }) =>
        path.startsWith("settings.photo"),
      ),
    ]

    const broken = photoCopy
      .filter(({ value }) => /\?/.test(value))
      .map(({ path, value }) => `${path}: ${value}`)

    expect(broken).toEqual([])
  })

  it.each(Object.entries(locales))("%s has curated tag filter and saving keys", (_locale, messages) => {
    const missing = requiredLocaleKeys.filter((key) => {
      const value = readLocaleKey(messages, key)
      return typeof value !== "string" || value.trim() === ""
    })

    expect(missing).toEqual([])
  })

  it.each(Object.entries(locales))(
    "%s actor library visible copy does not advertise actor tags",
    (_locale, messages) => {
      const visibleActorCopy = [
        readLocaleKey(messages, "actors.searchPlaceholder"),
        readLocaleKey(messages, "actors.subtitle"),
      ].join(" ")

      expect(visibleActorCopy).not.toMatch(/\bactor tags?\b|\btags?\b|标签|タグ/i)
    },
  )
})

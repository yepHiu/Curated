import { expect, test, type Page, type Route } from "@playwright/test"

const MOCK_BASE_URL = "http://127.0.0.1:4173"
const WEB_BASE_URL = "http://127.0.0.1:4174"

function apiPath(url: string): string {
  const parsed = new URL(url)
  return `${parsed.pathname}${parsed.search}`
}

async function hideDevPerformanceBar(page: Page) {
  await page.addInitScript(() => {
    window.localStorage.setItem("curated-dev-performance-bar-hidden-v1", "true")
  })
}

async function stubEventSource(page: Page) {
  await page.addInitScript(() => {
    class TestEventSource extends EventTarget {
      static readonly CLOSED = 2
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      readonly CLOSED = 2
      readonly CONNECTING = 0
      readonly OPEN = 1
      readonly readyState = TestEventSource.OPEN
      readonly url: string
      readonly withCredentials = false
      onerror: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onopen: ((event: Event) => void) | null = null

      constructor(url: string | URL) {
        super()
        this.url = String(url)
      }

      close() {}
    }

    Object.defineProperty(window, "EventSource", {
      configurable: true,
      value: TestEventSource,
    })
  })
}

test("Mock navigation stays entirely behind the Mock adapter", async ({ page }) => {
  const backendRequests: string[] = []
  page.on("request", (request) => {
    const url = new URL(request.url())
    if (url.pathname.startsWith("/api/") || url.port === "8080") {
      backendRequests.push(request.url())
    }
  })
  await hideDevPerformanceBar(page)

  await page.goto(`${MOCK_BASE_URL}/#/library`, { waitUntil: "domcontentloaded" })
  await expect(page.locator("#app")).toBeVisible()
  await expect(page.locator("[data-sidebar-nav-link]").first()).toBeVisible()

  await page.locator('[data-sidebar-nav-link][href$="/actors"]').first().click()
  await expect(page).toHaveURL(/#\/actors$/)

  await page.locator('[data-sidebar-nav-link][href$="/settings"]').first().click()
  await expect(page).toHaveURL(/#\/settings$/)

  expect(backendRequests).toEqual([])
})

test("375px library controls remain touchable without clipping or horizontal overflow", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await hideDevPerformanceBar(page)
  await page.goto(`${MOCK_BASE_URL}/#/library`, { waitUntil: "domcontentloaded" })

  await expect(page.locator("h1")).toHaveCount(1)
  await expect(page.locator("h1")).not.toHaveText("")
  await expect(page.locator("[data-library-filter-tabs]")).toBeVisible()

  const hasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(hasHorizontalOverflow).toBe(false)

  for (const trigger of await page.locator("[data-library-tab-trigger]").all()) {
    const box = await trigger.boundingBox()
    expect(box?.height ?? 0).toBeGreaterThanOrEqual(44)
  }

  const batchToggleBox = await page.locator("[data-library-batch-toggle]").boundingBox()
  expect(batchToggleBox?.height ?? 0).toBeGreaterThanOrEqual(44)

  const overflowBadge = page.locator("[data-movie-tag-overflow]").first()
  await expect(overflowBadge).toBeVisible()
  const tagRow = overflowBadge.locator("xpath=ancestor::*[@data-movie-tag-row]")
  const [overflowBox, tagRowBox] = await Promise.all([
    overflowBadge.boundingBox(),
    tagRow.boundingBox(),
  ])
  expect((overflowBox?.x ?? 0) + (overflowBox?.width ?? 0)).toBeLessThanOrEqual(
    (tagRowBox?.x ?? 0) + (tagRowBox?.width ?? 0) + 1,
  )

  await page.locator("[data-library-batch-toggle]").click()
  const movieBatchToggle = page.locator("[data-movie-batch-toggle]").first()
  await expect(movieBatchToggle).toBeVisible()
  const movieBatchToggleBox = await movieBatchToggle.boundingBox()
  expect(movieBatchToggleBox?.height ?? 0).toBeGreaterThanOrEqual(44)
  expect(movieBatchToggleBox?.width ?? 0).toBeGreaterThanOrEqual(44)

  for (const selector of [
    "[data-mobile-menu-trigger]",
    "[data-import-trigger]",
    "[data-notification-trigger]",
    "[data-mobile-theme-toggle]",
  ]) {
    const box = await page.locator(selector).boundingBox()
    expect(box?.height ?? 0).toBeGreaterThanOrEqual(44)
  }
})

test("locked startup defers protected hydration until a successful unlock", async ({ page }) => {
  let unlocked = false
  const protectedRequests: Array<{ path: string; unlocked: boolean }> = []
  const allApiRequests: string[] = []
  const unknownApiRequests: string[] = []
  const consoleErrors: string[] = []
  const pageErrors: string[] = []

  page.on("console", (message) => {
    if (message.type() === "error") {
      consoleErrors.push(message.text())
    }
  })
  page.on("pageerror", (error) => pageErrors.push(error.message))
  await hideDevPerformanceBar(page)
  await stubEventSource(page)

  await page.route(/^https?:\/\/[^/]+\/api\//, async (route: Route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    allApiRequests.push(`${request.method()} ${apiPath(request.url())}`)

    if (
      path === "/api/library/movies" ||
      path === "/api/playback/progress" ||
      path === "/api/library/played-movies"
    ) {
      protectedRequests.push({ path: apiPath(request.url()), unlocked })
    }

    if (path === "/api/auth/status") {
      await route.fulfill({
        json: {
          pinEnabled: true,
          unlocked,
          setupRequired: false,
          pinLength: 4,
          trustedForever: false,
          sessionTtlMinutes: 60,
          lanRequiresPin: true,
          lockOnRestart: true,
        },
      })
      return
    }

    if (path === "/api/auth/unlock" && request.method() === "POST") {
      unlocked = true
      await route.fulfill({
        json: {
          pinEnabled: true,
          unlocked: true,
          setupRequired: false,
          pinLength: 4,
          trustedForever: false,
          sessionTtlMinutes: 60,
          lanRequiresPin: true,
          lockOnRestart: true,
        },
      })
      return
    }

    if (path === "/api/library/movies") {
      await route.fulfill({ json: { items: [], limit: 500, offset: 0, total: 0 } })
      return
    }
    if (path === "/api/playback/progress") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/library/played-movies") {
      await route.fulfill({ json: { movieIds: [] } })
      return
    }
    if (path === "/api/health") {
      await route.fulfill({ json: { status: "ok", version: "e2e", channel: "test" } })
      return
    }
    if (path === "/api/library/paths/storage-status/check") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/curated-frames/stats") {
      await route.fulfill({ json: { total: 0 } })
      return
    }
    if (path === "/api/tasks/recent") {
      await route.fulfill({ json: { tasks: [] } })
      return
    }

    unknownApiRequests.push(`${request.method()} ${apiPath(request.url())}`)
    await route.fulfill({
      status: 404,
      json: { code: "E2E_UNSTUBBED", message: "Unstubbed e2e request", retryable: false },
    })
  })

  await page.goto(`${WEB_BASE_URL}/#/library`, { waitUntil: "domcontentloaded" })
  await page.waitForTimeout(500)
  expect({ allApiRequests, consoleErrors, pageErrors }).toEqual({
    allApiRequests: ["GET /api/auth/status"],
    consoleErrors: [],
    pageErrors: [],
  })
  await expect(page).toHaveURL(/#\/lock\?redirect=/)
  await expect(page.locator("[data-pin-input]")).toBeAttached()

  expect(protectedRequests).toEqual([])

  await page.locator("[data-pin-input]").fill("1234")
  await page.locator('form button[type="submit"]').click()
  await expect(page).toHaveURL(/#\/library$/)

  await expect
    .poll(() => protectedRequests.map((request) => request.path).sort())
    .toEqual([
      "/api/library/played-movies",
      "/api/library/movies?limit=500&offset=0",
      "/api/playback/progress",
    ].sort())
  expect(protectedRequests.every((request) => request.unlocked)).toBe(true)
  expect(unknownApiRequests).toEqual([])
  expect(consoleErrors).toEqual([])
})

test("maintenance backup flow creates verifies and preflights without online restore", async ({ page }) => {
  const consoleErrors: string[] = []
  const pageErrors: string[] = []
  const unknownApiRequests: string[] = []
  const backupPaths: string[] = []
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text())
  })
  page.on("pageerror", (error) => pageErrors.push(error.message))
  await hideDevPerformanceBar(page)
  await stubEventSource(page)

  const manifest = {
    format: "curated-backup",
    formatVersion: 1,
    createdAt: "2026-07-20T02:00:00Z",
    appVersion: "1.4.11",
    appChannel: "test",
    scope: {
      databaseIncluded: true,
      libraryConfigIncluded: true,
      userAssetsIncluded: false,
      mediaFilesIncluded: false,
    },
    schemaMigrations: ["0001_init.sql", "0027_auth_session_public_ids.sql"],
    files: [
      {
        kind: "database",
        path: "database/curated.db",
        sizeBytes: 1_048_576,
        sha256: "a".repeat(64),
      },
    ],
  }
  const verification = {
    valid: true,
    checkedAt: "2026-07-20T02:01:00Z",
    manifest,
    databaseIntegrity: { quickCheck: "ok", foreignKeyViolations: 0 },
    errors: [],
    warnings: [],
  }

  await page.route(/^https?:\/\/[^/]+\/api\//, async (route: Route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path === "/api/auth/status") {
      await route.fulfill({
        json: {
          pinEnabled: true,
          unlocked: true,
          setupRequired: false,
          pinLength: 4,
          trustedForever: false,
          sessionTtlMinutes: 60,
          lanRequiresPin: true,
          lockOnRestart: true,
        },
      })
      return
    }
    if (path === "/api/library/movies") {
      await route.fulfill({ json: { items: [], limit: 500, offset: 0, total: 0 } })
      return
    }
    if (path === "/api/playback/progress") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/library/played-movies") {
      await route.fulfill({ json: { movieIds: [] } })
      return
    }
    if (path === "/api/settings") {
      await route.fulfill({
        json: {
          libraryPaths: [],
          defaultImportLibraryPathId: "",
          player: {
            hardwareDecode: false,
            hardwareEncoder: "auto",
            nativePlayerPreset: "custom",
            nativePlayerEnabled: false,
            streamPushEnabled: false,
            forceStreamPush: false,
            preferNativePlayer: false,
            seekForwardStepSec: 10,
            seekBackwardStepSec: 10,
          },
          organizeLibrary: false,
          autoLibraryWatch: false,
          autoActorProfileScrape: false,
          autoDownloadUpdates: false,
          launchAtLogin: false,
          launchAtLoginSupported: false,
          curatedFrameExportFormat: "jpg",
          metadataMovieProvider: "",
          metadataMovieProviders: [],
          metadataMovieProviderChain: [],
          metadataMovieScrapeMode: "auto",
          metadataMovieStrategy: "auto-global",
          proxy: { enabled: false },
          backendLog: { logDir: "", logLevel: "info" },
        },
      })
      return
    }
    if (path === "/api/library/paths/storage-status" || path === "/api/library/paths/storage-status/check") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/curated-frames/stats") {
      await route.fulfill({ json: { total: 0 } })
      return
    }
    if (path === "/api/tasks/recent") {
      await route.fulfill({ json: { tasks: [] } })
      return
    }
    if (path === "/api/app-update/status") {
      await route.fulfill({ json: { supported: false, status: "unsupported" } })
      return
    }
    if (path === "/api/health") {
      await route.fulfill({
        json: {
          name: "curated-e2e",
          version: "e2e",
          channel: "test",
          transport: "http",
          databasePath: "D:\\Curated\\curated.db",
        },
      })
      return
    }
    if (path === "/api/connected-clients") {
      await route.fulfill({
        json: {
          clients: [],
          total: 0,
          localCount: 0,
          remoteCount: 0,
          sampledAt: "2026-07-20T02:00:00Z",
        },
      })
      return
    }
    if (path === "/api/playback/watch-time/daily") {
      await route.fulfill({
        json: {
          items: [],
          totalWatchedSec: 0,
          activeDays: 0,
          maxDayWatchedSec: 0,
          longestStreakDays: 0,
        },
      })
      return
    }
    if (path === "/api/dev/performance") {
      await route.fulfill({
        json: {
          supported: true,
          sampledAt: "2026-07-20T02:00:00Z",
          systemCpuPercent: 0,
          backendCpuPercent: 0,
        },
      })
      return
    }
    if (path === "/api/maintenance/backups" && request.method() === "POST") {
      const body = request.postDataJSON() as { destinationPath: string }
      backupPaths.push(body.destinationPath)
      await route.fulfill({ status: 201, json: manifest })
      return
    }
    if (path === "/api/maintenance/backups/verify") {
      await route.fulfill({ json: verification })
      return
    }
    if (path === "/api/maintenance/backups/preflight") {
      await route.fulfill({
        json: {
          canRestore: true,
          checkedAt: "2026-07-20T02:02:00Z",
          verification,
          targetDatabase: "D:\\Curated\\curated.db",
          targetDatabaseExists: true,
          targetConfig: "D:\\Curated\\library-config.cfg",
          targetConfigExists: true,
          requiredBytes: 2_097_152,
          availableBytes: 10_737_418_240,
          availableBytesKnown: true,
          unsupportedMigrations: [],
          errors: [],
          warnings: ["backup does not include media source files"],
        },
      })
      return
    }

    unknownApiRequests.push(`${request.method()} ${apiPath(request.url())}`)
    await route.fulfill({
      status: 404,
      json: { code: "E2E_UNSTUBBED", message: "Unstubbed e2e request", retryable: false },
    })
  })

  await page.goto(`${WEB_BASE_URL}/#/settings?section=maintenance`, {
    waitUntil: "domcontentloaded",
  })
  const pathInput = page.locator("[data-settings-backup-path]")
  await expect(pathInput).toBeVisible()
  await pathInput.fill("D:\\Backups\\curated-e2e")
  await page.locator("[data-settings-backup-create]").click()
  await expect(page.getByText(/验证通过|Verified|検証済み/)).toBeVisible()
  expect(backupPaths).toEqual(["D:\\Backups\\curated-e2e.curated-backup"])

  await page.locator("[data-settings-backup-preflight]").click()
  await expect(page.getByText(/可以离线恢复|Ready for offline restore|オフライン復元可能/)).toBeVisible()
  await expect(page.locator("[data-settings-backup-restore]")).toHaveCount(0)

  const desktopHasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(desktopHasHorizontalOverflow).toBe(false)

  await page.setViewportSize({ width: 375, height: 812 })
  await expect(pathInput).toBeVisible()
  const mobileHasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(mobileHasHorizontalOverflow).toBe(false)
  for (const selector of [
    "[data-settings-backup-pick]",
    "[data-settings-backup-create]",
    "[data-settings-backup-verify]",
    "[data-settings-backup-preflight]",
  ]) {
    const bounds = await page.locator(selector).boundingBox()
    expect(bounds?.height, selector).toBeGreaterThanOrEqual(44)
  }
  expect(unknownApiRequests).toEqual([])
  expect(consoleErrors).toEqual([])
  expect(pageErrors).toEqual([])

  const screenshotPath = process.env.CURATED_BACKUP_E2E_SCREENSHOT
  if (screenshotPath) {
    await page.screenshot({ path: screenshotPath, fullPage: false })
  }
})

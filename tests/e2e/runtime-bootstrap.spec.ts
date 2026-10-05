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

for (const returnAction of ["header", "escape", "history"] as const) {
  test(`movie detail returns via ${returnAction} without retaining its layout in the library`, async ({ page }) => {
    await hideDevPerformanceBar(page)
    await page.goto(`${MOCK_BASE_URL}/#/library`, { waitUntil: "domcontentloaded" })
    await page.locator("[data-movie-card-id] > button:visible").first().click()
    await expect(page).toHaveURL(/#\/detail\//)
    await expect(page.locator("[data-detail-media-column]")).toBeVisible()
    await expect(page.locator("[data-movie-scroll-region]")).toHaveCount(0)

    // Observe every painted frame, including the two frames a CSS Transition
    // normally retains its outgoing node even when no animation is defined.
    await page.evaluate(() => {
      const frame = document.querySelector<HTMLElement>("[data-router-view-frame]")
      const detail = frame?.firstElementChild
      if (!frame || !detail) throw new Error("Detail route has no content root")

      let sampledFrames = 0
      let overlapFrames = 0
      const sample = () => {
        const libraryVisible = Boolean(frame.querySelector("[data-movie-scroll-region]"))
        if (libraryVisible && detail.isConnected) overlapFrames++
        frame.dataset.detailReturnOverlap = String(overlapFrames)
        if (libraryVisible && !detail.isConnected) {
          frame.dataset.detailReturnComplete = "true"
        } else if (++sampledFrames < 120) {
          requestAnimationFrame(sample)
        }
      }
      requestAnimationFrame(sample)
    })

    if (returnAction === "header") {
      await page.locator("[data-shell-header] a").click()
    } else if (returnAction === "escape") {
      await page.keyboard.press("Escape")
    } else {
      await page.goBack()
    }

    await expect(page).toHaveURL(/#\/library(?:\?|$)/)
    await expect(page.locator("[data-movie-scroll-region]")).toBeVisible()
    const frame = page.locator("[data-router-view-frame]")
    await expect(frame).toHaveAttribute("data-detail-return-complete", "true")
    await expect(frame).toHaveAttribute("data-detail-return-overlap", "0")
  })
}

test("375px library controls remain touchable without clipping or horizontal overflow", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await hideDevPerformanceBar(page)
  await page.goto(`${MOCK_BASE_URL}/#/library`, { waitUntil: "domcontentloaded" })

  await expect(page.locator("h1")).toHaveCount(1)
  await expect(page.locator("h1")).not.toHaveText("")
  await expect(page.locator("[data-library-filter-tabs]")).toHaveCount(0)
  await expect(page.locator("[data-library-batch-toggle]")).toBeVisible()

  const hasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(hasHorizontalOverflow).toBe(false)

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
    "[data-theme-toggle]",
  ]) {
    const box = await page.locator(selector).boundingBox()
    expect(box?.height ?? 0).toBeGreaterThanOrEqual(44)
  }
})

test("home and movie library stay on their pages during boundary scrolling", async ({ page }) => {
  await hideDevPerformanceBar(page)
  await page.goto(`${MOCK_BASE_URL}/#/`, { waitUntil: "domcontentloaded" })
  const home = page.locator("[data-home-scroll-region]")
  await expect(home).toBeVisible()
  await home.evaluate((el) => { el.scrollTop = el.scrollHeight })
  await expect.poll(() => home.evaluate((el) => el.scrollTop)).toBeGreaterThan(0)
  await home.hover()
  await page.mouse.wheel(0, 120)
  await home.evaluate((el) => {
    el.dispatchEvent(new WheelEvent("wheel", { bubbles: true, deltaY: 120 }))
    el.dispatchEvent(new TouchEvent("touchstart", { bubbles: true, touches: [new Touch({ identifier: 1, target: el, clientX: 100, clientY: 200 })] }))
    el.dispatchEvent(new TouchEvent("touchmove", { bubbles: true, touches: [new Touch({ identifier: 1, target: el, clientX: 100, clientY: 80 })] }))
  })
  await expect(page).toHaveURL(/#\/$/)
  await expect(home).toBeVisible()

  await page.locator("[data-home-browse-library]").click()
  await expect(page).toHaveURL(/#\/library$/)
  await expect(home).toHaveCount(0)

  const movies = page.locator("[data-movie-scroll-region]")
  await expect(movies).toBeVisible()
  await movies.evaluate((el) => { el.scrollTop = 0 })
  await movies.hover()
  await page.mouse.wheel(0, -120)
  await movies.evaluate((el) => {
    el.dispatchEvent(new WheelEvent("wheel", { bubbles: true, deltaY: -120 }))
    el.dispatchEvent(new TouchEvent("touchstart", { bubbles: true, touches: [new Touch({ identifier: 1, target: el, clientX: 100, clientY: 100 })] }))
    el.dispatchEvent(new TouchEvent("touchmove", { bubbles: true, touches: [new Touch({ identifier: 1, target: el, clientX: 100, clientY: 220 })] }))
  })
  await expect(page).toHaveURL(/#\/library$/)
  await expect(movies).toBeVisible()

  await page.locator('[data-sidebar-nav-link][href="#/"]').click()
  await expect(page).toHaveURL(/#\/$/)
  await expect(movies).toHaveCount(0)
  await expect(home).toBeVisible()
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
      path === "/api/wishlist/items" ||
      path === "/api/settings" ||
      path === "/api/library/movies" ||
      path === "/api/library/saved-views" ||
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

    if (path === "/api/settings") {
      await route.fulfill({ json: { libraryPaths: [], comicLibraryEnabled: false, photoLibraryEnabled: false } })
      return
    }
    // 侧栏愿望计数与现有 AI 治理启动读取均使用受保护接口。
    if (path === "/api/wishlist/items") {
      await route.fulfill({ json: { items: [], total: 0, pendingCount: 0 } })
      return
    }
    if (path === "/api/ai/tag-organizations") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/ai/settings") {
      await route.fulfill({ json: { enabled: false } })
      return
    }
    if (path === "/api/library/movies") {
      await route.fulfill({ json: { items: [], limit: 500, offset: 0, total: 0 } })
      return
    }
    if (path === "/api/library/saved-views") {
      await route.fulfill({ json: { items: [] } })
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
    .poll(() => [...new Set(protectedRequests.map((request) => request.path))].sort())
    .toEqual([
      "/api/library/played-movies",
      "/api/library/movies?limit=500&offset=0",
      "/api/library/saved-views",
      "/api/playback/progress",
      "/api/settings",
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
  const verifiedPaths: string[] = []
  const preflightPaths: string[] = []
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
    // 侧栏愿望计数与现有 AI 治理启动读取均使用受保护接口。
    if (path === "/api/wishlist/items") {
      await route.fulfill({ json: { items: [], total: 0, pendingCount: 0 } })
      return
    }
    if (path === "/api/ai/tag-organizations") {
      await route.fulfill({ json: { items: [] } })
      return
    }
    if (path === "/api/ai/settings") {
      await route.fulfill({ json: { enabled: false } })
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
          backupDirectory: "",
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
          canManageLibraryPaths: true,
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
    if (path === "/api/maintenance/backups/latest") {
      await route.fulfill({ json: { backupPath: "" } })
      return
    }
    if (path === "/api/maintenance/backups/verify") {
      verifiedPaths.push((request.postDataJSON() as { backupPath: string }).backupPath)
      await route.fulfill({ json: verification })
      return
    }
    if (path === "/api/maintenance/backups/preflight") {
      preflightPaths.push((request.postDataJSON() as { backupPath: string }).backupPath)
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
  await expect(pathInput).not.toBeVisible()
  const directoryInput = page.locator("[data-settings-backup-directory]")
  await expect(directoryInput).toBeVisible()
  await directoryInput.fill("D:\\Backups")
  await page.locator("[data-settings-backup-create]").click()
  await expect(page.locator("[data-settings-backup-created-path]")).toBeVisible()
  await expect(page.locator("[data-settings-backup-create]")).toBeEnabled()
  expect(backupPaths).toHaveLength(1)
  expect(backupPaths[0]).toMatch(/^D:\\Backups\\curated-\d{8}-\d{6}Z\.curated-backup$/)

  await page.locator("[data-settings-backup-existing]").click()
  await expect(pathInput).toHaveValue(backupPaths[0]!)
  await expect(pathInput).toHaveAttribute("readonly", "")
  await page.locator("[data-settings-backup-preflight]").click()
  await expect(page.locator("[data-settings-backup-check-result]")).toBeVisible()
  await expect(page.locator("[data-settings-backup-check-result]")).toContainText("D:\\Curated\\curated.db")
  expect(verifiedPaths).toEqual(backupPaths)
  expect(preflightPaths).toEqual(backupPaths)
  await expect(page.locator("[data-settings-backup-restore]")).toHaveCount(0)
  await expect(page.locator("[data-settings-backup-pick]")).toHaveCount(0)
  await expect(page.locator("[data-settings-backup-verify]")).toHaveCount(0)

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
    "[data-settings-backup-pick-file]",
    "[data-settings-backup-create]",
    "[data-settings-backup-existing]",
    "[data-settings-backup-preflight]",
  ]) {
    await expect.poll(async () => {
      const bounds = await page.locator(selector).boundingBox()
      return bounds?.height ?? 0
    }, { message: selector }).toBeGreaterThanOrEqual(44)
  }
  expect(unknownApiRequests).toEqual([])
  expect(consoleErrors).toEqual([])
  expect(pageErrors).toEqual([])

  const screenshotPath = process.env.CURATED_BACKUP_E2E_SCREENSHOT
  if (screenshotPath) {
    await page.screenshot({ path: screenshotPath, fullPage: false })
  }
})

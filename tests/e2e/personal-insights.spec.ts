import { expect, test } from "@playwright/test"

const MOCK_BASE_URL = "http://127.0.0.1:4173"

test("personal insights renders bounded local aggregates at desktop and 375px", async ({ page }) => {
  const backendRequests: string[] = []
  const consoleErrors: string[] = []
  const pageErrors: string[] = []
  page.on("request", (request) => {
    const url = new URL(request.url())
    if (url.pathname.startsWith("/api/") || url.port === "8080") backendRequests.push(request.url())
  })
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text())
  })
  page.on("pageerror", (error) => pageErrors.push(error.message))

  await page.addInitScript(() => {
    window.localStorage.setItem("curated-dev-performance-bar-hidden-v1", "true")
    const now = new Date()
    const dayKey = [
      now.getFullYear(),
      String(now.getMonth() + 1).padStart(2, "0"),
      String(now.getDate()).padStart(2, "0"),
    ].join("-")
    window.localStorage.setItem(
      "curated-playback-watch-time-daily-v1",
      JSON.stringify({ [dayKey]: { "mkb-100": 7200 } }),
    )
    window.localStorage.setItem(
      "jav-library-playback-progress-v1",
      JSON.stringify({
        "mkb-100": {
          movieId: "mkb-100",
          positionSec: 95,
          durationSec: 100,
          updatedAt: now.toISOString(),
        },
      }),
    )
  })

  await page.goto(`${MOCK_BASE_URL}/#/insights`, { waitUntil: "domcontentloaded" })
  await expect(page.locator("[data-personal-insights-page]")).toBeVisible()
  await expect(page.locator("h1")).toHaveCount(1)
  await expect(page.locator("h1")).not.toHaveText("")
  await expect(page.locator("[data-insights-metric]")).toHaveCount(6)
  await expect(page.locator("[data-insights-breakdown]")).toHaveCount(3)
  await expect(page.locator('[data-insights-breakdown="actor"]')).toContainText("Mina Kaze")
  await expect(page.locator('[data-insights-metric="completed"]')).toContainText("1")
  await expect(page.locator('[data-insights-metric="completion-rate"]')).toContainText("100%")
  await expect(page.locator("body")).not.toContainText("NaN")
  await expect(page.locator("body")).not.toContainText("Infinity")

  const ranges = page.locator("[data-insights-range-selector] label")
  await expect(ranges).toHaveCount(4)
  await ranges.nth(1).click()
  await expect(page.locator('input[value="90d"]')).toBeChecked()
  await expect(page.locator("[data-insights-metric]")).toHaveCount(6)

  const desktopHasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(desktopHasHorizontalOverflow).toBe(false)

  await page.setViewportSize({ width: 375, height: 812 })
  await expect(page.locator("[data-personal-insights-page]")).toBeVisible()
  const mobileHasHorizontalOverflow = await page.evaluate(() => {
    const root = document.documentElement
    return root.scrollWidth > root.clientWidth + 1
  })
  expect(mobileHasHorizontalOverflow).toBe(false)
  for (const control of await ranges.all()) {
    const bounds = await control.boundingBox()
    expect(bounds?.height ?? 0).toBeGreaterThanOrEqual(44)
  }

  expect(backendRequests).toEqual([])
  expect(consoleErrors).toEqual([])
  expect(pageErrors).toEqual([])
})

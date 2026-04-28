import { chromium, expect, firefox, test, type Browser } from "@playwright/test"

import { displayScalingRoutes } from "./routes"

const baseUrl = process.env.DISPLAY_SCALING_BASE_URL ?? "http://127.0.0.1:5173"
const chromeChannel = process.env.DISPLAY_SCALING_CHROME_CHANNEL ?? "chrome"

const browserEngines: Array<{ name: "chrome" | "firefox"; launch: () => Promise<Browser> }> = [
  { name: "chrome", launch: () => chromium.launch({ channel: chromeChannel }) },
  { name: "firefox", launch: () => firefox.launch() },
]

const viewports = [
  { name: "phone-se", width: 320, height: 568, deviceScaleFactor: 2, isMobile: true },
  { name: "phone", width: 375, height: 812, deviceScaleFactor: 3, isMobile: true },
  { name: "large-phone", width: 430, height: 932, deviceScaleFactor: 3, isMobile: true },
  { name: "tablet", width: 768, height: 1024, deviceScaleFactor: 2, isMobile: true },
  { name: "small-desktop", width: 1024, height: 768, deviceScaleFactor: 1 },
  { name: "macbook-air", width: 1280, height: 832, deviceScaleFactor: 2 },
  { name: "desktop", width: 1440, height: 900, deviceScaleFactor: 1 },
  { name: "macbook-pro", width: 1512, height: 982, deviceScaleFactor: 2 },
  { name: "external-hd", width: 1920, height: 1080, deviceScaleFactor: 1.5 },
] as const

function buildUrl(path: string): string {
  return new URL(path, `${baseUrl}/`).toString()
}

function screenshotName(
  browserName: string,
  routeName: string,
  viewport: (typeof viewports)[number],
): string {
  const dpr = String(viewport.deviceScaleFactor).replace(".", "-")
  return `test-results/display-scaling/${browserName}-${routeName}-${viewport.name}-${viewport.width}x${viewport.height}-dpr${dpr}.png`
}

for (const browserEngine of browserEngines) {
  test.describe(`${browserEngine.name} display scaling`, () => {
    test.describe.configure({ mode: "serial" })

    for (const viewport of viewports) {
      for (const route of displayScalingRoutes) {
        test(`${route.name} ${viewport.name} ${viewport.width}x${viewport.height} dpr${viewport.deviceScaleFactor}`, async () => {
          const browser = await browserEngine.launch()
          const context = await browser.newContext({
            deviceScaleFactor: viewport.deviceScaleFactor,
            ...(browserEngine.name === "chrome"
              ? { isMobile: Boolean("isMobile" in viewport && viewport.isMobile) }
              : {}),
            viewport: { width: viewport.width, height: viewport.height },
          })
          const page = await context.newPage()

          try {
            await page.goto(buildUrl(route.path), { waitUntil: "domcontentloaded" })
            await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => undefined)
            await expect(page.locator("#app")).toBeVisible()

            const horizontalOverflow = await page.evaluate(() => {
              const doc = document.documentElement
              return doc.scrollWidth > doc.clientWidth + 1
            })
            expect(horizontalOverflow, `${route.name} should not create document-level horizontal overflow`).toBe(
              false,
            )

            const visibleInteractiveCount = await page
              .locator("button:visible, a:visible, input:visible, textarea:visible, [role='button']:visible")
              .count()
            expect(visibleInteractiveCount, `${route.name} should expose visible interactive UI`).toBeGreaterThan(0)

            const clippedCriticalElements = await page.evaluate(() => {
              const elements = Array.from(document.querySelectorAll("[data-scaling-critical]"))
              return elements
                .map((node) => {
                  const element = node as HTMLElement
                  const rect = element.getBoundingClientRect()
                  if (rect.width <= 0 || rect.height <= 0) {
                    return null
                  }
                  const clipped =
                    element.scrollWidth > element.clientWidth + 2 ||
                    element.scrollHeight > element.clientHeight + 2
                  if (!clipped) {
                    return null
                  }
                  return {
                    className: element.className,
                    height: Math.round(rect.height),
                    tagName: element.tagName.toLowerCase(),
                    text: element.textContent?.trim().replace(/\s+/g, " ").slice(0, 80) ?? "",
                    width: Math.round(rect.width),
                  }
                })
                .filter(Boolean)
            })
            expect(clippedCriticalElements).toEqual([])

            await page.screenshot({
              fullPage: false,
              path: screenshotName(browserEngine.name, route.name, viewport),
            })
          } finally {
            await context.close()
            await browser.close()
          }
        })
      }
    }
  })
}

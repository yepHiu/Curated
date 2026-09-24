import { describe, expect, it } from "vitest"
import html from "../index.html?raw"
import projectLicense from "../LICENSE?raw"
import mainSource from "./main.ts?raw"
import bundledProjectLicense from "./assets/licenses/Curated_LICENSE.txt?raw"
import thirdPartyNotices from "./assets/licenses/ThirdParty_NOTICES.txt?raw"

describe("offline desktop resources", () => {
  it("does not reference Google Fonts in the HTML shell", () => {
    expect(html).not.toContain("fonts.googleapis.com")
    expect(html).not.toContain("fonts.gstatic.com")
  })

  it("loads the Noto families from local fontsource assets", () => {
    expect(mainSource).toContain('import "@fontsource-variable/noto-sans/wght.css"')
    expect(mainSource).toContain('import "@fontsource-variable/noto-sans-jp/wght.css"')
  })

  it("keeps the bundled project license in sync with the repository license", () => {
    expect(bundledProjectLicense).toBe(projectLicense)
  })

  it("bundles the audited third-party notices for offline access", () => {
    expect(thirdPartyNotices).toContain("Frontend | hls.js | 1.6.16 | Apache-2.0")
    expect(thirdPartyNotices).toContain("Frontend | dompurify | 3.4.13 | Apache-2.0 OR MPL-2.0")
    expect(thirdPartyNotices).toContain("Backend | gorm.io/gorm | v1.30.1 | MIT")
    expect(thirdPartyNotices).toContain("Desktop | FFmpeg | 8.0.1-full_build-www.gyan.dev | GPL-3.0-or-later")
  })
})

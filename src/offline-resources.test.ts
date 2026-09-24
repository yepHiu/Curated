import { describe, expect, it } from "vitest"
import html from "../index.html?raw"
import projectLicense from "../LICENSE?raw"
import mainSource from "./main.ts?raw"
import bundledProjectLicense from "./assets/licenses/Curated_LICENSE.txt?raw"

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
})

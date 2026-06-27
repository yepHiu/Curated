import { describe, expect, it } from "vitest"
import html from "../index.html?raw"
import mainSource from "./main.ts?raw"

describe("offline desktop resources", () => {
  it("does not reference Google Fonts in the HTML shell", () => {
    expect(html).not.toContain("fonts.googleapis.com")
    expect(html).not.toContain("fonts.gstatic.com")
  })

  it("loads Outfit from local fontsource assets", () => {
    expect(mainSource).toContain('import "@fontsource/outfit/400.css"')
    expect(mainSource).toContain('import "@fontsource/outfit/500.css"')
    expect(mainSource).toContain('import "@fontsource/outfit/600.css"')
    expect(mainSource).toContain('import "@fontsource/outfit/700.css"')
  })
})

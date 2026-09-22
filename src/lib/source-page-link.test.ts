import { describe, expect, it } from "vitest"
import { sourcePageLink } from "./source-page-link"

describe("source page links", () => {
  it.each([
    ["https://javdb.com/v/abc", "JAVDB"],
    ["https://www.javdb.com/v/abc", "JAVDB"],
    ["https://jable.tv/videos/test/", "Jable"],
    ["https://www.javbus.com/TEST-001", "JavBus"],
    ["https://example.org/movies/1?q=test", "example.org"],
    ["https://javdb.com.example.org/v/1", "javdb.com.example.org"],
  ])("labels %s with its site name", (url, label) => {
    expect(sourcePageLink(url)).toEqual({ url, label })
  })

  it.each([undefined, "", " ", "/relative", "javascript:alert(1)", "data:text/html,test", "https://user:pass@javdb.com/v/abc"])("omits invalid source %s", (url) => {
    expect(sourcePageLink(url)).toBeUndefined()
  })
})

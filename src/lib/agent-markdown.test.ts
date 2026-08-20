import { describe, expect, it } from "vitest"
import { renderAgentMarkdown, stabilizeAgentMarkdown } from "./agent-markdown"

describe("renderAgentMarkdown", () => {
  it("renders GFM emphasis, code, and lists", () => {
    const html = renderAgentMarkdown("**粗体** 和 `code`\n\n- 一项")
    expect(html).toContain("<strong>粗体</strong>")
    expect(html).toContain("<code>code</code>")
    expect(html).toContain("<li>")
    expect(html).toContain("一项")
  })

  it("strips raw HTML and unsafe links", () => {
    const html = renderAgentMarkdown('<script>alert(1)</script>[点我](javascript:alert(1))')
    expect(html.toLowerCase()).not.toContain("<script")
    expect(html.toLowerCase()).not.toMatch(/href\s*=\s*["']?javascript:/i)
  })

  it("keeps http links and opens them in a new tab", () => {
    const html = renderAgentMarkdown("[Curated](https://example.com)")
    expect(html).toContain('href="https://example.com"')
    expect(html).toContain('target="_blank"')
    expect(html).toContain("noopener")
  })

  it("closes an unfinished code fence while streaming", () => {
    expect(stabilizeAgentMarkdown("```js\nconst a = 1")).toContain("```")
    const html = renderAgentMarkdown("```js\nconst a = 1")
    expect(html).toContain("<pre>")
    expect(html).toContain("const a = 1")
  })
})

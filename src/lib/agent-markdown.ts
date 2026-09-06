import { Marked } from "marked"
import DOMPurify from "dompurify"

const ALLOWED_TAGS = [
  "a",
  "blockquote",
  "br",
  "code",
  "del",
  "em",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "hr",
  "input",
  "li",
  "ol",
  "p",
  "pre",
  "strong",
  "table",
  "tbody",
  "td",
  "th",
  "thead",
  "tr",
  "ul",
]

const ALLOWED_ATTR = ["checked", "class", "disabled", "href", "rel", "target", "title", "type"]

const SAFE_HREF = /^(https?:|mailto:|#)/i

const parser = new Marked({
  gfm: true,
  breaks: true,
  renderer: {
    html() {
      return ""
    },
    image() {
      return ""
    },
  },
})

let hooksInstalled = false

function ensureSanitizerHooks() {
  if (hooksInstalled || typeof window === "undefined") return
  hooksInstalled = true
  DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (!(node instanceof Element) || node.tagName !== "A") return
    const href = node.getAttribute("href") ?? ""
    if (!SAFE_HREF.test(href.trim())) {
      node.removeAttribute("href")
    }
    node.setAttribute("target", "_blank")
    node.setAttribute("rel", "noopener noreferrer")
  })
}

/** Close an unfinished fenced code block so streaming deltas still render. */
export function stabilizeAgentMarkdown(source: string): string {
  const text = source.replace(/\r\n/g, "\n")
  const fences = text.match(/```/g)
  if (fences && fences.length % 2 === 1) {
    return `${text}\n\`\`\``
  }
  return text
}

/** Render GFM to sanitized HTML for Agent chat (LLM output is untrusted). */
export function renderAgentMarkdown(source: string): string {
  if (!source) return ""
  const html = parser.parse(stabilizeAgentMarkdown(source), { async: false })
  if (typeof window === "undefined") return ""
  ensureSanitizerHooks()
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS,
    ALLOWED_ATTR,
    FORBID_TAGS: ["script", "style", "iframe", "object", "embed", "form", "img"],
    FORBID_ATTR: ["style", "src", "srcset"],
  })
}

/**
 * Normalizes scraped movie summary for display as plain text (no v-html).
 * Strips HTML, then folds line breaks into spaces so the blurb reads as one
 * flowing paragraph (scrapers often spam <br> / newlines between short phrases).
 */
export function formatMovieSummaryForDisplay(raw: string): string {
  const s = raw.trim()
  if (!s) return ""

  let out = s
  // <br> → space (same intent as a soft break, not a visual paragraph)
  out = out.replace(/<br\s*\/?>/gi, " ")
  // Remove any remaining HTML tags
  out = out.replace(/<[^>]+>/g, "")
  // Common entities
  out = out
    .replace(/&nbsp;/gi, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")

  out = out.replace(/\r\n/g, "\n").replace(/\r/g, "\n")
  // Any run of newlines → single space (no blank lines / tall stacks of lines)
  out = out.replace(/\n+/g, " ")
  // Collapse repeated spaces (ASCII / tab / fullwidth space)
  out = out.replace(/[ \t\u3000]+/g, " ")

  return out.trim()
}

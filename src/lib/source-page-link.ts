/** Display a source site name while retaining the exact movie page as the link target. */
export function sourcePageLink(value?: string): { url: string; label: string } | undefined {
  if (!value?.trim()) return undefined
  try {
    const url = new URL(value.trim())
    if (!["https:", "http:"].includes(url.protocol) || url.username || url.password) return undefined
    const host = url.hostname.toLowerCase().replace(/^www\./, "")
    const sites: Record<string, string> = { "javdb.com": "JAVDB", "javbus.com": "JavBus", "jable.tv": "Jable" }
    const site = Object.keys(sites).find((domain) => host === domain || host.endsWith(`.${domain}`))
    return { url: value.trim(), label: site ? sites[site]! : host }
  } catch {
    return undefined
  }
}

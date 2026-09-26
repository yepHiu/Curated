import type { DesktopInfo } from "../../electron/desktop-contract"

/** A local-looking renderer can proxy a remote server. Check the actual API
 * target and, in Desktop, the target owned by main. Unknown means read-only. */
export function isLocalServerTarget(apiUrl: string, pageOrigin: string, desktop: boolean, info: DesktopInfo | null): boolean {
  try {
    const target = new URL(apiUrl, pageOrigin)
    const page = new URL(pageOrigin)
    const loopback = (url: URL) => ["http:", "https:"].includes(url.protocol)
      && ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname)
      && !url.username && !url.password
    if (!loopback(target) || !loopback(page)) return false
    if (!desktop) return true
    if (!info?.serverOrigin) return false
    return new URL(info.serverOrigin).origin === target.origin
  } catch {
    return false
  }
}

/** Standalone Desktop always uses its own update channel, even on the same host. */
export function isLocalUpdateTarget(apiUrl: string, pageOrigin: string, desktop: boolean, info: DesktopInfo | null): boolean {
  return (!desktop || info?.distribution === "legacy") && isLocalServerTarget(apiUrl, pageOrigin, desktop, info)
}

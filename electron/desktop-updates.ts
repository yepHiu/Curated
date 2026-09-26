import type { DesktopInfo, DesktopUpdateResult } from "./desktop-contract.js"

export const desktopInfoChannel = "curated:desktop-info"
export const desktopUpdateChannel = "curated:desktop-check-update"
const numericVersion = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/

export function isVersion(value: unknown): value is string {
  return typeof value === "string" && numericVersion.test(value) && value.split(".").every((part) => Number.isSafeInteger(Number(part)))
}

function compareVersions(left: string, right: string): number {
  const a = left.split(".").map(Number)
  const b = right.split(".").map(Number)
  return a[0] - b[0] || a[1] - b[1] || a[2] - b[2]
}

function httpsUrl(value: unknown): value is string {
  if (typeof value !== "string") return false
  try {
    const url = new URL(value)
    return url.protocol === "https:" && !url.username && !url.password
  } catch {
    return false
  }
}

export function selectDesktopUpdate(payload: unknown, info: DesktopInfo): DesktopUpdateResult {
  if (!payload || typeof payload !== "object" || !("schema" in payload) || payload.schema !== 1 || !("artifacts" in payload) || !Array.isArray(payload.artifacts)) {
    throw new Error("Invalid component update manifest")
  }
  const format = info.platform === "windows" ? "exe" : "dmg"
  const candidates: { version: string; url: string }[] = []
  for (const entry of payload.artifacts) {
    if (!entry || typeof entry !== "object") throw new Error("Invalid artifact")
    if (entry.component !== "desktop" || entry.variant !== "standalone" || entry.channel !== "stable" || entry.platform !== info.platform || entry.arch !== info.arch || entry.format !== format) continue
    if (!isVersion(entry.version) || !httpsUrl(entry.url) || typeof entry.sha256 !== "string" || !/^[a-fA-F0-9]{64}$/.test(entry.sha256)) {
      throw new Error("Invalid Desktop artifact")
    }
    candidates.push({ version: entry.version, url: entry.url })
  }
  candidates.sort((a, b) => compareVersions(b.version, a.version))
  const latest = candidates[0]
  if (!latest) return { status: "no-artifact" }
  if (compareVersions(latest.version, info.version) <= 0) return { status: "up-to-date" }
  return { status: "update-available", latestVersion: latest.version, downloadUrl: latest.url }
}

export async function checkDesktopUpdate(info: DesktopInfo, feed: string | null, fetcher: (url: string, init?: RequestInit) => Promise<Response>): Promise<DesktopUpdateResult> {
  if (info.development) return { status: "development" }
  if (info.distribution === "legacy") return { status: "bundled" }
  if (!feed) return { status: "not-configured" }
  if (!["windows", "macos"].includes(info.platform) || !["x64", "arm64"].includes(info.arch)) return { status: "unsupported" }
  try {
    if (!isVersion(info.version) || !httpsUrl(feed)) throw new Error("Invalid update configuration")
    const response = await fetcher(feed, { signal: AbortSignal.timeout(15_000), redirect: "error", cache: "no-store" })
    if (!response.ok || !response.body) throw new Error("Feed unavailable")
    const reader = response.body.getReader()
    const chunks: Uint8Array[] = []
    let size = 0
    try {
      while (true) {
        const chunk = await reader.read()
        if (chunk.done) break
        size += chunk.value.byteLength
        if (size > 1024 * 1024) throw new Error("Feed too large")
        chunks.push(chunk.value)
      }
    } finally {
      await reader.cancel()
    }
    return selectDesktopUpdate(JSON.parse(Buffer.concat(chunks).toString("utf8")), info)
  } catch {
    return { status: "error" }
  }
}

export function isTrustedDesktopSender(senderId: number, mainWindowId: number | undefined, frameUrl: string | undefined, isMainFrame: boolean, baseUrl: string | undefined): boolean {
  if (senderId !== mainWindowId || !isMainFrame || !frameUrl || !baseUrl) return false
  try {
    return new URL(frameUrl).origin === new URL(baseUrl).origin
  } catch {
    return false
  }
}

const releaseAPI = "https://api.github.com/repos/yepHiu/Curated/releases/latest"
const downloadPrefix = "https://github.com/yepHiu/Curated/releases/download/"
export interface DesktopUpdate { version: string; url: string }
export function desktopUpdateAsset(payload: unknown, platform: string, arch: string): DesktopUpdate | undefined {
  if (platform !== "win32" || arch !== "x64" || !payload || typeof payload !== "object") return
  const release = payload as Record<string, unknown>
  if (release.draft || release.prerelease || typeof release.tag_name !== "string" || !Array.isArray(release.assets)) return
  const version = release.tag_name.replace(/^v/, "")
  if (!/^\d+\.\d+\.\d+$/.test(version)) return
  const name = `Curated-Desktop-Setup-${version}-windows-x64.exe`
  for (const asset of release.assets as Record<string, unknown>[]) {
    if (asset.name !== name || typeof asset.browser_download_url !== "string") continue
    const url = asset.browser_download_url
    if (url === `${downloadPrefix}${encodeURIComponent(release.tag_name)}/${name}`) return { version, url }
  }
}
/** 使用调用方的网络栈，使客户端代理同样覆盖更新检查。 */
export async function checkDesktopUpdate(fetchImpl = fetch): Promise<DesktopUpdate | undefined> {
  const response = await fetchImpl(releaseAPI, { signal: AbortSignal.timeout(8000), redirect: "error", headers: { Accept: "application/vnd.github+json" } })
  if (!response.ok) throw new Error(`更新检查失败（HTTP ${response.status}）。`)
  const text = await response.text()
  if (text.length > 1024 * 1024) throw new Error("更新响应过大。")
  return desktopUpdateAsset(JSON.parse(text), process.platform, process.arch)
}

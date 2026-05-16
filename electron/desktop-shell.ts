import { existsSync } from "node:fs"
import { release as osRelease } from "node:os"
import path from "node:path"

export const pickDirectoryChannel = "curated:pick-directory"
export const curatedDesktopClientHeaderName = "X-Curated-Client"
export const curatedDesktopClientHeaderValue = "desktop-electron"
export const curatedDesktopClientVersionHeaderName = "X-Curated-Client-Version"
export const curatedDesktopClientOSHeaderName = "X-Curated-OS"
export const curatedDesktopClientOSVersionHeaderName = "X-Curated-OS-Version"

export interface NativeDirectoryPickResult {
  path: string
}

export interface DesktopOSInfo {
  os: string
  version: string
}

export type TrayMenuActionId =
  | "app-title"
  | "backend-status"
  | "open-curated"
  | "open-browser"
  | "open-settings"
  | "quit"

export type TrayMenuModelItem =
  | {
      id: TrayMenuActionId
      type: "normal"
      label: string
      enabled?: boolean
      url?: string
    }
  | { type: "separator" }

interface DirectoryDialogResult {
  canceled: boolean
  filePaths: string[]
}

export function resolveAppIconPath(
  appPath: string,
  pathExists: (candidate: string) => boolean = existsSync,
): string | undefined {
  const candidates = [
    path.join(appPath, "curated.ico"),
    path.join(appPath, "backend", "internal", "assets", "curated.ico"),
    path.join(appPath, "public", "Curated-icon.png"),
    path.join(appPath, "icon", "curated-icon-rg-dark-pink.png"),
  ]

  return candidates.find((candidate) => pathExists(candidate))
}

export function selectedDirectoryFromOpenDialogResult(
  result: DirectoryDialogResult,
): NativeDirectoryPickResult | null {
  if (result.canceled) {
    return null
  }
  const selectedPath = result.filePaths[0]?.trim()
  return selectedPath ? { path: selectedPath } : null
}

export function buildTrayMenuModel(options: {
  baseUrl: string
  attachedToExistingBackend: boolean
}): TrayMenuModelItem[] {
  const baseUrl = normalizeBaseUrl(options.baseUrl)
  return [
    { id: "app-title", type: "normal", label: "Curated", enabled: false },
    {
      id: "backend-status",
      type: "normal",
      label: options.attachedToExistingBackend ? "Connected to existing backend" : "Desktop service running",
      enabled: false,
    },
    { type: "separator" },
    { id: "open-curated", type: "normal", label: "Open Curated" },
    {
      id: "open-browser",
      type: "normal",
      label: "Open Web App in Browser",
      url: baseUrl,
    },
    {
      id: "open-settings",
      type: "normal",
      label: "Open Settings",
      url: `${baseUrl}/#/settings`,
    },
    { type: "separator" },
    { id: "quit", type: "normal", label: "Quit Curated" },
  ]
}

export function shouldHideWindowOnClose(options: { isQuitting: boolean }): boolean {
  return !options.isQuitting
}

export function shouldStopBackendOnQuit(options: { attachedToExistingBackend: boolean }): boolean {
  return !options.attachedToExistingBackend
}

export function shouldUseApplicationMenu(platform: string = process.platform): boolean {
  return platform === "darwin"
}

export function shouldMarkCuratedDesktopRequest(candidateUrl: string, backendBaseUrl: string): boolean {
  try {
    return new URL(candidateUrl).origin === new URL(backendBaseUrl).origin
  } catch {
    return false
  }
}

export function withCuratedDesktopRequestHeaders(
  requestHeaders: Record<string, string>,
  appVersion: string,
  osInfo: DesktopOSInfo = desktopOSInfo(),
): Record<string, string> {
  const headers: Record<string, string> = {
    ...requestHeaders,
    [curatedDesktopClientHeaderName]: curatedDesktopClientHeaderValue,
    [curatedDesktopClientVersionHeaderName]: appVersion,
  }
  if (osInfo.os) {
    headers[curatedDesktopClientOSHeaderName] = osInfo.os
  }
  if (osInfo.version) {
    headers[curatedDesktopClientOSVersionHeaderName] = osInfo.version
  }
  return headers
}

export function desktopOSInfo(platform: NodeJS.Platform = process.platform, releaseText: string = osRelease()): DesktopOSInfo {
  if (platform === "win32") {
    return { os: "Windows", version: windowsDisplayVersion(releaseText) }
  }
  if (platform === "darwin") {
    return { os: "macOS", version: releaseText.trim() }
  }
  if (platform === "linux") {
    return { os: "Linux", version: releaseText.trim() }
  }
  return { os: platform, version: releaseText.trim() }
}

function windowsDisplayVersion(releaseText: string): string {
  const parts = releaseText.trim().split(".")
  const build = Number(parts[2] ?? "")
  if (Number.isFinite(build) && build >= 22000) {
    return "11"
  }
  if (Number.isFinite(build) && build >= 10240) {
    return "10"
  }
  return releaseText.trim()
}

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, "")
}

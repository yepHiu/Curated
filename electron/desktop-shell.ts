import { existsSync } from "node:fs"
import path from "node:path"

export const pickDirectoryChannel = "curated:pick-directory"

export interface NativeDirectoryPickResult {
  path: string
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

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, "")
}

export interface DesktopInfo {
  /** Actual backend origin selected by main; absent on older Desktop versions. */
  serverOrigin?: string
  version: string
  buildStamp: string
  development: boolean
  distribution: "legacy" | "desktop"
  platform: string
  arch: string
}

export type DesktopUpdateStatus = "development" | "bundled" | "not-configured" | "unsupported" | "no-artifact" | "up-to-date" | "update-available" | "error"

export interface DesktopUpdateResult {
  status: DesktopUpdateStatus
  latestVersion?: string
  downloadUrl?: string
}

/** Read-only connection summary exposed to the selected Server UI. */
export interface DesktopServerConnections {
  servers: { id: string; name: string; url: string }[]
  currentServerUrl?: string
  connecting: boolean
}

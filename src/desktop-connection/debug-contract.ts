export interface DesktopDebugInfo {
  version: string
  buildStamp: string
  platform: string
  arch: string
  activeUrl?: string
  lastUrl?: string
  savedCount: number
  proxyMode: "system" | "direct" | "manual"
}

export interface DesktopDebugProbe {
  url: string
  name: string
  version: string
  serverId: string
  legacy: boolean
  latencyMs: number
}

export interface DesktopDebugAPI {
  readDebugInfo(): Promise<DesktopDebugInfo>
  probeDebugServer(url: string): Promise<DesktopDebugProbe>
  openDebugDevTools(): Promise<void>
}

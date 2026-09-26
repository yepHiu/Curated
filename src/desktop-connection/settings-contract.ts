/** 本地设置 IPC 的可编辑字段；与 Server Settings 合约分离。 */
export interface DesktopSettingsInput {
  proxyMode: "system" | "direct" | "manual"
  proxyUrl: string
  launchAtLogin: boolean
}
export interface DesktopSettingsState extends DesktopSettingsInput {
  loginSupported: boolean
  restartRequired: boolean
}
export interface DesktopSettingsAPI {
  readSettings(): Promise<DesktopSettingsState>
  saveSettings(value: DesktopSettingsInput): Promise<DesktopSettingsState>
}

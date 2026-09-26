import { mkdirSync, readFileSync, renameSync, writeFileSync } from "node:fs"
import path from "node:path"

export interface DesktopPreferences { proxyMode: "system" | "direct" | "manual"; proxyUrl: string }
export interface DesktopSettings extends DesktopPreferences { launchAtLogin: boolean; loginSupported: boolean; restartRequired: boolean }
export const defaultPreferences: DesktopPreferences = { proxyMode: "system", proxyUrl: "" }

/** 校验主进程收到的代理配置，不接受凭据、PAC、规则注入或额外路径。 */
export function validatePreferences(value: unknown): DesktopPreferences {
  if (!value || typeof value !== "object") throw new Error("无效的 Desktop 设置。")
  const input = value as Record<string, unknown>
  if (!["system", "direct", "manual"].includes(String(input.proxyMode)) || typeof input.proxyUrl !== "string" || input.proxyUrl.length > 2048) throw new Error("无效的代理设置。")
  const mode = input.proxyMode as DesktopPreferences["proxyMode"]
  const raw = input.proxyUrl.trim()
  if (!raw) {
    if (mode === "manual") throw new Error("请输入代理地址。")
    return { proxyMode: mode, proxyUrl: "" }
  }
  let url: URL
  try { url = new URL(raw) } catch { throw new Error("代理地址格式无效。") }
  if (!["http:", "https:", "socks5:"].includes(url.protocol) || !url.hostname || url.username || url.password || url.pathname !== "/" && url.pathname !== "" || url.search || url.hash || /[\s;,]/.test(raw)) throw new Error("代理仅支持无账号密码的 HTTP、HTTPS 或 SOCKS5 地址。")
  const port = url.port || (url.protocol === "https:" ? "443" : url.protocol === "http:" ? "80" : "1080")
  if (Number(port) < 1 || Number(port) > 65535) throw new Error("代理端口无效。")
  return { proxyMode: mode, proxyUrl: `${url.protocol}//${url.hostname}:${port}` }
}

/** Chromium 的代理配置，手动模式保留本机与常见私网直连。 */
export function proxyConfiguration(settings: DesktopPreferences): Electron.ProxyConfig {
  if (settings.proxyMode !== "manual") return { mode: settings.proxyMode }
  return { mode: "fixed_servers", proxyRules: settings.proxyUrl, proxyBypassRules: "<local>;127.0.0.1;[::1];10.0.0.0/8;172.16.0.0/12;192.168.0.0/16;169.254.0.0/16;[fc00::]/7;[fe80::]/10" }
}

/** 只存客户端网络偏好，自启动状态由操作系统提供。 */
export class DesktopPreferencesStore {
  private readonly file: string
  /** 使用已有 userData，避免与 Server 配置目录混淆。 */
  constructor(directory: string) { this.file = path.join(directory, "desktop-settings.json") }
  /** 缺文件使用系统代理；损坏文件不静默覆盖。 */
  read(): DesktopPreferences {
    try { return validatePreferences(JSON.parse(readFileSync(this.file, "utf8"))) }
    catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") return { ...defaultPreferences }
      throw error
    }
  }
  /** 原子替换已校验配置，不保存代理账号密码。 */
  write(value: DesktopPreferences): void {
    const settings = validatePreferences(value)
    mkdirSync(path.dirname(this.file), { recursive: true })
    writeFileSync(`${this.file}.tmp`, JSON.stringify(settings, null, 2), { mode: 0o600 })
    renameSync(`${this.file}.tmp`, this.file)
  }
}

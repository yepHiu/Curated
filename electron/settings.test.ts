import { mkdtempSync, rmSync, writeFileSync } from "node:fs"
import os from "node:os"
import path from "node:path"
import { afterEach, expect, it } from "vitest"
import { DesktopPreferencesStore, proxyConfiguration, validatePreferences } from "./settings"

const directories: string[] = []
// 每例清理独立目录，不碰现有客户端设置。
afterEach(() => { for (const directory of directories.splice(0)) rmSync(directory, { recursive: true, force: true }) })

// 拒绝可能被当成额外 Chromium 规则或认证信息的输入。
it("validates and normalizes manual proxies without allowing credentials or rule injection", () => {
  expect(validatePreferences({ proxyMode: "manual", proxyUrl: "http://localhost:7890/" }).proxyUrl).toBe("http://localhost:7890")
  expect(validatePreferences({ proxyMode: "manual", proxyUrl: "socks5://[::1]" }).proxyUrl).toBe("socks5://[::1]:1080")
  for (const proxyUrl of ["", "file:///tmp/a", "http://user:pass@host", "http://host/path", "http://host?pac=x", "http://host;direct://", "http://host:0"]) {
    expect(() => validatePreferences({ proxyMode: "manual", proxyUrl })).toThrow()
  }
})

// 禁用手动代理不能残留 fixed_servers，私网仍按配置直连。
it("maps all modes to explicit Chromium configuration", () => {
  expect(proxyConfiguration({ proxyMode: "direct", proxyUrl: "http://localhost:7890" })).toEqual({ mode: "direct" })
  expect(proxyConfiguration({ proxyMode: "system", proxyUrl: "" })).toEqual({ mode: "system" })
  const manual = proxyConfiguration({ proxyMode: "manual", proxyUrl: "socks5://localhost:1080" })
  expect(manual.mode).toBe("fixed_servers")
  expect(manual.proxyRules).toBe("socks5://localhost:1080")
  expect(manual.proxyBypassRules).toContain("192.168.0.0/16")
})

// 原子配置重读和损坏保护避免重启后默默丢失设置。
it("persists local preferences and surfaces damaged configuration", () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), "curated-settings-")); directories.push(directory)
  const store = new DesktopPreferencesStore(directory)
  expect(store.read()).toEqual({ proxyMode: "system", proxyUrl: "" })
  store.write({ proxyMode: "manual", proxyUrl: "http://localhost:7890" })
  expect(new DesktopPreferencesStore(directory).read().proxyUrl).toBe("http://localhost:7890")
  writeFileSync(path.join(directory, "desktop-settings.json"), "broken")
  expect(() => store.read()).toThrow()
})

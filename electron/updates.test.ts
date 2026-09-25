import { expect, it } from "vitest"
import { desktopUpdateAsset } from "./updates"
it("only opens a matching Desktop artifact from the official release", () => {
  const name = "Curated-Desktop-Setup-2.0.0-windows-x64.exe"
  const release = { tag_name: "v2.0.0", assets: [{name, browser_download_url:`https://github.com/yepHiu/Curated/releases/download/v2.0.0/${name}`}] }
  expect(desktopUpdateAsset(release,"win32","x64")?.version).toBe("2.0.0")
  expect(desktopUpdateAsset(release,"darwin","arm64")).toBeUndefined()
  expect(desktopUpdateAsset({...release, assets:[{name, browser_download_url:"https://untrusted.example/installer.exe"}]},"win32","x64")).toBeUndefined()
  expect(desktopUpdateAsset({...release, assets:[{...release.assets[0], name:name.replace("Desktop","Server")}]},"win32","x64")).toBeUndefined()
})

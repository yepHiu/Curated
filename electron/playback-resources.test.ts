import path from "node:path"
import { describe, expect, it } from "vitest"
import { playbackResources } from "./playback-resources"

const options = { directory: path.resolve("compiled"), appPath: path.resolve("app"), packaged: false }
const modulePath = path.join(options.directory, "native-player-macos.node")
const page = path.join(options.directory, "player/index.html")
const library = "/opt/homebrew/opt/mpv/lib/libmpv.dylib"
const exists = (files: string[]) => (file: string) => files.includes(file)
describe("native playback resources", () => {
  it("enables a Mac with a local libmpv, Cocoa module and bundled controls", () => {
    expect(playbackResources(options, "darwin", {}, exists([library, modulePath, page])))
      .toMatchObject({ available: true, executable: library, helper: modulePath, reason: undefined })
  })
  it("reports missing engine, module and page separately", () => {
    expect(playbackResources(options, "darwin", {}, exists([])).reason).toBe("engine")
    expect(playbackResources(options, "darwin", {}, exists([library])).reason).toBe("host")
    expect(playbackResources(options, "darwin", {}, exists([library, modulePath])).reason).toBe("page")
  })
  it("honors explicit absolute paths but never masks a broken or relative override", () => {
    const custom = path.resolve("custom/libmpv.dylib")
    const files = [library, custom, modulePath, page]
    expect(playbackResources(options, "darwin", { CURATED_NATIVE_LIBMPV: custom }, exists(files)).executable).toBe(custom)
    for (const override of ["relative/libmpv.dylib", "/missing/libmpv.dylib"]) {
      expect(playbackResources(options, "darwin", { CURATED_NATIVE_LIBMPV: override }, exists(files)))
        .toMatchObject({ available: false, reason: "engine" })
    }
  })
  it("keeps packaged Windows engines isolated from development overrides", () => {
    const executable = path.join(options.appPath, "native-player/mpv.exe")
    const helper = path.join(options.directory, "native-player-host.exe")
    expect(playbackResources({ ...options, packaged: true }, "win32", { CURATED_NATIVE_MPV: "/dev/mpv" }, exists([executable, helper, page])))
      .toMatchObject({ available: true, executable, helper })
    expect(playbackResources(options, "linux", {}, () => true).reason).toBe("platform")
  })
})

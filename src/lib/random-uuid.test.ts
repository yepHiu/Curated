import { afterEach, describe, expect, it, vi } from "vitest"
import { randomUuid } from "./random-uuid"

afterEach(() => vi.unstubAllGlobals())

describe("randomUuid", () => {
  it("preserves the native secure-context implementation", () => {
    const native = vi.fn(() => "native-uuid")
    vi.stubGlobal("crypto", { randomUUID: native })
    expect(randomUuid()).toBe("native-uuid")
    expect(native).toHaveBeenCalledOnce()
  })

  it("creates distinct UUID v4 IDs using Web Crypto on LAN HTTP", () => {
    const source = globalThis.crypto
    vi.stubGlobal("crypto", { getRandomValues: source.getRandomValues.bind(source) })
    const ids = Array.from({ length: 100 }, randomUuid)
    expect(new Set(ids).size).toBe(ids.length)
    for (const id of ids) expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  })
})

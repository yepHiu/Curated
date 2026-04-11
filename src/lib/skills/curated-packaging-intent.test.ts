import { describe, expect, it } from "vitest"
import { detectPackagingIntent } from "@/lib/skills/curated-packaging-intent"

describe("curated packaging intent", () => {
  it("detects publish mode from Mandarin input", () => {
    expect(detectPackagingIntent("\u6253\u751f\u4ea7\u5305")).toEqual({
      mode: "publish",
      baseChange: null,
    })
  })

  it("detects installer-only mode", () => {
    expect(detectPackagingIntent("\u6253\u5b89\u88c5\u5305")).toEqual({
      mode: "installer",
      baseChange: null,
    })
  })

  it("detects portable-only mode", () => {
    expect(detectPackagingIntent("\u6253\u4fbf\u643a\u5305")).toEqual({
      mode: "portable",
      baseChange: null,
    })
  })

  it("detects preview mode", () => {
    expect(detectPackagingIntent("\u9884\u89c8\u4e00\u4e0b")).toEqual({
      mode: "preview",
      baseChange: null,
    })
  })

  it("detects publish mode with minor base change", () => {
    expect(
      detectPackagingIntent("\u628a minor \u5347\u5230 2 \u518d\u6253\u751f\u4ea7\u5305")
    ).toEqual({
      mode: "publish",
      baseChange: { major: null, minor: 2 },
    })
  })

  it("prefers portable when installer is negated", () => {
    expect(
      detectPackagingIntent(
        "\u4e0d\u8981\u5b89\u88c5\u5305\u3001\u53ea\u6253\u4fbf\u643a\u5305"
      )
    ).toEqual({
      mode: "portable",
      baseChange: null,
    })
  })

  it("prefers installer when portable is negated", () => {
    expect(
      detectPackagingIntent("\u4e0d\u8981\u4fbf\u643a\u5305\u3001\u53ea\u6253\u5b89\u88c5\u5305")
    ).toEqual({
      mode: "installer",
      baseChange: null,
    })
  })

  it("handles more explicit portable negation", () => {
    expect(
      detectPackagingIntent("\u4e0d\u8981\u6253\u4fbf\u643a\u5305\u3001\u53ea\u6253\u5b89\u88c5\u5305")
    ).toEqual({
      mode: "installer",
      baseChange: null,
    })
  })

  it("understands English publish package", () => {
    expect(detectPackagingIntent("publish package")).toEqual({
      mode: "publish",
      baseChange: null,
    })
  })

  it("handles English base change before publish", () => {
    expect(detectPackagingIntent("bump minor to 2 then publish")).toEqual({
      mode: "publish",
      baseChange: { major: null, minor: 2 },
    })
  })
})

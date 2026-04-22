import { describe, expect, it } from "vitest"

import {
  isValidActorExternalLink,
  normalizeActorExternalLinkDraft,
} from "./actor-external-links"

describe("actor external links", () => {
  it("trims surrounding whitespace before save", () => {
    expect(normalizeActorExternalLinkDraft("  https://example.com/a  ")).toBe("https://example.com/a")
  })

  it("accepts absolute http and https urls", () => {
    expect(isValidActorExternalLink("http://example.com")).toBe(true)
    expect(isValidActorExternalLink("https://example.com/path")).toBe(true)
  })

  it("rejects relative and non-http urls", () => {
    expect(isValidActorExternalLink("/actor/1")).toBe(false)
    expect(isValidActorExternalLink("ftp://example.com")).toBe(false)
    expect(isValidActorExternalLink("javascript:alert(1)")).toBe(false)
  })
})
